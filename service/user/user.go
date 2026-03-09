package user

import (
	"context"
	"time"

	"github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/graphql/dataloader"
	"github.com/mutuals/go-mutuals/service/auth"
	"github.com/mutuals/go-mutuals/service/persist"
	"github.com/mutuals/go-mutuals/util"
)

// CreateUser creates a new user with associated linked accounts based on privy auth data in the context
func CreateUser(ctx context.Context, queries *coredb.Queries) (user coredb.User, err error) {
	gc := util.MustGetGinContext(ctx)
	userId := auth.GetUserIdFromCtx(gc)
	linkedAccounts := auth.GetLinkedAccountsFromCtx(gc)

	params := coredb.CreateUserParams{UserID: userId}
	for _, a := range linkedAccounts {
		params.ID = append(params.ID, a.Id)
		params.Type = append(params.Type, a.Type)
		params.Address = append(params.Address, a.Address)
		params.ChainType = append(params.ChainType, a.ChainType)
		params.WalletClientType = append(params.WalletClientType, a.WalletClientType)
		params.LinkedAt = append(params.LinkedAt, time.Unix(a.Lv, 0))
	}

	user, err = queries.CreateUser(ctx, params)
	if err != nil {
		return coredb.User{}, err
	}

	return user, nil
}

type GetUserByIdInput struct {
	UserId persist.DBID
}

// GetUserById retrieves a user by their ID using the dataloader
func GetUserById(ctx context.Context, loaders *dataloader.Loaders, input GetUserByIdInput) (*coredb.User, error) {
	user, err := loaders.GetUserByIdBatch.Load(input.UserId)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

type GetUsersByIdsInput struct {
	UserIds       []persist.DBID
	Limit         int32
	CurBeforeTime time.Time
	CurBeforeID   persist.DBID
	CurAfterTime  time.Time
	CurAfterID    persist.DBID
	PagingForward bool
}

// GetUsersByIds retrieves multiple users by their IDs with pagination
func GetUsersByIds(ctx context.Context, queries *coredb.Queries, input GetUsersByIdsInput) ([]coredb.User, error) {
	return queries.GetUsersByIds(ctx, coredb.GetUsersByIdsParams{
		Limit:         input.Limit,
		UserIds:       input.UserIds,
		CurBeforeTime: input.CurBeforeTime,
		CurBeforeID:   input.CurBeforeID,
		CurAfterTime:  input.CurAfterTime,
		CurAfterID:    input.CurAfterID,
		PagingForward: input.PagingForward,
	})
}

type GetUserByAddressInput struct {
	Address persist.Address
}

// GetUserByAddress retrieves a user by their address (currently not implemented)
func GetUserByAddress(ctx context.Context, queries *coredb.Queries, input GetUserByAddressInput) (*coredb.User, error) {
	// TODO: Implement this function when needed
	return nil, nil
}

type GetUserRolesByUserIdInput struct {
	UserId persist.DBID
}

// GetUserRolesByUserId retrieves the roles for a specific user
func GetUserRolesByUserId(ctx context.Context, queries *coredb.Queries, input GetUserRolesByUserIdInput) ([]persist.Role, error) {
	return auth.RolesByUserId(ctx, queries, input.UserId)
}

type PaginateUsersWithRoleInput struct {
	Role          persist.Role
	Limit         int32
	CurBeforeKey  string
	CurBeforeID   persist.DBID
	CurAfterKey   string
	CurAfterID    persist.DBID
	PagingForward bool
}

// PaginateUsersWithRole retrieves users with a specific role, with pagination
func PaginateUsersWithRole(ctx context.Context, queries *coredb.Queries, input PaginateUsersWithRoleInput) ([]coredb.User, error) {
	return queries.GetUsersWithRolePaginate(ctx, coredb.GetUsersWithRolePaginateParams{
		Role:          input.Role,
		Limit:         input.Limit,
		CurBeforeKey:  input.CurBeforeKey,
		CurBeforeID:   input.CurBeforeID,
		CurAfterKey:   input.CurAfterKey,
		CurAfterID:    input.CurAfterID,
		PagingForward: input.PagingForward,
	})
}

type BlockUserInput struct {
	ViewerId persist.DBID
	UserId   persist.DBID
}

// BlockUser blocks a user for the authenticated viewer
func BlockUser(ctx context.Context, queries *coredb.Queries, input BlockUserInput) (blocklist coredb.UserBlocklist, err error) {
	blocklist, err = queries.BlockUser(ctx, coredb.BlockUserParams{
		ID:            persist.GenerateID(),
		UserID:        input.ViewerId,
		BlockedUserID: input.UserId,
	})
	if err != nil {
		return coredb.UserBlocklist{}, err
	}
	return blocklist, nil
}

type UnblockUserInput struct {
	ViewerId persist.DBID
	UserId   persist.DBID
}

// UnblockUser unblocks a user for the authenticated viewer
func UnblockUser(ctx context.Context, queries *coredb.Queries, input UnblockUserInput) (blocklist coredb.UserBlocklist, err error) {
	return queries.UnblockUser(ctx, coredb.UnblockUserParams{UserID: input.ViewerId, BlockedUserID: input.UserId})
}
