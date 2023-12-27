package tokenprocessing

import (
	"context"
	"github.com/SplitFi/go-splitfi/service/multichain"
	"github.com/SplitFi/go-splitfi/service/persist/postgres"
	"github.com/SplitFi/go-splitfi/service/throttle"
	"github.com/gin-gonic/gin"
)

var (
	OwnersGroupPath                   = "/owner"
	ProcessAssetsForOwnerPathRelative = "/assets"
	ProcessAssetsForOwnerPath         = OwnersGroupPath + ProcessAssetsForOwnerPathRelative
)

func handlersInitServer(ctx context.Context, router *gin.Engine, mc *multichain.Provider, repos *postgres.Repositories, throttler *throttle.Locker) *gin.Engine {

	ownersGroup := router.Group(OwnersGroupPath)
	ownersGroup.POST(ProcessAssetsForOwnerPathRelative, processAssetsForOwner(mc, mc.Queries))

	return router
}
