package publicapi

import (
	"context"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-playground/validator/v10"
	db "github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/graphql/dataloader"
	"github.com/mutuals/go-mutuals/graphql/model"
	"github.com/mutuals/go-mutuals/service/persist"
	"github.com/mutuals/go-mutuals/service/persist/postgres"
	"github.com/mutuals/go-mutuals/validate"
)

type PoolAPI struct {
	repos     *postgres.Repositories
	queries   *db.Queries
	loaders   *dataloader.Loaders
	validator *validator.Validate
	ethClient *ethclient.Client
}

func (api PoolAPI) GetViewerPoolById(ctx context.Context, poolID persist.DBID) (*db.Pool, error) {
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"poolID": validate.WithTag(poolID, "required"),
	}); err != nil {
		return nil, err
	}

	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}

	pool, err := api.queries.GetPoolByUserID(ctx, db.GetPoolByUserIDParams{
		UserID: userID,
		PoolID: poolID,
	})
	if err != nil {
		return nil, err
	}

	return &pool, nil
}

func (api PoolAPI) GetPoolsByUserID(ctx context.Context, userID persist.DBID) ([]db.Pool, error) {
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"userID": validate.WithTag(userID, "required"),
	}); err != nil {
		return nil, err
	}

	pools, err := api.loaders.GetPoolsByUserIDBatch.Load(userID)
	if err != nil {
		return nil, err
	}

	return pools, nil
}

func (api PoolAPI) GetPoolById(ctx context.Context, poolID persist.DBID) (*db.Pool, error) {
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"poolID": validate.WithTag(poolID, "required"),
	}); err != nil {
		return nil, err
	}

	pool, err := api.loaders.GetPoolByIdBatch.Load(poolID)
	if err != nil {
		return nil, err
	}

	return &pool, nil
}

func (api PoolAPI) GetPoolsByIds(ctx context.Context, poolIDs []persist.DBID) ([]*db.Pool, []error) {
	poolThunk := func(poolID persist.DBID) func() (db.Pool, error) {
		if err := validate.ValidateFields(api.validator, validate.ValidationMap{
			"poolIDs": validate.WithTag(poolID, "required"),
		}); err != nil {
			return func() (db.Pool, error) { return db.Pool{}, err }
		}

		return api.loaders.GetPoolByIdBatch.LoadThunk(poolID)
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
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"name":        validate.WithTag(input.Name, "max=200"),
		"description": validate.WithTag(input.Description, "max=600"),
		"slug":        validate.WithTag(input.Slug, "max=100"),
	}); err != nil {
		return db.Pool{}, err
	}

	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return db.Pool{}, err
	}

	// Begin transaction
	tx, err := api.repos.BeginTx(ctx)
	if err != nil {
		return db.Pool{}, err
	}
	defer tx.Rollback(ctx)

	q := api.queries.WithTx(tx)

	poolID := persist.GenerateID()

	private := false
	if input.Private != nil {
		private = *input.Private
	}

	pool, err := q.CreatePool(ctx, db.CreatePoolParams{
		ID:          poolID,
		Name:        *input.Name,
		Description: *input.Description,
		Image:       *input.Image,
		Slug:        *input.Slug,
		Private:     private,
		OwnerID:     userID,
	})
	if err != nil {
		return db.Pool{}, err
	}

	// Create claims if provided
	if input.AddClaims != nil && len(input.AddClaims) > 0 {
		/*	for _, claimInput := range input.AddClaims {
			_, err := q.CreateClaims(ctx, db.CreateClaimsParams{
				ID:               persist.GenerateID(),
				PoolID:           poolID,
				RecipientAddress: claimInput.RecipientAddress.String(),
				StateID:          claimInput.StateId,
				StrategyID:       claimInput.StrategyId,
				Data:             claimInput.Data,
				ParentID:         "", // TODO: Handle nested claims if needed
			})
			if err != nil {
				return db.Pool{}, err
			}
		}*/
	}

	// Commit transaction
	err = tx.Commit(ctx)
	if err != nil {
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

	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return db.Pool{}, err
	}

	// Verify user owns the pool
	pool, err := api.queries.GetPoolByUserID(ctx, db.GetPoolByUserIDParams{
		UserID: userID,
		PoolID: id,
	})
	if err != nil {
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
	private := pool.Private
	if input.Private != nil {
		private = *input.Private
	}

	name := pool.Name
	if input.Name != nil {
		name = *input.Name
	}

	description := pool.Description
	if input.Description != nil {
		description = *input.Description
	}

	slug := pool.Slug
	if input.Slug != nil {
		slug = *input.Slug
	}

	image := pool.Image
	if input.Image != nil {
		image = *input.Image
	}

	donationBps := int32(0)
	if input.DonationBps != nil {
		donationBps = int32(*input.DonationBps)
	}

	updatedPool, err := q.UpdatePool(ctx, db.UpdatePoolParams{
		ID:          id,
		Name:        name,
		Description: description,
		Image:       image,
		Slug:        slug,
		Private:     private,
		DonationBps: donationBps,
		OwnerID:     userID,
	})
	if err != nil {
		return db.Pool{}, err
	}

	// Handle add claims
	if input.AddClaims != nil && len(input.AddClaims) > 0 {
		/*for _, claimInput := range input.AddClaims {
				_, err := q.CreateClaims(ctx, db.CreateClaimsParams{
					ID:               persist.GenerateID(),
					PoolID:           id,
					RecipientAddress: claimInput.RecipientAddress.String(),
					StateID:          claimInput.StateId,
					StrategyID:       claimInput.StrategyId,
					Data:             claimInput.Data,
					ParentID:         "", // TODO: Handle nested claims
				})
				if err != nil {
					return db.Pool{}, err
				}
		}*/
	}

	// Handle update claims
	if input.UpdateClaims != nil && len(input.UpdateClaims) > 0 {
		/*		for _, claimInput := range input.UpdateClaims {
					err := q.UpdateClaims(ctx, db.UpdateClaimsParams{
						ID:               claimInput.ClaimId,
						RecipientAddress: claimInput.RecipientAddress.String(),
						StateID:          claimInput.StateId,
						StrategyID:       claimInput.StrategyId,
						Data:             claimInput.Data,
					})
					if err != nil {
						return db.Pool{}, err
					}
				}
		*/
	}

	if input.RemoveClaims != nil && len(input.RemoveClaims) > 0 {
		/*		for _, claimID := range input.RemoveClaims {
					err := q.DeleteClaim(ctx, claimID)
					if err != nil {
						return db.Pool{}, err
					}
				}
		*/
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

	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return err
	}

	// Verify user owns the pool
	_, err = api.queries.GetPoolByUserID(ctx, db.GetPoolByUserIDParams{
		UserID: userID,
		PoolID: poolID,
	})
	if err != nil {
		return err
	}

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
