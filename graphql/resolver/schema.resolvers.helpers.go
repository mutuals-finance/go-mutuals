package graphql

// schema.resolvers.go gets updated when generating gqlgen bindings and should not contain
// helper functions. schema.resolvers.helpers.go is a companion file that can contain
// helper functions without interfering with code generation.

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/SplitFi/go-splitfi/env"
	"github.com/SplitFi/go-splitfi/graphql/model"
	"github.com/SplitFi/go-splitfi/service/emails"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/mediamapper"
	"github.com/SplitFi/go-splitfi/service/notifications"
	"github.com/SplitFi/go-splitfi/service/socialauth"
	"github.com/SplitFi/go-splitfi/service/twitter"
	"github.com/SplitFi/go-splitfi/validate"
	"github.com/gammazero/workerpool"
	"github.com/magiclabs/magic-admin-go/token"

	"github.com/SplitFi/go-splitfi/debugtools"

	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/publicapi"
	"github.com/SplitFi/go-splitfi/service/auth"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/util"
)

var errNoAuthMechanismFound = fmt.Errorf("no auth mechanism found")

var nodeFetcher = model.NodeFetcher{
	OnSplit:            resolveSplitBySplitID,
	OnSplitFiUser:      resolveSplitFiUserByUserID,
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
		if env.GetString("ENV") == "local" && m.Debug != nil {
			return authApi.NewDebugAuthenticator(ctx, *m.Debug)
		}
	}

	if m.Eoa != nil && m.Eoa.ChainPubKey != nil {
		return authApi.NewNonceAuthenticator(*m.Eoa.ChainPubKey, m.Eoa.Nonce, m.Eoa.Signature, persist.WalletTypeEOA), nil
	}

	if m.GnosisSafe != nil {
		// GnosisSafe passes an empty signature
		return authApi.NewNonceAuthenticator(persist.NewChainPubKey(persist.PubKey(m.GnosisSafe.Address), persist.ChainETH), m.GnosisSafe.Nonce, "0x", persist.WalletTypeGnosis), nil
	}

	if m.MagicLink != nil && m.MagicLink.Token != "" {
		t, err := token.NewToken(m.MagicLink.Token)
		if err != nil {
			return nil, err
		}
		return authApi.NewMagicLinkAuthenticator(*t), nil
	}

	return nil, errNoAuthMechanismFound
}

// authMechanismToAuthenticator takes a GraphQL AuthMechanism and returns an Authenticator that can be used for auth
func (r *Resolver) socialAuthMechanismToAuthenticator(ctx context.Context, m model.SocialAuthMechanism) (socialauth.Authenticator, error) {

	if debugtools.Enabled {
		if env.GetString("ENV") == "local" && m.Debug != nil {
			return debugtools.NewDebugSocialAuthenticator(m.Debug.Provider, m.Debug.ID, map[string]interface{}{"username": m.Debug.Username}), nil
		}
	}

	if m.Twitter != nil {
		authedUserID := publicapi.For(ctx).User.GetLoggedInUserId(ctx)
		return publicapi.For(ctx).Social.NewTwitterAuthenticator(authedUserID, m.Twitter.Code), nil
	}

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
}

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

func resolveViewerExperiencesByUserID(ctx context.Context, userID persist.DBID) ([]*model.UserExperience, error) {
	return publicapi.For(ctx).User.GetUserExperiences(ctx, userID)
}

func resolveViewerSocialsByUserID(ctx context.Context, userID persist.DBID) (*model.SocialAccounts, error) {
	return publicapi.For(ctx).User.GetSocials(ctx, userID)
}

func resolveUserSocialsByUserID(ctx context.Context, userID persist.DBID) (*model.SocialAccounts, error) {
	return publicapi.For(ctx).User.GetDisplayedSocials(ctx, userID)
}

func resolveTokenByTokenID(ctx context.Context, tokenID persist.DBID) (*model.Token, error) {
	token, err := publicapi.For(ctx).Token.GetTokenById(ctx, tokenID)

	if err != nil {
		return nil, err
	}

	return tokenToModel(ctx, *token), nil
}

func resolveTokensByWalletID(ctx context.Context, walletID persist.DBID) ([]*model.Token, error) {
	tokens, err := publicapi.For(ctx).Token.GetTokensByWalletID(ctx, walletID)

	if err != nil {
		return nil, err
	}

	return tokensToModel(ctx, tokens), nil
}

func resolveTokensByUserIDAndContractID(ctx context.Context, userID, contractID persist.DBID) ([]*model.Token, error) {

	tokens, err := publicapi.For(ctx).Token.GetTokensByUserIDAndContractID(ctx, userID, contractID)
	if err != nil {
		return nil, err
	}

	return tokensToModel(ctx, tokens), nil
}

