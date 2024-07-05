package tokenprocessing

import (
	"cloud.google.com/go/profiler"
	"fmt"
	"github.com/SplitFi/go-splitfi/service/logger"
	sentryutil "github.com/SplitFi/go-splitfi/service/sentry"
	"github.com/SplitFi/go-splitfi/streamer"
	"google.golang.org/appengine"
	"net/http"
	"os"
)

func main() {
	defer sentryutil.RecoverAndRaise(nil)

	cfg := profiler.Config{
		Service:        "tokenprocessing",
		ServiceVersion: "1.0.0",
		MutexProfiling: true,
	}

	// Profiler initialization, best done as early as possible.
	if err := profiler.Start(cfg); err != nil {
		logger.For(nil).Warnf("failed to start cloud profiler due to error: %s\n", err)
	}

	streamer.InitServer()
	if appengine.IsAppEngine() {
		appengine.Main()
	} else {
		port := "6500"
		if it := os.Getenv("PORT"); it != "" {
			port = it
		}
		http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
	}
}
