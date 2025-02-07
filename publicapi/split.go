package publicapi

import (
	"context"
	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/graphql/dataloader"
	"github.com/SplitFi/go-splitfi/graphql/model"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/persist/postgres"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/SplitFi/go-splitfi/validate"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-playground/validator/v10"
)

type SplitAPI struct {
	repos     *postgres.Repositories
	queries   *db.Queries
	loaders   *dataloader.Loaders
	validator *validator.Validate
	ethClient *ethclient.Client
}

func (api SplitAPI) CreateSplit(ctx context.Context, name, description, logoUrl *string) (db.Split, error) {

	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"name":        {name, "max=200"},
		"description": {description, "max=600"},
		"logoUrl":     {logoUrl, "max=200"},
	}); err != nil {
		return db.Split{}, err
	}

	split, err := api.queries.CreateSplit(ctx, db.CreateSplitParams{
		ID:          persist.GenerateID(),
		Name:        util.FromPointer(name),
		Description: util.FromPointer(description),
	})
	if err != nil {
		return db.Split{}, err
	}

	return split, nil
}

func (api SplitAPI) PublishSplit(ctx context.Context, update model.PublishSplitInput) error {

	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"splitID": {update.SplitID, "required"},
		"editID":  {update.EditID, "required"},
	}); err != nil {
		return err
	}

	//err := publishEventGroup(ctx, update.EditID, persist.ActionSplitUpdated, update.Caption)
	//if err != nil {
	//	return err
	//}

	return nil
}

func (api SplitAPI) GetViewerSplitById(ctx context.Context, splitID persist.DBID) (*db.Split, error) {

	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"splitID": validate.WithTag(splitID, "required"),
	}); err != nil {
		return nil, err
	}

	userID, err := getAuthenticatedUserID(ctx)

	if err != nil {
		return nil, err
	}

	split, err := api.queries.GetSplitByUserID(ctx, db.GetSplitByUserIDParams{
		UserID:  userID,
		SplitID: splitID,
	})
	if err != nil {
		return nil, err
	}

	return &split, nil
}

func (api SplitAPI) GetSplitsByUserID(ctx context.Context, userID persist.DBID) ([]db.Split, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"userID": validate.WithTag(userID, "required"),
	}); err != nil {
		return nil, err
	}

	splits, err := api.loaders.GetSplitsByUserIDBatch.Load(userID)
	if err != nil {
		return nil, err
	}

	return splits, nil
}

func (api SplitAPI) GetSplitById(ctx context.Context, splitID persist.DBID) (*db.Split, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"splitID": {splitID, "required"},
	}); err != nil {
		return nil, err
	}

	split, err := api.loaders.GetSplitByIdBatch.Load(splitID)
	if err != nil {
		return nil, err
	}

	return &split, nil
}

func (api SplitAPI) GetSplitsByIds(ctx context.Context, splitIDs []persist.DBID) ([]*db.Split, []error) {
	splitThunk := func(splitID persist.DBID) func() (db.Split, error) {
		if err := validate.ValidateFields(api.validator, validate.ValidationMap{
			"splitIDs": {splitID, "required"},
		}); err != nil {
			return func() (db.Split, error) { return db.Split{}, err }
		}

		return api.loaders.GetSplitByIdBatch.LoadThunk(splitID)
	}

	// A "thunk" will add this request to a batch, and then return a function that will block to fetch
	// data when called. By creating all of the thunks first (without invoking the functions they return),
	// we're setting up a batch that will eventually fetch all of these requests at the same time when
	// their functions are invoked. "LoadAll" would accomplish something similar, but wouldn't let us
	// validate each splitID parameter first.
	thunks := make([]func() (db.Split, error), len(splitIDs))

	for i, splitID := range splitIDs {
		thunks[i] = splitThunk(splitID)
	}

	splits := make([]*db.Split, len(splitIDs))
	errors := make([]error, len(splitIDs))

	for i := range splitIDs {
		split, err := thunks[i]()
		if err == nil {
			splits[i] = &split
		} else {
			errors[i] = err
		}
	}

	return splits, errors
}

