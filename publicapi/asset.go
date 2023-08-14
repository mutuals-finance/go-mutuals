package publicapi

import (
	"context"
	"fmt"
	"time"

	"github.com/SplitFi/go-splitfi/service/persist/postgres"

	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/graphql/dataloader"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/multichain"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/throttle"
	"github.com/SplitFi/go-splitfi/validate"
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

func (api AssetAPI) GetAssetsByOwnerAddress(ctx context.Context, walletID persist.DBID) ([]db.Token, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"walletID": {walletID, "required"},
	}); err != nil {
		return nil, err
	}

	tokens, err := api.loaders.TokensByWalletID.Load(walletID)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

func (api AssetAPI) GetAssetsBySplitIdPaginate(ctx context.Context, contractID persist.DBID, before, after *string, first, last *int, onlySplitFiUsers *bool) ([]db.Token, PageInfo, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"splitID": {contractID, "required"},
	}); err != nil {
		return nil, PageInfo{}, err
	}

	if err := validatePaginationParams(api.validator, first, last); err != nil {
		return nil, PageInfo{}, err
	}

	ogu := false
	if onlySplitFiUsers != nil {
		ogu = *onlySplitFiUsers
	}

	queryFunc := func(params boolTimeIDPagingParams) ([]interface{}, error) {

		logger.For(ctx).Infof("GetAssetsBySplitIdPaginate: %+v", params)
		tokens, err := api.queries.GetTokensByContractIdPaginate(ctx, db.GetTokensByContractIdPaginateParams{
			Contract:           contractID,
			Limit:              params.Limit,
			SplitfiUsersOnly:   ogu,
			CurBeforeUniversal: params.CursorBeforeBool,
			CurAfterUniversal:  params.CursorAfterBool,
			CurBeforeTime:      params.CursorBeforeTime,
			CurBeforeID:        params.CursorBeforeID,
			CurAfterTime:       params.CursorAfterTime,
			CurAfterID:         params.CursorAfterID,
			PagingForward:      params.PagingForward,
		})
		if err != nil {
			return nil, err
		}

		results := make([]interface{}, len(tokens))
		for i, token := range tokens {
			results[i] = token
		}

		return results, nil
	}

	countFunc := func() (int, error) {
		total, err := api.queries.CountTokensByContractId(ctx, db.CountTokensByContractIdParams{
			Contract:         contractID,
			SplitfiUsersOnly: ogu,
		})
		return int(total), err
	}

	cursorFunc := func(i interface{}) (bool, time.Time, persist.DBID, error) {
		if token, ok := i.(db.Token); ok {
			owner, err := api.loaders.OwnerByTokenID.Load(token.ID)
			if err != nil {
				return false, time.Time{}, "", err
			}
			return owner.Universal, token.CreatedAt, token.ID, nil
		}
		return false, time.Time{}, "", fmt.Errorf("interface{} is not a token")
	}

	paginator := boolTimeIDPaginator{
		QueryFunc:  queryFunc,
		CursorFunc: cursorFunc,
		CountFunc:  countFunc,
	}

	results, pageInfo, err := paginator.paginate(before, after, first, last)

	if err != nil {
		return nil, PageInfo{}, err
	}

	tokens := make([]db.Token, len(results))
	for i, result := range results {
		if token, ok := result.(db.Token); ok {
			tokens[i] = token
		} else {
			return nil, PageInfo{}, fmt.Errorf("interface{} is not a token: %T", token)
		}
	}

	return tokens, pageInfo, nil
}
