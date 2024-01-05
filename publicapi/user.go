package publicapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SplitFi/go-splitfi/event"
	"github.com/SplitFi/go-splitfi/service/redis"
	sentryutil "github.com/SplitFi/go-splitfi/service/sentry"
	"github.com/SplitFi/go-splitfi/service/task"
	"github.com/jackc/pgx/v4"
	"time"

	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/multichain"
	"github.com/SplitFi/go-splitfi/service/persist/postgres"
	"github.com/SplitFi/go-splitfi/service/user"
	"github.com/jackc/pgtype"

	"cloud.google.com/go/storage"
	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/graphql/dataloader"
	"github.com/SplitFi/go-splitfi/graphql/model"
	"github.com/SplitFi/go-splitfi/service/auth"
	"github.com/SplitFi/go-splitfi/service/emails"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/SplitFi/go-splitfi/validate"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/everFinance/goar"
	"github.com/go-playground/validator/v10"
	shell "github.com/ipfs/go-ipfs-api"
)

type UserAPI struct {
	repos              *postgres.Repositories
	queries            *db.Queries
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
	return auth.GetUserIDFromCtx(gc)
}

func (api UserAPI) IsUserLoggedIn(ctx context.Context) bool {
	gc := util.MustGetGinContext(ctx)
	return auth.GetUserAuthedFromCtx(gc)
}

func (api UserAPI) GetUserById(ctx context.Context, userID persist.DBID) (*db.User, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"userID": validate.WithTag(userID, "required"),
	}); err != nil {
		return nil, err
	}

	user, err := api.loaders.GetUserByIdBatch.Load(userID)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (api UserAPI) GetUserByVerifiedEmailAddress(ctx context.Context, emailAddress persist.Email) (*db.User, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"emailAddress": validate.WithTag(emailAddress, "required"),
	}); err != nil {
		return nil, err
	}

	// Intentionally using queries here instead of a dataloader. Caching a user by email address is tricky
	// because the key (email address) isn't part of the user object, and this method isn't currently invoked
	// in a way that would benefit from dataloaders or caching anyway.
	user, err := api.queries.GetUserByVerifiedEmailAddress(ctx, emailAddress.String())

	if err != nil {
		if err == pgx.ErrNoRows {
			err = persist.ErrUserNotFound{Email: emailAddress}
		}
		return nil, err
	}

	return &user, nil
}

// GetUserWithPII returns the current user and their associated personally identifiable information
func (api UserAPI) GetUserWithPII(ctx context.Context) (*db.PiiUserView, error) {
	// Nothing to validate

	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}

	userWithPII, err := api.queries.GetUserWithPIIByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &userWithPII, nil
}

func (api UserAPI) GetUsersByIDs(ctx context.Context, userIDs []persist.DBID, before, after *string, first, last *int) ([]db.User, PageInfo, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"userIDs": validate.WithTag(userIDs, "required"),
	}); err != nil {
		return nil, PageInfo{}, err
	}

	if err := validatePaginationParams(api.validator, first, last); err != nil {
		return nil, PageInfo{}, err
	}

	queryFunc := func(params timeIDPagingParams) ([]db.User, error) {
		return api.queries.GetUsersByIDs(ctx, db.GetUsersByIDsParams{
			Limit:         params.Limit,
			UserIds:       userIDs,
			CurBeforeTime: params.CursorBeforeTime,
			CurBeforeID:   params.CursorBeforeID,
			CurAfterTime:  params.CursorAfterTime,
			CurAfterID:    params.CursorAfterID,
			PagingForward: params.PagingForward,
		})
	}

	countFunc := func() (int, error) {
		return len(userIDs), nil
	}

	cursorFunc := func(u db.User) (time.Time, persist.DBID, error) {
		return u.CreatedAt, u.ID, nil
	}

	paginator := timeIDPaginator[db.User]{
		QueryFunc:  queryFunc,
		CursorFunc: cursorFunc,
		CountFunc:  countFunc,
	}

	return paginator.paginate(before, after, first, last)
}

func (api UserAPI) paginatorFromCursorStr(ctx context.Context, curStr string) (positionPaginator[db.User], error) {
	cur := cursors.NewPositionCursor()
	err := cur.Unpack(curStr)
	if err != nil {
		return positionPaginator[db.User]{}, err
	}
	return api.paginatorFromCursor(ctx, cur), nil
}

