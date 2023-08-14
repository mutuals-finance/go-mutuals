package publicapi

import (
	"context"
	"errors"

	admin "github.com/SplitFi/go-splitfi/adminapi"
	"github.com/SplitFi/go-splitfi/graphql/apq"
	magicclient "github.com/magiclabs/magic-admin-go/client"

	"github.com/SplitFi/go-splitfi/service/persist/postgres"
	"github.com/SplitFi/go-splitfi/service/redis"

	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/event"
	"github.com/gin-gonic/gin"

	gcptasks "cloud.google.com/go/cloudtasks/apiv2"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/storage"
	"github.com/SplitFi/go-splitfi/graphql/dataloader"
	"github.com/SplitFi/go-splitfi/service/auth"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/multichain"
	"github.com/SplitFi/go-splitfi/service/persist"
	sentryutil "github.com/SplitFi/go-splitfi/service/sentry"
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
	Token         *TokenAPI
	Asset         *AssetAPI
	Wallet        *WalletAPI
	Misc          *MiscAPI
	Notifications *NotificationsAPI
	Admin         *admin.AdminAPI
	Social        *SocialAPI
	Search        *SearchAPI
}

func New(ctx context.Context, disableDataloaderCaching bool, repos *postgres.Repositories, queries *db.Queries, ethClient *ethclient.Client, ipfsClient *shell.Shell,
	arweaveClient *goar.Client, storageClient *storage.Client, multichainProvider *multichain.Provider, taskClient *gcptasks.Client, throttler *throttle.Locker, secrets *secretmanager.Client, apq *apq.APQCache, socialCache *redis.Cache, magicClient *magicclient.API) *PublicAPI {
	loaders := dataloader.NewLoaders(ctx, queries, disableDataloaderCaching)
	validator := validate.WithCustomValidators()

	return &PublicAPI{
		repos:     repos,
		queries:   queries,
		loaders:   loaders,
		validator: validator,
		APQ:       apq,

		Auth:          &AuthAPI{repos: repos, queries: queries, loaders: loaders, validator: validator, ethClient: ethClient, multiChainProvider: multichainProvider, magicLinkClient: magicClient},
		Split:         &SplitAPI{repos: repos, queries: queries, loaders: loaders, validator: validator, ethClient: ethClient},
		User:          &UserAPI{repos: repos, queries: queries, loaders: loaders, validator: validator, ethClient: ethClient, ipfsClient: ipfsClient, arweaveClient: arweaveClient, storageClient: storageClient, multichainProvider: multichainProvider},
		Token:         &TokenAPI{repos: repos, queries: queries, loaders: loaders, validator: validator, ethClient: ethClient, multichainProvider: multichainProvider, throttler: throttler},
		Asset:         &AssetAPI{repos: repos, queries: queries, loaders: loaders, validator: validator, ethClient: ethClient, multichainProvider: multichainProvider, throttler: throttler},
		Wallet:        &WalletAPI{repos: repos, queries: queries, loaders: loaders, validator: validator, ethClient: ethClient, multichainProvider: multichainProvider},
		Misc:          &MiscAPI{repos: repos, queries: queries, loaders: loaders, validator: validator, ethClient: ethClient, storageClient: storageClient},
		Notifications: &NotificationsAPI{queries: queries, loaders: loaders, validator: validator},
		Admin:         admin.NewAPI(repos, queries, validator, multichainProvider),
		Social:        &SocialAPI{repos: repos, queries: queries, loaders: loaders, validator: validator, redis: socialCache},
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
	gc := util.GinContextFromContext(ctx)
	return gc.Value(apiContextKey).(*PublicAPI)
}

func getAuthenticatedUserID(ctx context.Context) (persist.DBID, error) {
	gc := util.GinContextFromContext(ctx)
	authError := auth.GetAuthErrorFromCtx(gc)

	if authError != nil {
		return "", authError
	}

	userID := auth.GetUserIDFromCtx(gc)
	return userID, nil
}

func publishEventGroup(ctx context.Context, groupID string, action persist.Action, caption *string) error {
	return event.DispatchGroup(sentryutil.NewSentryHubGinContext(ctx), groupID, action, caption)
}

func dispatchEvent(ctx context.Context, evt db.Event, v *validator.Validate, caption *string) error {
	ctx = sentryutil.NewSentryHubGinContext(ctx)
	if err := v.Struct(evt); err != nil {
		return err
	}

	if caption != nil {
		evt.Caption = persist.StrPtrToNullStr(caption)
		return event.DispatchImmediate(ctx, []db.Event{evt})
	}

	go pushEvent(ctx, evt)
	return nil
}

func dispatchEvents(ctx context.Context, evts []db.Event, v *validator.Validate, editID *string, caption *string) error {

	if len(evts) == 0 {
		return nil
	}

	ctx = sentryutil.NewSentryHubGinContext(ctx)
	for i, evt := range evts {
		evt.GroupID = persist.StrPtrToNullStr(editID)
		if err := v.Struct(evt); err != nil {
			return err
		}
		evts[i] = evt
	}

	if caption != nil {
		for i, evt := range evts {
			evt.Caption = persist.StrPtrToNullStr(caption)
			evts[i] = evt
		}
		return event.DispatchImmediate(ctx, evts)
	}

	for _, evt := range evts {
		go pushEvent(ctx, evt)
	}
	return nil
}

func pushEvent(ctx context.Context, evt db.Event) {
	if hub := sentryutil.SentryHubFromContext(ctx); hub != nil {
		sentryutil.SetEventContext(hub.Scope(), persist.NullStrToDBID(evt.ActorID), evt.SubjectID, evt.Action)
	}
	if err := event.DispatchDelayed(ctx, evt); err != nil {
		logger.For(ctx).Error(err)
		sentryutil.ReportError(ctx, err)
	}
}
