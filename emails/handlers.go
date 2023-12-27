package emails

import (
	"time"

	"github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/graphql/dataloader"
	"github.com/SplitFi/go-splitfi/middleware"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/sendgrid/sendgrid-go"
)

func handlersInitServer(router *gin.Engine, loaders *dataloader.Loaders, queries *coredb.Queries, s *sendgrid.Client, r *redis.Client) *gin.Engine {
	sendGroup := router.Group("/send")

	sendGroup.POST("/notifications", middleware.AdminRequired(), adminSendNotificationEmail(queries, s))

	// Return 200 on auth failures to prevent task/job retries
	// TODO
	//authOpts := middleware.BasicAuthOptionBuilder{}
	//basicAuthHandler := middleware.BasicHeaderAuthRequired(env.GetString("EMAILS_TASK_SECRET"), authOpts.WithFailureStatus(http.StatusOK))
	//sendGroup.POST("/process/add-to-mailing-list", basicAuthHandler, middleware.TaskRequired(), processAddToMailingList(queries))

	verificationLimiter := middleware.RateLimited(middleware.NewKeyRateLimiter(1, time.Second*5, r))
	sendGroup.POST("/verification", verificationLimiter, sendVerificationEmail(loaders, queries, s))

	router.POST("/subscriptions", updateSubscriptions(queries))
	router.POST("/unsubscribe", unsubscribe(queries))
	router.POST("/resubscribe", resubscribe(queries))

	router.POST("/verify", verifyEmail(queries))
	preVerifyLimiter := middleware.RateLimited(middleware.NewKeyRateLimiter(1, time.Millisecond*500, r))
	router.GET("/preverify", preVerifyLimiter, preverifyEmail())
	return router
}
