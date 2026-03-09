package publicapi

import (
	"context"
	"fmt"
	"time"

	"github.com/mutuals/go-mutuals/event"
	"github.com/mutuals/go-mutuals/service/redis"
	"github.com/mutuals/go-mutuals/service/task"

	"github.com/mutuals/go-mutuals/service/logger"
	"github.com/mutuals/go-mutuals/service/multichain"
	"github.com/mutuals/go-mutuals/service/persist/postgres"
	userService "github.com/mutuals/go-mutuals/service/user"

	"cloud.google.com/go/storage"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/everFinance/goar"
	"github.com/go-playground/validator/v10"
	shell "github.com/ipfs/go-ipfs-api"
	"github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/graphql/dataloader"
	"github.com/mutuals/go-mutuals/service/auth"
	"github.com/mutuals/go-mutuals/service/persist"
	"github.com/mutuals/go-mutuals/util"
	"github.com/mutuals/go-mutuals/validate"
)

type UserAPI struct {
	repos              *postgres.Repositories
	queries            *coredb.Queries
	loaders            *dataloader.Loaders
	validator          *validator.Validate
	ethClient          *ethclient.Client
	ipfsClient         *shell.Shell
	arweaveClient      *goar.Client
	storageClient      *storage.Client
	multichainProvider *multichain.Provider
	taskClient         *task.Client
	cache              *redis.Cache
}

func (api UserAPI) GetLoggedInUserId(ctx context.Context) persist.DBID {
	gc := util.MustGetGinContext(ctx)

	return auth.GetUserIdFromCtx(gc)
}

func (api UserAPI) IsUserLoggedIn(ctx context.Context) bool {
	gc := util.MustGetGinContext(ctx)

	logger.For(ctx).Infof("%v", gc.Request.Header)
	logger.For(ctx).Infof("%v", gc.Request.Host)

	return auth.GetUserAuthedFromCtx(gc)
}

func (api UserAPI) GetUserById(ctx context.Context, userId persist.DBID) (*coredb.User, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"userId": validate.WithTag(userId, "required"),
	}); err != nil {
		return nil, err
	}

	return userService.GetUserById(ctx, api.loaders, userService.GetUserByIdInput{UserId: userId})
}

func (api UserAPI) GetViewer(ctx context.Context) (*coredb.User, error) {
	userId := api.GetLoggedInUserId(ctx)
	return api.GetUserById(ctx, userId)
}

func (api UserAPI) GetUsersByIds(ctx context.Context, userIds []persist.DBID, before, after *string, first, last *int) ([]coredb.User, PageInfo, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"userIds": validate.WithTag(userIds, "required"),
	}); err != nil {
		return nil, PageInfo{}, err
	}

	if err := validatePaginationParams(api.validator, first, last); err != nil {
		return nil, PageInfo{}, err
	}

	queryFunc := func(params timeIDPagingParams) ([]coredb.User, error) {
		return userService.GetUsersByIds(ctx, api.queries, userService.GetUsersByIdsInput{
			Limit:         params.Limit,
			UserIds:       userIds,
			CurBeforeTime: params.CursorBeforeTime,
			CurBeforeID:   params.CursorBeforeID,
			CurAfterTime:  params.CursorAfterTime,
			CurAfterID:    params.CursorAfterID,
			PagingForward: params.PagingForward,
		})
	}

	countFunc := func() (int, error) {
		return len(userIds), nil
	}

	cursorFunc := func(u coredb.User) (time.Time, persist.DBID, error) {
		return u.CreatedAt, u.ID, nil
	}

	paginator := timeIDPaginator[coredb.User]{
		QueryFunc:  queryFunc,
		CursorFunc: cursorFunc,
		CountFunc:  countFunc,
	}

	return paginator.paginate(before, after, first, last)
}

func (api UserAPI) paginatorFromCursorStr(ctx context.Context, curStr string) (positionPaginator[coredb.User], error) {
	cur := cursors.NewPositionCursor()
	err := cur.Unpack(curStr)
	if err != nil {
		return positionPaginator[coredb.User]{}, err
	}
	return api.paginatorFromCursor(ctx, cur), nil
}

func (api UserAPI) paginatorFromCursor(ctx context.Context, c *positionCursor) positionPaginator[coredb.User] {
	return api.paginatorWithQuery(c, func(p positionPagingParams) ([]coredb.User, error) {
		params := coredb.GetUsersByPositionPaginateBatchParams{
			UserIds: util.MapWithoutError(c.IDs, func(id persist.DBID) string { return id.String() }),
			// Postgres uses 1-based indexing
			CurBeforePos: p.CursorBeforePos + 1,
			CurAfterPos:  p.CursorAfterPos + 1,
		}
		return api.loaders.GetUsersByPositionPaginateBatch.Load(params)
	})
}

