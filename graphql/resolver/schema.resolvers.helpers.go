package graphql

// schema.resolvers.go gets updated when generating gqlgen bindings and should not contain
// helper functions. schema.resolvers.helpers.go is a companion file that can contain
// helper functions without interfering with code generation.

import (
	"context"
	"fmt"
	"time"

	"github.com/gammazero/workerpool"
	db "github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/db/gen/indexerdb"
	"github.com/mutuals/go-mutuals/graphql/model"
	"github.com/mutuals/go-mutuals/publicapi"
	"github.com/mutuals/go-mutuals/service/auth"
	"github.com/mutuals/go-mutuals/service/logger"
	"github.com/mutuals/go-mutuals/service/notifications"
	"github.com/mutuals/go-mutuals/service/persist"
	"github.com/mutuals/go-mutuals/validate"
)

var nodeFetcher = model.NodeFetcher{
	OnClaim:       resolveClaimByID,
	OnDeletedNode: resolveDeletedNodeByID,
	OnPool:        resolvePoolByID,
	OnUser:        resolveUserByUserID,
	OnViewer:      resolveViewerByID,
}

func init() {
	nodeFetcher.ValidateHandlers()
}

// errorToGraphqlType converts a golang error to its matching type from our GraphQL schema.
// If no matching type is found, ok will return false.
// This function is used by middleware to remap errors to their GraphQL union types.
func errorToGraphqlType(ctx context.Context, err error, gqlTypeName string) (gqlModel interface{}, ok bool) {
	message := err.Error()
	var mappedErr model.Error = nil

	switch e := err.(type) {
	case auth.ErrAuthenticationFailed:
		mappedErr = model.ErrAuthenticationFailed{Message: message}
	case persist.ErrUserNotFound:
		mappedErr = model.ErrUserNotFound{Message: message}
	case persist.ErrUserAlreadyExists:
		mappedErr = model.ErrUserAlreadyExists{Message: message}
	case persist.ErrUsernameNotAvailable:
		mappedErr = model.ErrUsernameNotAvailable{Message: message}
	case persist.ErrTokenNotFoundByID:
		mappedErr = model.ErrTokenNotFound{Message: message}
	case persist.ErrAddressOwnedByUser:
		mappedErr = model.ErrAddressOwnedByUser{Message: message}
	case validate.ErrInvalidInput:
		mappedErr = model.ErrInvalidInput{
			Message:    message,
			Parameters: e.Parameters,
			Reasons:    e.Reasons,
		}
	case persist.ErrPoolNotFound:
		mappedErr = model.ErrPoolNotFound{Message: message}
	}

	if mappedErr != nil {
		if converted, ok := model.ConvertToModelType(mappedErr, gqlTypeName); ok {
			return converted, true
		}
	}

	return nil, false
}

func resolveUserByUserID(ctx context.Context, userId persist.DBID) (*model.User, error) {
	user, err := publicapi.For(ctx).User.GetUserById(ctx, userId)
	if err != nil {
		return nil, err
	}
	return userToModel(ctx, *user), nil
}

func resolveUserByAddress(ctx context.Context, chainAddress persist.ChainAddress) (*model.User, error) {
	user, err := publicapi.For(ctx).User.GetUserByAddress(ctx, chainAddress)
	if err != nil {
		return nil, err
	}
	return userToModel(ctx, *user), nil
}

func resolvePool(ctx context.Context, id *model.GqlID, slug *string, contractID *model.GqlID) (model.PoolResult, error) {
	poolDBID := id.DBID()
	contractDBID := contractID.DBID()

	pool, err := publicapi.For(ctx).Pool.GetPool(ctx, &poolDBID, slug, &contractDBID)
	if err != nil {
		return nil, err
	}

	return poolToModel(ctx, *pool), nil
}

func resolvePoolByID(ctx context.Context, id persist.DBID) (*model.Pool, error) {
	pool, err := publicapi.For(ctx).Pool.GetPool(ctx, &id, nil, nil)
	if err != nil {
		return nil, err
	}
	return poolToModel(ctx, *pool), nil
}

func resolveViewerPools(ctx context.Context) ([]*model.Pool, error) {
	pools, err := publicapi.For(ctx).Pool.GetViewerPools(ctx)
	if err != nil {
		return nil, err
	}
	return poolsToModels(ctx, *pools), nil
}

