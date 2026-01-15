package publicapi

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4"
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
	"github.com/mutuals/go-mutuals/service/emails"
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

	user, err := api.loaders.GetUserByIdBatch.Load(userId)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (api UserAPI) GetViewer(ctx context.Context) (*coredb.User, error) {
	userId := api.GetLoggedInUserId(ctx)
	return api.GetUserById(ctx, userId)
}

func (api UserAPI) VerifiedEmailAddressExists(ctx context.Context, emailAddress persist.Email) (bool, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"emailAddress": validate.WithTag(emailAddress, "required"),
	}); err != nil {
		return false, err
	}

	// Intentionally using queries here instead of a dataloader. Caching a user by email address is tricky
	// because the key (email address) isn't part of the user object, and this method isn't currently invoked
	// in a way that would benefit from dataloaders or caching anyway.
	/*_, err := api.queries.GetUserByVerifiedEmailAddress(ctx, emailAddress.String())

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}*/

	return true, nil
}

// GetUserWithPII returns the current user and their associated personally identifiable information
func (api UserAPI) GetUserWithPII(ctx context.Context) (*coredb.PiiUserView, error) {
	// Nothing to validate
	/*
		userId, err := getAuthenticatedUserId(ctx)
		if err != nil {
			return nil, err
		}

			userWithPII, err := api.queries.GetUserWithPIIById(ctx, userId)
			if err != nil {
				return nil, err
			}

			return &userWithPII, nil
	*/

	return nil, nil
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
		return api.queries.GetUsersByIds(ctx, coredb.GetUsersByIdsParams{
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

func (api UserAPI) GetUserByAddress(ctx context.Context, chainAddress persist.ChainAddress) (*coredb.User, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"chainAddress": validate.WithTag(chainAddress, "required"),
	}); err != nil {
		return nil, err
	}

	/*	dbUser, err := api.queries.GetUsersByIds(ctx, chainAddress.Address())
		if err != nil {
			return nil, err
		}
	*/
	return nil, nil
}

func (api *UserAPI) GetUserRolesByUserId(ctx context.Context, userId persist.DBID) ([]persist.Role, error) {
	return auth.RolesByUserId(ctx, api.queries, userId)
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
		return api.queries.GetUsersWithRolePaginate(ctx, coredb.GetUsersWithRolePaginateParams{
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

func (api UserAPI) UpdateUserInfo(ctx context.Context, username string) error {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"username": validate.WithTag(username, "required,username"),
	}); err != nil {
		return err
	}

	userId, err := getAuthenticatedUserId(ctx)
	if err != nil {
		return err
	}

	err = userService.UpdateUserInfo(ctx, userId, username, api.repos.UserRepository, api.ethClient)
	if err != nil {
		return err
	}

	return nil
}

func (api UserAPI) UpdateUserEmailWithManualVerification(ctx context.Context, email persist.Email) error {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"email": validate.WithTag(email, "required"),
	}); err != nil {
		return err
	}

	userId, err := getAuthenticatedUserId(ctx)
	if err != nil {
		return err
	}
	err = api.queries.UpdateUserUnverifiedEmail(ctx, coredb.UpdateUserUnverifiedEmailParams{
		UserID:       userId,
		EmailAddress: email,
	})
	if err != nil {
		return err
	}

	err = emails.RequestVerificationEmail(ctx, userId)
	if err != nil {
		return err
	}

	return nil
}

func (api UserAPI) UpdateUserEmail(ctx context.Context, email persist.Email) error {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"email": validate.WithTag(email, "required"),
	}); err != nil {
		return err
	}

	userId, err := getAuthenticatedUserId(ctx)
	if err != nil {
		return err
	}

	err = api.queries.UpdateUserVerifiedEmail(ctx, coredb.UpdateUserVerifiedEmailParams{
		UserID:       userId,
		EmailAddress: email,
	})

	if err != nil {
		return err
	}

	return nil
}

func (api UserAPI) UpdateUserEmailNotificationSettings(ctx context.Context, settings persist.EmailUnsubscriptions) error {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"settings": validate.WithTag(settings, "required"),
	}); err != nil {
		return err
	}

	userId, err := getAuthenticatedUserId(ctx)
	if err != nil {
		return err
	}

	// update unsubscriptions

	return emails.UpdateUnsubscriptionsByUserID(ctx, userId, settings)

}

func (api UserAPI) GetCurrentUserEmailNotificationSettings(ctx context.Context) (persist.EmailUnsubscriptions, error) {

	userId, err := getAuthenticatedUserId(ctx)
	if err != nil {
		return persist.EmailUnsubscriptions{}, err
	}

	// update unsubscriptions

	return emails.GetCurrentUnsubscriptionsByUserID(ctx, userId)

}

