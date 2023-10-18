package tokenprocessing

import (
	"context"
	"fmt"
	"github.com/SplitFi/go-splitfi/db/gen/coredb"
	"net/http"
	"time"

	"github.com/SplitFi/go-splitfi/service/persist/postgres"
	"github.com/ethereum/go-ethereum/ethclient"

	"cloud.google.com/go/storage"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/media"
	"github.com/SplitFi/go-splitfi/service/multichain"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/task"
	"github.com/SplitFi/go-splitfi/service/throttle"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/everFinance/goar"
	"github.com/gammazero/workerpool"
	"github.com/gin-gonic/gin"
	shell "github.com/ipfs/go-ipfs-api"
	"github.com/sirupsen/logrus"
)

type ProcessMediaForTokenInput struct {
	TokenID           persist.TokenID `json:"token_id" binding:"required"`
	ContractAddress   persist.Address `json:"contract_address" binding:"required"`
	Chain             persist.Chain   `json:"chain"`
	OwnerAddress      persist.Address `json:"owner_address" binding:"required"`
	ImageKeywords     []string        `json:"image_keywords" binding:"required"`
	AnimationKeywords []string        `json:"animation_keywords" binding:"required"`
}

func processMediaForUsersTokensOfChain(mc *multichain.Provider, tokenRepo *postgres.TokenSplitRepository, contractRepo *postgres.ContractSplitRepository, walletRepo persist.WalletRepository, ethClient *ethclient.Client, ipfsClient *shell.Shell, arweaveClient *goar.Client, stg *storage.Client, tokenBucket string, throttler *throttle.Locker) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input task.TokenProcessingUserMessage
		if err := c.ShouldBindJSON(&input); err != nil {
			util.ErrResponse(c, http.StatusOK, err)
			return
		}

		ctx := logger.NewContextWithFields(c, logrus.Fields{"userID": input.UserID})

		if err := throttler.Lock(ctx, input.UserID.String()); err != nil {
			// Reply with a non-200 status so that the message is tried again later on
			util.ErrResponse(c, http.StatusTooManyRequests, err)
			return
		}
		defer throttler.Unlock(ctx, input.UserID.String())

		wp := workerpool.New(100)
		for _, tokenID := range input.TokenIDs {
			t, err := tokenRepo.GetByID(ctx, tokenID)
			if err != nil {
				logger.For(ctx).Errorf("failed to fetch tokenID=%s: %s", tokenID, err)
				continue
			}

			contract, err := contractRepo.GetByID(ctx, t.Contract)
			if err != nil {
				logger.For(ctx).Errorf("Error getting contract: %s", err)
			}

			wp.Submit(func() {
				key := fmt.Sprintf("%s-%s-%d", t.TokenID, contract.Address, t.Chain)
				imageKeywords, animationKeywords := t.Chain.BaseKeywords()
				err := processToken(ctx, key, t, contract.Address, "", mc, ethClient, ipfsClient, arweaveClient, stg, tokenBucket, tokenRepo, imageKeywords, animationKeywords, false)
				if err != nil {
					logger.For(c).Errorf("Error processing token: %s", err)
				}
			})
		}

		wp.StopWait()
		logger.For(ctx).Infof("Processing Media: %s - Finished", input.UserID)

		c.JSON(http.StatusOK, util.SuccessResponse{Success: true})
	}
}

func processMediaForToken(mc *multichain.Provider, tokenRepo *postgres.TokenSplitRepository, userRepo *postgres.UserRepository, walletRepo *postgres.WalletRepository, ethClient *ethclient.Client, ipfsClient *shell.Shell, arweaveClient *goar.Client, stg *storage.Client, tokenBucket string, throttler *throttle.Locker) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input ProcessMediaForTokenInput
		if err := c.ShouldBindJSON(&input); err != nil {
			util.ErrResponse(c, http.StatusBadRequest, err)
			return
		}
		key := fmt.Sprintf("%s-%s-%d", input.TokenID, input.ContractAddress, input.Chain)
		if err := throttler.Lock(c, key); err != nil {
			util.ErrResponse(c, http.StatusTooManyRequests, err)
			return
		}
		defer throttler.Unlock(c, key)

		wallet, err := walletRepo.GetByChainAddress(c, persist.NewChainAddress(input.OwnerAddress, input.Chain))
		if err != nil {
			util.ErrResponse(c, http.StatusInternalServerError, err)
			return
		}

		user, err := userRepo.GetByWalletID(c, wallet.ID)
		if err != nil {
			util.ErrResponse(c, http.StatusInternalServerError, err)
			return
		}

		ctx := logger.NewContextWithFields(c, logrus.Fields{"userID": user.ID})

		t, err := tokenRepo.GetByFullIdentifiers(ctx, input.TokenID, input.ContractAddress, input.Chain, user.ID)
		if err != nil {
			util.ErrResponse(c, http.StatusInternalServerError, err)
			return
		}

		err = processToken(ctx, key, t, input.ContractAddress, input.OwnerAddress, mc, ethClient, ipfsClient, arweaveClient, stg, tokenBucket, tokenRepo, input.ImageKeywords, input.AnimationKeywords, true)
		if err != nil {
			util.ErrResponse(c, http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, util.SuccessResponse{Success: true})
	}
}

