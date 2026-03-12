//go:build debug_tools
// +build debug_tools

// debugtools_enabled.go is only compiled when the `debug_tools` build tag is set.
// Anything that should be debug-only can be added here. Additionally, because the
// 'Enabled' bool is a const, code in other files that is conditional on Enabled
// will also be compiled out of builds.

package debugtools

import (
	"context"
	"crypto/subtle"
	"errors"

	"github.com/mutuals/go-mutuals/env"
	"github.com/mutuals/go-mutuals/service/auth"
)

const Enabled bool = true

func init() {
	// An additional safeguard against running debug tools in production
	if env.GetString("ENV") == "production" {
		panic(errors.New("debug tools may not be enabled in a production environment"))
	}
}

func isValidPassword(password string) bool {
	envPassword := env.GetString("DEBUG_TOOLS_PASSWORD")
	if env.GetString("ENV") != "local" && envPassword == "" {
		panic(errors.New("debug tools password must be set in non-local environments"))
	}
	return subtle.ConstantTimeCompare([]byte(envPassword), []byte(password)) == 1
}

func (d DebugAuthenticator) Authenticate(ctx context.Context) (*auth.AuthResult, error) {
	if !IsDebugEnv() {
		return nil, errors.New("DebugAuthenticator may only be used in a local and development environments")
	}
	if !isValidPassword(d.DebugToolsPassword) {
		return nil, errors.New("invalid debug tools password")
	}
	return &auth.AuthResult{User: d.User}, nil
}
