package graphql

// schema.resolvers.go gets updated when generating gqlgen bindings and should not contain
// helper functions. schema.resolvers.helpers.go is a companion file that can contain
// helper functions without interfering with code generation.

import (
	"context"
	"fmt"
	"github.com/gammazero/workerpool"
	"github.com/magiclabs/magic-admin-go/token"
	"github.com/mutuals/go-mutuals/db/gen/indexerdb"
	"github.com/mutuals/go-mutuals/debugtools"
	"github.com/mutuals/go-mutuals/graphql/model"
	"github.com/mutuals/go-mutuals/service/emails"
	"github.com/mutuals/go-mutuals/service/logger"
	"github.com/mutuals/go-mutuals/service/notifications"
	"github.com/mutuals/go-mutuals/util"
	"github.com/mutuals/go-mutuals/validate"
	"time"

	db "github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/publicapi"
	"github.com/mutuals/go-mutuals/service/auth"
	"github.com/mutuals/go-mutuals/service/persist"
)

var errNoAuthMechanismFound = fmt.Errorf("no auth mechanism found")

var nodeFetcher = model.NodeFetcher{
	OnMutualsUser: resolveMutualsUserByUserID,
	OnViewer:      resolveViewerByID,
	OnDeletedNode: resolveDeletedNodeByID,
}

func init() {
	nodeFetcher.ValidateHandlers()
}

// errorToGraphqlType converts a golang error to its matching type from our GraphQL schema.
// If no matching type is found, ok will return false
func errorToGraphqlType(ctx context.Context, err error, gqlTypeName string) (gqlModel interface{}, ok bool) {
	message := err.Error()
	var mappedErr model.Error = nil

	// TODO: Add model.ErrNotAuthorized mapping once auth handling is moved to the publicapi layer

	switch err.(type) {
	case auth.ErrAuthenticationFailed:
		mappedErr = model.ErrAuthenticationFailed{Message: message}
	case auth.ErrDoesNotOwnRequiredNFT:
		mappedErr = model.ErrDoesNotOwnRequiredToken{Message: message}
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
		validationErr, _ := err.(validate.ErrInvalidInput)
		mappedErr = model.ErrInvalidInput{Message: message, Parameters: validationErr.Parameters, Reasons: validationErr.Reasons}
	//case persist.ErrUnknownAction:
	//	mappedErr = model.ErrUnknownAction{Message: message}
	case persist.ErrPoolNotFound:
		mappedErr = model.ErrPoolNotFound{Message: message}
	}
	// TODO add missing errors
	if mappedErr != nil {
		if converted, ok := model.ConvertToModelType(mappedErr, gqlTypeName); ok {
			return converted, true
		}
	}

	return nil, false
}

// authMechanismToAuthenticator takes a GraphQL AuthMechanism and returns an Authenticator that can be used for auth
func (r *Resolver) authMechanismToAuthenticator(ctx context.Context, m model.AuthMechanism) (auth.Authenticator, error) {

	authApi := publicapi.For(ctx).Auth

	if debugtools.Enabled {
		if debugtools.IsDebugEnv() && m.Debug != nil {
			return authApi.NewDebugAuthenticator(ctx, *m.Debug)
		}
	}

	if m.Eoa != nil && m.Eoa.ChainPubKey != nil {
		return authApi.NewNonceAuthenticator(*m.Eoa.ChainPubKey, m.Eoa.Nonce, m.Eoa.Message, m.Eoa.Signature, persist.WalletTypeEOA), nil
	}

	if m.GnosisSafe != nil {
		// GnosisSafe passes an empty signature
		return authApi.NewNonceAuthenticator(persist.NewChainPubKey(persist.PubKey(m.GnosisSafe.Address), persist.ChainETH), m.GnosisSafe.Nonce, m.GnosisSafe.Message, "0x", persist.WalletTypeGnosis), nil
	}

	if m.MagicLink != nil && m.MagicLink.Token != "" {
		t, err := token.NewToken(m.MagicLink.Token)
		if err != nil {
			return nil, err
		}
		return authApi.NewMagicLinkAuthenticator(*t), nil
	}

	if m.OneTimeLoginToken != nil && m.OneTimeLoginToken.Token != "" {
		return authApi.NewOneTimeLoginTokenAuthenticator(m.OneTimeLoginToken.Token), nil
	}

	/*
	   TODO
	   if m.Privy != nil && m.Privy.Token != "" {
	   		return authApi.NewPrivyAuthenticator(m.Privy.Token), nil
	   	}
	*/
	return nil, errNoAuthMechanismFound
}

