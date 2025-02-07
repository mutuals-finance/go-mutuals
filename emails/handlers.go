package emails

import (
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"context"

	"github.com/bsm/redislock"
	"github.com/gin-gonic/gin"
	"github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/env"
	"github.com/mutuals/go-mutuals/event"
	"github.com/mutuals/go-mutuals/graphql/dataloader"
	"github.com/mutuals/go-mutuals/middleware"
	"github.com/mutuals/go-mutuals/publicapi"
	"github.com/mutuals/go-mutuals/service/limiters"
	"github.com/mutuals/go-mutuals/service/notifications"
	"github.com/mutuals/go-mutuals/service/redis"
	"github.com/mutuals/go-mutuals/service/task"
	"github.com/sendgrid/sendgrid-go"
	"net/http"
	"time"
)

func handlersInitServer(router *gin.Engine, loaders *dataloader.Loaders, queries *coredb.Queries, s *sendgrid.Client, r *redis.Cache, stg *storage.Client, papi *publicapi.PublicAPI, psub *pubsub.Client, t *task.Client, notifLock *redislock.Client) *gin.Engine {
	sendGroup := router.Group("/send")

	sendGroup.POST("/notifications", middleware.AdminRequired(), adminSendNotificationEmail(queries, s))

	// Return 200 on auth failures to prevent task/job retries
	authOpts := middleware.BasicAuthOptionBuilder{}
	basicAuthHandler := middleware.BasicHeaderAuthRequired(env.GetString("EMAILS_TASK_SECRET"), authOpts.WithFailureStatus(http.StatusOK))
	sendGroup.POST("/process/add-to-mailing-list", basicAuthHandler, middleware.TaskRequired(), processAddToMailingList(queries))

	limiterCtx := context.Background()
	limiterCache := redis.NewCache(redis.EmailRateLimitersCache)

	verificationLimiter := limiters.NewKeyRateLimiter(limiterCtx, limiterCache, "verification", 1, time.Second*5)
	sendGroup.POST("/verification", middleware.IPRateLimited(verificationLimiter), sendVerificationEmail(loaders, queries, s))

	router.POST("/subscriptions", updateSubscriptions(queries))
	router.POST("/unsubscribe", unsubscribe(queries))
	router.POST("/resubscribe", resubscribe(queries))

	router.POST("/verify", verifyEmail(queries))
	preverifyLimiter := limiters.NewKeyRateLimiter(limiterCtx, limiterCache, "preverify", 1, time.Millisecond*500)
	router.GET("/preverify", middleware.IPRateLimited(preverifyLimiter), preverifyEmail())

	notificationsGroup := router.Group("/notifications")
	notificationsGroup.GET("/send", middleware.CloudSchedulerMiddleware, sendNotificationEmails(queries, s, r))
	notificationsGroup.POST("/announcement", middleware.RetoolAuthRequired, useEventHandler(queries, psub, t, notifLock), sendAnnouncementNotification(queries))

	return router
}

func useEventHandler(q *coredb.Queries, p *pubsub.Client, t *task.Client, l *redislock.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		event.AddTo(c, false, notifications.New(q, p, t, l, false), q, t)
		c.Next()
	}
}
