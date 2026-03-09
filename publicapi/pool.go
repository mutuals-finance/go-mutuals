package publicapi

import (
	"context"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-playground/validator/v10"
	db "github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/graphql/dataloader"
	"github.com/mutuals/go-mutuals/graphql/model"
	"github.com/mutuals/go-mutuals/service/auth/privy"
	claimService "github.com/mutuals/go-mutuals/service/claim"
	"github.com/mutuals/go-mutuals/service/persist"
	"github.com/mutuals/go-mutuals/service/persist/allocation"
	"github.com/mutuals/go-mutuals/service/persist/postgres"
	poolService "github.com/mutuals/go-mutuals/service/pool"
	"github.com/mutuals/go-mutuals/util"
	"github.com/mutuals/go-mutuals/validate"
)

type PoolAPI struct {
	repos     *postgres.Repositories
	queries   *db.Queries
	loaders   *dataloader.Loaders
	validator *validator.Validate
	ethClient *ethclient.Client
}

func (api PoolAPI) GetPool(ctx context.Context, poolID *persist.DBID, slug *string, contractID *persist.DBID) (*db.Pool, error) {
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"poolID | slug | contractID": validate.WithTag([]any{poolID, slug, contractID}, "at_least_one"),
	}); err != nil {
		return nil, err
	}

	pool, err := api.loaders.GetPoolBatch.Load(db.GetPoolBatchParams{
		PoolID:     persist.DBIDPtrToNullStr(poolID),
		Slug:       persist.StrPtrToNullStr(slug),
		ContractID: persist.DBIDPtrToNullStr(contractID),
	})
	if err != nil {
		return nil, err
	}

	return &pool, nil
}

func (api PoolAPI) GetViewerPools(ctx context.Context) (*[]db.Pool, error) {
	viewerId, err := getAuthenticatedUserId(ctx)
	if err != nil {
		return nil, err
	}

	viewerAccounts, err := getAuthenticatedLinkedAccounts(ctx)
	if err != nil {
		return nil, err
	}

	return api.GetPoolsByUserIdAndLinkedAccounts(ctx, viewerId, viewerAccounts)
}

func (api PoolAPI) GetPoolsByUserIdAndLinkedAccounts(ctx context.Context, userId persist.DBID, linkedAccounts []privy.LinkedAccount) (*[]db.Pool, error) {
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"userId": validate.WithTag(userId, "required"),
	}); err != nil {
		return nil, err
	}

	params := db.GetPoolsByAddressesOrOwnerBatchParams{
		OwnerID: userId,
	}

	for _, account := range linkedAccounts {
		params.Addresses = append(params.Addresses, account.Address)
	}

	pools, err := api.loaders.GetPoolsByAddressesOrOwnerBatch.Load(params)
	if err != nil {
		return nil, err
	}

	return &pools, nil
}

func (api PoolAPI) GetPoolsByIds(ctx context.Context, poolIDs []persist.DBID) ([]*db.Pool, []error) {
	poolThunk := func(poolID persist.DBID) func() (db.Pool, error) {
		if err := validate.ValidateFields(api.validator, validate.ValidationMap{
			"poolIDs": validate.WithTag(poolID, "required"),
		}); err != nil {
			return func() (db.Pool, error) { return db.Pool{}, err }
		}

		return api.loaders.GetPoolBatch.LoadThunk(db.GetPoolBatchParams{
			PoolID: persist.DBIDToNullStr(poolID),
		})
	}

	thunks := make([]func() (db.Pool, error), len(poolIDs))
	for i, poolID := range poolIDs {
		thunks[i] = poolThunk(poolID)
	}

	pools := make([]*db.Pool, len(poolIDs))
	errors := make([]error, len(poolIDs))

	for i := range poolIDs {
		pool, err := thunks[i]()
		if err == nil {
			pools[i] = &pool
		} else {
			errors[i] = err
		}
	}

	return pools, errors
}

