package auth

import (
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
)

const authContextName = "auth context"

func SetAuthContext(scope *sentry.Scope, gc *gin.Context) {
	var authCtx sentry.Context
	var userCtx sentry.User

	if GetUserAuthedFromCtx(gc) {
		userId := GetUserIdFromCtx(gc)
		appId := GetAppIDFromCtx(gc)
		authCtx = sentry.Context{
			"Authenticated": true,
			"UserId":        userId,
			"AppId":         appId,
		}
		userCtx = sentry.User{ID: userId.String()}
	} else {
		authCtx = sentry.Context{
			"AuthError": GetAuthErrorFromCtx(gc),
		}
		userCtx = sentry.User{}
	}

	scope.SetContext(authContextName, authCtx)
	scope.SetUser(userCtx)
}

func ScrubEventCookies(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
	if event == nil || event.Request == nil {
		return event
	}

	var scrubbed []string
	for _, c := range strings.Split(event.Request.Cookies, "; ") {
		if strings.HasPrefix(c, AuthCookieKey) {
			scrubbed = append(scrubbed, AuthCookieKey+"=[filtered]")
		} else if strings.HasPrefix(c, RefreshCookieKey) {
			scrubbed = append(scrubbed, RefreshCookieKey+"=[filtered]")
		} else {
			scrubbed = append(scrubbed, c)
		}
	}
	cookies := strings.Join(scrubbed, "; ")

	event.Request.Cookies = cookies
	event.Request.Headers["Cookie"] = cookies
	return event
}
