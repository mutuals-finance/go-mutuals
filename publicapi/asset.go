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

type AssetAPI struct {
	repos              *postgres.Repositories
	queries            *db.Queries
	loaders            *dataloader.Loaders
	validator          *validator.Validate
	ethClient          *ethclient.Client
	multichainProvider *multichain.Provider
	throttler          *throttle.Locker
}

func (api AssetAPI) GetAssetsByOwnerChainAddressPaginate(ctx context.Context, ownerChainAddress persist.ChainAddress, before, after *string, first, last *int, onlySplitfiUsers bool) ([]any, PageInfo, error) {

	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"ownerChainAddress": validate.WithTag(ownerChainAddress, "required"),
	}); err != nil {
		return nil, PageInfo{}, err
	}

	if err := validatePaginationParams(api.validator, first, last); err != nil {
		return nil, PageInfo{}, err
	}

	/*
		TODO external call api

		queryFunc := func(params boolTimeIDPagingParams) ([]db.GetAssetsByOwnerChainAddressPaginateRow, error) {
				return api.queries.GetAssetsByOwnerChainAddressPaginate(ctx, db.GetAssetsByOwnerChainAddressPaginateParams{
					Chain:         ownerChainAddress.Chain(),
					OwnerAddress:  ownerChainAddress.Address(),
					Limit:         params.Limit,
					CurBeforeTime: params.CursorBeforeTime,
					CurBeforeID:   params.CursorBeforeID,
					CurAfterTime:  params.CursorAfterTime,
					CurAfterID:    params.CursorAfterID,
					PagingForward: params.PagingForward,
				})
			}

			countFunc := func() (int, error) {
				total, err := api.queries.CountAssetsByOwnerChainAddress(ctx, db.CountAssetsByOwnerChainAddressParams{
					Chain:        ownerChainAddress.Chain(),
					OwnerAddress: ownerChainAddress.Address(),
				})
				return int(total), err
			}

			cursorFunc := func(r db.GetAssetsByOwnerChainAddressPaginateRow) (bool, time.Time, persist.DBID, error) {
				// TODO remove bool return val (`true`) from cursor
				return true, r.Asset.CreatedAt, r.Asset.ID, nil
			}

			paginator := boolTimeIDPaginator[db.GetAssetsByOwnerChainAddressPaginateRow]{
				QueryFunc:  queryFunc,
				CursorFunc: cursorFunc,
				CountFunc:  countFunc,
			}

			results, pageInfo, err := paginator.paginate(before, after, first, last)
			assets := util.MapWithoutError(results, func(r db.GetAssetsByOwnerChainAddressPaginateRow) db.Asset { return r.Asset })
			return assets, pageInfo, err
	*/
	return nil, PageInfo{}, nil
}