func (api UserAPI) paginatorFromResults(ctx context.Context, c *positionCursor, users []coredb.User) positionPaginator[coredb.User] {
	queryF := func(positionPagingParams) ([]coredb.User, error) { return users, nil }
	return api.paginatorWithQuery(c, queryF)
}

func (api UserAPI) paginatorWithQuery(c *positionCursor, queryF func(positionPagingParams) ([]coredb.User, error)) positionPaginator[coredb.User] {
	var paginator positionPaginator[coredb.User]
	paginator.QueryFunc = queryF
	paginator.CursorFunc = func(u coredb.User) (int64, []persist.DBID, error) { return c.Positions[u.ID], c.IDs, nil }
	return paginator
}

func (api UserAPI) GetUserByAddress(ctx context.Context, address persist.Address) (*coredb.User, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"address": validate.WithTag(address, "required"),
	}); err != nil {
		return nil, err
	}

	return userService.GetUserByAddress(ctx, api.queries, userService.GetUserByAddressInput{Address: address})
}

func (api *UserAPI) GetUserRolesByUserId(ctx context.Context, userId persist.DBID) ([]persist.Role, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"userId": validate.WithTag(userId, "required"),
	}); err != nil {
		return nil, err
	}

	return userService.GetUserRolesByUserId(ctx, api.queries, userService.GetUserRolesByUserIdInput{UserId: userId})
}

func (api *UserAPI) UserIsAdmin(ctx context.Context) bool {
	for _, role := range getUserRoles(ctx) {
		if role == persist.RoleAdmin {
			return true
		}
	}
	return false
}

func (api UserAPI) PaginateUsersWithRole(ctx context.Context, role persist.Role, before *string, after *string, first *int, last *int) ([]coredb.User, PageInfo, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"role": validate.WithTag(role, "required,role"),
	}); err != nil {
		return nil, PageInfo{}, err
	}

	if err := validatePaginationParams(api.validator, first, last); err != nil {
		return nil, PageInfo{}, err
	}

	queryFunc := func(params lexicalPagingParams) ([]coredb.User, error) {
		return userService.PaginateUsersWithRole(ctx, api.queries, userService.PaginateUsersWithRoleInput{
			Role:          role,
			Limit:         params.Limit,
			CurBeforeKey:  params.CursorBeforeKey,
			CurBeforeID:   params.CursorBeforeID,
			CurAfterKey:   params.CursorAfterKey,
			CurAfterID:    params.CursorAfterID,
			PagingForward: params.PagingForward,
		})
	}

	cursorFunc := func(u coredb.User) (string, persist.DBID, error) {
		return u.ID.String(), u.ID, nil
	}

	paginator := lexicalPaginator[coredb.User]{
		QueryFunc:  queryFunc,
		CursorFunc: cursorFunc,
	}

	return paginator.paginate(before, after, first, last)
}

func (api UserAPI) CreateUser(ctx context.Context) (user coredb.User, err error) {
	user, err = userService.CreateUser(ctx, api.queries)
	if err != nil {
		return coredb.User{}, err
	}

	// Send event
	err = event.Dispatch(ctx, coredb.Event{
		ActorID:        persist.DBIDToNullStr(user.ID),
		Action:         persist.ActionUserCreated,
		ResourceTypeID: persist.ResourceTypeUser,
		UserID:         user.ID,
		SubjectID:      user.ID,
		Data:           persist.EventData{},
	})

	if err != nil {
		logger.For(ctx).Errorf("failed to dispatch event: %s", err)
	}

	return user, nil
}

// BlockUser adds the specified user to the authenticated user's blocklist
func (api UserAPI) BlockUser(ctx context.Context, userId persist.DBID) (user coredb.UserBlocklist, err error) {
	viewerId, err := getAuthenticatedUserId(ctx)
	if err != nil {
		return coredb.UserBlocklist{}, err
	}

	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"userId": validate.WithTag(userId, fmt.Sprintf("required,ne=%s", viewerId)),
	}); err != nil {
		return coredb.UserBlocklist{}, err
	}

	return userService.BlockUser(ctx, api.queries, userService.BlockUserInput{ViewerId: viewerId, UserId: userId})
}

// UnblockUser removes the specified user from the authenticated user's blocklist
func (api UserAPI) UnblockUser(ctx context.Context, userId persist.DBID) (user coredb.UserBlocklist, err error) {
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"userId": validate.WithTag(userId, "required"),
	}); err != nil {
		return coredb.UserBlocklist{}, err
	}

	viewerId, err := getAuthenticatedUserId(ctx)
	if err != nil {
		return coredb.UserBlocklist{}, err
	}

	return userService.UnblockUser(ctx, api.queries, userService.UnblockUserInput{ViewerId: viewerId, UserId: userId})
}