func resolveTokensByUserID(ctx context.Context, userID persist.DBID) ([]*model.Token, error) {
	tokens, err := publicapi.For(ctx).Token.GetTokensByUserID(ctx, userID)

	if err != nil {
		return nil, err
	}

	return tokensToModel(ctx, tokens), nil
}

func resolveTokenOwnerByTokenID(ctx context.Context, tokenID persist.DBID) (*model.SplitFiUser, error) {
	token, err := publicapi.For(ctx).Token.GetTokenById(ctx, tokenID)

	if err != nil {
		return nil, err
	}

	return resolveSplitFiUserByUserID(ctx, token.OwnerUserID)
}

func resolveWalletByWalletID(ctx context.Context, walletID persist.DBID) (*model.Wallet, error) {
	wallet, err := publicapi.For(ctx).Wallet.GetWalletByID(ctx, walletID)
	if err != nil {
		return nil, err
	}

	return walletToModelSqlc(ctx, *wallet), nil
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

func resolveGeneralAllowlist(ctx context.Context) ([]*persist.ChainAddress, error) {
	addresses, err := publicapi.For(ctx).Misc.GetGeneralAllowlist(ctx)

	if err != nil {
		return nil, err
	}

	output := make([]*persist.ChainAddress, 0, len(addresses))

	for _, address := range addresses {
		chainAddress := persist.NewChainAddress(persist.Address(address), persist.ChainETH)
		output = append(output, &chainAddress)
	}

	return output, nil
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
	settings := user.NotificationSettings
	return &model.NotificationSettings{
		SomeoneFollowedYou:     settings.SomeoneFollowedYou,
		SomeoneViewedYourSplit: settings.SomeoneViewedYourSplit,
	}
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
	_, _ = token.TokenMetadata.MarshalJSON()
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

func getMediaForToken(ctx context.Context, token db.Token) model.MediaSubtype {
	med := token.Media
	switch med.MediaType {
	case persist.MediaTypeImage, persist.MediaTypeSVG:
		return getImageMedia(ctx, med)
	case persist.MediaTypeGIF:
		return getGIFMedia(ctx, med)
	case persist.MediaTypeVideo:
		return getVideoMedia(ctx, med)
	case persist.MediaTypeAudio:
		return getAudioMedia(ctx, med)
	case persist.MediaTypeHTML:
		return getHtmlMedia(ctx, med)
	case persist.MediaTypeAnimation:
		return getGltfMedia(ctx, med)
	case persist.MediaTypeJSON:
		return getJsonMedia(ctx, med)
	case persist.MediaTypeText, persist.MediaTypeBase64Text:
		return getTextMedia(ctx, med)
	case persist.MediaTypePDF:
		return getPdfMedia(ctx, med)
	case persist.MediaTypeUnknown:
		return getUnknownMedia(ctx, med)
	default:
		return getInvalidMedia(ctx, med)
	}

}

func getPreviewUrls(ctx context.Context, media persist.Media, options ...mediamapper.Option) *model.PreviewURLSet {
	url := media.ThumbnailURL.String()
	if (media.MediaType == persist.MediaTypeImage || media.MediaType == persist.MediaTypeSVG || media.MediaType == persist.MediaTypeGIF) && url == "" {
		url = media.MediaURL.String()
	}
	preview := remapLargeImageUrls(url)
	mm := mediamapper.For(ctx)

	live := media.LivePreviewURL.String()
	if media.LivePreviewURL == "" {
		live = media.MediaURL.String()
	}

	return &model.PreviewURLSet{
		Raw:        &preview,
		Thumbnail:  util.ToPointer(mm.GetThumbnailImageUrl(preview, options...)),
		Small:      util.ToPointer(mm.GetSmallImageUrl(preview, options...)),
		Medium:     util.ToPointer(mm.GetMediumImageUrl(preview, options...)),
		Large:      util.ToPointer(mm.GetLargeImageUrl(preview, options...)),
		SrcSet:     util.ToPointer(mm.GetSrcSet(preview, options...)),
		LiveRender: &live,
	}
}

func getImageMedia(ctx context.Context, media persist.Media) model.ImageMedia {
	url := remapLargeImageUrls(media.MediaURL.String())

	return model.ImageMedia{
		PreviewURLs:      getPreviewUrls(ctx, media),
		MediaURL:         util.ToPointer(media.MediaURL.String()),
		MediaType:        (*string)(&media.MediaType),
		ContentRenderURL: &url,
		Dimensions:       mediaToDimensions(media),
	}
}

func getGIFMedia(ctx context.Context, media persist.Media) model.GIFMedia {
	url := remapLargeImageUrls(media.MediaURL.String())

	return model.GIFMedia{
		PreviewURLs:       getPreviewUrls(ctx, media),
		StaticPreviewURLs: getPreviewUrls(ctx, media, mediamapper.WithStaticImage()),
		MediaURL:          util.ToPointer(media.MediaURL.String()),
		MediaType:         (*string)(&media.MediaType),
		ContentRenderURL:  &url,
		Dimensions:        mediaToDimensions(media),
	}
}

// Temporary method for handling the large "dead ringers" NFT image. This remapping
// step should actually happen as part of generating resized images with imgix.
func remapLargeImageUrls(url string) string {
	return url
}

func getVideoMedia(ctx context.Context, media persist.Media) model.VideoMedia {
	asString := media.MediaURL.String()
	videoUrls := model.VideoURLSet{
		Raw:    &asString,
		Small:  &asString,
		Medium: &asString,
		Large:  &asString,
	}

	return model.VideoMedia{
		PreviewURLs:       getPreviewUrls(ctx, media),
		MediaURL:          util.ToPointer(media.MediaURL.String()),
		MediaType:         (*string)(&media.MediaType),
		ContentRenderURLs: &videoUrls,
		Dimensions:        mediaToDimensions(media),
	}
}

func getAudioMedia(ctx context.Context, media persist.Media) model.AudioMedia {
	return model.AudioMedia{
		PreviewURLs:      getPreviewUrls(ctx, media),
		MediaURL:         util.ToPointer(media.MediaURL.String()),
		MediaType:        (*string)(&media.MediaType),
		ContentRenderURL: (*string)(&media.MediaURL),
		Dimensions:       mediaToDimensions(media),
	}
}

func getTextMedia(ctx context.Context, media persist.Media) model.TextMedia {
	return model.TextMedia{
		PreviewURLs:      getPreviewUrls(ctx, media),
		MediaURL:         util.ToPointer(media.MediaURL.String()),
		MediaType:        (*string)(&media.MediaType),
		ContentRenderURL: (*string)(&media.MediaURL),
		Dimensions:       mediaToDimensions(media),
	}
}

func getPdfMedia(ctx context.Context, media persist.Media) model.PDFMedia {
	return model.PDFMedia{
		PreviewURLs:      getPreviewUrls(ctx, media),
		MediaURL:         util.ToPointer(media.MediaURL.String()),
		MediaType:        (*string)(&media.MediaType),
		ContentRenderURL: (*string)(&media.MediaURL),
		Dimensions:       mediaToDimensions(media),
	}
}

func getHtmlMedia(ctx context.Context, media persist.Media) model.HTMLMedia {
	return model.HTMLMedia{
		PreviewURLs:      getPreviewUrls(ctx, media),
		MediaURL:         util.ToPointer(media.MediaURL.String()),
		MediaType:        (*string)(&media.MediaType),
		ContentRenderURL: (*string)(&media.MediaURL),
		Dimensions:       mediaToDimensions(media),
	}
}

func getJsonMedia(ctx context.Context, media persist.Media) model.JSONMedia {
	return model.JSONMedia{
		PreviewURLs:      getPreviewUrls(ctx, media),
		MediaURL:         util.ToPointer(media.MediaURL.String()),
		MediaType:        (*string)(&media.MediaType),
		ContentRenderURL: (*string)(&media.MediaURL),
		Dimensions:       mediaToDimensions(media),
	}
}

func getGltfMedia(ctx context.Context, media persist.Media) model.GltfMedia {
	return model.GltfMedia{
		PreviewURLs:      getPreviewUrls(ctx, media),
		MediaURL:         util.ToPointer(media.MediaURL.String()),
		MediaType:        (*string)(&media.MediaType),
		ContentRenderURL: (*string)(&media.MediaURL),
		Dimensions:       mediaToDimensions(media),
	}
}

func getUnknownMedia(ctx context.Context, media persist.Media) model.UnknownMedia {
	return model.UnknownMedia{
		PreviewURLs:      getPreviewUrls(ctx, media),
		MediaURL:         util.ToPointer(media.MediaURL.String()),
		MediaType:        (*string)(&media.MediaType),
		ContentRenderURL: (*string)(&media.MediaURL),
		Dimensions:       mediaToDimensions(media),
	}
}

func getInvalidMedia(ctx context.Context, media persist.Media) model.InvalidMedia {
	return model.InvalidMedia{
		PreviewURLs:      getPreviewUrls(ctx, media),
		MediaURL:         util.ToPointer(media.MediaURL.String()),
		MediaType:        (*string)(&media.MediaType),
		ContentRenderURL: (*string)(&media.MediaURL),
		Dimensions:       mediaToDimensions(media),
	}
}

func mediaToDimensions(media persist.Media) *model.MediaDimensions {
	var aspect float64
	if media.Dimensions.Height > 0 && media.Dimensions.Width > 0 {
		aspect = float64(media.Dimensions.Width) / float64(media.Dimensions.Height)
	}

	return &model.MediaDimensions{
		Width:       &media.Dimensions.Height,
		Height:      &media.Dimensions.Width,
		AspectRatio: &aspect,
	}
}