func resolvePoolContractByPoolContractID(ctx context.Context, contractID persist.DBID) (*model.PoolContract, error) {
	pool := &model.PoolContract{
		ID:          "",
		Address:     "",
		ChainID:     0,
		Status:      "",
		PoolFactory: nil, // handled by dedicated resolver
		Account:     nil, // handled by dedicated resolver
		Owner:       nil, // handled by dedicated resolver
		DayBalance:  nil, // handled by dedicated resolver
		HourBalance: nil, // handled by dedicated resolver
		Deposits:    nil, // handled by dedicated resolver
		Withdrawals: nil, // handled by dedicated resolver
		CreatedAt:   time.Time{},
		UpdatedAt:   time.Time{},
	}
	return pool, nil
}

func resolveClaimByID(ctx context.Context, id persist.DBID) (*model.Claim, error) {
	claim, err := publicapi.For(ctx).Claim.GetClaimById(ctx, id)
	if err != nil {
		return nil, err
	}
	return claimToModel(ctx, *claim), nil
}

func resolveClaimsByPoolID(ctx context.Context, poolID persist.DBID) ([]*model.Claim, error) {
	claims, err := publicapi.For(ctx).Claim.GetClaimsByPoolID(ctx, poolID)
	if err != nil {
		return nil, err
	}
	return claimsToModels(ctx, claims), nil
}

func resolveTokenByTokenID(ctx context.Context, tokenID persist.DBID) (*model.Token, error) {
	// TODO: implement
	return nil, nil
}

func resolveTokenBalanceByTokenBalanceID(ctx context.Context, assetID persist.DBID) (*model.TokenBalance, error) {
	// TODO: implement
	return &model.TokenBalance{}, nil
}

func resolveViewer(ctx context.Context) model.ViewerResult {
	if !publicapi.For(ctx).User.IsUserLoggedIn(ctx) {
		return nil
	}

	// User id assignment works through the ID() method via @goGqlId directive
	return &model.Viewer{
		User:  nil, // handled by dedicated resolver
		Pools: nil, // handled by dedicated resolver
	}
}

func resolveViewerEmail(ctx context.Context) *model.UserEmail {
	userWithPII, err := publicapi.For(ctx).User.GetUserWithPII(ctx)
	if err != nil {
		return nil
	}
	return userWithPIIToEmailModel(userWithPII)
}

func userWithPIIToEmailModel(user *db.PiiUserView) *model.UserEmail {
	var verificationStatus persist.EmailVerificationStatus
	var email persist.Email

	if user.PiiVerifiedEmailAddress.String() != "" {
		email = user.PiiVerifiedEmailAddress
		verificationStatus = persist.EmailVerificationStatusVerified
	} else {
		email = user.PiiUnverifiedEmailAddress
		verificationStatus = persist.EmailVerificationStatusUnverified
	}

	return &model.UserEmail{
		Email:              &email,
		VerificationStatus: &verificationStatus,
		EmailNotificationSettings: &model.EmailNotificationSettings{
			UnsubscribedFromAll:           user.EmailUnsubscriptions.All.Bool(),
			UnsubscribedFromNotifications: user.EmailUnsubscriptions.Notifications.Bool(),
		},
	}
}

func resolveViewerNotifications(ctx context.Context, before *string, after *string, first *int, last *int) (*model.NotificationsConnection, error) {
	notifs, pageInfo, unseen, err := publicapi.For(ctx).Notifications.GetViewerNotifications(ctx, before, after, first, last)
	if err != nil {
		return nil, err
	}

	edges, err := notificationsToEdges(notifs)
	if err != nil {
		return nil, err
	}

	return &model.NotificationsConnection{
		Edges:       edges,
		PageInfo:    pageInfoToModel(ctx, pageInfo),
		UnseenCount: &unseen,
	}, nil
}

func notificationsToEdges(notifs []db.Notification) ([]*model.NotificationEdge, error) {
	edges := make([]*model.NotificationEdge, len(notifs))

	for i, notif := range notifs {
		node, err := notificationToModel(notif)
		if err != nil {
			return nil, err
		}
		edges[i] = &model.NotificationEdge{
			Node: node,
		}
	}

	return edges, nil
}

