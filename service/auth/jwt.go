package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/mutuals/go-mutuals/service/auth/privy"
)

type IdTokenClaims = privy.IdTokenClaims

func ParseIdToken(ctx context.Context, token string) (IdTokenClaims, error) {
	return privy.ParseIdToken(ctx, token)
}

// keyFunc returns a key function for verifying Privy ES256 tokens
func keyFunc(verificationKey string) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != "ES256" {
			return nil, fmt.Errorf("unexpected JWT signing method=%v", token.Header["alg"])
		}

		parsed := strings.ReplaceAll("-----BEGIN PUBLIC KEY-----\\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAERFWiFvDlBs1m5ZCaNNwzxizIlmOlYgdKw0DjFVsZhJYfTcOc/gqMfz8WEOJZginOWCfy/Cyydj9xg9xWtHxJpg==\\n-----END PUBLIC KEY-----", "\\n", "\n")
		key, err := jwt.ParseECPublicKeyFromPEM([]byte(parsed))
		if err != nil {
			return nil, err
		}
		return key, nil
	}
}
