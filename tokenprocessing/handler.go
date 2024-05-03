package tokenprocessing

import (
	"context"
	"github.com/SplitFi/go-splitfi/service/multichain"
	"github.com/SplitFi/go-splitfi/service/persist/postgres"
	sentryutil "github.com/SplitFi/go-splitfi/service/sentry"
	"github.com/SplitFi/go-splitfi/service/throttle"
	"github.com/gin-gonic/gin"
	"time"
)

func handlersInitServer(ctx context.Context, router *gin.Engine, mc *multichain.Provider, repos *postgres.Repositories, throttler *throttle.Locker) *gin.Engine {

	ownersGroup := router.Group("/owners")
	ownersGroup.POST("/wallet-removal", processWalletRemoval(mc.Queries))

	assetsGroup := router.Group("/assets")
	assetsGroup.POST("/", processAssets(mc, mc.Queries))
	// assetsGroup.POST("/spam", detectSpamAssets(mc.Queries))

	tokensGroup := router.Group("/tokens")
	tokensGroup.POST("/", func(c *gin.Context) {
		if hub := sentryutil.SentryHubFromContext(c); hub != nil {
			hub.Scope().AddEventProcessor(sentryutil.SpanFilterEventProcessor(c, 1000, 1*time.Millisecond, 8, true))
		}
		processTokens(mc)(c)
	})
	// tokensGroup.POST("/spam", detectSpamTokens(mc.Queries))

	return router
}
