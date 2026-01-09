package privy

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v4"
)

type LinkedAccount struct {
	Id               string `json:"id"`
	Type             string `json:"type"`
	Address          string `json:"address"`
	ChainType        string `json:"chain_type,omitempty"`
	WalletClientType string `json:"wallet_client_type,omitempty"`
	Lv               int64  `json:"lv"`
}

type IdTokenClaims struct {
	UserId         string          `json:"sub,omitempty"`
	AppId          string          `json:"aud,omitempty"`
	LinkedAccounts []LinkedAccount `json:"linked_accounts,omitempty"`
	Expiration     uint64          `json:"exp,omitempty"`
	IssuedAt       uint64          `json:"iat,omitempty"`
	Issuer         string          `json:"iss,omitempty"`
	jwt.RegisteredClaims
}

func (c *IdTokenClaims) UnmarshalJSON(data []byte) error {
	type Alias IdTokenClaims
	aux := &struct {
		LinkedAccounts string `json:"linked_accounts,omitempty"`
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	if aux.LinkedAccounts != "" {
		if err := json.Unmarshal([]byte(aux.LinkedAccounts), &c.LinkedAccounts); err != nil {
			return fmt.Errorf("failed to unmarshal linked_accounts: %w", err)
		}
	}

	return nil
}

func ParseIdToken(ctx context.Context, token string) (IdTokenClaims, error) {
	claims := IdTokenClaims{}
	verificationKey := "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAERFWiFvDlBs1m5ZCaNNwzxizIlmOlYgdKw0DjFVsZhJYfTcOc/gqMfz8WEOJZginOWCfy/Cyydj9xg9xWtHxJpg==\n-----END PUBLIC KEY-----\n"
	//env.GetString("PRIVY_VERIFICATION_KEY")
	//fmt.Printf("PRIVY_VERIFICATION_KEY length: %d, content: %q\n", len(verificationKey), verificationKey)

	parsedToken, err := jwt.ParseWithClaims(token, &claims, keyFunc(verificationKey))

	if err != nil || !parsedToken.Valid {
		return IdTokenClaims{}, fmt.Errorf("invalid JWT: %w", err)
	}

	return claims, nil
}

func keyFunc(verificationKey string) jwt.Keyfunc {
	return func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != "ES256" {
			return nil, fmt.Errorf("unexpected JWT signing method=%v", token.Header["alg"])
		}

		parsed := strings.ReplaceAll(verificationKey, "\\n", "\n")
		key, err := jwt.ParseECPublicKeyFromPEM([]byte(parsed))
		if err != nil {
			return nil, err
		}
		return key, nil
	}
}
