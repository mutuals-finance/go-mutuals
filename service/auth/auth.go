package auth

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	db "github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/service/redis"

	"github.com/magiclabs/magic-admin-go"
	magicclient "github.com/magiclabs/magic-admin-go/client"
	"github.com/magiclabs/magic-admin-go/token"
	"github.com/mutuals/go-mutuals/env"

	"github.com/mutuals/go-mutuals/service/logger"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/mutuals/go-mutuals/service/multichain"
	"github.com/mutuals/go-mutuals/service/persist"
)

// AuthenticatedAddress contains address information that has been successfully verified
// by an authenticator.
type AuthenticatedAddress struct {
	// A ChainAddress that has had its ownership successfully verified by an authenticator
	ChainAddress persist.ChainAddress

	// The WalletType of the verified ChainAddress
	WalletType persist.WalletType
}

const (
	// Context keys for auth data
	userAuthedContextKey = "auth.authenticated"
	userIdContextKey     = "auth.user_id"
	appIDContextKey      = "auth.app_id"
	authErrorContextKey  = "auth.auth_error"
	userRolesContextKey  = "auth.roles"
)

const cookieExpires = 1 * time.Hour

// NoncePrepend is prepended to a nonce to make our default signing message
const NoncePrepend = "Mutuals uses this cryptographic signature in place of a password: "

// AuthCookieKey is the key used to store the auth token in the cookie
const AuthCookieKey = "privy-token"

// RefreshCookieKey is the key used to store the refresh token in the cookie
const RefreshCookieKey = "SPLITFI_REFRESH_JWT"

// ErrNonceMismatch is returned when the nonce does not match the expected nonce
var ErrNonceMismatch = errors.New("incorrect nonce input")

// ErrMessageDoesNotContainNonce is returned when a nonce authenticator's message does not contain its nonce
var ErrMessageDoesNotContainNonce = errors.New("message does not contain nonce")

// ErrInvalidJWT is returned when the JWT is invalid
var ErrInvalidJWT = errors.New("invalid or expired auth token")

// ErrNoCookie is returned when there is no JWT in the request
var ErrNoCookie = errors.New("no jwt passed as cookie")

var ErrSessionInvalidated = errors.New("session has been invalidated")

// ErrSignatureInvalid is returned when the signed nonce's signature is invalid
var ErrSignatureInvalid = errors.New("signature invalid")

var ErrInvalidMagicLink = errors.New("invalid magic link")

// ErrEmailUnverified
// TODO: Figure out a better scheme for handling user-facing errors
var ErrEmailUnverified = errors.New("The email address you provided is unverified. Login with QR code instead, or verify your email at gallery.so/settings.")

var ErrEmailAlreadyUsed = errors.New("email already in use")

type Authenticator interface {
	// GetDescription returns information about the authenticator for error and logging purposes.
	// NOTE: GetDescription should NOT include any sensitive data (passwords, auth tokens, etc)
	// that we wouldn't want showing up in logs!
	GetDescription() string

	Authenticate(context.Context) (*AuthResult, error)

	UserRegistered(context.Context) (bool, error)
}

type AuthResult struct {
	User      *db.User
	Addresses []AuthenticatedAddress
	Email     *persist.Email
	PrivyDID  *string
}

