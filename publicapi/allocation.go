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

type AllocationAPI struct {
	repos     *postgres.Repositories
	queries   *db.Queries
	loaders   *dataloader.Loaders
	validator *validator.Validate
	ethClient *ethclient.Client
}

func (api PoolAPI) CreateAllocation(ctx context.Context, name, description, logoUrl *string) (db.Allocation, error) {

	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"name":        {name, "max=200"},
		"description": {description, "max=600"},
		"logoUrl":     {logoUrl, "max=200"},
	}); err != nil {
		return db.Allocation{}, err
	}

	pool, err := api.queries.CreatePool(ctx, db.CreatePoolParams{
		ID:          persist.GenerateID(),
		Name:        util.FromPointer(name),
		Description: util.FromPointer(description),
	})
	if err != nil {
		return db.Allocation{}, err
	}

	return db.Allocation{PoolID: pool.ID}, nil
}

func (api PoolAPI) PublishAllocation(ctx context.Context, update model.PublishPoolInput) error {

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