func notificationToModel(notif db.Notification) (model.Notification, error) {
	switch notif.Action {
	// TODO extend with custom notification actions
	default:
		return nil, fmt.Errorf("unknown notification action: %s", notif.Action)
	}
}

func resolveViewerNotificationSettings(ctx context.Context) (model.NotificationSettingsUpdateResult, error) {
	userId := publicapi.For(ctx).User.GetLoggedInUserId(ctx)
	user, err := publicapi.For(ctx).User.GetUserById(ctx, userId)
	if err != nil {
		return nil, err
	}

	return &model.NotificationSettingsUpdatePayload{
		NotificationSettings: notificationSettingsToModel(ctx, user),
	}, nil
}

func notificationSettingsToModel(ctx context.Context, user *db.User) *model.NotificationSettings {
	// TODO notifications
	return &model.NotificationSettings{}
}

func resolveNewNotificationSubscription(ctx context.Context) <-chan model.Notification {
	userId := publicapi.For(ctx).User.GetLoggedInUserId(ctx)
	notifDispatcher := notifications.For(ctx)
	notifs := notifDispatcher.GetNewNotificationsForUser(userId)
	logger.For(ctx).Info("new notification subscription for ", userId)

	result := make(chan model.Notification)

	go func() {
		for notif := range notifs {
			asModel, err := notificationToModel(notif)
			if err != nil {
				logger.For(nil).Errorf("error converting notification to model: %v", err)
				return
			}
			select {
			case result <- asModel:
				logger.For(nil).Debug("sent new notification to subscription")
			default:
				logger.For(nil).Errorf("notification subscription channel full, dropping notification")
				notifDispatcher.UnsubscribeNewNotificationsForUser(userId)
			}
		}
	}()

	return result
}

func resolveUpdatedNotificationSubscription(ctx context.Context) <-chan model.Notification {
	userId := publicapi.For(ctx).User.GetLoggedInUserId(ctx)
	notifDispatcher := notifications.For(ctx)
	notifs := notifDispatcher.GetUpdatedNotificationsForUser(userId)

	result := make(chan model.Notification)
	wp := workerpool.New(10)

	go func() {
		for notif := range notifs {
			n := notif
			wp.Submit(func() {
				asModel, err := notificationToModel(n)
				if err != nil {
					logger.For(nil).Errorf("error converting notification to model: %v", err)
					return
				}
				select {
				case result <- asModel:
					logger.For(nil).Debug("sent updated notification to subscription")
				default:
					logger.For(nil).Errorf("notification subscription channel full, dropping notification")
					notifDispatcher.UnsubscribeUpdatedNotificationsForUser(userId)
				}
			})
		}
		wp.StopWait()
	}()

	return result
}

func resolveGroupNotificationUsersConnectionByUserIDs(ctx context.Context, userIds persist.DBIDList, before *string, after *string, first *int, last *int) (*model.GroupNotificationUsersConnection, error) {
	if len(userIds) == 0 {
		return &model.GroupNotificationUsersConnection{
			Edges:    []*model.GroupNotificationUserEdge{},
			PageInfo: &model.PageInfo{},
		}, nil
	}

	users, pageInfo, err := publicapi.For(ctx).User.GetUsersByIds(ctx, userIds, before, after, first, last)
	if err != nil {
		return nil, err
	}

	edges := make([]*model.GroupNotificationUserEdge, len(users))
	for i, user := range users {
		edges[i] = &model.GroupNotificationUserEdge{
			Node:   userToModel(ctx, user),
			Cursor: nil,
		}
	}

	return &model.GroupNotificationUsersConnection{
		Edges:    edges,
		PageInfo: pageInfoToModel(ctx, pageInfo),
	}, nil
}

func resolveNotificationByID(ctx context.Context, id persist.DBID) (model.Notification, error) {
	notification, err := publicapi.For(ctx).Notifications.GetById(ctx, id)
	if err != nil {
		return nil, err
	}
	return notificationToModel(notification)
}

