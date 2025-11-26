package auth

import (
	"context"

	db "github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/service/persist"
)

func RolesByUserId(ctx context.Context, queries *db.Queries, userID persist.DBID) ([]persist.Role, error) {
	return queries.GetUserRolesByUserId(ctx, userID)
}
