package publicapi

import (
	"github.com/SplitFi/go-splitfi/service/persist/postgres"

	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/graphql/dataloader"
	"github.com/SplitFi/go-splitfi/service/multichain"
	"github.com/SplitFi/go-splitfi/service/throttle"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-playground/validator/v10"
)

type AssetAPI struct {
	repos              *postgres.Repositories
	queries            *db.Queries
	loaders            *dataloader.Loaders
	validator          *validator.Validate
	ethClient          *ethclient.Client
	multichainProvider *multichain.Provider
	throttler          *throttle.Locker
}

/*
TODO implement
func (api AssetAPI) GetAssetsByChainAddress(ctx context.Context, walletID persist.DBID) ([]db.Token, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"walletID": {walletID, "required"},
	}); err != nil {
		return nil, err
	}

	tokens, err := api.loaders.AssetsByChainAddress.Load(walletID)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}
*/
