package publicapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/storage"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/everFinance/goar"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	shell "github.com/ipfs/go-ipfs-api"
	"github.com/mutuals/go-mutuals/adminapi"
	"github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/db/gen/indexerdb"
	"github.com/mutuals/go-mutuals/graphql/apq"
	coreData "github.com/mutuals/go-mutuals/graphql/dataloader"
	indexerData "github.com/mutuals/go-mutuals/graphql/dataloader/indexerdb"
	"github.com/mutuals/go-mutuals/service/auth"
	"github.com/mutuals/go-mutuals/service/auth/privy"
	"github.com/mutuals/go-mutuals/service/persist"
	"github.com/mutuals/go-mutuals/service/persist/postgres"
	"github.com/mutuals/go-mutuals/service/redis"
	"github.com/mutuals/go-mutuals/service/task"
	"github.com/mutuals/go-mutuals/service/throttle"
	"github.com/mutuals/go-mutuals/service/tracing"
	"github.com/mutuals/go-mutuals/util"
	"github.com/mutuals/go-mutuals/validate"
)

var errBadCursorFormat = errors.New("bad cursor format")

const apiContextKey = "publicapi.api"

type PublicAPI struct {
	repos          *postgres.Repositories
	coreQueries    *coredb.Queries
	indexerQueries *indexerdb.Queries
	coreLoaders    *coreData.Loaders
	indexerLoaders *indexerData.Loaders
	validator      *validator.Validate
	APQ            *apq.APQCache
	Auth           *AuthAPI
	Pool           *PoolAPI
	User           *UserAPI
	Claim          *ClaimAPI
	Wallet         *WalletAPI
	PushToken      *PushTokenAPI
	Admin          *adminapi.AdminAPI
	Search         *SearchAPI
}

func New(ctx context.Context, disableDataloaderCaching bool, repos *postgres.Repositories, coreQueries *coredb.Queries, indexerQueries *indexerdb.Queries, httpClient *http.Client, ethClient *ethclient.Client, ipfsClient *shell.Shell, arweaveClient *goar.Client, storageClient *storage.Client, taskClient *task.Client, throttler *throttle.Locker, secrets *secretmanager.Client, apq *apq.APQCache, authRefreshCache *redis.Cache) *PublicAPI {
	coreLoaders := coreData.NewLoaders(ctx, coreQueries, disableDataloaderCaching, tracing.DataloaderPreFetchHook, tracing.DataloaderPostFetchHook)
	indexerLoaders := indexerData.NewLoaders(ctx, indexerQueries, disableDataloaderCaching, tracing.DataloaderPreFetchHook, tracing.DataloaderPostFetchHook)
	validator := validate.WithCustomValidators()

	//privyClient := privy.NewPrivyClient(httpClient)

	return &PublicAPI{
		repos:          repos,
		coreQueries:    coreQueries,
		indexerQueries: indexerQueries,
		coreLoaders:    coreLoaders,
		indexerLoaders: indexerLoaders,
		validator:      validator,
		APQ:            apq,
		Auth:           &AuthAPI{repos: repos, queries: coreQueries, loaders: coreLoaders, validator: validator, ethClient: ethClient, authRefreshCache: authRefreshCache},
		Pool:           &PoolAPI{repos: repos, queries: coreQueries, loaders: coreLoaders, validator: validator, ethClient: ethClient},
		User:           &UserAPI{repos: repos, queries: coreQueries, loaders: coreLoaders, validator: validator, ethClient: ethClient, ipfsClient: ipfsClient, arweaveClient: arweaveClient, storageClient: storageClient},
		Claim:          &ClaimAPI{repos: repos, queries: coreQueries, loaders: coreLoaders, validator: validator, ethClient: ethClient},
		Wallet:         &WalletAPI{repos: repos, queries: coreQueries, coreLoaders: coreLoaders, indexerLoaders: indexerLoaders, validator: validator, ethClient: ethClient},
		PushToken:      &PushTokenAPI{queries: coreQueries, validator: validator},
		Admin:          adminapi.NewAPI(repos, coreQueries, authRefreshCache, validator),
		Search:         &SearchAPI{queries: coreQueries, loaders: coreLoaders, validator: validator},
	}
}

// AddTo adds the specified PublicAPI to a gin context
func AddTo(ctx *gin.Context, api *PublicAPI) {
	ctx.Set(apiContextKey, api)
}

// PushTo pushes the specified PublicAPI onto the context stack and returns the new context
func PushTo(ctx context.Context, api *PublicAPI) context.Context {
	return context.WithValue(ctx, apiContextKey, api)
}

func For(ctx context.Context) *PublicAPI {
	// See if a newer PublicAPI instance has been pushed to the context stack
	if api, ok := ctx.Value(apiContextKey).(*PublicAPI); ok {
		return api
	}

	// If not, fall back to the one added to the gin context
	gc := util.MustGetGinContext(ctx)
	return gc.Value(apiContextKey).(*PublicAPI)
}

func getAuthenticatedUserId(ctx context.Context) (persist.DBID, error) {
	gc := util.MustGetGinContext(ctx)
	authError := auth.GetAuthErrorFromCtx(gc)

	if authError != nil {
		return "", authError
	}

	userID := auth.GetUserIdFromCtx(gc)
	return userID, nil
}

func getAuthenticatedLinkedAccounts(ctx context.Context) ([]privy.LinkedAccount, error) {
	gc := util.MustGetGinContext(ctx)
	authError := auth.GetAuthErrorFromCtx(gc)

	if authError != nil {
		return []privy.LinkedAccount{}, authError
	}

	accounts := auth.GetLinkedAccountsFromCtx(gc)
	return accounts, nil
}

func getUserRoles(ctx context.Context) []persist.Role {
	gc := util.MustGetGinContext(ctx)
	return auth.GetRolesFromCtx(gc)
}

// dbidCache is a lazy cache that stores DBIDs from expensive queries
type dbidCache struct {
	*redis.LazyCache
}

func newDBIDCache(cfg redis.CacheConfig, key string, ttl time.Duration, f func(context.Context) ([]persist.DBID, error)) dbidCache {
	lc := &redis.LazyCache{Cache: redis.NewCache(cfg), Key: key, TTL: ttl}
	lc.CalcFunc = func(ctx context.Context) ([]byte, error) {
		ids, err := f(ctx)
		if err != nil {
			return nil, err
		}
		cur := cursors.NewPositionCursor()
		cur.CurrentPosition = 0
		cur.IDs = ids
		b, err := cur.Pack()
		return []byte(b), err
	}
	return dbidCache{lc}
}

func (d dbidCache) Load(ctx context.Context) ([]persist.DBID, error) {
	b, err := d.LazyCache.Load(ctx)
	if err != nil {
		return nil, err
	}
	cur := cursors.NewPositionCursor()
	err = cur.Unpack(string(b))
	return cur.IDs, err
}