func (api SplitAPI) GetSplitByChainAddress(ctx context.Context, chainAddress persist.ChainAddress) (*db.Split, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"chainAddress": {chainAddress, "required"},
	}); err != nil {
		return nil, err
	}

	split, err := api.loaders.GetSplitByChainAddressBatch.Load(db.GetSplitByChainAddressBatchParams{
		Address: chainAddress.Address(),
		Chain:   chainAddress.Chain(),
	})
	if err != nil {
		return nil, err
	}

	return &split, nil
}

func (api SplitAPI) UpdateSplitInfo(ctx context.Context, splitID persist.DBID, name, description, logoUrl *string) error {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"splitID":     {splitID, "required"},
		"name":        {name, "max=200"},
		"description": {description, "max=600"},
		"logoUrl":     {logoUrl, "max=200"},
	}); err != nil {
		return err
	}

	return nil
}

/*
	func (api SplitAPI) UpdateSplitHidden(ctx context.Context, splitID persist.DBID, hidden bool) (db.Split, error) {
		// Validate
		if err := validate.ValidateFields(api.validator, validate.ValidationMap{
			"splitID": validate.WithTag(splitID, "required"),
		}); err != nil {
			return db.Split{}, err
		}

		split, err := api.queries.UpdateSplitHidden(ctx, db.UpdateSplitHiddenParams{
			ID:     splitID,
			Hidden: hidden,
		})
		if err != nil {
			return db.Split{}, err
		}

		return split, nil
	}
*/

func (api SplitAPI) UpsertSplit(ctx context.Context, input model.UpsertSplitInput) (db.Split, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"name":        validate.WithTag(input.Name, "max=200"),
		"description": validate.WithTag(input.Description, "max=600"),
	}); err != nil {
		return db.Split{}, err
	}

	splitID := input.SplitID
	if splitID == nil {
		splitID = util.ToPointer(persist.GenerateID())
	}

	tx, err := api.repos.BeginTx(ctx)
	if err != nil {
		return db.Split{}, err
	}
	defer tx.Rollback(ctx)

	q := api.queries.WithTx(tx)

	split, err := api.queries.UpsertSplit(ctx, db.UpsertSplitParams{
		ID:          *splitID,
		Name:        *input.Name,
		Description: *input.Description,
		Status:      int32(persist.SplitStatusDraft),
	})

	if err != nil {
		return db.Split{}, err
	}

	if len(input.Allocations) > 0 {
		allocationParams, aggregationParams := processAllocations(splitID, input.Allocations)

		_, err = q.UpsertSplitAllocations(ctx, allocationParams)
		if err != nil {
			return db.Split{}, err
		}

		_, err = q.UpsertSplitAggregatedAllocations(ctx, aggregationParams)
		if err != nil {
			return db.Split{}, err
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return db.Split{}, err
	}

	return split, nil
}

func processAllocations(splitID *persist.DBID, a []*model.SplitAllocationInput) (allocationParams db.UpsertSplitAllocationsParams, aggregationParams db.UpsertSplitAggregatedAllocationsParams) {
	recipientToAllocations := make(map[persist.Address][]*model.SplitAllocationInput)

	allocationParams.SplitID = *splitID
	aggregationParams.SplitID = *splitID

	var traverse func(node *model.SplitAllocationInput, path string)
	traverse = func(node *model.SplitAllocationInput, parentPath string) {
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

func (api SplitAPI) GetAllocationById(ctx context.Context, allocationID persist.DBID) (*db.Allocation, error) {
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

func (api SplitAPI) GetAllocationAggregationById(ctx context.Context, allocationAggregationID persist.DBID) (*db.AllocationAggregation, error) {
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
