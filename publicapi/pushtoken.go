package publicapi

import (
	"context"

	"github.com/go-playground/validator/v10"
	"github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/service/pushtoken"
	"github.com/mutuals/go-mutuals/validate"
)

type PushTokenAPI struct {
	queries   *coredb.Queries
	validator *validator.Validate
}

// CreatePushTokenForUser adds a push token to a user, or returns the existing push token if it's already been
// added to this user. If the token can't be added because it belongs to another user, an error is returned.
func (api PushTokenAPI) CreatePushTokenForUser(ctx context.Context, token string) (coredb.PushNotificationToken, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"pushToken": validate.WithTag(token, "required,min=1,max=255"),
	}); err != nil {
		return coredb.PushNotificationToken{}, err
	}

	userId, err := getAuthenticatedUserId(ctx)
	if err != nil {
		return coredb.PushNotificationToken{}, err
	}

	return pushtoken.CreatePushToken(ctx, api.queries, pushtoken.CreatePushTokenInput{
		UserID:    userId,
		PushToken: token,
	})
}

// DeletePushTokenByPushToken removes a push token from a user, or does nothing if the token doesn't exist.
// If the token can't be removed because it belongs to another user, an error is returned.
func (api PushTokenAPI) DeletePushTokenByPushToken(ctx context.Context, token string) error {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"pushToken": validate.WithTag(token, "required,min=1,max=255"),
	}); err != nil {
		return err
	}

	userId, err := getAuthenticatedUserId(ctx)
	if err != nil {
		return err
	}

	return pushtoken.DeletePushToken(ctx, api.queries, pushtoken.DeletePushTokenInput{
		UserID:    userId,
		PushToken: token,
	})
}
