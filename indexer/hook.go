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

func newAssetHooks(tasks *gcptasks.Client, bQueries *coredb.Queries) []DBHook[persist.AssetDB] {
	return []DBHook[persist.AssetDB]{
		func(ctx context.Context, it []persist.AssetDB, statsID persist.DBID) error {

			owners, _ := util.Map(it, func(a persist.AssetDB) (string, error) {
				return a.OwnerAddress.String(), nil
			})

			chains, _ := util.Map(it, func(a persist.AssetDB) (int32, error) {
				return int32(a.Chain), nil
			})

			// get all splits associated with any of the assets
			splits, err := bQueries.GetSplitsByChainsAndAddresses(ctx, coredb.GetSplitsByChainsAndAddressesParams{
				Chains:    chains,
				Addresses: owners,
			})
			if err != nil {
				return err
			}

			// create performant structure for verifying owner (split) existence
			ownerExists := make(map[persist.Address]bool)
			for _, s := range splits {
				ownerExists[s.Address] = true
			}

			assetsForOwner := make(map[persist.Address]map[persist.TokenChainAddress]persist.NullInt32)

			for _, a := range it {
				owner := a.OwnerAddress
				// check if the asset corresponds to any token of an owner
				if exists, ok := ownerExists[owner]; ok {
					if _, ok := assetsForOwner[owner]; exists && !ok {
						assetsForOwner[owner] = make(task.TokenIdentifiersQuantities)
					}
					// add asset (token balance) for current owner
					tid := persist.NewTokenChainAddress(a.TokenAddress, a.Chain)
					cur := assetsForOwner[owner]
					cur[tid] = a.Balance

					assetsForOwner[owner] = cur
				}
			}

			logger.For(ctx).Infof("submitting %d tasks to process assets for owners", len(assetsForOwner))
			for owner, assets := range assetsForOwner {
				for tid, balance := range assets {
					logger.For(ctx).WithFields(logrus.Fields{"owner_address": owner.String(), "asset": tid.String(), "balance": balance}).Debug("asset for owner")
				}
				// send each asset grouped by their owner to the task queue
				logger.For(ctx).WithFields(logrus.Fields{"owner_address": owner.String(), "asset_count": len(assets)}).Infof("submitting task for owner %s with %d assets", oaddr.String(), len(assets))
				err = task.CreateTaskForAssetProcessing(ctx, task.AssetProcessingMessage{
					OwnerAddress: owner,
					Assets:       assets,
				}, tasks)
				if err != nil {
					return err
				}
			}

			return nil
		},
	}
}