// CreatePool creates a new pool with optional claims
func (api PoolAPI) CreatePool(ctx context.Context, input model.PoolCreateInput) (db.Pool, error) {
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"name":        validate.WithTag(input.Name, "max=200"),
		"description": validate.WithTag(input.Description, "max=600"),
		"slug":        validate.WithTag(input.Slug, "max=100"),
	}); err != nil {
		return db.Pool{}, err
	}

	tx, err := api.repos.BeginTx(ctx)
	if err != nil {
		return db.Pool{}, err
	}
	defer tx.Rollback(ctx)

	q := api.queries.WithTx(tx)

	pool, err := poolService.CreatePool(ctx, q, poolService.CreatePoolInput{
		Name:        input.Name,
		Description: input.Description,
		Image:       util.FromPointer(input.Image),
		Slug:        input.Slug,
		Private:     util.FromPointer(input.Private),
	})
	if err != nil {
		return db.Pool{}, err
	}

	if len(input.AddClaims) > 0 {
		claims := make([]allocation.Claim, len(input.AddClaims))
		for i, c := range input.AddClaims {
			claims[i] = allocation.Claim{
				Label:          c.Label,
				Data:           c.Data,
				ParentID:       c.Parent,
				ChildIDs:       c.Children,
				ValidationID:   c.ValidationID,
				DistributionID: c.DistributionID,
			}
		}

		_, err = claimService.CreateClaimsWithAllocationTreeForPool(ctx, q, claimService.CreateClaimsWithAllocationTreeForPoolInput{
			PoolID: pool.ID,
			Claims: claims,
		})
		if err != nil {
			return db.Pool{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return db.Pool{}, err
	}

	return pool, nil
}

// UpdatePool updates an existing pool and manages claims
func (api PoolAPI) UpdatePool(ctx context.Context, id persist.DBID, input model.PoolUpdateInput) (db.Pool, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"id":          validate.WithTag(id, "required"),
		"name":        validate.WithTag(input.Name, "max=200"),
		"description": validate.WithTag(input.Description, "max=600"),
		"slug":        validate.WithTag(input.Slug, "max=100"),
	}); err != nil {
		return db.Pool{}, err
	}

	// Begin transaction
	tx, err := api.repos.BeginTx(ctx)
	if err != nil {
		return db.Pool{}, err
	}
	defer tx.Rollback(ctx)

	q := api.queries.WithTx(tx)

	// Update pool basic info
	updatedPool, err := poolService.UpdatePool(ctx, q, poolService.UpdatePoolInput{
		ID:          id,
		Name:        input.Name,
		Description: input.Description,
		Image:       input.Image,
		Slug:        input.Slug,
		Private:     input.Private,
	})
	if err != nil {
		return db.Pool{}, err
	}

	if input.AddClaims != nil && len(input.AddClaims) > 0 {
		claims := make([]allocation.Claim, len(input.AddClaims))
		for i, c := range input.AddClaims {
			claims[i] = allocation.Claim{
				Label:          c.Label,
				Data:           c.Data,
				ParentID:       c.Parent,
				ChildIDs:       c.Children,
				ValidationID:   c.ValidationID,
				DistributionID: c.DistributionID,
			}
		}

		_, err := claimService.CreateClaimsWithAllocationTreeForPool(ctx, q, claimService.CreateClaimsWithAllocationTreeForPoolInput{
			PoolID: id,
			Claims: claims,
		})
		if err != nil {
			return db.Pool{}, err
		}
	}

	if input.UpdateClaims != nil && len(input.UpdateClaims) > 0 {
		updateInputs := make([]claimService.UpdateClaimInput, len(input.UpdateClaims))
		for i, c := range input.UpdateClaims {
			var children []persist.DBID
			if c.Children != nil {
				children = make([]persist.DBID, len(c.Children))
				for j, child := range c.Children {
					children[j] = child.DBID()
				}
			}

			updateInputs[i] = claimService.UpdateClaimInput{
				ClaimID:        c.ClaimID.DBID(),
				Data:           persist.JSONToJSONB(c.Data),
				Parent:         persist.DBIDPtrToSQLNullString(c.Parent.DBIDPtr()),
				Children:       persist.DBIDSliceToStringSlice(children),
				ValidationID:   persist.DBID(*c.ValidationID),
				DistributionID: persist.DBID(*c.DistributionID),
			}
		}

		_, err := claimService.BulkUpdateClaimsForPool(ctx, q, claimService.BulkUpdateClaimsForPoolInput{
			PoolID: id,
			Claims: updateInputs,
		})
		if err != nil {
			return db.Pool{}, err
		}
	}

	if input.RemoveClaims != nil && len(input.RemoveClaims) > 0 {
		claimIDs := make([]persist.DBID, len(input.RemoveClaims))
		for i, claimID := range input.RemoveClaims {
			claimIDs[i] = claimID.DBID()
		}

		_, err := claimService.BulkDeleteClaimsForPool(ctx, q, claimService.BulkDeleteClaimsForPoolInput{
			PoolID:   id,
			ClaimIDs: claimIDs,
		})
		if err != nil {
			return db.Pool{}, err
		}
	}

	// Commit transaction
	err = tx.Commit(ctx)
	if err != nil {
		return db.Pool{}, err
	}

	return updatedPool, nil
}

// DeletePool deletes a pool
func (api PoolAPI) DeletePool(ctx context.Context, poolID persist.DBID) error {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"poolID": validate.WithTag(poolID, "required"),
	}); err != nil {
		return err
	}

	/*
		userId, err := getAuthenticatedUserId(ctx)
		if err != nil {
			return err
		}

		// Verify user owns the pool
		_, err = api.queries.GetPoolByUserId(ctx, db.GetPoolByUserIdParams{
			UserId: userId,
			PoolID: poolID,
		})
		if err != nil {
			return err
		}
	*/
	// Begin transaction
	tx, err := api.repos.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	/*q := api.queries.WithTx(tx)

	// Delete all claims associated with the pool
	err = q.DeleteClaimsByPoolID(ctx, poolID)
	if err != nil {
		return err
	}

	// Delete the pool
	err = q.DeletePool(ctx, poolID)
	if err != nil {
		return err
	}
	*/
	// Commit transaction
	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}
