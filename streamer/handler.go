package streamer

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/mutuals/go-mutuals/env"
	"github.com/mutuals/go-mutuals/middleware"
	"github.com/mutuals/go-mutuals/util"
	"net/http"

	"github.com/mutuals/go-mutuals/service/multichain"
	"github.com/mutuals/go-mutuals/service/persist/postgres"
	"github.com/mutuals/go-mutuals/service/task"
	"github.com/mutuals/go-mutuals/service/throttle"
)

func handlersInitServer(ctx context.Context, router *gin.Engine, s *streamer, mc *multichain.Provider, repos *postgres.Repositories, throttler *throttle.Locker, taskClient *task.Client) *gin.Engine {

	router.GET("/alive", util.HealthCheckHandler())

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
