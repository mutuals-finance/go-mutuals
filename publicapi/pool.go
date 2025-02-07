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

func (api PoolAPI) CreatePool(ctx context.Context, name, description, logoUrl *string) (db.Pool, error) {

	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"name":        {name, "max=200"},
		"description": {description, "max=600"},
		"logoUrl":     {logoUrl, "max=200"},
	}); err != nil {
		return db.Pool{}, err
	}

	pool, err := api.queries.CreatePool(ctx, db.CreatePoolParams{
		ID:          persist.GenerateID(),
		Name:        util.FromPointer(name),
		Description: util.FromPointer(description),
	})
	if err != nil {
		return db.Pool{}, err
	}

	return pool, nil
}

func (api PoolAPI) PublishPool(ctx context.Context, update model.PublishPoolInput) error {

	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"poolID": {update.PoolID, "required"},
		"editID": {update.EditID, "required"},
	}); err != nil {
		return err
	}

	//err := publishEventGroup(ctx, update.EditID, persist.ActionPoolUpdated, update.Caption)
	//if err != nil {
	//	return err
	//}

	return nil
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

func (api PoolAPI) GetPoolByChainAddress(ctx context.Context, chainAddress persist.ChainAddress) (*db.Pool, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"chainAddress": {chainAddress, "required"},
	}); err != nil {
		return nil, err
	}

	pool, err := api.loaders.GetPoolByChainAddressBatch.Load(db.GetPoolByChainAddressBatchParams{
		Address: chainAddress.Address(),
		Chain:   chainAddress.Chain(),
	})
	if err != nil {
		return nil, err
	}

	return &pool, nil
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

/*
	func (api PoolAPI) UpdatePoolHidden(ctx context.Context, poolID persist.DBID, hidden bool) (db.Pool, error) {
		// Validate
		if err := validate.ValidateFields(api.validator, validate.ValidationMap{
			"poolID": validate.WithTag(poolID, "required"),
		}); err != nil {
			return db.Pool{}, err
		}

		pool, err := api.queries.UpdatePoolHidden(ctx, db.UpdatePoolHiddenParams{
			ID:     poolID,
			Hidden: hidden,
		})
		if err != nil {
			return db.Pool{}, err
		}

		return pool, nil
	}
*/

func (api PoolAPI) UpsertPool(ctx context.Context, input model.UpsertPoolInput) (db.Pool, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"name":        validate.WithTag(input.Name, "max=200"),
		"description": validate.WithTag(input.Description, "max=600"),
	}); err != nil {
		return db.Pool{}, err
	}

	poolID := input.PoolID
	if poolID == nil {
		poolID = util.ToPointer(persist.GenerateID())
	}

	tx, err := api.repos.BeginTx(ctx)
	if err != nil {
		return db.Pool{}, err
	}
	defer tx.Rollback(ctx)

	q := api.queries.WithTx(tx)

	pool, err := api.queries.UpsertPool(ctx, db.UpsertPoolParams{
		ID:          *poolID,
		Name:        *input.Name,
		Description: *input.Description,
		Status:      int32(persist.PoolStatusDraft),
	})

	if err != nil {
		return db.Pool{}, err
	}

	if len(input.Allocations) > 0 {
		allocationParams, aggregationParams := processAllocations(poolID, input.Allocations)

		_, err = q.UpsertPoolAllocations(ctx, allocationParams)
		if err != nil {
			return db.Pool{}, err
		}

		_, err = q.UpsertPoolAggregatedAllocations(ctx, aggregationParams)
		if err != nil {
			return db.Pool{}, err
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return db.Pool{}, err
	}

	return pool, nil
}

func processAllocations(poolID *persist.DBID, a []*model.PoolAllocationInput) (allocationParams db.UpsertPoolAllocationsParams, aggregationParams db.UpsertPoolAggregatedAllocationsParams) {
	recipientToAllocations := make(map[persist.Address][]*model.PoolAllocationInput)

	allocationParams.PoolID = *poolID
	aggregationParams.PoolID = *poolID

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

		allocationParams.Ids = append(allocationParams.Ids, id.String())
		allocationParams.RecipientAddress = append(allocationParams.RecipientAddress, node.RecipientAddress.String())
		allocationParams.RecipientType = append(allocationParams.RecipientType, int32(node.RecipientType[0]))
		allocationParams.CalculationType = append(allocationParams.CalculationType, int32(node.CalculationType[0]))
		allocationParams.Value = append(allocationParams.Value, persist.MustHexString(node.Value.String()).String())
		allocationParams.Expression = append(allocationParams.Expression, "")
		allocationParams.Label = append(allocationParams.Label, label)
		allocationParams.Path = append(allocationParams.Path, path)

		if node.RecipientType[0] == persist.RecipientTypeDefaultItem && node.RecipientAddress != nil {
			recipientToAllocations[*node.RecipientAddress] = append(recipientToAllocations[*node.RecipientAddress], node)
		}

		// Recursively process children
		for _, child := range node.Children {
			traverse(child, path)
		}
	}

	// Process each top-level node
	for _, allocation := range a {
		traverse(allocation, "")
	}

	// Calculate aggregation
	for recipient, inputs := range recipientToAllocations {
		expression := ""
		for _, input := range inputs {
			expression = expression + input.Value.String()
		}
		aggregationParams.ID = append(aggregationParams.RecipientAddress, persist.GenerateID().String())
		aggregationParams.RecipientAddress = append(aggregationParams.RecipientAddress, recipient.String())
		aggregationParams.Expression = append(aggregationParams.Expression, expression)
	}

	return allocationParams, aggregationParams
}

func (api PoolAPI) GetAllocationById(ctx context.Context, allocationID persist.DBID) (*db.Allocation, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"allocationID": {allocationID, "required"},
	}); err != nil {
		return nil, err
	}

	allocation, err := api.loaders.GetAllocationByIdBatch.Load(allocationID)
	if err != nil {
		return nil, err
	}

	return &allocation, nil
}

func (api PoolAPI) GetAllocationAggregationById(ctx context.Context, allocationAggregationID persist.DBID) (*db.AllocationAggregation, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"allocationAggregationID": {allocationAggregationID, "required"},
	}); err != nil {
		return nil, err
	}

	allocationAggregation, err := api.loaders.GetAllocationAggregationByIdBatch.Load(allocationAggregationID)
	if err != nil {
		return nil, err
	}

	return &allocationAggregation, nil
}