func (api UserAPI) paginatorFromCursor(ctx context.Context, c *positionCursor) positionPaginator[db.User] {
	return api.paginatorWithQuery(c, func(p positionPagingParams) ([]db.User, error) {
		params := db.GetUsersByPositionPaginateBatchParams{
			UserIds: util.MapWithoutError(c.IDs, func(id persist.DBID) string { return id.String() }),
			// Postgres uses 1-based indexing
			CurBeforePos: p.CursorBeforePos + 1,
			CurAfterPos:  p.CursorAfterPos + 1,
		}
		return api.loaders.GetUsersByPositionPaginateBatch.Load(params)
	})
}

func (api UserAPI) paginatorFromResults(ctx context.Context, c *positionCursor, users []db.User) positionPaginator[db.User] {
	queryF := func(positionPagingParams) ([]db.User, error) { return users, nil }
	return api.paginatorWithQuery(c, queryF)
}

func (api UserAPI) paginatorWithQuery(c *positionCursor, queryF func(positionPagingParams) ([]db.User, error)) positionPaginator[db.User] {
	var paginator positionPaginator[db.User]
	paginator.QueryFunc = queryF
	paginator.CursorFunc = func(u db.User) (int64, []persist.DBID, error) { return c.Positions[u.ID], c.IDs, nil }
	return paginator
}

func (api UserAPI) GetUserByUsername(ctx context.Context, username string) (*db.User, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"username": validate.WithTag(username, "required"),
	}); err != nil {
		return nil, err
	}

	user, err := api.loaders.GetUserByUsernameBatch.Load(username)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (api UserAPI) GetUserByAddress(ctx context.Context, chainAddress persist.ChainAddress) (*db.User, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"chainAddress": validate.WithTag(chainAddress, "required"),
	}); err != nil {
		return nil, err
	}

	user, err := api.loaders.GetUserByChainAddressBatch.Load(db.GetUserByChainAddressBatchParams{
		Address: chainAddress.Address(),
		Chain:   chainAddress.Chain(),
	})
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (api *UserAPI) OptInForRoles(ctx context.Context, roles []persist.Role) (*db.User, error) {
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"roles": validate.WithTag(roles, "required,min=1,unique,dive,role,opt_in_role"),
	}); err != nil {
		return nil, err
	}

	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}

	// The opt_in_role validator already checks this, but let's be explicit about not letting
	// users opt in for the admin role.
	for _, role := range roles {
		if role == persist.RoleAdmin {
			err = errors.New("cannot opt in for admin role")
			sentryutil.ReportError(ctx, err)
			return nil, err
		}
	}

	newRoles := util.MapWithoutError(roles, func(role persist.Role) string { return string(role) })
	ids := util.MapWithoutError(roles, func(role persist.Role) string { return persist.GenerateID().String() })

	err = api.queries.AddUserRoles(ctx, db.AddUserRolesParams{
		UserID: userID,
		Ids:    ids,
		Roles:  newRoles,
	})

	if err != nil {
		return nil, err
	}

	// Even though the user's roles have changed in the database, it could take a while before
	// the new roles are reflected in their auth token. Forcing an auth token refresh will
	// make the roles appear immediately.
	err = For(ctx).Auth.ForceAuthTokenRefresh(ctx, userID)
	if err != nil {
		logger.For(ctx).Errorf("error forcing auth token refresh for user %s: %s", userID, err)
	}

	user, err := api.queries.GetUserById(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &user, err
}

func (api *UserAPI) OptOutForRoles(ctx context.Context, roles []persist.Role) (*db.User, error) {
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"roles": validate.WithTag(roles, "required,min=1,unique,dive,role,opt_in_role"),
	}); err != nil {
		return nil, err
	}

	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}

	// The opt_in_role validator already checks this, but let's be explicit about not letting
	// users opt out of the admin role.
	for _, role := range roles {
		if role == persist.RoleAdmin {
			err := errors.New("cannot opt out of admin role")
			sentryutil.ReportError(ctx, err)
			return nil, err
		}
	}

	err = api.queries.DeleteUserRoles(ctx, db.DeleteUserRolesParams{
		Roles:  roles,
		UserID: userID,
	})

	if err != nil {
		return nil, err
	}

	// Even though the user's roles have changed in the database, it could take a while before
	// the new roles are reflected in their auth token. Forcing an auth token refresh will
	// make the roles appear immediately.
	err = For(ctx).Auth.ForceAuthTokenRefresh(ctx, userID)
	if err != nil {
		logger.For(ctx).Errorf("error forcing auth token refresh for user %s: %s", userID, err)
	}

	user, err := api.queries.GetUserById(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &user, err
}