func resolveMutualsUserByUserID(ctx context.Context, userID persist.DBID) (*model.MutualsUser, error) {
	user, err := publicapi.For(ctx).User.GetUserById(ctx, userID)
	if err != nil {
		return nil, err
	}
	return userToModel(ctx, *user), nil
}

func resolveMutualsUserByAddress(ctx context.Context, chainAddress persist.ChainAddress) (*model.MutualsUser, error) {
	user, err := publicapi.For(ctx).User.GetUserByAddress(ctx, chainAddress)

	if err != nil {
		return nil, err
	}

	return userToModel(ctx, *user), nil
}

func resolveMutualsUserByUsername(ctx context.Context, username string) (*model.MutualsUser, error) {
	user, err := publicapi.For(ctx).User.GetUserByUsername(ctx, username)

	if err != nil {
		return nil, err
	}

	return userToModel(ctx, *user), nil
}

func resolvePoolByPoolID(ctx context.Context, poolID persist.DBID) (*model.Pool, error) {
	pool, err := publicapi.For(ctx).Pool.GetPoolById(ctx, poolID)
	if err != nil {
		return nil, err
	}

	return poolToModel(ctx, *pool), nil
}

func resolvePoolContractByPoolContractID(ctx context.Context, contractID persist.DBID) (*model.PoolContract, error) {
	//dbPool, err := publicapi.For(ctx).Pool.GetPoolById(ctx, contractID)
	//if err != nil {
	//	return nil, err
	//}
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

func resolveViewerPoolByPoolID(ctx context.Context, poolID persist.DBID) (*model.ViewerPool, error) {
	pool, err := publicapi.For(ctx).Pool.GetViewerPoolById(ctx, poolID)

	if err != nil {
		return nil, err
	}

	return &model.ViewerPool{
		Pool: poolToModel(ctx, *pool),
	}, nil
}

func resolvePoolsByUserID(ctx context.Context, userID persist.DBID) ([]*model.Pool, error) {
	pools, err := publicapi.For(ctx).Pool.GetPoolsByUserID(ctx, userID)

	if err != nil {
		return nil, err
	}

	return poolsToModels(ctx, pools), nil
}

func resolveClaimsByPoolID(ctx context.Context, poolID persist.DBID) ([]*model.Claim, error) {
	claims, err := publicapi.For(ctx).Claim.GetClaimsByPoolID(ctx, poolID)

	if err != nil {
		return nil, err
	}

	return claimsToModels(ctx, claims), nil
}

func resolveViewerExperiencesByUserID(ctx context.Context, userID persist.DBID) ([]*model.UserExperience, error) {
	return publicapi.For(ctx).User.GetUserExperiences(ctx, userID)
}

func resolveTokenByTokenID(ctx context.Context, tokenID persist.DBID) (*model.Token, error) {
	/*
		TODO
		asset, err := publicapi.For(ctx).Token.GetTokenById(ctx, tokenID)

			if err != nil {
				return nil, err
			}

			return tokenToModel(ctx, *asset), nil
	*/
	return nil, nil
}

func resolveTokenBalanceByTokenBalanceID(ctx context.Context, assetID persist.DBID) (*model.TokenBalance, error) {
	/*
		TODO
		asset, err := publicapi.For(ctx).Asset.GetAssetById(ctx, assetID)

			if err != nil {
				return nil, err
			}

			return assetToModel(ctx, *asset), nil
	*/
	return &model.TokenBalance{}, nil
}

func resolveViewer(ctx context.Context) *model.Viewer {

	if !publicapi.For(ctx).User.IsUserLoggedIn(ctx) {
		return nil
	}

	userID := publicapi.For(ctx).User.GetLoggedInUserId(ctx)

	viewer := &model.Viewer{
		HelperViewerData: model.HelperViewerData{
			UserId: userID,
		},
		User:        nil, // handled by dedicated resolver
		ViewerPools: nil, // handled by dedicated resolver
	}

	return viewer
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

func resolveWalletsByUserID(ctx context.Context, userID persist.DBID) ([]*model.Account, error) {
	userAccounts, err := publicapi.For(ctx).Wallet.GetUserAccountsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	addresses := util.MapWithoutError(userAccounts, func(wallet db.UserAccount) string {
		return wallet.Address.String()
	})

	accounts, err := publicapi.For(ctx).Wallet.GetAccountsByAddresses(ctx, addresses)
	if err != nil {
		return nil, err
	}

	return walletsToModels(ctx, accounts), nil
}

func resolvePrimaryWalletByUserID(ctx context.Context, userID persist.DBID) (*model.Account, error) {

	user, err := publicapi.For(ctx).User.GetUserById(ctx, userID)
	if err != nil {
		return nil, err
	}

	userAccount, err := publicapi.For(ctx).Wallet.GetUserAccountByID(ctx, user.PrimaryAccountID)
	if err != nil {
		return nil, err
	}

	account, err := publicapi.For(ctx).Wallet.GetAccountByAddress(ctx, userAccount.Address)
	if err != nil {
		return nil, err
	}

	return walletToModel(ctx, *account), nil
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

func resolveViewerNotificationSettings(ctx context.Context) (*model.NotificationSettings, error) {

	userID := publicapi.For(ctx).User.GetLoggedInUserId(ctx)

	user, err := publicapi.For(ctx).User.GetUserById(ctx, userID)

	if err != nil {
		return nil, err
	}

	return notificationSettingsToModel(ctx, user), nil

}

func notificationSettingsToModel(ctx context.Context, user *db.User) *model.NotificationSettings {
	// TODO notifications
	// settings := user.NotificationSettings
	return &model.NotificationSettings{}
}

func resolveNewNotificationSubscription(ctx context.Context) <-chan model.Notification {
	userID := publicapi.For(ctx).User.GetLoggedInUserId(ctx)
	notifDispatcher := notifications.For(ctx)
	notifs := notifDispatcher.GetNewNotificationsForUser(userID)
	logger.For(ctx).Info("new notification subscription for ", userID)

	result := make(chan model.Notification)

	go func() {
		for notif := range notifs {
			// use async to prevent blocking the dispatcher
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
				notifDispatcher.UnsubscribeNewNotificationsForUser(userID)
			}
		}
	}()

	return result
}

func resolveUpdatedNotificationSubscription(ctx context.Context) <-chan model.Notification {
	userID := publicapi.For(ctx).User.GetLoggedInUserId(ctx)
	notifDispatcher := notifications.For(ctx)
	notifs := notifDispatcher.GetUpdatedNotificationsForUser(userID)

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
					notifDispatcher.UnsubscribeUpdatedNotificationsForUser(userID)
				}
			})
		}
		wp.StopWait()
	}()

	return result
}

