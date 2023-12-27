package tokenprocessing

import (
	"github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/event"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/multichain"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/task"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
)

func processAssetsForOwner(mc *multichain.Provider, queries *coredb.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input task.AssetProcessingMessage
		if err := c.ShouldBindJSON(&input); err != nil {
			util.ErrResponse(c, http.StatusOK, err)
			return
		}

		logger.For(c).WithFields(logrus.Fields{"owner_address": input.OwnerAddress, "total_assets": len(input.Assets), "asset_ids": input.Assets}).Infof("Processing: %s - Processing Assets Refresh (total: %d)", input.OwnerAddress, len(input.Assets))
		newAssets, err := mc.SyncAssetsByOwnerAndTokenChainAddress(c, input.OwnerAddress, util.MapKeys(input.Assets))
		if err != nil {
			util.ErrResponse(c, http.StatusInternalServerError, err)
			return
		}

		if len(newAssets) > 0 {

			for _, asset := range newAssets {

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
						TokenID:         asset.Token.ID,
						NewTokenBalance: asset.Balance,
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
