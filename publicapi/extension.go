package publicapi

import (
	"context"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-playground/validator/v10"
	db "github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/graphql/dataloader"
	"github.com/mutuals/go-mutuals/service/multichain"
	"github.com/mutuals/go-mutuals/service/persist"
	"github.com/mutuals/go-mutuals/service/persist/postgres"
	"github.com/mutuals/go-mutuals/validate"
)

type ExtensionAPI struct {
	repos              *postgres.Repositories
	queries            *db.Queries
	loaders            *dataloader.Loaders
	validator          *validator.Validate
	ethClient          *ethclient.Client
	multichainProvider *multichain.Provider
}

func (api WalletAPI) GetExtensionByID(ctx context.Context, extensionID persist.DBID) (*db.Extension, error) {
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"extensionID": validate.WithTag(extensionID, "required"),
	}); err != nil {
		return nil, err
	}

	address, err := api.loaders.GetExtensionByIDBatch.Load(extensionID)
	if err != nil {
		return nil, err
	}

	return &address, nil
}

func (api WalletAPI) GetExtensions(ctx context.Context) ([]db.Extension, error) {
	a, err := api.loaders.GetExtensionsBatch.Load()
	if err != nil {
		return nil, err
	}

	return a, nil
}
