package graphql

// schema.resolvers.go gets updated when generating gqlgen bindings and should not contain
// helper functions. schema.resolvers.helpers.go is a companion file that can contain
// helper functions without interfering with code generation.

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/SplitFi/go-splitfi/graphql/model"
	"github.com/SplitFi/go-splitfi/service/emails"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/notifications"
	"github.com/SplitFi/go-splitfi/service/twitter"
	"github.com/SplitFi/go-splitfi/validate"
	"github.com/gammazero/workerpool"
	"github.com/magiclabs/magic-admin-go/token"

	"github.com/SplitFi/go-splitfi/debugtools"

	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/publicapi"
	"github.com/SplitFi/go-splitfi/service/auth"
	"github.com/SplitFi/go-splitfi/service/persist"
)

var errNoAuthMechanismFound = fmt.Errorf("no auth mechanism found")

var nodeFetcher = model.NodeFetcher{
	OnAsset:            resolveAssetByAssetID,
	OnSplit:            resolveSplitBySplitID,
	OnSplitFiUser:      resolveSplitFiUserByUserID,
	OnRecipient:        resolveRecipientByID,
	OnToken:            resolveTokenByTokenID,
	OnWallet:           resolveWalletByAddress,
	OnViewer:           resolveViewerByID,
	OnDeletedNode:      resolveDeletedNodeByID,
	OnSocialConnection: resolveSocialConnectionByIdentifiers,
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
	case publicapi.ErrTokenRefreshFailed:
		mappedErr = model.ErrSyncFailed{Message: message}
	case validate.ErrInvalidInput:
		validationErr, _ := err.(validate.ErrInvalidInput)
		mappedErr = model.ErrInvalidInput{Message: message, Parameters: validationErr.Parameters, Reasons: validationErr.Reasons}
	//case persist.ErrUnknownAction:
	//	mappedErr = model.ErrUnknownAction{Message: message}
	case persist.ErrSplitNotFound:
		mappedErr = model.ErrSplitNotFound{Message: message}
	case twitter.ErrInvalidRefreshToken:
		mappedErr = model.ErrNeedsToReconnectSocial{SocialAccountType: persist.SocialProviderTwitter, Message: message}
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

func resolveSplitFiUserByUserID(ctx context.Context, userID persist.DBID) (*model.SplitFiUser, error) {
	user, err := publicapi.For(ctx).User.GetUserById(ctx, userID)

	if err != nil {
		return nil, err
	}

	return userToModel(ctx, *user), nil
}

func resolveSplitFiUserByAddress(ctx context.Context, chainAddress persist.ChainAddress) (*model.SplitFiUser, error) {
	user, err := publicapi.For(ctx).User.GetUserByAddress(ctx, chainAddress)

	if err != nil {
		return nil, err
	}

	return userToModel(ctx, *user), nil
}

func resolveSplitFiUserByUsername(ctx context.Context, username string) (*model.SplitFiUser, error) {
	user, err := publicapi.For(ctx).User.GetUserByUsername(ctx, username)

	if err != nil {
		return nil, err
	}

	return userToModel(ctx, *user), nil
}

/*
TODO
func resolveSplitsByUserID(ctx context.Context, userID persist.DBID) ([]*model.Split, error) {
	splits, err := publicapi.For(ctx).Split.GetSplitsByUserId(ctx, userID)

	if err != nil {
		return nil, err
	}

	var output = make([]*model.Split, len(splits))
	for i, split := range splits {
		output[i] = splitToModel(ctx, split)
	}

	return output, nil
}*/

func resolveSplitBySplitID(ctx context.Context, splitID persist.DBID) (*model.Split, error) {
	dbSplit, err := publicapi.For(ctx).Split.GetSplitById(ctx, splitID)
	if err != nil {
		return nil, err
	}
	split := &model.Split{
		Dbid:        splitID,
		Name:        &dbSplit.Name,
		Description: &dbSplit.Description,
		//TODO return full split data
	}

	return split, nil
}

func resolveViewerSplitBySplitID(ctx context.Context, splitID persist.DBID) (*model.ViewerSplit, error) {
	split, err := publicapi.For(ctx).Split.GetViewerSplitById(ctx, splitID)

	if err != nil {
		return nil, err
	}

	return &model.ViewerSplit{
		Split: splitToModel(ctx, *split),
	}, nil
}

func resolveRecipientByID(ctx context.Context, recipientID persist.DBID) (*model.Recipient, error) {
	/*
		TODO
		recipient, err := publicapi.For(ctx).Split.GetRecipientById(ctx, recipientID)

			if err != nil {
				return nil, err
			}

			return recipientToModel(ctx, *recipient), nil
	*/
	return &model.Recipient{}, nil
}

/*
TODO
func resolveViewerExperiencesByUserID(ctx context.Context, userID persist.DBID) ([]*model.UserExperience, error) {
	return publicapi.For(ctx).User.GetUserExperiences(ctx, userID)
}
*/

func resolveAssetByAssetID(ctx context.Context, assetID persist.DBID) (*model.Asset, error) {
	/*
		TODO
		asset, err := publicapi.For(ctx).Asset.GetAssetById(ctx, assetID)

			if err != nil {
				return nil, err
			}

			return assetToModel(ctx, *asset), nil
	*/
	return &model.Asset{}, nil
}

func resolveTokenByTokenID(ctx context.Context, tokenID persist.DBID) (*model.Token, error) {
	token, err := publicapi.For(ctx).Token.GetTokenById(ctx, tokenID)

	if err != nil {
		return nil, err
	}

	return tokenToModel(ctx, *token), nil
}

func resolveWalletByAddress(ctx context.Context, address persist.DBID) (*model.Wallet, error) {

	wallet := model.Wallet{
		// TODO
	}

	return &wallet, nil
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
		User:         nil, // handled by dedicated resolver
		ViewerSplits: nil, // handled by dedicated resolver
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

	return &model.UserEmail{
		Email:              &user.PiiEmailAddress,
		VerificationStatus: &user.EmailVerified,
		EmailNotificationSettings: &model.EmailNotificationSettings{
			UnsubscribedFromAll:           user.EmailUnsubscriptions.All.Bool(),
			UnsubscribedFromNotifications: user.EmailUnsubscriptions.Notifications.Bool(),
		},
	}

}

func resolveWalletsByUserID(ctx context.Context, userID persist.DBID) ([]*model.Wallet, error) {
	wallets, err := publicapi.For(ctx).Wallet.GetWalletsByUserID(ctx, userID)

	if err != nil {
		return nil, err
	}

	output := make([]*model.Wallet, 0, len(wallets))

	for _, wallet := range wallets {
		output = append(output, walletToModelSqlc(ctx, wallet))
	}

	return output, nil
}

func resolvePrimaryWalletByUserID(ctx context.Context, userID persist.DBID) (*model.Wallet, error) {

	user, err := publicapi.For(ctx).User.GetUserById(ctx, userID)
	if err != nil {
		return nil, err
	}

	wallet, err := publicapi.For(ctx).Wallet.GetWalletByID(ctx, user.PrimaryWalletID)
	if err != nil {
		return nil, err
	}

	return walletToModelSqlc(ctx, *wallet), nil
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
		User:         nil, // handled by dedicated resolver
		ViewerSplits: nil, // handled by dedicated resolver
	}, nil
}

