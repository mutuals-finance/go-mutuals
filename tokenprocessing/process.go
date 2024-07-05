package tokenprocessing

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4"
	"github.com/sirupsen/logrus"
	concpool "github.com/sourcegraph/conc/pool"
	"math/big"
	"net/http"

	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/event"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/multichain"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/task"
	"github.com/SplitFi/go-splitfi/util"
)

func processTokenTransfers(mc *multichain.Provider, queries *db.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input task.TokenTransferProcessingMessage

		if err := c.ShouldBindJSON(&input); err != nil {
			util.ErrResponse(c, http.StatusOK, err)
			return
		}

		var pPoolAddresses []persist.Address
		var pTokenAddresses []persist.Address
		var pChains []persist.Chain

		for _, transfer := range input.Transfers {
			pPoolAddresses = append(pPoolAddresses, transfer.FromAddress, transfer.ToAddress)
			pTokenAddresses = append(pTokenAddresses, transfer.Token.Address, transfer.Token.Address)
			pChains = append(pChains, transfer.Token.Chain, transfer.Token.Chain)
		}

		beforeBalances, err := queries.GetPoolTokensByTokenIdentifiers(c, db.GetPoolTokensByTokenIdentifiersParams{
			PoolAddresses:  pPoolAddresses,
			TokenAddresses: pTokenAddresses,
			Chains:         pChains,
		})

		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			logger.For(c).Errorf("error querying tokens: %s", err)
			util.ErrResponse(c, http.StatusInternalServerError, err)
			return
		}

		ownerTokensMap := make(map[persist.ChainAddress]map[persist.TokenChainAddress]persist.HexString)

		for _, b := range beforeBalances {
			ownerTokensMap[persist.NewChainAddress(b.OwnerAddress, b.Chain)][persist.NewTokenChainAddress(b.TokenAddress, b.Chain)] = b.Balance
		}

		for _, transfer := range input.Transfers {
			token := transfer.Token
			chain := transfer.Token.Chain
			from, to := persist.NewChainAddress(transfer.FromAddress, chain), persist.NewChainAddress(transfer.ToAddress, chain)
			_, ok := ownerTokensMap[from]
			if ok {
				ownerTokensMap[from][token] = ownerTokensMap[from][token].Sub(transfer.Amount)
			}

			_, ok = ownerTokensMap[to]
			if ok {
				ownerTokensMap[to][token].Add(transfer.Amount)
			}
		}

		wp := concpool.New().WithMaxGoroutines(50).WithErrors()

		for owner, tokens := range ownerTokensMap {
			wp.Go(func() error {

				ctx := logger.NewContextWithFields(c, logrus.Fields{
					"address": owner.Address(),
					"chain":   owner.Chain,
				})

				logger.For(ctx).Infof("Owner=%s - Processing Token", owner)

				tAddresses := make([]persist.Address, len(tokens))
				tChains := make([]persist.Chain, len(tokens))
				tBalances := make([]persist.HexString, len(tokens))

				for tID, tBalance := range tokens {
					tAddresses = append(tAddresses, tID.Address)
					tChains = append(tChains, tID.Chain)
					tBalances = append(tBalances, tBalance)
				}

				updatedTokens, err := mc.UpdateTokensForPoolUnchecked(ctx, owner, tAddresses, tChains, tBalances)

				if err != nil {
					logger.For(ctx).Errorf("error syncing tokens: %s", err)
					util.ErrResponse(c, http.StatusInternalServerError, err)
					return err
				}

				if len(updatedTokens) == 0 {
					logger.For(ctx).Infof("no tokens updated for owner=%s", owner)
				} else {
					logger.For(ctx).Infof("updated %d tokens for owner=%s", len(updatedTokens), owner)
				}

				for _, t := range updatedTokens {
					logger.For(ctx).Infof("added tokenDBID=%s to owner=%s", t.Instance.ID, t.Instance.OwnerAddress)

					if t.Instance.Balance.BigInt().Cmp(big.NewInt(0)) <= 0 {
						logger.For(ctx).Infof("token balance is 0 or less, skipping")
						continue
					}

					// one event per token balance update
					// TODO update for pool token type
					err = event.Dispatch(ctx, db.Event{
						ID: persist.GenerateID(),
						//ActorID:        owner.,
						ResourceTypeID: persist.ResourceTypeToken,
						SubjectID:      t.Instance.ID,
						//PoolID:         owner.ID,
						//TokenID: t.Instance.ID,
						//Action: persist.ActionTokenTransfer,
						Data: persist.EventData{
							//TokenID: t.Instance.ID,
							//Balance: t.Instance.Balance,
						},
					})
					if err != nil {
						logger.For(ctx).Errorf("error dispatching event: %s", err)
					}
				}

				return nil
			})

		}

		wp.Wait()

		c.JSON(http.StatusOK, util.SuccessResponse{Success: true})
	}
}

func processWalletRemoval() gin.HandlerFunc {
	return func(c *gin.Context) {
		var input task.TokenProcessingWalletRemovalMessage
		if err := c.ShouldBindJSON(&input); err != nil {
			util.ErrResponse(c, http.StatusOK, err)
			return
		}

		logger.For(c).Infof("Processing wallet removal: UserID=%s, WalletIDs=%v", input.UserID, input.WalletIDs)

		c.JSON(http.StatusOK, util.SuccessResponse{Success: true})
	}
}
