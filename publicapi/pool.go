package publicapi

import (
	"context"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-playground/validator/v10"
	db "github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/graphql/dataloader"
	"github.com/mutuals/go-mutuals/graphql/model"
	claimsService "github.com/mutuals/go-mutuals/service/claims"
	"github.com/mutuals/go-mutuals/service/logger"
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
	viewerAccounts, err := getAuthenticatedLinkedAccounts(ctx)
	logger.For(ctx).Infof("VIEWER: %s, %v", viewerId, viewerAccounts)

	if err != nil {
		return nil, err
	}

	params := db.GetPoolsByAddressesOrOwnerBatchParams{
		OwnerID: viewerId,
	}

	for _, account := range viewerAccounts {
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

	pool, err := poolService.CreatePool(ctx, q, input.Name, input.Description, util.FromPointer(input.Image), input.Slug, util.FromPointer(input.Private))
	if err != nil {
		return db.Pool{}, err
	}

	if len(input.AddClaims) > 0 {
		var claims []allocation.Claim

		for _, c := range input.AddClaims {
			claims = append(claims, allocation.Claim{
				Label:            c.Label,
				RecipientAddress: c.RecipientAddress,
				Data:             c.Data,
				ParentID:         c.Parent,
				ChildIDs:         c.Children,
				StateID:          c.StateID,
				StrategyID:       c.StrategyID,
			})
		}

		_, err := claimsService.CreateClaims(ctx, q, pool.ID, claims)
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
	updatedPool, err := poolService.UpdatePool(ctx, q, id, input.Name, input.Description, input.Image, input.Slug, input.Private)
	if err != nil {
		return db.Pool{}, err
	}

	// Handle add claims
	if input.AddClaims != nil && len(input.AddClaims) > 0 {
		// TODO: Implement claim additions
	}

	// Handle update claims
	if input.UpdateClaims != nil && len(input.UpdateClaims) > 0 {
		// TODO: Implement claim updates
	}

	if input.RemoveClaims != nil && len(input.RemoveClaims) > 0 {
		// TODO: Implement claim removals
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
