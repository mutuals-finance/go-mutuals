package tokenprocessing

import (
	"cloud.google.com/go/storage"
	"context"
	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/everFinance/goar"
	shell "github.com/ipfs/go-ipfs-api"
	"net/http"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/SplitFi/go-splitfi/env"
	"github.com/SplitFi/go-splitfi/event"
	"github.com/SplitFi/go-splitfi/middleware"
	"github.com/SplitFi/go-splitfi/server"
	"github.com/SplitFi/go-splitfi/service/auth"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/multichain"
	"github.com/SplitFi/go-splitfi/service/notifications"
	"github.com/SplitFi/go-splitfi/service/redis"
	"github.com/SplitFi/go-splitfi/service/throttle"
	"github.com/SplitFi/go-splitfi/service/tracing"
	"github.com/SplitFi/go-splitfi/util"
)

const sentryTokenContextName = "NFT context" // Sentry excludes contexts that contain "token" so we use "NFT" instead

// InitServer initializes the tokenprocessing server
func InitServer() {
	setDefaults()
	ctx := context.Background()
	c := server.ClientInit(ctx)
	mc := multichain.NewMultichainProvider(ctx, c.Repos, c.Queries, c.EthClient, c.TaskClient)
	router := CoreInitServer(ctx, c, mc)
	logger.For(nil).Info("Starting tokenprocessing server...")
	http.Handle("/", router)
}

func CoreInitServer(ctx context.Context, clients *server.Clients, mc *multichain.Provider) *gin.Engine {
	InitSentry()
	logger.InitWithGCPDefaults()

	router := gin.Default()

	router.Use(middleware.GinContextToContext(), middleware.Sentry(true), middleware.Tracing(), middleware.HandleCORS(), middleware.ErrLogger())

	notificationsHandler := notifications.New(clients.Queries, clients.PubSubClient, clients.TaskClient, redis.NewLockClient(redis.NewCache(redis.NotificationLockCache)), false)

	router.Use(func(c *gin.Context) {
		event.AddTo(c, false, notificationsHandler, clients.Queries, clients.TaskClient)
	})

	if env.GetString("ENV") != "production" {
		gin.SetMode(gin.DebugMode)
		logrus.SetLevel(logrus.DebugLevel)
	}

	logger.For(nil).Info("Registering handlers...")

	t := newThrottler()

	// tokenprocessing tends to create many connections to many different hosts.
	// Since a connection is unlikely to get re-used, we don't leave any idle connections around
	// to avoid having too many open connections.
	if tr, ok := http.DefaultTransport.(*http.Transport); ok {
		(*tr).MaxIdleConns = -1
		(*tr).DisableKeepAlives = true
	} else if tr, ok := http.DefaultTransport.(*tracing.TracingTransport); ok {
		t := tr.RoundTripper.(*http.Transport)
		(*t).MaxIdleConns = -1
		(*t).DisableKeepAlives = true
	}

	tp := NewTokenProcessor(clients.Queries, http.DefaultClient, clients.IPFSClient, clients.ArweaveClient, clients.StorageClient, env.GetString("GCLOUD_TOKEN_CONTENT_BUCKET"))

	return handlersInitServer(ctx, router, tp, mc, clients.Repos, t, clients.TaskClient)
}

type tokenProcessor struct {
	queries       *db.Queries
	httpClient    *http.Client
	ipfsClient    *shell.Shell
	arweaveClient *goar.Client
	stg           *storage.Client
	tokenBucket   string
}

func NewTokenProcessor(queries *db.Queries, httpClient *http.Client, ipfsClient *shell.Shell, arweaveClient *goar.Client, stg *storage.Client, tokenBucket string) *tokenProcessor {
	return &tokenProcessor{
		queries:       queries,
		httpClient:    httpClient,
		ipfsClient:    ipfsClient,
		arweaveClient: arweaveClient,
		stg:           stg,
		tokenBucket:   tokenBucket,
	}
}

