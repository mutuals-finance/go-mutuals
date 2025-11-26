package publicapi

import (
	"context"

	"github.com/mutuals/go-mutuals/db/gen/indexerdb"
	"github.com/mutuals/go-mutuals/service/persist/postgres"
	"github.com/mutuals/go-mutuals/validate"

	db "github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/service/multichain"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-playground/validator/v10"
	coreData "github.com/mutuals/go-mutuals/graphql/dataloader"
	indexerData "github.com/mutuals/go-mutuals/graphql/dataloader/indexerdb"
	"github.com/mutuals/go-mutuals/service/persist"
)

type WalletAPI struct {
	repos              *postgres.Repositories
	queries            *db.Queries
	indexerLoaders     *indexerData.Loaders
	coreLoaders        *coreData.Loaders
	validator          *validator.Validate
	ethClient          *ethclient.Client
	multichainProvider *multichain.Provider
}

func (api WalletAPI) GetAccountsByAddresses(ctx context.Context, addresses []string) ([]indexerdb.Account, error) {
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"addresses": validate.WithTag(addresses, "required"),
	}); err != nil {
		return nil, err
	}
	// TODO use Address type instead of string
	result, err := api.indexerLoaders.GetAccountsByAddressesBatch.Load(addresses)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (api WalletAPI) GetAccountById(ctx context.Context, id persist.DBID) (*indexerdb.Account, error) {
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"id": validate.WithTag(id, "required"),
	}); err != nil {
		return nil, err
	}

	result, err := api.indexerLoaders.GetAccountByIdBatch.Load(id)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (api WalletAPI) GetAccountByAddress(ctx context.Context, accountAddress persist.Address) (*indexerdb.Account, error) {
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"accountAddress": validate.WithTag(accountAddress, "required"),
	}); err != nil {
		return nil, err
	}

	account, err := api.indexerLoaders.GetAccountByAddressBatch.Load(accountAddress.String())
	if err != nil {
		return nil, err
	}

	return &account, nil
}