func resolveDeletedNodeByID(ctx context.Context, id persist.DBID) (*model.DeletedNode, error) {
	return &model.DeletedNode{
		Dbid: id,
	}, nil
}

func resolveSocialConnectionByIdentifiers(ctx context.Context, socialId string, socialType persist.SocialProvider) (*model.SocialConnection, error) {
	return &model.SocialConnection{
		SocialID:   socialId,
		SocialType: socialType,
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

func updateUserEmail(ctx context.Context, email persist.Email) (*model.UpdateEmailPayload, error) {
	err := publicapi.For(ctx).User.UpdateUserEmail(ctx, email)
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

func splitToModel(ctx context.Context, split db.Split) *model.Split {

	return &model.Split{
		Dbid:        split.ID,
		Name:        &split.Name,
		Description: &split.Description,
		//TODO return full split data
	}
}

func splitsToModels(ctx context.Context, splits []db.Split) []*model.Split {
	models := make([]*model.Split, len(splits))
	for i, split := range splits {
		models[i] = splitToModel(ctx, split)
	}

	return models
}

// userToModel converts a db.User to a model.User
func userToModel(ctx context.Context, user db.User) *model.SplitFiUser {
	userApi := publicapi.For(ctx).User
	isAuthenticatedUser := userApi.IsUserLoggedIn(ctx) && userApi.GetLoggedInUserId(ctx) == user.ID

	wallets := make([]*model.Wallet, len(user.Wallets))
	for i, wallet := range user.Wallets {
		wallets[i] = walletToModelPersist(ctx, wallet)
	}

	return &model.SplitFiUser{
		HelperSplitFiUserData: model.HelperSplitFiUserData{
			UserID:          user.ID,
			FeaturedSplitID: user.FeaturedSplit,
		},
		Dbid:      user.ID,
		Username:  &user.Username.String,
		Bio:       &user.Bio.String,
		Wallets:   wallets,
		Universal: &user.Universal,

		// each handled by dedicated resolver
		Splits: nil,
		Roles:  nil,

		IsAuthenticatedUser: &isAuthenticatedUser,
	}
}

func usersToModels(ctx context.Context, users []db.User) []*model.SplitFiUser {
	models := make([]*model.SplitFiUser, len(users))
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

func walletToModelPersist(ctx context.Context, wallet persist.Wallet) *model.Wallet {
	chainAddress := persist.NewChainAddress(wallet.Address, wallet.Chain)

	return &model.Wallet{
		Dbid:         wallet.ID,
		WalletType:   &wallet.WalletType,
		ChainAddress: &chainAddress,
		Chain:        &wallet.Chain,
	}
}

func walletToModelSqlc(ctx context.Context, wallet db.Wallet) *model.Wallet {
	chain := wallet.Chain
	chainAddress := persist.NewChainAddress(wallet.Address, chain)

	return &model.Wallet{
		Dbid:         wallet.ID,
		WalletType:   &wallet.WalletType,
		ChainAddress: &chainAddress,
		Chain:        &wallet.Chain,
	}
}

func tokenToModel(ctx context.Context, token db.Token) *model.Token {
	chain := token.Chain
	// _, _ = token.TokenMetadata.MarshalJSON()
	blockNumber := fmt.Sprint(token.BlockNumber.Int64)
	tokenType := model.TokenType(token.TokenType.String)

	//var isSpamByUser *bool
	//if token.IsUserMarkedSpam.Valid {
	//	isSpamByUser = &token.IsUserMarkedSpam.Bool
	//}
	//
	//var isSpamByProvider *bool
	//if token.IsProviderMarkedSpam.Valid {
	//	isSpamByProvider = &token.IsProviderMarkedSpam.Bool
	//}

	return &model.Token{
		Dbid:         token.ID,
		CreationTime: &token.CreatedAt,
		LastUpdated:  &token.LastUpdated,
		TokenType:    &tokenType,
		Chain:        &chain,
		Name:         &token.Name.String,
		BlockNumber:  &blockNumber, // TODO: later
		//TODO return full token data
	}
}

func tokensToModel(ctx context.Context, token []db.Token) []*model.Token {
	res := make([]*model.Token, len(token))
	for i, token := range token {
		res[i] = tokenToModel(ctx, token)
	}
	return res
}

func getUrlExtension(url string) string {
	return strings.ToLower(strings.TrimPrefix(filepath.Ext(url), "."))
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