func setDefaults() {
	viper.SetDefault("IPFS_URL", "https://gallery.infura-ipfs.io")
	viper.SetDefault("IPFS_API_URL", "https://ipfs.infura.io:5001")
	viper.SetDefault("IPFS_PROJECT_ID", "")
	viper.SetDefault("IPFS_PROJECT_SECRET", "")
	viper.SetDefault("CHAIN", 0)
	viper.SetDefault("ENV", "local")
	viper.SetDefault("RPC_URL", "https://eth-goerli.g.alchemy.com/v2/_2u--i79yarLYdOT4Bgydqa0dBceVRLD")
	viper.SetDefault("GCLOUD_TOKEN_LOGS_BUCKET", "dev-eth-token-logs")
	viper.SetDefault("GCLOUD_TOKEN_CONTENT_BUCKET", "dev-token-content")
	viper.SetDefault("POSTGRES_HOST", "0.0.0.0")
	viper.SetDefault("POSTGRES_PORT", 5432)
	viper.SetDefault("POSTGRES_USER", "postgres")
	viper.SetDefault("POSTGRES_PASSWORD", "postgres")
	viper.SetDefault("POSTGRES_DB", "postgres")
	viper.SetDefault("ALLOWED_ORIGINS", "http://localhost:3000")
	viper.SetDefault("REDIS_URL", "localhost:6379")
	viper.SetDefault("SENTRY_DSN", "")
	viper.SetDefault("IMGIX_API_KEY", "")
	viper.SetDefault("VERSION", "")
	viper.SetDefault("ALCHEMY_API_URL", "")
	viper.SetDefault("ALCHEMY_OPTIMISM_API_URL", "")
	viper.SetDefault("ALCHEMY_POLYGON_API_URL", "")
	viper.SetDefault("ALCHEMY_BASE_SEPOLIA_API_URL", "")
	viper.SetDefault("POAP_API_KEY", "")
	viper.SetDefault("POAP_AUTH_TOKEN", "")
	viper.SetDefault("TOKEN_PROCESSING_URL", "http://localhost:6500")
	viper.SetDefault("TOKEN_PROCESSING_QUEUE", "projects/gallery-local/locations/here/queues/token-processing")
	viper.SetDefault("TASK_QUEUE_HOST", "")
	viper.SetDefault("GOOGLE_CLOUD_PROJECT", "gallery-dev-322005")
	viper.SetDefault("PUBSUB_EMULATOR_HOST", "")
	viper.SetDefault("PUBSUB_TOPIC_NEW_NOTIFICATIONS", "dev-new-notifications")
	viper.SetDefault("PUBSUB_TOPIC_UPDATED_NOTIFICATIONS", "dev-updated-notifications")
	viper.SetDefault("PUBSUB_SUB_NEW_NOTIFICATIONS", "dev-new-notifications-sub")
	viper.SetDefault("PUBSUB_SUB_UPDATED_NOTIFICATIONS", "dev-updated-notifications-sub")
	viper.SetDefault("RASTERIZER_URL", "http://localhost:3000")
	viper.SetDefault("TEZOS_API_URL", "https://api.tzkt.io")
	viper.SetDefault("ALCHEMY_WEBHOOK_SECRET_ARBITRUM", "")
	viper.SetDefault("ALCHEMY_WEBHOOK_SECRET_ETH", "")
	viper.SetDefault("OPENSEA_WEBHOOK_SECRET", "")
	viper.SetDefault("MINT_PROCESSING_QUEUE", "projects/gallery-local/locations/here/queues/mint-processing")
	viper.SetDefault("SIMPLEHASH_API_KEY", "")

	viper.AutomaticEnv()

	if env.GetString("ENV") != "local" {
		logger.For(nil).Info("running in non-local environment, skipping environment configuration")
	} else {
		fi := "local"
		if len(os.Args) > 1 {
			fi = os.Args[1]
		}
		envFile := util.ResolveEnvFile("tokenprocessing", fi)
		util.LoadEncryptedEnvFile(envFile)
	}

	if env.GetString("ENV") != "local" {
		util.VarNotSetTo("SENTRY_DSN", "")
		util.VarNotSetTo("VERSION", "")
	}
}

func newThrottler() *throttle.Locker {
	return throttle.NewThrottleLocker(redis.NewCache(redis.TokenProcessingThrottleCache), time.Minute*30)
}

func InitSentry() {
	if env.GetString("ENV") == "local" {
		logger.For(nil).Info("skipping sentry init")
		return
	}

	logger.For(nil).Info("initializing sentry...")

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              env.GetString("SENTRY_DSN"),
		Environment:      env.GetString("ENV"),
		TracesSampleRate: env.GetFloat64("SENTRY_TRACES_SAMPLE_RATE"),
		Release:          env.GetString("VERSION"),
		AttachStacktrace: true,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			event = auth.ScrubEventCookies(event, hint)
			event = excludeTokenSpamEvents(event, hint)
			//event = excludeBadTokenEvents(event, hint)
			return event
		},
	})

	if err != nil {
		logger.For(nil).Fatalf("failed to start sentry: %s", err)
	}
}

// excludeTokenSpamEvents excludes events for tokens marked as spam.
func excludeTokenSpamEvents(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
	isSpam, ok := event.Contexts[sentryTokenContextName]["IsSpam"].(bool)
	if ok && isSpam {
		return nil
	}
	return event
}