func (api UserAPI) ResendEmailVerification(ctx context.Context) error {

	userId, err := getAuthenticatedUserId(ctx)
	if err != nil {
		return err
	}

	err = emails.RequestVerificationEmail(ctx, userId)
	if err != nil {
		return err
	}

	return nil
}

func (api UserAPI) UpdateUserNotificationSettings(ctx context.Context, notificationSettings persist.UserNotificationSettings) error {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"notification_settings": validate.WithTag(notificationSettings, "required"),
	}); err != nil {
		return err
	}

	userId, err := getAuthenticatedUserId(ctx)
	if err != nil {
		return err
	}

	return api.queries.UpdateNotificationSettingsById(ctx, coredb.UpdateNotificationSettingsByIdParams{ID: userId, NotificationSettings: notificationSettings})
}

// CreatePushTokenForUser adds a push token to a user, or returns the existing push token if it's already been
// added to this user. If the token can't be added because it belongs to another user, an error is returned.
func (api UserAPI) CreatePushTokenForUser(ctx context.Context, pushToken string) (coredb.PushNotificationToken, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"pushToken": validate.WithTag(pushToken, "required,min=1,max=255"),
	}); err != nil {
		return coredb.PushNotificationToken{}, err
	}

	userId, err := getAuthenticatedUserId(ctx)
	if err != nil {
		return coredb.PushNotificationToken{}, err
	}

	// Does the token already exist?
	token, err := api.queries.GetPushTokenByPushToken(ctx, pushToken)

	if err == nil {
		// If the token exists and belongs to the current user, return it. Attempting to re-add
		// a token that you've already registered is a no-op.
		if token.UserID == userId {
			return token, nil
		}

		// Otherwise, the token belongs to another user. Return an error.
		return coredb.PushNotificationToken{}, persist.ErrPushTokenBelongsToAnotherUser{PushToken: pushToken}
	}

	// ErrNoRows is expected and means we can continue with creating the token. If we see any other
	// error, return it.
	if err != pgx.ErrNoRows {
		return coredb.PushNotificationToken{}, err
	}

	token, err = api.queries.CreatePushTokenForUser(ctx, coredb.CreatePushTokenForUserParams{
		ID:        persist.GenerateID(),
		UserID:    userId,
		PushToken: pushToken,
	})

	if err != nil {
		return coredb.PushNotificationToken{}, err
	}

	return token, nil
}

// DeletePushTokenByPushToken removes a push token from a user, or does nothing if the token doesn't exist.
// If the token can't be removed because it belongs to another user, an error is returned.
func (api UserAPI) DeletePushTokenByPushToken(ctx context.Context, pushToken string) error {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"pushToken": validate.WithTag(pushToken, "required,min=1,max=255"),
	}); err != nil {
		return err
	}

	userId, err := getAuthenticatedUserId(ctx)
	if err != nil {
		return err
	}

	existingToken, err := api.queries.GetPushTokenByPushToken(ctx, pushToken)
	if err == nil {
		// If the token exists and belongs to the current user, let them delete it.
		if existingToken.UserID == userId {
			return api.queries.DeletePushTokensByIds(ctx, []persist.DBID{existingToken.ID})
		}

		// Otherwise, the token belongs to another user. Return an error.
		return persist.ErrPushTokenBelongsToAnotherUser{PushToken: pushToken}
	}

	// ErrNoRows is okay and means the token doesn't exist. Unregistering it is a no-op
	// and doesn't return an error.
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}

	return err
}

func (api UserAPI) BlockUser(ctx context.Context, userID persist.DBID) error {
	// Validate
	viewerId, err := getAuthenticatedUserId(ctx)
	if err != nil {
		return err
	}
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"userID": validate.WithTag(userID, fmt.Sprintf("required,ne=%s", viewerId)),
	}); err != nil {
		return err
	}
	_, err = api.queries.BlockUser(ctx, coredb.BlockUserParams{
		ID:            persist.GenerateID(),
		UserID:        viewerId,
		BlockedUserID: userID,
	})
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		return persist.ErrUserNotFound{UserID: userID}
	}
	return err
}

func (api UserAPI) UnblockUser(ctx context.Context, userID persist.DBID) error {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"userID": validate.WithTag(userID, "required"),
	}); err != nil {
		return err
	}
	viewerID, err := getAuthenticatedUserId(ctx)
	if err != nil {
		return err
	}
	return api.queries.UnblockUser(ctx, coredb.UnblockUserParams{UserID: persist.DBID(viewerID), BlockedUserID: userID})
}
