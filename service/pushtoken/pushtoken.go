package pushtoken

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v4"
	"github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/service/persist"
)

type CreatePushTokenInput struct {
	UserID    persist.DBID
	PushToken string
}

// CreatePushToken adds a push token to a user, or returns the existing push token if it's already been
// added to this user. If the token can't be added because it belongs to another user, an error is returned.
func CreatePushToken(ctx context.Context, queries *coredb.Queries, input CreatePushTokenInput) (coredb.PushNotificationToken, error) {
	// Does the token already exist?
	token, err := queries.GetPushTokenByPushToken(ctx, input.PushToken)

	if err == nil {
		// If the token exists and belongs to the current user, return it. Attempting to re-add
		// a token that you've already registered is a no-op.
		if token.UserID == input.UserID {
			return token, nil
		}

		// Otherwise, the token belongs to another user. Return an error.
		return coredb.PushNotificationToken{}, persist.ErrPushTokenBelongsToAnotherUser{PushToken: input.PushToken}
	}

	// ErrNoRows is expected and means we can continue with creating the token. If we see any other
	// error, return it.
	if err != pgx.ErrNoRows {
		return coredb.PushNotificationToken{}, err
	}

	token, err = queries.CreatePushTokenForUser(ctx, coredb.CreatePushTokenForUserParams{
		ID:        persist.GenerateID(),
		UserID:    input.UserID,
		PushToken: input.PushToken,
	})

	if err != nil {
		return coredb.PushNotificationToken{}, err
	}

	return token, nil
}

type DeletePushTokenInput struct {
	UserID    persist.DBID
	PushToken string
}

// DeletePushToken removes a push token from a user, or does nothing if the token doesn't exist.
// If the token can't be removed because it belongs to another user, an error is returned.
func DeletePushToken(ctx context.Context, queries *coredb.Queries, input DeletePushTokenInput) error {
	existingToken, err := queries.GetPushTokenByPushToken(ctx, input.PushToken)
	if err == nil {
		// If the token exists and belongs to the current user, let them delete it.
		if existingToken.UserID == input.UserID {
			return queries.DeletePushTokensByIds(ctx, []persist.DBID{existingToken.ID})
		}

		// Otherwise, the token belongs to another user. Return an error.
		return persist.ErrPushTokenBelongsToAnotherUser{PushToken: input.PushToken}
	}

	// ErrNoRows is okay and means the token doesn't exist. Unregistering it is a no-op
	// and doesn't return an error.
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}

	return err
}
