package auth

import (
	"context"
	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/service/persist"
)

func RolesByUserID(ctx context.Context, queries *db.Queries, userID persist.DBID) ([]persist.Role, error) {
	return queries.GetUserRolesByUserId(ctx, userID)
}
