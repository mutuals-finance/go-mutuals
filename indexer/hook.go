package indexer

import (
	"context"
	"github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/task"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/sirupsen/logrus"
)

type DBHook[T any] func(ctx context.Context, it []T, statsID persist.DBID) error

func newTokenHooks(taskClient *task.Client, bQueries *coredb.Queries) []DBHook[persist.Token] {
	return []DBHook[persist.Token]{
		func(ctx context.Context, tokens []persist.Token, statsID persist.DBID) error {

			tids, err := util.Map(tokens, func(t persist.Token) (coredb.GetTokensByChainAddressBatchParams, error) {
				return coredb.GetTokensByChainAddressBatchParams{ContractAddress: t.ContractAddress, Chain: t.Chain}, nil
			})
			if err != nil {
				return err
			}

			tokenLookup := make(map[persist.TokenChainAddress]bool)
			bQueries.GetTokensByChainAddressBatch(ctx, tids).Query(func(i int, existingTokens []coredb.Token, err error) {
				for _, token := range existingTokens {
					tokenLookup[persist.NewTokenChainAddress(token.ContractAddress, token.Chain)] = true
				}
			})

			filterFunc := func(p coredb.GetTokensByChainAddressBatchParams) bool {
				_, ok := tokenLookup[persist.NewTokenChainAddress(p.ContractAddress, p.Chain)]
				return !ok
			}

			mapFunc := func(p coredb.GetTokensByChainAddressBatchParams) (persist.TokenChainAddress, error) {
				return persist.NewTokenChainAddress(p.ContractAddress, p.Chain), nil
			}

			newTids, err := util.Map(util.Filter(tids, filterFunc, false), mapFunc)
			if err != nil {
				return err
			}

			for _, t := range newTids {
				logger.For(ctx).WithFields(logrus.Fields{"token address": t.Address, "chain": t.Chain}).Debug("token")
			}

			// send each asset grouped by their owner to the task queue
			logger.For(ctx).WithFields(logrus.Fields{"token_count": len(newTids)}).Infof("submitting task with %d tokens", len(newTids))
			// TODO batch ID needed?
			message := task.TokenProcessingBatchMessage{BatchID: persist.GenerateID(), Tokens: newTids}
			err = taskClient.CreateTaskForTokenProcessing(ctx, message)
			if err != nil {
				return err
			}
			return nil
		},
	}
}

func newAssetHooks(taskClient *task.Client, queries *coredb.Queries) []DBHook[persist.AssetDB] {
	return []DBHook[persist.AssetDB]{
		func(ctx context.Context, it []persist.AssetDB, statsID persist.DBID) error {

			// get all splits associated with any of the assets

			owners, _ := util.Map(it, func(a persist.AssetDB) (string, error) {
				return a.OwnerAddress.String(), nil
			})

			chains, _ := util.Map(it, func(a persist.AssetDB) (int32, error) {
				return int32(a.Chain), nil
			})

			splits, err := queries.GetSplitsByChainsAndAddresses(ctx, coredb.GetSplitsByChainsAndAddressesParams{
				Chains:    chains,
				Addresses: owners,
			})
			if err != nil {
				return err
			}

			// verify split existence

			splitExists := make(map[persist.Address]bool)
			for _, s := range splits {
				splitExists[s.Address] = true
			}

			assetsForSplit := make(map[persist.Address]task.TokenIdentifierBalances)

			for _, a := range it {
				splitAddress := a.OwnerAddress
				// check if the asset corresponds to any token of an owner
				if exists, ok := splitExists[splitAddress]; ok {
					if _, ok := assetsForSplit[splitAddress]; exists && !ok {
						assetsForSplit[splitAddress] = make(task.TokenIdentifierBalances)
					}

					// add asset (token balance) for current owner
					splitAssets := assetsForSplit[splitAddress]

					splitAssets[persist.NewTokenChainAddress(
						a.TokenAddress,
						a.Chain,
					)] = a.Balance

					assetsForSplit[splitAddress] = splitAssets
				}
			}

			// submit tasks for processing assets for owner

			logger.For(ctx).Infof("submitting %d tasks to process assets for owners", len(assetsForSplit))
			for splitAddress, splitAssets := range assetsForSplit {
				for tID, balance := range splitAssets {
					logger.For(ctx).WithFields(logrus.Fields{"split": splitAddress.String(), "token": tID.String(), "balance": balance}).Debug("asset for split")
				}
				// send each asset grouped by their split to the task queue
				logger.For(ctx).WithFields(logrus.Fields{"owner_address": splitAddress.String(), "asset_count": len(splitAssets)}).Infof("submitting task for owner %s with %d assets", splitAddress, len(splitAssets))
				err = taskClient.CreateTaskForAssetProcessing(ctx, task.TokenProcessingAssetMessage{
					OwnerAddress: splitAddress,
					Assets:       splitAssets,
				})
				if err != nil {
					return err
				}
			}

			return nil
		},
	}
}
