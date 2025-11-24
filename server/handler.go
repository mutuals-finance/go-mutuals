package server

import (
	"context"
	"net/http"
	"time"

	"github.com/mutuals/go-mutuals/db/gen/indexerdb"
	"github.com/mutuals/go-mutuals/service/task"
	"github.com/vektah/gqlparser/v2/ast"

	"github.com/mutuals/go-mutuals/env"
	"github.com/mutuals/go-mutuals/graphql/apq"
	"github.com/mutuals/go-mutuals/service/auth"
	"github.com/mutuals/go-mutuals/service/redis"

	"github.com/bsm/redislock"
	magicclient "github.com/magiclabs/magic-admin-go/client"
	"github.com/mutuals/go-mutuals/service/persist/postgres"

	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/gorilla/websocket"

	"cloud.google.com/go/pubsub"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/storage"
	gqlgen "github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/everFinance/goar"
	sentry "github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	shell "github.com/ipfs/go-ipfs-api"
	"github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/event"
	"github.com/mutuals/go-mutuals/graphql/generated"
	graphql "github.com/mutuals/go-mutuals/graphql/resolver"
	"github.com/mutuals/go-mutuals/middleware"
	"github.com/mutuals/go-mutuals/publicapi"
	"github.com/mutuals/go-mutuals/service/mediamapper"
	"github.com/mutuals/go-mutuals/service/notifications"
	sentryutil "github.com/mutuals/go-mutuals/service/sentry"
	"github.com/mutuals/go-mutuals/service/throttle"
	"github.com/mutuals/go-mutuals/util"
)

func HandlersInit(router *gin.Engine, repos *postgres.Repositories, coreQueries *coredb.Queries, indexerQueries *indexerdb.Queries, httpClient *http.Client, ethClient *ethclient.Client, ipfsClient *shell.Shell, arweaveClient *goar.Client, storageClient *storage.Client, throttler *throttle.Locker, taskClient *task.Client, pub *pubsub.Client, lock *redislock.Client, secrets *secretmanager.Client, graphqlAPQCache, authRefreshCache, oneTimeLoginCache *redis.Cache, magicClient *magicclient.API) *gin.Engine {
	router.GET("/alive", util.HealthCheckHandler())
	apqCache := &apq.APQCache{Cache: graphqlAPQCache}
	publicapiF := func(ctx context.Context, disableDataloaderCaching bool) *publicapi.PublicAPI {
		api := publicapi.New(ctx, disableDataloaderCaching, repos, coreQueries, indexerQueries, httpClient, ethClient, ipfsClient, arweaveClient, storageClient, taskClient, throttler, secrets, apqCache, authRefreshCache, oneTimeLoginCache, magicClient)
		return api
	}
	GraphqlHandlersInit(router, coreQueries, indexerQueries, taskClient, pub, lock, apqCache, authRefreshCache, publicapiF)
	return router
}

func GraphqlHandlersInit(router *gin.Engine, coreQueries *coredb.Queries, indexerQueries *indexerdb.Queries, taskClient *task.Client, pub *pubsub.Client, lock *redislock.Client, apqCache *apq.APQCache, authRefreshCache *redis.Cache, publicapiF func(ctx context.Context, disableDataloaderCaching bool) *publicapi.PublicAPI) {
	graphqlGroup := router.Group("/mutuals/graphql")
	graphqlHandler := GraphQLHandler(coreQueries, taskClient, pub, lock, apqCache, publicapiF)
	graphqlGroup.Any("/query", middleware.VerifySession(coreQueries, authRefreshCache), graphqlHandler)
	graphqlGroup.Any("/query/:operationName", middleware.VerifySession(coreQueries, authRefreshCache), graphqlHandler)
	graphqlGroup.GET("/playground", graphqlPlaygroundHandler())
}

