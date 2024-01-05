package auth

import (
	"context"
	"strings"

	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/service/persist"
)

func RolesByUserID(ctx context.Context, queries *db.Queries, userID persist.DBID) ([]persist.Role, error) {
	return queries.GetUserRolesByUserId(ctx, db.GetUserRolesByUserIdParams{
		UserID: userID,
		// TODO add roles if any?
		//MembershipAddress:     persist.Address(membershipAddress),
		//MembershipTokenIds:    memberTokens,
		//GrantedMembershipRole: string(persist.RoleEarlyAccess), // Role granted if user carries a matching token
		//Chain:  persist.ChainETH,
	})
}

// parseAddressTokens returns a contract and tokens from a string encoded as '<address>=[<tokenID>,<tokenID>,...<tokenID>]'.
// It's helpful for parsing contract and tokens passed as environment variables.
func parseAddressTokens(s string) (string, []string) {
	addressTokens := strings.Split(s, "=")
	if len(addressTokens) != 2 {
		panic("invalid address tokens format")
	}
	address, tokens := addressTokens[0], addressTokens[1]
	tokens = strings.TrimLeft(tokens, "[")
	tokens = strings.TrimRight(tokens, "]")
	return address, strings.Split(tokens, ",")
}
