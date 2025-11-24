package publicapi

import (
	"context"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-playground/validator/v10"
	magicclient "github.com/magiclabs/magic-admin-go/client"
	db "github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/graphql/dataloader"
	"github.com/mutuals/go-mutuals/service/auth"
	"github.com/mutuals/go-mutuals/service/multichain"
	"github.com/mutuals/go-mutuals/service/persist"
	"github.com/mutuals/go-mutuals/service/persist/postgres"
	"github.com/mutuals/go-mutuals/service/redis"
)

type AuthAPI struct {
	repos              *postgres.Repositories
	queries            *db.Queries
	loaders            *dataloader.Loaders
	validator          *validator.Validate
	ethClient          *ethclient.Client
	multiChainProvider *multichain.Provider
	magicLinkClient    *magicclient.API
	oneTimeLoginCache  *redis.Cache
	authRefreshCache   *redis.Cache
}

func (api AuthAPI) ForceAuthTokenRefresh(ctx context.Context, userID persist.DBID) error {
	return auth.ForceAuthTokenRefresh(ctx, api.authRefreshCache, userID)
}