func processToken(c context.Context, key string, t persist.TokenSplit, contractAddress, ownerAddress persist.Address, mc *multichain.Provider, ethClient *ethclient.Client, ipfsClient *shell.Shell, arweaveClient *goar.Client, stg *storage.Client, tokenBucket string, tokenRepo *postgres.TokenSplitRepository, imageKeywords, animationKeywords []string, forceRefresh bool) error {
	ctx := logger.NewContextWithFields(c, logrus.Fields{
		"tokenDBID":       t.ID,
		"tokenID":         t.TokenID,
		"contractDBID":    t.Contract,
		"contractAddress": contractAddress,
		"chain":           t.Chain,
	})
	totalTime := time.Now()
	ctx, cancel := context.WithTimeout(ctx, time.Hour)
	defer cancel()

	newMetadata := t.TokenMetadata

	if len(newMetadata) == 0 || forceRefresh {
		mcMetadata, err := mc.GetTokenMetadataByTokenIdentifiers(ctx, contractAddress, t.TokenID, ownerAddress, t.Chain)
		if err != nil {
			logger.For(ctx).Errorf("error getting metadata from chain: %s", err)
		} else if mcMetadata != nil && len(mcMetadata) > 0 {
			logger.For(ctx).Infof("got metadata from chain: %v", mcMetadata)
			newMetadata = mcMetadata
		}
	}

	image, animation := media.KeywordsForChain(t.Chain, imageKeywords, animationKeywords)

	name, description := media.FindNameAndDescription(ctx, newMetadata)

	if name == "" {
		name = t.Name.String()
	}

	if description == "" {
		description = t.Description.String()
	}

	totalTimeOfMedia := time.Now()
	newMedia, err := media.MakePreviewsForMetadata(ctx, newMetadata, contractAddress, persist.TokenID(t.TokenID.String()), t.TokenURI, t.Chain, ipfsClient, arweaveClient, stg, tokenBucket, image, animation)
	if err != nil {
		logger.For(ctx).Errorf("error processing media for %s: %s", key, err)
		newMedia = persist.Media{
			MediaType: persist.MediaTypeUnknown,
		}
	}
	logger.For(ctx).Infof("processing media took %s", time.Since(totalTimeOfMedia))

	// Don't replace existing usable media if tokenprocessing failed to get new media
	if t.Media.IsServable() && !newMedia.IsServable() {
		logger.For(ctx).Debugf("not replacing existing media for %s: cur %v new %v", key, t.Media.IsServable(), newMedia.IsServable())
		return nil
	}

	if newMedia.MediaType.IsAnimationLike() && !persist.TokenURI(newMedia.ThumbnailURL).IsRenderable() && persist.TokenURI(t.Media.ThumbnailURL).IsRenderable() {
		newMedia.ThumbnailURL = t.Media.ThumbnailURL
	}

	up := persist.TokenUpdateAllURIDerivedFieldsInput{
		Media:       newMedia,
		Metadata:    newMetadata,
		Name:        persist.NullString(name),
		Description: persist.NullString(description),
		LastUpdated: persist.LastUpdatedTime{},
	}
	if err := tokenRepo.UpdateByTokenIdentifiersUnsafe(ctx, t.TokenID, contractAddress, t.Chain, up); err != nil {
		logger.For(ctx).Errorf("error updating media for %s-%s-%d: %s", t.TokenID, contractAddress, t.Chain, err)
		return err
	}

	logger.For(ctx).Infof("total processing took %s", time.Since(totalTime))
	return nil
}

func processAssetsForOwner(mc *multichain.Provider, queries *coredb.Queries, validator *validator.Validate) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input task.TokenProcessingAssetsMessage
		if err := c.ShouldBindJSON(&input); err != nil {
			util.ErrResponse(c, http.StatusOK, err)
			return
		}

		logger.For(c).WithFields(logrus.Fields{"owner_address": input.OwnerAddress, "total_tokens": len(input.Balances), "token_ids": input.Balances}).Infof("Processing: %s - Processing User Tokens Refresh (total: %d)", input.OwnerAddress.String(), len(input.Balances))
		newAssets, err := mc.SyncAssetsByOwnerAndTokenChainAddress(c, input.OwnerAddress, util.MapKeys(input.Balances))
		if err != nil {
			util.ErrResponse(c, http.StatusInternalServerError, err)
			return
		}

		if len(newAssets) > 0 {

			for _, asset := range newAssets {
				var curTotal persist.HexString
				dbUniqueAssetIDs, err := queries.GetUniqueTokenIdentifiersByTokenID(c, asset.ID)
				if err != nil {
					logger.For(c).Errorf("error getting unique token identifiers: %s", err)
					continue
				}
				for _, q := range dbUniqueAssetIDs.OwnerAddresses {
					curTotal = input.Balances[persist.TokenChainAddress{
						Chain:        dbUniqueAssetIDs.Chain,
						Address:      dbUniqueAssetIDs.ContractAddress,
						OwnerAddress: persist.Address(q),
					}].Add(curTotal)
				}

				// verify the total is less than or equal to the total in the db
				if curTotal.BigInt().Cmp(token.Quantity.BigInt()) > 0 {
					logger.For(c).Errorf("error: total quantity of tokens in db is greater than total quantity of tokens on chain")
					continue
				}

				// one event per token identifier
				_, err = event.DispatchEvent(c, coredb.Event{
					ID:             persist.GenerateID(),
					ActorID:        persist.DBIDToNullStr(input.UserID),
					ResourceTypeID: persist.ResourceTypeToken,
					SubjectID:      asset.ID,
					UserID:         input.UserID,
					TokenID:        asset.Token.ID,
					Action:         persist.ActionNewTokensReceived,
					Data: persist.EventData{
						NewTokenID:      token.ID,
						NewTokenBalance: curTotal,
					},
				}, validator, nil)
				if err != nil {
					logger.For(c).Errorf("error dispatching event: %s", err)
				}
			}
		}

		c.JSON(http.StatusOK, util.SuccessResponse{Success: true})
	}
}