func resolveGroupNotificationUsersConnectionByUserIDs(ctx context.Context, userIDs persist.DBIDList, before *string, after *string, first *int, last *int) (*model.GroupNotificationUsersConnection, error) {
	if len(userIDs) == 0 {
		return &model.GroupNotificationUsersConnection{
			Edges:    []*model.GroupNotificationUserEdge{},
			PageInfo: &model.PageInfo{},
		}, nil
	}
	users, pageInfo, err := publicapi.For(ctx).User.GetUsersByIDs(ctx, userIDs, before, after, first, last)
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
		HelperGroupNotificationUsersConnectionData: model.HelperGroupNotificationUsersConnectionData{
			UserIDs: userIDs,
		},
	}, nil
}

func resolveNotificationByID(ctx context.Context, id persist.DBID) (model.Notification, error) {
	notification, err := publicapi.For(ctx).Notifications.GetByID(ctx, id)

	if err != nil {
		return nil, err
	}

	return notificationToModel(notification)
}

func resolveViewerByID(ctx context.Context, id string) (*model.Viewer, error) {

	if !publicapi.For(ctx).User.IsUserLoggedIn(ctx) {
		return nil, nil
	}
	userID := publicapi.For(ctx).User.GetLoggedInUserId(ctx)

	if userID.String() != id {
		return nil, nil
	}

	return &model.Viewer{
		HelperViewerData: model.HelperViewerData{
			UserId: userID,
		},
		User:        nil, // handled by dedicated resolver
		ViewerPools: nil, // handled by dedicated resolver
	}, nil
}

func resolveDeletedNodeByID(ctx context.Context, id persist.DBID) (*model.DeletedNode, error) {
	return &model.DeletedNode{
		Dbid: id,
	}, nil
}

func verifyEmail(ctx context.Context, token string) (*model.VerifyEmailPayload, error) {
	output, err := emails.VerifyEmail(ctx, token)
	if err != nil {
		return nil, err
	}

	return &model.VerifyEmailPayload{
		Email: output.Email,
	}, nil

}

