package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/mutuals/go-mutuals/service/logger"

	"github.com/mutuals/go-mutuals/env"
	"github.com/mutuals/go-mutuals/service/persist"
)

type TokenType string

const (
	TokenTypeAuth              TokenType = "auth"
	TokenTypeRefresh           TokenType = "refresh"
	TokenTypeOneTimeLogin      TokenType = "one_time_login"
	TokenTypeEmailVerification TokenType = "email_verification"
)

type MutualsClaims struct {
	jwt.RegisteredClaims
}

type AuthTokenClaims struct {
	UserDID string `json:"sub,omitempty"`
	AppId   string `json:"aud,omitempty"`
	MutualsClaims
}

type RefreshTokenClaims struct {
	ID       string       `json:"id"`        // The refresh token's ID
	ParentID string       `json:"parent_id"` // The parent refresh token this child refresh token was generated from
	UserID   persist.DBID `json:"user_id"`
	AppID    persist.DBID `json:"session_id"` // The session this refresh token belongs to
	MutualsClaims
}

type oneTimeLoginClaims struct {
	UserID string `json:"user_id"`
	Source string `json:"source"`
	MutualsClaims
}

type emailVerificationClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	MutualsClaims
}

func GenerateAuthToken(ctx context.Context, userID persist.DBID, appID persist.DBID, refreshID string, roles []persist.Role) (string, error) {
	secret := env.GetString("AUTH_JWT_SECRET")
	validFor := time.Duration(env.GetInt64("AUTH_JWT_TTL")) * time.Second

	claims := AuthTokenClaims{
		UserDID:       userID.String(),
		AppId:         appID.String(),
		MutualsClaims: newMutualsClaims(TokenTypeAuth, validFor),
	}

	return generateJWT(claims, secret)
}

func ParseAuthToken(ctx context.Context, token string) (AuthTokenClaims, error) {
	claims := AuthTokenClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, &claims, keyFunc(env.GetString("PRIVY_AUTH_JWT_SECRET")))
	logger.For(ctx).Infof("token %s, secret %s; parsedToken %v; err %s", token, env.GetString("PRIVY_AUTH_JWT_SECRET"), parsedToken, err)

	if err != nil || !parsedToken.Valid {
		return AuthTokenClaims{}, ErrInvalidJWT
	}

	return claims, nil
}

func GenerateRefreshToken(ctx context.Context, ID string, parentID string, userID persist.DBID, appID persist.DBID) (string, time.Time, error) {
	secret := env.GetString("REFRESH_JWT_SECRET")
	validFor := time.Duration(env.GetInt64("REFRESH_JWT_TTL")) * time.Second

	claims := RefreshTokenClaims{
		ID:            ID,
		ParentID:      parentID,
		UserID:        userID,
		AppID:         appID,
		MutualsClaims: newMutualsClaims(TokenTypeRefresh, validFor),
	}

	jwt, err := generateJWT(claims, secret)
	expiresAt := time.Now().Add(validFor)

	return jwt, expiresAt, err
}

func ParseRefreshToken(ctx context.Context, token string) (RefreshTokenClaims, error) {
	claims := RefreshTokenClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, &claims, keyFunc(env.GetString("REFRESH_JWT_SECRET")))

	if err != nil || !parsedToken.Valid {
		return RefreshTokenClaims{}, ErrInvalidJWT
	}

	return claims, nil
}

func ParseOneTimeLoginToken(ctx context.Context, token string) (persist.DBID, time.Time, error) {
	claims := oneTimeLoginClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, &claims, keyFunc(env.GetString("ONE_TIME_LOGIN_JWT_SECRET")))

	if err != nil || !parsedToken.Valid {
		return "", time.Time{}, ErrInvalidJWT
	}

	return persist.DBID(claims.UserID), claims.ExpiresAt.Time, nil
}

func GenerateEmailVerificationToken(ctx context.Context, userID persist.DBID, email string) (string, error) {
	secret := env.GetString("EMAIL_VERIFICATION_JWT_SECRET")
	validFor := time.Duration(env.GetInt64("EMAIL_VERIFICATION_JWT_TTL")) * time.Second

	claims := emailVerificationClaims{
		UserID:        userID.String(),
		Email:         email,
		MutualsClaims: newMutualsClaims(TokenTypeEmailVerification, validFor),
	}

	return generateJWT(claims, secret)
}

func ParseEmailVerificationToken(ctx context.Context, token string) (persist.DBID, string, error) {
	claims := emailVerificationClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, &claims, keyFunc(env.GetString("EMAIL_VERIFICATION_JWT_SECRET")))

	if err != nil || !parsedToken.Valid {
		return "", "", ErrInvalidJWT
	}

	return persist.DBID(claims.UserID), claims.Email, nil
}

func newMutualsClaims(tokenType TokenType, validFor time.Duration) MutualsClaims {
	claims := MutualsClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(validFor)),
			Issuer:    "mutuals",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	return claims
}

func generateJWT(claims jwt.Claims, jwtSecret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	jwtToken, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", err
	}

	return jwtToken, nil
}

func keyFunc(verificationKey string) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {

		if token.Method.Alg() != "ES256" {
			return nil, fmt.Errorf("unexpected JWT signing method=%v", token.Header["alg"])
		}
		// https://pkg.go.dev/github.com/dgrijalva/jwt-go#ParseECPublicKeyFromPEM
		key, err := jwt.ParseECPublicKeyFromPEM([]byte("-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAERFWiFvDlBs1m5ZCaNNwzxizIlmOlYgdKw0DjFVsZhJYfTcOc/gqMfz8WEOJZginOWCfy/Cyydj9xg9xWtHxJpg==\n-----END PUBLIC KEY-----"))
		if err != nil {
			return nil, err
		}
		return key, nil
	}
}