func resolveViewerByID(ctx context.Context, id persist.DBID) (*model.Viewer, error) {
	if !publicapi.For(ctx).User.IsUserLoggedIn(ctx) {
		return nil, nil
	}

	userId := publicapi.For(ctx).User.GetLoggedInUserId(ctx)
	if userId != id {
		return nil, nil
	}

	return &model.Viewer{
		User:  nil, // handled by dedicated resolver
		Pools: nil, // handled by dedicated resolver
	}, nil
}

func resolveDeletedNodeByID(ctx context.Context, id persist.DBID) (*model.DeletedNode, error) {
	return &model.DeletedNode{}, nil
}

func poolToModel(ctx context.Context, pool db.Pool) *model.Pool {
	return &model.Pool{
		Name:        pool.Name,
		Description: pool.Description,
		Image:       pool.Image,
		DonationBps: 0, // TODO: add to db.Pool
		Slug:        pool.Slug,
		Status:      model.PoolStatusDraft, // TODO: map pool.Status
		CreatedAt:   pool.CreatedAt,
		UpdatedAt:   pool.UpdatedAt,
		Owner:       nil, // handled by dedicated resolver
		Contract:    nil, // handled by dedicated resolver
		Claims:      nil, // handled by dedicated resolver
	}
}

func poolsToModels(ctx context.Context, pools []db.Pool) []*model.Pool {
	models := make([]*model.Pool, len(pools))
	for i, pool := range pools {
		models[i] = poolToModel(ctx, pool)
	}
	return models
}

func claimToModel(ctx context.Context, claim db.Claim) *model.Claim {
	return &model.Claim{
		Data:          claim.Data.Bytes,
		Label:         claim.Label,
		Path:          persist.NullStrToStr(claim.Path),
		ChildrenCount: 0, // TODO: calculate or fetch
		CreatedAt:     claim.CreatedAt,
		UpdatedAt:     claim.UpdatedAt,
		Parent:        nil, // handled by dedicated resolver
		Pool:          nil, // handled by dedicated resolver
		Recipient:     nil, // handled by dedicated resolver
		State:         nil, // handled by dedicated resolver
		Strategy:      nil, // handled by dedicated resolver
	}
}

func claimsToModels(ctx context.Context, claims []db.Claim) []*model.Claim {
	models := make([]*model.Claim, len(claims))
	for i, claim := range claims {
		models[i] = claimToModel(ctx, claim)
	}
	return models
}

func userToModel(ctx context.Context, user db.User) *model.User {
	return &model.User{
		Pools: nil, // handled by dedicated resolver
		Roles: nil, // handled by dedicated resolver
	}
}

func usersToModels(ctx context.Context, users []db.User) []*model.User {
	models := make([]*model.User, len(users))
	for i, user := range users {
		models[i] = userToModel(ctx, user)
	}
	return models
}

func usersToEdges(ctx context.Context, users []db.User) []*model.UserEdge {
	edges := make([]*model.UserEdge, len(users))
	for i, user := range users {
		edges[i] = &model.UserEdge{
			Node:   userToModel(ctx, user),
			Cursor: nil,
		}
	}
	return edges
}

func evmAccountToModel(ctx context.Context, account indexerdb.Account) *model.EVMAccount {
	return &model.EVMAccount{
		Address:     persist.Address(account.Address),
		AccountType: model.EVMAccountType(account.AccountType),
		CreatedAt:   account.CreatedAt,
		UpdatedAt:   account.UpdatedAt,
		SelfPools:   nil, // handled by dedicated resolver
		Balances:    nil, // handled by dedicated resolver
	}
}

func evmAccountsToModels(ctx context.Context, wallets []indexerdb.Account) []*model.EVMAccount {
	models := make([]*model.EVMAccount, len(wallets))
	for i, wallet := range wallets {
		models[i] = evmAccountToModel(ctx, wallet)
	}
	return models
}

func pageInfoToModel(ctx context.Context, pageInfo publicapi.PageInfo) *model.PageInfo {
	return &model.PageInfo{
		Total:           pageInfo.Total,
		Size:            pageInfo.Size,
		HasPreviousPage: pageInfo.HasPreviousPage,
		HasNextPage:     pageInfo.HasNextPage,
		StartCursor:     pageInfo.StartCursor,
		EndCursor:       pageInfo.EndCursor,
	}
}
