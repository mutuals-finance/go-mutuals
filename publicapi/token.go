package publicapi

import (
	"context"
	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/graphql/dataloader"
	"github.com/SplitFi/go-splitfi/service/multichain"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/persist/postgres"
	"github.com/SplitFi/go-splitfi/service/throttle"
	"github.com/SplitFi/go-splitfi/validate"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-playground/validator/v10"
)

type TokenAPI struct {
	repos              *postgres.Repositories
	queries            *db.Queries
	loaders            *dataloader.Loaders
	validator          *validator.Validate
	ethClient          *ethclient.Client
	multichainProvider *multichain.Provider
	throttler          *throttle.Locker
}

// ErrTokenRefreshFailed is a generic error that wraps all sync failures.
type ErrTokenRefreshFailed struct {
	Message string
}

func (e ErrTokenRefreshFailed) Error() string {
	return e.Message
}

func (api TokenAPI) GetTokenById(ctx context.Context, tokenID persist.DBID) (*db.Token, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"tokenID": {tokenID, "required"},
	}); err != nil {
		return nil, err
	}

	token, err := api.loaders.GetTokenByIdBatch.Load(tokenID)
	if err != nil {
		return nil, err
	}

	return &token, nil
}

func (api TokenAPI) GetTokenByChainAddress(ctx context.Context, chainAddress persist.ChainAddress) (*db.Token, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"contractAddress": validate.WithTag(chainAddress, "required"),
	}); err != nil {
		return nil, err
	}

	contract, err := api.loaders.GetTokenByChainAddressBatch.Load(db.GetTokenByChainAddressBatchParams{
		ContractAddress: chainAddress.Address(),
		Chain:           chainAddress.Chain(),
	})
	if err != nil {
		return nil, err
	}

	return &contract, nil
}

func (api TokenAPI) GetTokensByIDs(ctx context.Context, tokenIDs []persist.DBID) ([]db.Token, error) {
	tokens, errs := api.loaders.GetTokenByIdBatch.LoadAll(tokenIDs)
	foundTokens := make([]db.Token, 0, len(tokens))

	for i, token := range tokens {
		if errs[i] == nil {
			foundTokens = append(foundTokens, token)
		} else if _, ok := errs[i].(persist.ErrTokenNotFoundByID); !ok {
			return []db.Token{}, errs[i]
		}
	}

	return foundTokens, nil
}

/*func (api TokenAPI) SetSpamPreference(ctx context.Context, tokens []persist.DBID, isSpam bool) error {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"tokens": validate.WithTag(tokens, "required,unique"),
	}); err != nil {
		return err
	}

	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return err
	}

	return api.queries.UpdateTokensAsUserMarkedSpam(ctx, db.UpdateTokensAsUserMarkedSpamParams{
		IsUserMarkedSpam: sql.NullBool{Bool: isSpam, Valid: true},
		OwnerUserID:      userID,
		TokenIds:         tokens,
	})
}
*/
// RefreshToken refreshes the metadata for a given token DBID
func (api TokenAPI) RefreshToken(ctx context.Context, tokenID persist.DBID) error {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"tokenID": {tokenID, "required"},
	}); err != nil {
		return nil
	}

	token, err := api.loaders.GetTokenByIdBatch.Load(tokenID)
	if err != nil {
		return nil
	}

	err = api.multichainProvider.RefreshToken(ctx, persist.NewTokenChainAddress(token.ContractAddress, token.Chain))
	if err != nil {
		return ErrTokenRefreshFailed{Message: err.Error()}
	}

	return nil
}
