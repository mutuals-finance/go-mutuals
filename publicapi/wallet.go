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

func (api WalletAPI) GetUserAccountByID(ctx context.Context, userAccountID persist.DBID) (*db.UserAccount, error) {
	// Validate

	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"userAccountID": validate.WithTag(userAccountID, "required"),
	}); err != nil {
		return nil, err
	}

	account, err := api.coreLoaders.GetUserAccountByIdBatch.Load(userAccountID)
	if err != nil {
		return nil, err
	}

	return &account, nil
}

func (api WalletAPI) GetUserAccountsByUserID(ctx context.Context, userID persist.DBID) ([]db.UserAccount, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"userID": validate.WithTag(userID, "required"),
	}); err != nil {
		return nil, err
	}

	userAccounts, err := api.coreLoaders.GetUserAccountsByUserIdBatch.Load(userID)
	if err != nil {
		return nil, err
	}

	return userAccounts, nil
}
func (api WalletAPI) GetAccountsByAddresses(ctx context.Context, addresses []string) ([]indexerdb.Account, error) {
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"addresses": validate.WithTag(addresses, "required"),
	}); err != nil {
		return nil, err
	}
	// TODO use Address type instead of string
	accounts, err := api.indexerLoaders.GetAccountsByAddressesBatch.Load(addresses)
	if err != nil {
		return nil, err
	}

	return accounts, nil
}

func (api WalletAPI) GetAccountByID(ctx context.Context, accountID persist.DBID) (*indexerdb.Account, error) {
	// Validate

	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"walletID": validate.WithTag(accountID, "required"),
	}); err != nil {
		return nil, err
	}

	account, err := api.indexerLoaders.GetAccountByIdBatch.Load(accountID)
	if err != nil {
		return nil, err
	}

	return &account, nil
}

func (api WalletAPI) GetAccountByAddress(ctx context.Context, accountAddress persist.Address) (*indexerdb.Account, error) {
	// Validate

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
