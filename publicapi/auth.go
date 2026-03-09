package publicapi

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-playground/validator/v10"
	db "github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/graphql/dataloader"
	"github.com/mutuals/go-mutuals/service/persist/postgres"
	"github.com/mutuals/go-mutuals/service/redis"
)

type AuthAPI struct {
	repos            *postgres.Repositories
	queries          *db.Queries
	loaders          *dataloader.Loaders
	validator        *validator.Validate
	ethClient        *ethclient.Client
	authRefreshCache *redis.Cache
}
