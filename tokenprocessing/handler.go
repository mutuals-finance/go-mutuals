package tokenprocessing

import (
	"context"
	"github.com/gin-gonic/gin"

	"github.com/mutuals/go-mutuals/service/multichain"
	"github.com/mutuals/go-mutuals/service/persist/postgres"
	"github.com/mutuals/go-mutuals/service/task"
	"github.com/mutuals/go-mutuals/service/throttle"
)

func handlersInitServer(ctx context.Context, router *gin.Engine, tp *tokenProcessor, mc *multichain.Provider, repos *postgres.Repositories, throttler *throttle.Locker, taskClient *task.Client) *gin.Engine {
	// Handles retries and token state

	tokenGroup := router.Group("/token")
	tokenGroup.POST("/transfer", processTokenTransfers(mc, mc.Queries))

	ownersGroup := router.Group("/owner")
	ownersGroup.POST("/wallet-removal", processWalletRemoval())

	return router
}
