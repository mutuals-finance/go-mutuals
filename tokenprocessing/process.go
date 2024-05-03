package tokenprocessing

import (
	"github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/multichain"
	"github.com/SplitFi/go-splitfi/service/task"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/gin-gonic/gin"
	"net/http"
)

func processTokens(mc *multichain.Provider) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var input task.TokenProcessingBatchMessage

		if err := ctx.ShouldBindJSON(&input); err != nil {
			util.ErrResponse(ctx, http.StatusOK, err)
			return
		}

		// RefreshTokensByTokenIdentifiers syncs the token if it is found
		_, err := mc.RefreshTokensByTokenIdentifiers(ctx, input.Tokens)
		if err != nil {
			util.ErrResponse(ctx, http.StatusInternalServerError, err)
			return
		}

		ctx.JSON(http.StatusOK, util.SuccessResponse{Success: true})
	}
}

func processAssets(mc *multichain.Provider, queries *coredb.Queries) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var input task.AssetProcessingMessage
		if err := ctx.ShouldBindJSON(&input); err != nil {
			util.ErrResponse(ctx, http.StatusOK, err)
			return
		}

		split, err := queries.GetSplitByAddress(ctx, input.OwnerAddress)
		if err != nil || len(splits) <= 0 {
			// If the split doesn't exist, remove the message from the queue
			util.ErrResponse(ctx, http.StatusOK, err)
			return
		}

		assets, err := mc.SyncAssetsBySplitIDAndTokenIdentifiers(ctx, input.OwnerAddress, util.MapKeys(input.Tokens))
		if err != nil {
			util.ErrResponse(ctx, http.StatusInternalServerError, err)
			return
		}

		ctx.JSON(http.StatusOK, util.SuccessResponse{Success: true})
	}
}

func processWalletRemoval(queries *coredb.Queries) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var input task.TokenProcessingWalletRemovalMessage
		if err := ctx.ShouldBindJSON(&input); err != nil {
			util.ErrResponse(ctx, http.StatusOK, err)
			return
		}

		logger.For(ctx).Infof("Processing wallet removal: UserID=%s, WalletIDs=%v", input.UserID, input.WalletIDs)

		// We never actually remove multiple wallets at a time, but our API does allow it. If we do end up
		// processing multiple wallet removals, we'll just process them in a loop here, because tuning the
		// underlying query to handle multiple wallet removals at a time is difficult.
		for _, walletID := range input.WalletIDs {
			err := queries.RemoveWalletFromTokens(ctx, coredb.RemoveWalletFromTokensParams{
				WalletID: walletID.String(),
				UserID:   input.UserID,
			})

			if err != nil {
				util.ErrResponse(ctx, http.StatusInternalServerError, err)
				return
			}
		}

		ctx.JSON(http.StatusOK, util.SuccessResponse{Success: true})
	}
}
