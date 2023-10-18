package indexer

import (
	gcptasks "cloud.google.com/go/cloudtasks/apiv2"
	"context"
	"github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/db/gen/indexerdb"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/task"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sirupsen/logrus"
	"github.com/sourcegraph/conc/pool"
	"net/http"
)

type DBHook[T any] func(ctx context.Context, it []T, statsID persist.DBID) error

func newAssetHooks(tasks *gcptasks.Client, bQueries *coredb.Queries) []DBHook[persist.Asset] {
	return []DBHook[persist.Asset]{
		func(ctx context.Context, it []persist.Asset, statsID persist.DBID) error {

			assetOwners, _ := util.Map(it, func(a persist.Asset) (string, error) {
				return a.OwnerAddress.String(), nil
			})

			assetChains, _ := util.Map(it, func(a persist.Asset) (int32, error) {
				return int32(a.Token.Chain), nil
			})

			// get all splits associated with any of the assets
			splits, err := bQueries.GetSplitsByChainsAndAddresses(ctx, coredb.GetSplitsByChainsAndAddressesParams{
				Chains:    assetChains,
				Addresses: assetOwners,
			})

			if err != nil {
				return err
			}

			// map the chain address to the owner
			chainAddressToOwner := make(map[persist.ChainAddress]persist.Address)
			for _, s := range splits {
				chainAddressToOwner[persist.NewChainAddress(s.Address, s.Chain)] = s.Address
			}

			assetsForOwner := make(map[persist.Address]task.TokenIdentifiersQuantities)
			for _, a := range it {
				ca := persist.NewChainAddress(persist.Address(a.Token.ContractAddress), a.Token.Chain)
				// check if the token corresponds to a split
				if s, ok := chainAddressToOwner[ca]; ok {
					if _, ok := assetsForOwner[s]; !ok {
						assetsForOwner[s] = make(task.TokenIdentifiersQuantities)
					}
					cur := assetsForOwner[s]
					cur[persist.NewTokenChainAddress(persist.Address(a.Token.ContractAddress), a.Token.Chain)] = persist.HexString(a.Balance)

					assetsForOwner[s] = cur
				}
			}

			logger.For(ctx).Infof("submitting %d tasks to process tokens for users", len(assetsForOwner))
			for ownerAddress, balancesMap := range assetsForOwner {
				for a, b := range balancesMap {
					logger.For(ctx).WithFields(logrus.Fields{"owner_address": ownerAddress, "asset": a.String(), "balance": b}).Debug("asset for split")
				}
				// send each asset grouped by their owner to the task queue
				logger.For(ctx).WithFields(logrus.Fields{"owner_address": ownerAddress, "token_count": len(balancesMap)}).Infof("submitting task for owner %s with %d assets", ownerAddress, len(balancesMap))
				err = task.CreateTaskForAssetProcessing(ctx, task.TokenProcessingAssetsMessage{
					OwnerAddress: ownerAddress,
					Balances:     balancesMap,
				}, tasks)
				if err != nil {
					return err
				}
			}

			return nil
		},
	}
}

func newTokenHooks(queries *indexerdb.Queries, repo persist.TokenRepository, ethClient *ethclient.Client, httpClient *http.Client) []DBHook[persist.Token] {
	return []DBHook[persist.Token]{
		func(ctx context.Context, it []persist.Token, statsID persist.DBID) error {
			upChan := make(chan []persist.Token)
			go fillTokenFields(ctx, it, queries, repo, httpClient, ethClient, upChan, statsID)

			p := pool.New().WithErrors().WithContext(ctx).WithMaxGoroutines(10)
			for up := range upChan {
				up := up
				p.Go(func(ctx context.Context) error {
					logger.For(ctx).Info("bulk upserting tokens")
					if err := repo.BulkUpsert(ctx, up); err != nil {
						return err
					}
					return nil
				})
			}
			return p.Wait()
		},
	}
}
