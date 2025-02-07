package main

import (
	"net/http"

	"github.com/mutuals/go-mutuals/service/logger"
	sentryutil "github.com/mutuals/go-mutuals/service/sentry"

	"github.com/mutuals/go-mutuals/server"
	"google.golang.org/appengine"
)

func main() {
	defer sentryutil.RecoverAndRaise(nil)

	server.Init()
	if appengine.IsAppEngine() {
		logger.For(nil).Info("Running in App Engine Mode")
		appengine.Main()
	} else {
		logger.For(nil).Info("Running in Default Mode")
		http.ListenAndServe(":4000", nil)
	}

}