func (api *UserAPI) GetUserRolesByUserID(ctx context.Context, userID persist.DBID) ([]persist.Role, error) {
	return auth.RolesByUserID(ctx, api.queries, userID)
}

func (api *UserAPI) UserIsAdmin(ctx context.Context) bool {
	for _, role := range getUserRoles(ctx) {
		if role == persist.RoleAdmin {
			return true
		}
	}
	return false
}

func (api UserAPI) PaginateUsersWithRole(ctx context.Context, role persist.Role, before *string, after *string, first *int, last *int) ([]db.User, PageInfo, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"role": validate.WithTag(role, "required,role"),
	}); err != nil {
		return nil, PageInfo{}, err
	}

	if err := validatePaginationParams(api.validator, first, last); err != nil {
		return nil, PageInfo{}, err
	}

	queryFunc := func(params lexicalPagingParams) ([]db.User, error) {
		return api.queries.GetUsersWithRolePaginate(ctx, db.GetUsersWithRolePaginateParams{
			Role:          role,
			Limit:         params.Limit,
			CurBeforeKey:  params.CursorBeforeKey,
			CurBeforeID:   params.CursorBeforeID,
			CurAfterKey:   params.CursorAfterKey,
			CurAfterID:    params.CursorAfterID,
			PagingForward: params.PagingForward,
		})
	}

	cursorFunc := func(u db.User) (string, persist.DBID, error) {
		return u.UsernameIdempotent.String, u.ID, nil
	}

	paginator := lexicalPaginator[db.User]{
		QueryFunc:  queryFunc,
		CursorFunc: cursorFunc,
	}

	return paginator.paginate(before, after, first, last)
}

func (api UserAPI) AddWalletToUser(ctx context.Context, chainAddress persist.ChainAddress, authenticator auth.Authenticator) error {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"chainAddress":  validate.WithTag(chainAddress, "required"),
		"authenticator": validate.WithTag(authenticator, "required"),
	}); err != nil {
		return err
	}

	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return err
	}

	err = user.AddWalletToUser(ctx, userID, chainAddress, authenticator, api.repos.UserRepository, api.multichainProvider)
	if err != nil {
		return err
	}

	return nil
}

func (api UserAPI) RemoveWalletsFromUser(ctx context.Context, walletIDs []persist.DBID) error {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"walletIDs": validate.WithTag(walletIDs, "required,unique,dive,required"),
	}); err != nil {
		return err
	}

	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return err
	}

	removedIDs, removalErr := user.RemoveWalletsFromUser(ctx, userID, walletIDs, api.repos.UserRepository)

	// If any wallet IDs were successfully removed, we need to process those removals, even if we also
	// encountered an error.
	if len(removedIDs) > 0 {
		walletRemovalMessage := task.TokenProcessingWalletRemovalMessage{
			UserID:    userID,
			WalletIDs: removedIDs,
		}

		if err := api.taskClient.CreateTaskForWalletRemoval(ctx, walletRemovalMessage); err != nil {
			// Just log the error here. No need to return it -- the actual wallet removal DID succeed,
			// but tokens owned by the affected wallets won't be updated until the user's next sync.
			logger.For(ctx).WithError(err).Error("failed to create task to process wallet removal")
		}
	}

	return removalErr
}

