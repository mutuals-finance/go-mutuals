package adminapi

import (
	"context"
	"fmt"

	"github.com/mutuals/go-mutuals/service/auth/basicauth"
	"github.com/mutuals/go-mutuals/service/redis"

	"github.com/go-playground/validator/v10"
	db "github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/service/auth"
	"github.com/mutuals/go-mutuals/service/persist/postgres"
)

type AdminAPI struct {
	repos            *postgres.Repositories
	queries          *db.Queries
	authRefreshCache *redis.Cache
	validator        *validator.Validate
}

func NewAPI(repos *postgres.Repositories, queries *db.Queries, authRefreshCache *redis.Cache, validator *validator.Validate) *AdminAPI {
	return &AdminAPI{repos, queries, authRefreshCache, validator}
}

type authenticator struct {
	authMethod func(context.Context) (*auth.AuthResult, error)
}

func (a authenticator) GetDescription() string                           { return "" }
func (a authenticator) UserRegistered(ctx context.Context) (bool, error) { return false, nil }
func (a authenticator) Authenticate(ctx context.Context) (*auth.AuthResult, error) {
	return a.authMethod(ctx)
}

func requireRetoolAuthorized(ctx context.Context) {
	requireBasicAuth(ctx, []basicauth.AuthTokenType{basicauth.AuthTokenTypeRetool})
}

func requireBasicAuth(ctx context.Context, allowedTypes []basicauth.AuthTokenType) {
	if !basicauth.AuthorizeHeaderForAllowedTypes(ctx, allowedTypes) {
		panic(fmt.Errorf("basic auth: not authorized for allowedTypes: %v", allowedTypes))
	}
}
