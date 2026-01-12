package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	db "github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/service/auth/privy"
	"github.com/mutuals/go-mutuals/service/redis"
	"github.com/mutuals/go-mutuals/util"

	"github.com/mutuals/go-mutuals/env"

	"github.com/mutuals/go-mutuals/service/logger"

	"github.com/gin-gonic/gin"
	"github.com/mutuals/go-mutuals/service/persist"
)

const (
	// Context keys for auth data
	userAuthedContextKey     = "auth.authenticated"
	userIdContextKey         = "auth.user_id"
	appIdContextKey          = "auth.app_id"
	linkedAccountsContextKey = "auth.linked_accounts"
	authErrorContextKey      = "auth.auth_error"
	userRolesContextKey      = "auth.roles"
)

const cookieExpires = 1 * time.Hour

// AuthCookieKey is the key used to store the auth token in the cookie
const AuthCookieKey = "privy-token"

// IdCookieKey is the key used to store the id token in the cookie
const IdCookieKey = "privy-id-token"

// RefreshCookieKey is the key used to store the refresh token in the cookie
const RefreshCookieKey = "SPLITFI_REFRESH_JWT"

// ErrInvalidJWT is returned when the JWT is invalid
var ErrInvalidJWT = errors.New("invalid or expired auth token")

// ErrNoCookie is returned when there is no JWT in the request
var ErrNoCookie = errors.New("no jwt passed as cookie")

var ErrSessionInvalidated = errors.New("session has been invalidated")

type AuthResult struct {
	User     *db.User
	Email    *persist.Email
	PrivyDID *string
}

type ErrAuthenticationFailed struct {
	WrappedErr error
}

func (e ErrAuthenticationFailed) Unwrap() error {
	return e.WrappedErr
}

func (e ErrAuthenticationFailed) Error() string {
	return fmt.Sprintf("authentication failed: %s", e.WrappedErr.Error())
}

// GetAppIdFromCtx returns the session ID from the context
func GetAppIdFromCtx(c *gin.Context) string {
	return c.MustGet(appIdContextKey).(string)
}

// GetUserIdFromCtx returns the user ID from the context
func GetUserIdFromCtx(c *gin.Context) persist.DBID {
	return persist.DBID(c.MustGet(userIdContextKey).(string))
}

// GetAuthenticatedUserId returns the user ID from the context if the user is authenticated
func GetAuthenticatedUserId(ctx context.Context) (persist.DBID, error) {
	gc := util.MustGetGinContext(ctx)
	authError := GetAuthErrorFromCtx(gc)

	if authError != nil {
		return "", authError
	}

	userID := GetUserIdFromCtx(gc)
	return userID, nil
}

// GetLinkedAccountsFromCtx returns the linked accounts from the context
func GetLinkedAccountsFromCtx(c *gin.Context) []privy.LinkedAccount {
	return c.MustGet(linkedAccountsContextKey).([]privy.LinkedAccount)
}

// GetAuthenticatedLinkedAccounts returns the linked accounts from the context if the user is authenticated
func GetAuthenticatedLinkedAccounts(ctx context.Context) ([]privy.LinkedAccount, error) {
	gc := util.MustGetGinContext(ctx)
	authError := GetAuthErrorFromCtx(gc)

	if authError != nil {
		return []privy.LinkedAccount{}, authError
	}

	linkedAccounts := GetLinkedAccountsFromCtx(gc)
	return linkedAccounts, nil
}

// GetUserAuthedFromCtx queries the context to determine whether the user is authenticated
func GetUserAuthedFromCtx(c *gin.Context) bool {
	return c.GetBool(userAuthedContextKey)
}

func GetAuthErrorFromCtx(c *gin.Context) error {
	err := c.MustGet(authErrorContextKey)

	if err == nil {
		return nil
	}

	return err.(error)
}

func GetRolesFromCtx(c *gin.Context) []persist.Role {
	return c.MustGet(userRolesContextKey).([]persist.Role)
}

func setSessionStateForCtx(c *gin.Context, userId string, appId string, linkedAccounts []privy.LinkedAccount) {
	if userId == "" || appId == "" {
		logger.For(c).Errorf("attempted to set session state with missing values. userId: %s, appId: %s, linkedAccounts %v", userId, appId, linkedAccounts)
		err := errors.New("attempted to set session state with missing values")
		// We should never be trying to set a session with an empty userId or appId. If we find
		// ourselves here, clear the session and have the user log in again.
		clearSessionStateForCtx(c, err)
		clearSessionCookies(c)
		return
	}

	c.Set(userIdContextKey, userId)
	c.Set(appIdContextKey, appId)
	c.Set(linkedAccountsContextKey, linkedAccounts)
	c.Set(authErrorContextKey, nil)
	c.Set(userAuthedContextKey, true)
	c.Set(userRolesContextKey, []persist.Role{})
}

