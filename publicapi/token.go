package publicapi

import (
	"context"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-playground/validator/v10"
	db "github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/graphql/dataloader"
	"github.com/mutuals/go-mutuals/service/persist"
	"github.com/mutuals/go-mutuals/service/persist/postgres"
	"github.com/mutuals/go-mutuals/service/throttle"
	"github.com/mutuals/go-mutuals/validate"
)

type AssetAPI struct {
	repos     *postgres.Repositories
	queries   *db.Queries
	loaders   *dataloader.Loaders
	validator *validator.Validate
	ethClient *ethclient.Client
	throttler *throttle.Locker
}

func (api AssetAPI) GetAssetsByOwnerAddressPaginate(ctx context.Context, ownerAddress persist.Address, before, after *string, first, last *int) ([]any, PageInfo, error) {

	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"ownerAddress": validate.WithTag(ownerAddress, "required"),
	}); err != nil {
		return nil, PageInfo{}, err
	}

	if err := validatePaginationParams(api.validator, first, last); err != nil {
		return nil, PageInfo{}, err
	}

	// TODO: implement external call using ownerAddress
	return nil, PageInfo{}, nil
}
