package publicapi

import (
	"context"
	"errors"
	admin "github.com/SplitFi/go-splitfi/adminapi"
	"github.com/SplitFi/go-splitfi/graphql/apq"
	"github.com/SplitFi/go-splitfi/service/task"
	"github.com/SplitFi/go-splitfi/service/tracing"
	magicclient "github.com/magiclabs/magic-admin-go/client"
	"net/http"
	"time"

	"github.com/SplitFi/go-splitfi/service/persist/postgres"
	"github.com/SplitFi/go-splitfi/service/redis"

	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/gin-gonic/gin"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/storage"
	"github.com/SplitFi/go-splitfi/graphql/dataloader"
	"github.com/SplitFi/go-splitfi/service/auth"
	"github.com/SplitFi/go-splitfi/service/multichain"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/throttle"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/SplitFi/go-splitfi/validate"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/everFinance/goar"
	"github.com/go-playground/validator/v10"
	shell "github.com/ipfs/go-ipfs-api"
)

var errBadCursorFormat = errors.New("bad cursor format")

const apiContextKey = "publicapi.api"

type PublicAPI struct {
	repos     *postgres.Repositories
	queries   *db.Queries
	loaders   *dataloader.Loaders
	validator *validator.Validate
	APQ       *apq.APQCache

	Auth          *AuthAPI
	Split         *SplitAPI
	User          *UserAPI
	Asset         *AssetAPI
	Wallet        *WalletAPI
	Notifications *NotificationsAPI
	Admin         *admin.AdminAPI
	Search        *SearchAPI
}

func New(ctx context.Context, disableDataloaderCaching bool, repos *postgres.Repositories, queries *db.Queries, httpClient *http.Client, ethClient *ethclient.Client, ipfsClient *shell.Shell, arweaveClient *goar.Client, storageClient *storage.Client, taskClient *task.Client, throttler *throttle.Locker, secrets *secretmanager.Client, apq *apq.APQCache, authRefreshCache, oneTimeLoginCache *redis.Cache, magicClient *magicclient.API) *PublicAPI {
	multichainProvider := multichain.NewMultichainProvider(ctx, repos, queries, ethClient, taskClient)
	return NewWithMultichainProvider(ctx, disableDataloaderCaching, repos, queries, httpClient, ethClient, ipfsClient, arweaveClient, storageClient, taskClient, throttler, secrets, apq, authRefreshCache, oneTimeLoginCache, magicClient, multichainProvider)
}

func NewWithMultichainProvider(ctx context.Context, disableDataloaderCaching bool, repos *postgres.Repositories, queries *db.Queries, httpClient *http.Client, ethClient *ethclient.Client, ipfsClient *shell.Shell, arweaveClient *goar.Client, storageClient *storage.Client, taskClient *task.Client, throttler *throttle.Locker, secrets *secretmanager.Client, apq *apq.APQCache, authRefreshCache, oneTimeLoginCache *redis.Cache, magicClient *magicclient.API, multichainProvider *multichain.Provider) *PublicAPI {
	loaders := dataloader.NewLoaders(ctx, queries, disableDataloaderCaching, tracing.DataloaderPreFetchHook, tracing.DataloaderPostFetchHook)
	validator := validate.WithCustomValidators()

	//privyClient := privy.NewPrivyClient(httpClient)

	return &PublicAPI{
		repos:         repos,
		queries:       queries,
		loaders:       loaders,
		validator:     validator,
		APQ:           apq,
		Auth:          &AuthAPI{repos: repos, queries: queries, loaders: loaders, validator: validator, ethClient: ethClient, multiChainProvider: multichainProvider, magicLinkClient: magicClient, oneTimeLoginCache: oneTimeLoginCache, authRefreshCache: authRefreshCache}, // privyClient: privyClient
		Split:         &SplitAPI{repos: repos, queries: queries, loaders: loaders, validator: validator, ethClient: ethClient},
		User:          &UserAPI{repos: repos, queries: queries, loaders: loaders, validator: validator, ethClient: ethClient, ipfsClient: ipfsClient, arweaveClient: arweaveClient, storageClient: storageClient, multichainProvider: multichainProvider},
		Asset:         &AssetAPI{repos: repos, queries: queries, loaders: loaders, validator: validator, ethClient: ethClient, multichainProvider: multichainProvider, throttler: throttler},
		Wallet:        &WalletAPI{repos: repos, queries: queries, loaders: loaders, validator: validator, ethClient: ethClient, multichainProvider: multichainProvider},
		Notifications: &NotificationsAPI{queries: queries, loaders: loaders, validator: validator},
		Admin:         admin.NewAPI(repos, queries, authRefreshCache, validator, multichainProvider),
		Search:        &SearchAPI{queries: queries, loaders: loaders, validator: validator},
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

func getAuthenticatedUserID(ctx context.Context) (persist.DBID, error) {
	gc := util.MustGetGinContext(ctx)
	authError := auth.GetAuthErrorFromCtx(gc)

	if authError != nil {
		return "", authError
	}

	userID := auth.GetUserIDFromCtx(gc)
	return userID, nil
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