func clearSessionStateForCtx(c *gin.Context, err error) {
	c.Set(userIdContextKey, "")
	c.Set(appIdContextKey, "")
	c.Set(linkedAccountsContextKey, []privy.LinkedAccount{})
	c.Set(authErrorContextKey, err)
	c.Set(userAuthedContextKey, false)
	c.Set(userRolesContextKey, []persist.Role{})
}

// VerifySession checks the request cookies for an existing auth session.
// If the request is for an expired or invalid session, the user will be
// logged out. After calling VerifySession, the current auth state can be queried with
// functions like GetUserAuthedFromCtx(), GetUserIdFromCtx(), etc.
func VerifySession(c *gin.Context, queries *db.Queries, authRefreshCache *redis.Cache) error {
	// If the user has a valid auth cookie, we can set their auth state and be done
	// (unless something like updating roles triggered a forced refresh of the auth token)
	idClaims, idTokenErr := getAndParseIdToken(c)

	if idTokenErr != nil {
		clearSessionStateForCtx(c, idTokenErr)
		// The most common case here is that the user has no cookies at all, which is fine and expected.
		// If we encounter any other errors, log them and clear the user's cookies.
		if !errors.Is(idTokenErr, ErrNoCookie) {
			logger.For(c).Warnf("could not verify session: authTokenErr=%s", idTokenErr)
			clearSessionCookies(c)
		}

		return idTokenErr
	}

	setSessionStateForCtx(c, idClaims.UserId, idClaims.AppId, idClaims.LinkedAccounts)

	return nil
}

func clearSessionCookies(c *gin.Context) {
	clearCookie(c, AuthCookieKey)
	clearCookie(c, IdCookieKey)
	clearCookie(c, RefreshCookieKey)
}

func getAndParseIdToken(c *gin.Context) (IdTokenClaims, error) {
	idToken, err := getCookie(c, IdCookieKey)
	if err != nil {
		return IdTokenClaims{}, err
	}

	return ParseIdToken(c, idToken)
}

func getAndParseAuthToken(c *gin.Context) (AuthTokenClaims, error) {
	authToken, err := getCookie(c, AuthCookieKey)
	if err != nil {
		return AuthTokenClaims{}, err
	}

	return ParseAuthToken(c, authToken)
}

func getCookie(c *gin.Context, cookieName string) (string, error) {
	cookie, err := c.Cookie(cookieName)

	// Treat empty cookies the same way we treat missing cookies, since setting a cookie to the empty
	// string is how we "delete" them.
	if (err == nil && cookie == "") || errors.Is(err, http.ErrNoCookie) {
		err = ErrNoCookie
	}

	if err != nil {
		return "", err
	}

	return cookie, nil
}

func setCookie(c *gin.Context, cookieName string, value string) {
	mode := http.SameSiteStrictMode
	domain := ".mutuals.finance"
	httpOnly := true
	secure := true

	clientIsLocalhost := c.Request.Header.Get("Origin") == "http://localhost:3000"

	if env.GetString("ENV") != "production" || clientIsLocalhost {
		mode = http.SameSiteNoneMode
		domain = ""
		httpOnly = false
	}

	if env.GetString("ENV") == "local" {
		userAgent := c.GetHeader("User-Agent")

		// WebKit-based clients (e.g. Safari and our mobile app) won't set a secure cookie unless the
		// request uses HTTPS, but local development doesn't use HTTPS, so we need to disable secure
		// cookies for local environments when receiving requests from these platforms.

		// Mobile app
		if strings.Contains(userAgent, "MutualsLabs") && strings.Contains(userAgent, "Darwin") {
			secure = false
			logger.For(c).Info("Request is from mobile app, setting local auth cookie with secure=false")
		}

		// Safari mentions "Safari" in its User-Agent string, but it doesn't mention Chrome or Chromium.
		if strings.Contains(userAgent, "Safari") && !strings.Contains(userAgent, "Chrome") && !strings.Contains(userAgent, "Chromium") {
			secure = false
			logger.For(c).Info("Request is from Safari, setting local auth cookie with secure=false")
		}
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cookieName,
		Value:    value,
		Expires:  time.Now().Add(cookieExpires),
		Path:     "/",
		Secure:   secure,
		HttpOnly: httpOnly,
		SameSite: mode,
		Domain:   domain,
	})
}

func clearCookie(c *gin.Context, cookieName string) {
	setCookie(c, cookieName, "")
}
