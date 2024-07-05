package streamer

import (
	"fmt"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/persist"
	sentryutil "github.com/SplitFi/go-splitfi/service/sentry"
	"github.com/SplitFi/go-splitfi/service/task"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/gin-gonic/gin"
	"net/http"
)

func processTokenTransfer(taskClient *task.Client) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var input persist.AlchemyWebhookInput[persist.AlchemyAddressActivityEvent]

		if err := ctx.ShouldBindJSON(&input); err != nil {
			util.ErrResponse(ctx, http.StatusOK, err)
			return
		}

		transfers, _ := util.Map(input.Event.Activity, func(event persist.AlchemyAddressActivityEventItem) (task.TokenTransfer, error) {
			return task.TokenTransfer{
				FromAddress: event.FromAddress,
				ToAddress:   event.ToAddress,
				Token:       persist.NewTokenChainAddress(event.RawContract.Address, input.Event.Network),
				Amount:      event.Value,
			}, nil
		})

		go func() {
			err := taskClient.CreateTaskForTokenTransferProcessing(ctx, task.TokenTransferProcessingMessage{Transfers: transfers})
			if err != nil {
				err = fmt.Errorf("error creating task for token transfer processing: %w", err)
				logger.For(ctx).Error(err)
				sentryutil.ReportError(ctx, err)
			}
		}()

		ctx.JSON(http.StatusOK, util.SuccessResponse{Success: true})
	}
}

func processPoolPublish(taskClient *task.Client) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, util.SuccessResponse{Success: true})
	}
}

func processPoolDeactivate(taskClient *task.Client) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, util.SuccessResponse{Success: true})
	}
}

func processPoolActivate(taskClient *task.Client) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, util.SuccessResponse{Success: true})
	}
}

func processPoolRecipientCreate(taskClient *task.Client) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, util.SuccessResponse{Success: true})
	}
}

func processPoolRecipientUpdate(taskClient *task.Client) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, util.SuccessResponse{Success: true})
	}
}

func processPoolRecipientDelete(taskClient *task.Client) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, util.SuccessResponse{Success: true})
	}
}

func processPoolOwnerUpdate(taskClient *task.Client) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, util.SuccessResponse{Success: true})
	}
}

func processPoolOwnerDelete(taskClient *task.Client) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, util.SuccessResponse{Success: true})
	}
}