func (api UserAPI) CreateUser(ctx context.Context, authenticator auth.Authenticator, username string, email *persist.Email, bio, splitName, splitDesc, splitPos string) (userID persist.DBID, splitID persist.DBID, err error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"username": validate.WithTag(username, "required,username"),
		"bio":      validate.WithTag(bio, "bio"),
	}); err != nil {
		return "", "", err
	}

	createUserParams, err := createNewUserParamsWithAuth(ctx, authenticator, username, bio, email)
	if err != nil {
		return "", "", err
	}

	tx, err := api.repos.BeginTx(ctx)
	if err != nil {
		return "", "", err
	}
	queries := api.queries.WithTx(tx)
	defer tx.Rollback(ctx)

	userID, err = user.CreateUser(ctx, createUserParams, api.repos.UserRepository, queries)
	if err != nil {
		return "", "", err
	}

	gc := util.MustGetGinContext(ctx)
	err = queries.AddPiiAccountCreationInfo(ctx, db.AddPiiAccountCreationInfoParams{
		UserID:    userID,
		IpAddress: gc.ClientIP(),
	})
	if err != nil {
		logger.For(ctx).Warnf("failed to get IP address for userID %s: %s\n", userID, err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return "", "", err
	}

	if createUserParams.EmailStatus == persist.EmailVerificationStatusUnverified && email != nil {
		if err := emails.RequestVerificationEmail(ctx, userID); err != nil {
			// Just the log the error since the user can verify their email later
			logger.For(ctx).Warnf("failed to send verification email: %s", err)
		}
	}

	if createUserParams.EmailStatus == persist.EmailVerificationStatusVerified {
		if err := api.taskClient.CreateTaskForAddingEmailToMailingList(ctx, task.AddEmailToMailingListMessage{UserID: userID}); err != nil {
			// Report error to Sentry since there's not another way to subscribe the user to the mailing list
			sentryutil.ReportError(ctx, err)
			logger.For(ctx).Warnf("failed to send mailing list subscription task: %s", err)
		}
	}

	// Send event
	err = event.Dispatch(ctx, db.Event{
		ActorID:        persist.DBIDToNullStr(userID),
		Action:         persist.ActionUserCreated,
		ResourceTypeID: persist.ResourceTypeUser,
		UserID:         userID,
		SubjectID:      userID,
		Data:           persist.EventData{UserBio: bio},
	})
	if err != nil {
		logger.For(ctx).Errorf("failed to dispatch event: %s", err)
	}

	if err != nil {
		logger.For(ctx).Errorf("failed to create task for autosocial process users: %s", err)
	}

	return userID, splitID, nil
}

func (api UserAPI) UpdateUserInfo(ctx context.Context, username string, bio string) error {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"username": validate.WithTag(username, "required,username"),
		"bio":      validate.WithTag(bio, "bio"),
	}); err != nil {
		return err
	}

	// Sanitize
	bio = validate.SanitizationPolicy.Sanitize(bio)

	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return err
	}

	err = user.UpdateUserInfo(ctx, userID, username, bio, api.repos.UserRepository, api.ethClient)
	if err != nil {
		return err
	}

	return nil
}

func (api UserAPI) UpdateUserPrimaryWallet(ctx context.Context, primaryWalletID persist.DBID) error {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"primaryWalletID": validate.WithTag(primaryWalletID, "required"),
	}); err != nil {
		return err
	}

	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return err
	}

	err = api.queries.UpdateUserPrimaryWallet(ctx, db.UpdateUserPrimaryWalletParams{WalletID: primaryWalletID, UserID: userID})
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

	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return err
	}
	err = api.queries.UpdateUserEmail(ctx, db.UpdateUserEmailParams{
		UserID:       userID,
		EmailAddress: email,
	})
	if err != nil {
		return err
	}

	err = emails.RequestVerificationEmail(ctx, userID)
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

	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return err
	}

	// update unsubscriptions
	return emails.UpdateUnsubscriptionsByUserID(ctx, userID, settings)

}

func (api UserAPI) ResendEmailVerification(ctx context.Context) error {
	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return err
	}

	err = emails.RequestVerificationEmail(ctx, userID)
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

	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return err
	}

	return api.queries.UpdateNotificationSettingsByID(ctx, db.UpdateNotificationSettingsByIDParams{ID: userID, NotificationSettings: notificationSettings})
}

func (api UserAPI) GetUserExperiences(ctx context.Context, userID persist.DBID) ([]*model.UserExperience, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"userID": validate.WithTag(userID, "required"),
	}); err != nil {
		return nil, err
	}

	experiences, err := api.queries.GetUserExperiencesByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	asJSON := map[string]bool{}
	if err := experiences.AssignTo(&asJSON); err != nil {
		return nil, err
	}

	result := make([]*model.UserExperience, len(model.AllUserExperienceType))
	for i, experienceType := range model.AllUserExperienceType {
		result[i] = &model.UserExperience{
			Type:        experienceType,
			Experienced: asJSON[experienceType.String()],
		}
	}
	return result, nil
}

func (api UserAPI) UpdateUserExperience(ctx context.Context, experienceType model.UserExperienceType, value bool) error {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"experienceType": validate.WithTag(experienceType, "required"),
	}); err != nil {
		return err
	}

	curUserID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return err
	}

	in := map[string]interface{}{
		experienceType.String(): value,
	}

	marshalled, err := json.Marshal(in)
	if err != nil {
		return err
	}

	return api.queries.UpdateUserExperience(ctx, db.UpdateUserExperienceParams{
		Experience: pgtype.JSONB{
			Bytes:  marshalled,
			Status: pgtype.Present,
		},
		UserID: curUserID,
	})
}

