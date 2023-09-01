package indexer

import (
	gcptasks "cloud.google.com/go/cloudtasks/apiv2"
	"context"
	"github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/sirupsen/logrus"
)

type DBHook[T any] func(ctx context.Context, it []T) error

func newAssetHooks(tasks *gcptasks.Client, bQueries *coredb.Queries) []DBHook[persist.Asset] {
	return []DBHook[persist.Asset]{
		func(ctx context.Context, it []persist.Asset, statsID persist.DBID) error {

			wallets, _ := util.Map(it, func(a persist.Asset) (string, error) {
				return a.OwnerAddress.String(), nil
			})
			chains, _ := util.Map(it, func(a persist.Asset) (int32, error) {
				return int32(a.Token.Chain), nil
			})

			// get all splitfi users associated with any of the tokens
			users, err := bQueries.GetUsersByWalletAddressesAndChains(ctx, coredb.GetUsersByWalletAddressesAndChainsParams{
				WalletAddresses: wallets,
				Chains:          chains,
			})
			if err != nil {
				return err
			}

			// map the chain address to the user id
			addressToUser := make(map[persist.ChainAddress]persist.DBID)
			for _, u := range users {
				addressToUser[persist.NewChainAddress(u.Wallet.Address, u.Wallet.Chain)] = u.User.ID
			}

			assetsForUser := make(map[persist.DBID]map[persist.AssetIdentifiers]persist.NullInt32)
			for _, a := range it {
				ca := persist.NewChainAddress(persist.Address(a.OwnerAddress.String()), a.Token.Chain)
				// check if the token corresponds to a user
				if u, ok := addressToUser[ca]; ok {
					if _, ok := assetsForUser[u]; !ok {
						assetsForUser[u] = make(map[persist.AssetIdentifiers]persist.NullInt32)
					}
					cur := assetsForUser[u]
					cur[persist.NewAssetIdentifiers(a.Token.ContractAddress, a.OwnerAddress)] = a.Balance

					assetsForUser[u] = cur
				}
			}

			logger.For(ctx).Infof("submitting %d tasks to process tokens for users", len(assetsForUser))
			for userID, aids := range assetsForUser {
				for a, b := range aids {
					logger.For(ctx).WithFields(logrus.Fields{"user_id": userID, "asset": a.String(), "balance": b}).Debug("asset for user")
				}
				// send each token grouped by user ID to the task queue
				logger.For(ctx).WithFields(logrus.Fields{"user_id": userID, "token_count": len(aids)}).Infof("submitting task for user %s with %d assets", userID, len(aids))
				err = task.CreateTaskForUserAssetProcessing(ctx, task.AssetProcessingUserAssetsMessage{
					UserID:           userID,
					AssetIdentifiers: aids,
				}, tasks)
				if err != nil {
					return err
				}
			}

			return nil
		},
	}
}

func newTokenHooks(tasks *gcptasks.Client, bQueries *coredb.Queries) []DBHook[persist.Token] {
	return []DBHook[persist.Token]{
		func(ctx context.Context, it []persist.Token, statsID persist.DBID) error {

			wallets, _ := util.Map(it, func(t persist.Token) (string, error) {
				return t.O.String(), nil
			})
			chains, _ := util.Map(it, func(t persist.Token) (int32, error) {
				return int32(t.Chain), nil
			})

			// get all splitfi users associated with any of the tokens
			users, err := bQueries.GetUsersByWalletAddressesAndChains(ctx, coredb.GetUsersByWalletAddressesAndChainsParams{
				WalletAddresses: wallets,
				Chains:          chains,
			})
			if err != nil {
				return err
			}

			// map the chain address to the user id
			addressToUser := make(map[persist.ChainAddress]persist.DBID)
			for _, u := range users {
				addressToUser[persist.NewChainAddress(u.Wallet.Address, u.Wallet.Chain)] = u.User.ID
			}

			tokensForUser := make(map[persist.DBID]map[persist.TokenUniqueIdentifiers]persist.HexString)
			for _, t := range it {
				ca := persist.NewChainAddress(persist.Address(t.OwnerAddress.String()), t.Chain)
				// check if the token corresponds to a user
				if u, ok := addressToUser[ca]; ok {
					if _, ok := tokensForUser[u]; !ok {
						tokensForUser[u] = make(map[persist.TokenUniqueIdentifiers]persist.HexString)
					}
					cur := tokensForUser[u]
					cur[persist.TokenUniqueIdentifiers{
						Chain:           t.Chain,
						ContractAddress: persist.Address(t.ContractAddress.String()),
						TokenID:         t.TokenID,
						OwnerAddress:    persist.Address(t.OwnerAddress.String()),
					}] = t.Quantity

					tokensForUser[u] = cur
				}
			}

			logger.For(ctx).Infof("submitting %d tasks to process tokens for users", len(tokensForUser))
			for userID, tids := range tokensForUser {
				for t, q := range tids {
					logger.For(ctx).WithFields(logrus.Fields{"user_id": userID, "token_id": t.TokenID, "quantity": q}).Debug("token for user")
				}
				// send each token grouped by user ID to the task queue
				logger.For(ctx).WithFields(logrus.Fields{"user_id": userID, "token_count": len(tids)}).Infof("submitting task for user %s with %d tokens", userID, len(tids))
				err = task.CreateTaskForUserTokenProcessing(ctx, task.TokenProcessingUserTokensMessage{
					UserID:           userID,
					TokenIdentifiers: tids,
				}, tasks)
				if err != nil {
					return err
				}
			}

			return nil
		},
	}
}
