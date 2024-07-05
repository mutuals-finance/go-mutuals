package streamer

import (
	"context"
	"github.com/SplitFi/go-splitfi/env"
	"github.com/SplitFi/go-splitfi/middleware"
	"github.com/gin-gonic/gin"
	"net/http"

	"github.com/SplitFi/go-splitfi/service/multichain"
	"github.com/SplitFi/go-splitfi/service/persist/postgres"
	"github.com/SplitFi/go-splitfi/service/task"
	"github.com/SplitFi/go-splitfi/service/throttle"
)

func handlersInitServer(ctx context.Context, router *gin.Engine, s *streamer, mc *multichain.Provider, repos *postgres.Repositories, throttler *throttle.Locker, taskClient *task.Client) *gin.Engine {

	authOpts := middleware.BasicAuthOptionBuilder{}
	router.Use(middleware.BasicHeaderAuthRequired(env.GetString("ALCHEMY_WEBHOOK_SECRET"), authOpts.WithFailureStatus(http.StatusOK)))

	tokenGroup := router.Group("/token")
	tokenGroup.POST("/transfer", processTokenTransfer(taskClient))

	poolGroup := router.Group("/pool")
	poolGroup.POST("/publish", processPoolPublish(taskClient))
	poolGroup.POST("/deactivate", processPoolDeactivate(taskClient))
	poolGroup.POST("/activate", processPoolActivate(taskClient))

	poolRecipientGroup := poolGroup.Group("/recipient")
	poolRecipientGroup.POST("/create", processPoolRecipientCreate(taskClient))
	poolRecipientGroup.POST("/update", processPoolRecipientUpdate(taskClient))
	poolRecipientGroup.POST("/delete", processPoolRecipientDelete(taskClient))

	poolOwnerGroup := poolGroup.Group("/owner")
	poolOwnerGroup.POST("/update", processPoolOwnerUpdate(taskClient))
	poolOwnerGroup.POST("/delete", processPoolOwnerDelete(taskClient))

	return router
}