// CreatePushTokenForUser adds a push token to a user, or returns the existing push token if it's already been
// added to this user. If the token can't be added because it belongs to another user, an error is returned.
func (api UserAPI) CreatePushTokenForUser(ctx context.Context, pushToken string) (db.PushNotificationToken, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"pushToken": validate.WithTag(pushToken, "required,min=1,max=255"),
	}); err != nil {
		return db.PushNotificationToken{}, err
	}

	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return db.PushNotificationToken{}, err
	}

	// Does the token already exist?
	token, err := api.queries.GetPushTokenByPushToken(ctx, pushToken)

	if err == nil {
		// If the token exists and belongs to the current user, return it. Attempting to re-add
		// a token that you've already registered is a no-op.
		if token.UserID == userID {
			return token, nil
		}

		// Otherwise, the token belongs to another user. Return an error.
		return db.PushNotificationToken{}, persist.ErrPushTokenBelongsToAnotherUser{PushToken: pushToken}
	}

	// ErrNoRows is expected and means we can continue with creating the token. If we see any other
	// error, return it.
	if err != pgx.ErrNoRows {
		return db.PushNotificationToken{}, err
	}

	token, err = api.queries.CreatePushTokenForUser(ctx, db.CreatePushTokenForUserParams{
		ID:        persist.GenerateID(),
		UserID:    userID,
		PushToken: pushToken,
	})

	if err != nil {
		return db.PushNotificationToken{}, err
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

	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return err
	}

	existingToken, err := api.queries.GetPushTokenByPushToken(ctx, pushToken)
	if err == nil {
		// If the token exists and belongs to the current user, let them delete it.
		if existingToken.UserID == userID {
			return api.queries.DeletePushTokensByIDs(ctx, []persist.DBID{existingToken.ID})
		}

		// Otherwise, the token belongs to another user. Return an error.
		return persist.ErrPushTokenBelongsToAnotherUser{PushToken: pushToken}
	}

	// ErrNoRows is okay and means the token doesn't exist. Unregistering it is a no-op
	// and doesn't return an error.
	if err == pgx.ErrNoRows {
		return nil
	}

	return err
}

func (api UserAPI) BlockUser(ctx context.Context, userID persist.DBID) error {
	// Validate
	viewerID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return err
	}
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"userID": validate.WithTag(userID, fmt.Sprintf("required,ne=%s", viewerID)),
	}); err != nil {
		return err
	}
	_, err = api.queries.BlockUser(ctx, db.BlockUserParams{
		ID:            persist.GenerateID(),
		UserID:        viewerID,
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
	viewerID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return err
	}
	return api.queries.UnblockUser(ctx, db.UnblockUserParams{UserID: viewerID, BlockedUserID: userID})
}

func createNewUserParamsWithAuth(ctx context.Context, authenticator auth.Authenticator, username string, bio string, email *persist.Email) (persist.CreateUserInput, error) {
	authResult, err := authenticator.Authenticate(ctx)
	if err != nil && !util.ErrorIs[persist.ErrUserNotFound](err) {
		return persist.CreateUserInput{}, auth.ErrAuthenticationFailed{WrappedErr: err}
	}

	if authResult.User != nil && !authResult.User.Universal.Bool() {
		if _, ok := authenticator.(auth.MagicLinkAuthenticator); ok {
			// TODO: We currently only use MagicLink for email, but we may use it for other login methods like SMS later,
			// so this error may not always be applicable in the future.
			return persist.CreateUserInput{}, auth.ErrEmailAlreadyUsed
		}
		return persist.CreateUserInput{}, persist.ErrUserAlreadyExists{Authenticator: authenticator.GetDescription()}
	}

	var wallet auth.AuthenticatedAddress

	if len(authResult.Addresses) > 0 {
		// TODO: This currently takes the first authenticated address returned by the authenticator and creates
		// the user's account based on that address. This works because the only auth mechanism we have is nonce-based
		// auth and that supplies a single address. In the future, a user may authenticate in a way that makes
		// multiple authenticated addresses available for initial user creation, and we may want to add all of
		// those addresses to the user's account here.
		wallet = authResult.Addresses[0]
	}

	params := persist.CreateUserInput{
		Username:     username,
		Bio:          bio,
		Email:        email,
		EmailStatus:  persist.EmailVerificationStatusUnverified,
		ChainAddress: wallet.ChainAddress,
		WalletType:   wallet.WalletType,
	}

	// Override input email with verified email if available
	if authResult.Email != nil {
		params.Email = authResult.Email
		params.EmailStatus = persist.EmailVerificationStatusVerified
	}

	return params, nil
}