func GraphQLHandler(queries *coredb.Queries, taskClient *task.Client, pub *pubsub.Client, lock *redislock.Client, apqCache *apq.APQCache, publicapiF func(ctx context.Context, disableDataloaderCaching bool) *publicapi.PublicAPI) gin.HandlerFunc {
	config := generated.Config{Resolvers: &graphql.Resolver{}}
	config.Directives.AuthRequired = graphql.AuthRequiredDirectiveHandler()
	config.Directives.RestrictEnvironment = graphql.RestrictEnvironmentDirectiveHandler()
	config.Directives.BasicAuth = graphql.BasicAuthDirectiveHandler()
	config.Directives.FrontendBuildAuth = graphql.FrontendBuildAuthDirectiveHandler()
	config.Directives.Experimental = graphql.ExperimentalDirectiveHandler()

	schema := generated.NewExecutableSchema(config)
	h := handler.New(schema)

	// This code is ripped from ExecutableSchema.NewDefaultServer
	// We're not using NewDefaultServer anymore because we need a custom
	// WebSocket transport so we can modify the CheckOrigin function
	h.AddTransport(transport.Options{})
	h.AddTransport(transport.GET{})
	h.AddTransport(transport.POST{})
	h.AddTransport(transport.MultipartForm{})

	h.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	h.Use(extension.Introspection{})
	h.Use(extension.AutomaticPersistedQuery{
		Cache: apqCache,
	})

	// End code stolen from handler.NewDefaultServer

	h.AddTransport(&transport.Websocket{
		Upgrader: websocket.Upgrader{
			// This is okay to blindly return true since our
			// HandleCORS middleware function would block us
			// before arriving at this code path.
			CheckOrigin: func(r *http.Request) bool {
				requestOrigin := r.Header.Get("Origin")

				return middleware.IsOriginAllowed(requestOrigin)
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		KeepAlivePingInterval: 15 * time.Second,
	})

	// Request/response logging is spammy in a local environment and can typically be better handled via browser debug tools.
	// It might be worth logging top-level queries and mutations in a single log line, though.
	enableLogging := env.GetString("ENV") != "local"

	h.AroundOperations(graphql.RequestReporter(schema.Schema(), enableLogging, true))
	h.AroundResponses(graphql.ResponseReporter(enableLogging, true))
	h.AroundFields(graphql.FieldReporter(true))
	h.SetErrorPresenter(graphql.ErrorLogger)

	// Should happen after FieldReporter, so Sentry trace context is set up prior to error reporting
	h.AroundFields(graphql.RemapAndReportErrors)

	notificationsHandler := notifications.New(queries, pub, taskClient, lock, true)

	h.AroundFields(graphql.MutationCachingHandler(publicapiF))

	h.SetRecoverFunc(func(ctx context.Context, err interface{}) error {
		if hub := sentryutil.SentryHubFromContext(ctx); hub != nil {
			hub.Recover(err)
		}

		return gqlgen.DefaultRecover(ctx, err)
	})

	return func(c *gin.Context) {
		if hub := sentryutil.SentryHubFromContext(c); hub != nil {
			auth.SetAuthContext(hub.Scope(), c)

			hub.Scope().AddEventProcessor(func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
				// Filter the request body because queries may contain sensitive data. Other middleware (e.g. RequestReporter)
				// can update the request body later with an appropriately scrubbed version of the query.
				event.Request.Data = "[filtered]"
				return event
			})

			hub.Scope().AddEventProcessor(sentryutil.SpanFilterEventProcessor(c, 1000, 1*time.Millisecond, 8, true))
		}

		disableDataloaderCaching := false

		mediamapper.AddTo(c)
		event.AddTo(c, disableDataloaderCaching, notificationsHandler, queries, taskClient)
		notifications.AddTo(c, notificationsHandler)

		// Use the request context so dataloaders will add their traces to the request span
		publicapi.AddTo(c, publicapiF(c.Request.Context(), disableDataloaderCaching))

		h.ServeHTTP(c.Writer, c.Request)
	}
}

// GraphQL playground GUI for experimenting and debugging
func graphqlPlaygroundHandler() gin.HandlerFunc {
	h := playground.Handler("GraphQL", "/mutuals/graphql/query")

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