func (a *AuthResult) GetAuthenticatedAddress(chainAddress persist.ChainAddress) (AuthenticatedAddress, bool) {
	for _, address := range a.Addresses {
		if address.ChainAddress == chainAddress {
			return address, true
		}
	}

	return AuthenticatedAddress{}, false
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

type ErrSignatureVerificationFailed struct {
	WrappedErr error
}

func (e ErrSignatureVerificationFailed) Unwrap() error {
	return e.WrappedErr
}

func (e ErrSignatureVerificationFailed) Error() string {
	return fmt.Sprintf("signature verification failed: %s", e.WrappedErr.Error())
}

type ErrDoesNotOwnRequiredNFT struct {
	addresses []persist.ChainAddress
}

func (e ErrDoesNotOwnRequiredNFT) Error() string {
	return fmt.Sprintf("required tokens not owned by any addresses: %s", e.addresses)
}

type ErrNonceNotFound struct {
	L1ChainAddress persist.L1ChainAddress
}

func (e ErrNonceNotFound) Error() string {
	return fmt.Sprintf("nonce not found for address: %s", e.L1ChainAddress)
}

// GenerateNonce generates a random nonce to be signed by a wallet
func GenerateNonce() (string, error) {
	nonceBytes := make([]byte, 16)
	_, err := rand.Read(nonceBytes)
	if err != nil {
		return "", err
	}
	// Encode to a hex string
	nonceStr := hex.EncodeToString(nonceBytes)
	return nonceStr, nil
}

type NonceAuthenticator struct {
	ChainPubKey        persist.ChainPubKey
	Nonce              string
	Message            string
	Signature          string
	WalletType         persist.WalletType
	EthClient          *ethclient.Client
	MultichainProvider *multichain.Provider
	Queries            *db.Queries
}

func (e NonceAuthenticator) GetDescription() string {
	return fmt.Sprintf("NonceAuthenticator(address: %s, nonce: %s, message: %s, signature: %s, walletType: %v)", e.ChainPubKey, e.Nonce, e.Message, e.Signature, e.WalletType)
}

type MagicLinkAuthenticator struct {
	Token       token.Token
	MagicClient *magicclient.API
	Queries     *db.Queries
}

func NewMagicLinkClient() *magicclient.API {
	return magicclient.New(env.GetString("MAGIC_LINK_SECRET_KEY"), magic.NewDefaultClient())
}

type OneTimeLoginTokenAuthenticator struct {
	ConsumedTokenCache *redis.Cache
	Queries            *db.Queries
	LoginToken         string
}

func (a OneTimeLoginTokenAuthenticator) GetDescription() string {
	return "OneTimeLoginTokenAuthenticator"
}

func (a OneTimeLoginTokenAuthenticator) Authenticate(ctx context.Context) (*AuthResult, error) {
	userId, expiresAt, err := ParseOneTimeLoginToken(ctx, a.LoginToken)
	if err != nil {
		return nil, err
	}

	// Use redis to stop this token from being used again (and add an extra minute to the TTL account for clock differences)
	ttl := time.Until(expiresAt) + time.Minute
	success, err := a.ConsumedTokenCache.SetNX(ctx, a.LoginToken, []byte{1}, ttl)
	if err != nil {
		return nil, err
	}

	if !success {
		return nil, errors.New("token already used")
	}

	user, err := a.Queries.GetUserById(ctx, userId)
	if err != nil {
		return nil, err
	}

	authResult := AuthResult{
		Addresses: []AuthenticatedAddress{},
		User:      &user,
	}

	return &authResult, nil
}

func (a OneTimeLoginTokenAuthenticator) UserRegistered(ctx context.Context) (bool, error) {
	// TODO: implement
	return false, nil
}

// GetAppIDFromCtx returns the session ID from the context
func GetAppIDFromCtx(c *gin.Context) string {
	return c.MustGet(appIDContextKey).(string)
}

// GetUserIdFromCtx returns the user ID from the context
func GetUserIdFromCtx(c *gin.Context) persist.DBID {
	return c.MustGet(userIdContextKey).(persist.DBID)
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

func setSessionStateForCtx(c *gin.Context, userId string, appID string) {
	if userId == "" || appID == "" {
		logger.For(c).Errorf("attempted to set session state with missing values. userId: %s, appID: %s", userId, appID)
		err := errors.New("attempted to set session state with missing values")
		// We should never be trying to set a session with an empty userId or appID. If we find
		// ourselves here, clear the session and have the user log in again.
		clearSessionStateForCtx(c, err)
		clearSessionCookies(c)
		return
	}

	c.Set(userIdContextKey, userId)
	c.Set(appIDContextKey, appID)
	c.Set(authErrorContextKey, nil)
	c.Set(userAuthedContextKey, true)
	c.Set(userRolesContextKey, []persist.Role{})
}

func clearSessionStateForCtx(c *gin.Context, err error) {
	c.Set(userIdContextKey, "")
	c.Set(appIDContextKey, "")
	c.Set(authErrorContextKey, err)
	c.Set(userAuthedContextKey, false)
	c.Set(userRolesContextKey, []persist.Role{})
}

// ForceAuthTokenRefresh should be called whenever something happens that would result in existing auth
// tokens being out-of-date. For example, when a user's roles are changed, or a user logs out of a session,
// existing otherwise-valid auth tokens should be refreshed so they have the latest session state.
func ForceAuthTokenRefresh(ctx context.Context, authRefreshCache *redis.Cache, userId persist.DBID) error {
	// Keep the key long enough for any existing auth tokens to expire, plus an extra minute of wiggle room
	expiration := time.Duration(env.GetInt64("AUTH_JWT_TTL"))*time.Second + time.Minute
	return authRefreshCache.SetTime(ctx, userId.String(), time.Now(), expiration, true)
}

func mustRefreshAuthToken(ctx context.Context, authRefreshCache *redis.Cache, userId persist.DBID, issuedAt time.Time) bool {
	forceRefreshBefore, err := authRefreshCache.GetTime(ctx, userId.String())
	if err != nil {
		// If there's no key for this user, we don't need to force an auth token refresh
		var notFound redis.ErrKeyNotFound
		if errors.As(err, &notFound) {
			return false
		}

		// If we couldn't hit the redis cache, assume we need to refresh the auth token
		logger.For(ctx).Errorf("error checking auth refresh cache: %s", err)
		return true
	}

	return issuedAt.Before(forceRefreshBefore)
}

// VerifySession checks the request cookies for an existing auth session.
// If the request is for an expired or invalid session, the user will be
// logged out. After calling VerifySession, the current auth state can be queried with
// functions like GetUserAuthedFromCtx(), GetUserIdFromCtx(), etc.
func VerifySession(c *gin.Context, queries *db.Queries, authRefreshCache *redis.Cache) error {
	// If the user has a valid auth cookie, we can set their auth state and be done
	// (unless something like updating roles triggered a forced refresh of the auth token)
	authClaims, authTokenErr := getAndParseAuthToken(c)

	if authTokenErr != nil {
		clearSessionStateForCtx(c, authTokenErr)
		// The most common case here is that the user has no cookies at all, which is fine and expected.
		// If we encounter any other errors, log them and clear the user's cookies.
		if !errors.Is(authTokenErr, ErrNoCookie) {
			logger.For(c).Warnf("could not verify session: authTokenErr=%s", authTokenErr)
			clearSessionCookies(c)
		}

		return authTokenErr
	}

	setSessionStateForCtx(c, authClaims.UserId, authClaims.AppId)

	return nil
}

func clearSessionCookies(c *gin.Context) {
	clearCookie(c, AuthCookieKey)
	clearCookie(c, RefreshCookieKey)
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
