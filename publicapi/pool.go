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
	// Validate
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
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"poolID": {poolID, "required"},
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
			"poolIDs": {poolID, "required"},
		}); err != nil {
			return func() (db.Pool, error) { return db.Pool{}, err }
		}

		return api.loaders.GetPoolByIdBatch.LoadThunk(poolID)
	}

	// A "thunk" will add this request to a batch, and then return a function that will block to fetch
	// data when called. By creating all of the thunks first (without invoking the functions they return),
	// we're setting up a batch that will eventually fetch all of these requests at the same time when
	// their functions are invoked. "LoadAll" would accomplish something similar, but wouldn't let us
	// validate each poolID parameter first.
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

func (api PoolAPI) UpdatePoolInfo(ctx context.Context, poolID persist.DBID, name, description, logoUrl *string) error {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"poolID":      {poolID, "required"},
		"name":        {name, "max=200"},
		"description": {description, "max=600"},
		"logoUrl":     {logoUrl, "max=200"},
	}); err != nil {
		return err
	}

	return nil
}

func (api PoolAPI) UpdatePool(ctx context.Context, id persist.DBID, input model.PoolUpdateInput) (db.Pool, error) {
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"id":          validate.WithTag(id, "required"),
		"name":        validate.WithTag(input.Name, "max=200"),
		"description": validate.WithTag(input.Description, "max=600"),
	}); err != nil {
		return db.Pool{}, err
	}

	tx, err := api.repos.BeginTx(ctx)
	if err != nil {
		return db.Pool{}, err
	}
	defer tx.Rollback(ctx)

	q := api.queries.WithTx(tx)

	pool, err := q.UpsertPool(ctx, db.UpsertPoolParams{
		ID:          id,
		Name:        *input.Name,
		Description: *input.Description,
		Logo:        "", // TODO *input.Logo
		OwnerID:     "", // TODO *input.OwnerID
		ContractID:  "", //  TODO *input.ContractID
	})

	if err != nil {
		return db.Pool{}, err
	}

	if len(input.AddClaims) > 0 {
		/*		allocationParams := processClaims(&id, input.AddClaims)

				_, err = q.UpsertClaims(ctx, allocationParams)
				if err != nil {
					return db.Pool{}, err
				}
		*/
	}
	if len(input.RemoveClaims) > 0 {
		/*		allocationParams := processClaims(&id, input.AddClaims)

				_, err = q.DeleteClaims(ctx, allocationParams)
				if err != nil {
					return db.Pool{}, err
				}
		*/
	}

	err = tx.Commit(ctx)
	if err != nil {
		return db.Pool{}, err
	}

	return pool, nil
}

/*func processClaims(poolID *persist.DBID, a []*model.PoolAllocationInput) (allocationParams db.UpsertClaimsParams) {
	allocationParams.PoolID = *poolID

	var traverse func(node *model.PoolAllocationInput, path string)
	traverse = func(node *model.PoolAllocationInput, parentPath string) {
		id := node.ID
		if id == nil {
			id = util.ToPointer(persist.GenerateID())
		}
		// Determine the label: use RecipientAddress if not empty, otherwise use id
		label := node.RecipientAddress.String()
		if label == "" {
			label = id.String()
		}

		// Construct the current path
		path := parentPath
		if parentPath == "" {
			path = label
		} else {
			path = parentPath + "." + label
		}

		allocationParams.ID = append(allocationParams.ID, id.String())
		allocationParams.RecipientAddress = append(allocationParams.RecipientAddress, node.RecipientAddress.String())
		allocationParams.StrategyID = append(allocationParams.StrategyID, "") // TODO
		allocationParams.StateID = append(allocationParams.StrategyID, "")    // TODO
		allocationParams.Deleted = append(allocationParams.Deleted, false)    // TODO
		allocationParams.Value = append(allocationParams.Value, persist.MustHexString(node.Value.String()).String())
		allocationParams.Label = append(allocationParams.Label, label)
		allocationParams.Path = append(allocationParams.Path, path)

		// Recursively process children
		for _, child := range node.Children {
			traverse(child, path)
		}
	}

	// Process each top-level node
	for _, allocation := range a {
		traverse(allocation, "")
	}

	return allocationParams
}
*/