func updateUserEmail(ctx context.Context, email persist.Email, authenticator *auth.Authenticator) (*model.UpdateEmailPayload, error) {
	var err error

	if authenticator != nil {
		err = publicapi.For(ctx).User.UpdateUserEmailWithAuthenticator(ctx, email, *authenticator)
	} else {
		err = publicapi.For(ctx).User.UpdateUserEmailWithManualVerification(ctx, email)
	}

	if err != nil {
		return nil, err
	}

	return &model.UpdateEmailPayload{
		Viewer: resolveViewer(ctx),
	}, nil

}

func resendEmailVerification(ctx context.Context) (*model.ResendVerificationEmailPayload, error) {
	err := publicapi.For(ctx).User.ResendEmailVerification(ctx)
	if err != nil {
		return nil, err
	}

	return &model.ResendVerificationEmailPayload{
		Viewer: resolveViewer(ctx),
	}, nil

}

func updateUserEmailNotificationSettings(ctx context.Context, input model.UpdateEmailNotificationSettingsInput) (*model.UpdateEmailNotificationSettingsPayload, error) {
	err := publicapi.For(ctx).User.UpdateUserEmailNotificationSettings(ctx, persist.EmailUnsubscriptions{
		All:           persist.NullBool(input.UnsubscribedFromAll),
		Notifications: persist.NullBool(input.UnsubscribedFromNotifications),
	})
	if err != nil {
		return nil, err
	}

	return &model.UpdateEmailNotificationSettingsPayload{
		Viewer: resolveViewer(ctx),
	}, nil

}

func unsubscribeFromEmailType(ctx context.Context, input model.UnsubscribeFromEmailTypeInput) (*model.UnsubscribeFromEmailTypePayload, error) {

	if err := emails.UnsubscribeByJWT(ctx, input.Token, []model.EmailUnsubscriptionType{input.Type}); err != nil {
		return nil, err
	}

	return &model.UnsubscribeFromEmailTypePayload{
		Viewer: resolveViewer(ctx),
	}, nil

}

func poolToModel(ctx context.Context, pool db.Pool) *model.Pool {
	return &model.Pool{
		Dbid:        pool.ID,
		Name:        pool.Name,
		Description: pool.Description,
		Logo:        pool.Logo,
		Slug:        "", // TODO pool.Slug
		Status:      "", // TODO pool.Status
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
		Dbid:      claim.ID,
		Value:     persist.HexString(claim.Value), // TODO claim.Value as HexString
		Label:     claim.Label,
		Path:      persist.NullStrToStr(claim.Path),
		CreatedAt: claim.CreatedAt,
		UpdatedAt: claim.UpdatedAt,
		Parent:    nil, // handled by dedicated resolver
		Pool:      nil, // handled by dedicated resolver
		Recipient: nil, // handled by dedicated resolver
		State:     nil, // handled by dedicated resolver
		Strategy:  nil, // handled by dedicated resolver
	}
}

func claimsToModels(ctx context.Context, claims []db.Claim) []*model.Claim {
	models := make([]*model.Claim, len(claims))
	for i, claim := range claims {
		models[i] = claimToModel(ctx, claim)
	}

	return models
}

// userToModel converts a db.User to a model.User
func userToModel(ctx context.Context, user db.User) *model.MutualsUser {
	userApi := publicapi.For(ctx).User
	isAuthenticatedUser := userApi.IsUserLoggedIn(ctx) && userApi.GetLoggedInUserId(ctx) == user.ID

	return &model.MutualsUser{
		HelperMutualsUserData: model.HelperMutualsUserData{
			UserID: user.ID,
		},
		Dbid:      user.ID,
		Username:  &user.Username.String,
		Universal: &user.Universal,

		// each handled by dedicated resolver
		Pools: nil,
		Roles: nil,

		IsAuthenticatedUser: &isAuthenticatedUser,
	}
}

func usersToModels(ctx context.Context, users []db.User) []*model.MutualsUser {
	models := make([]*model.MutualsUser, len(users))
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
			Cursor: nil, // not used by relay, but relay will complain without this field existing
		}
	}
	return edges
}

func walletToModel(ctx context.Context, account indexerdb.Account) *model.Account {
	return &model.Account{
		Address:     persist.Address(account.Address),       // TODO account.Address
		AccountType: model.AccountType(account.AccountType), // TODO account.AccountType
		CreatedAt:   account.CreatedAt,
		UpdatedAt:   account.UpdatedAt,
		SelfPools:   nil, // handled by dedicated resolver
		Balances:    nil, // handled by dedicated resolver
	}
}

func walletsToModels(ctx context.Context, wallets []indexerdb.Account) []*model.Account {
	models := make([]*model.Account, len(wallets))
	for i, wallet := range wallets {
		models[i] = walletToModel(ctx, wallet)
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
