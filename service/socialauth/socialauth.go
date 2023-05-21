package socialauth

import (
	"context"

	"github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/redis"
	"github.com/SplitFi/go-splitfi/service/twitter"
	"github.com/SplitFi/go-splitfi/util"
)

type SocialAuthResult struct {
	Provider persist.SocialProvider `json:"provider,required" binding:"required"`
	ID       string                 `json:"id,required" binding:"required"`
	Metadata map[string]interface{} `json:"metadata"`
}

type Authenticator interface {
	Authenticate(context.Context) (*SocialAuthResult, error)
}

type TwitterAuthenticator struct {
	Queries *coredb.Queries
	Redis   *redis.Cache

	UserID   persist.DBID
	AuthCode string
}

func (a TwitterAuthenticator) Authenticate(ctx context.Context) (*SocialAuthResult, error) {
	tAPI := twitter.NewAPI(a.Queries, a.Redis)

	ids, access, err := tAPI.GetAuthedUserFromCode(ctx, a.AuthCode)
	if err != nil {
		return nil, err
	}

	err = a.Queries.UpsertSocialOAuth(ctx, coredb.UpsertSocialOAuthParams{
		ID:           persist.GenerateID(),
		UserID:       a.UserID,
		Provider:     persist.SocialProviderTwitter,
		AccessToken:  util.ToNullString(access.AccessToken, false),
		RefreshToken: util.ToNullString(access.RefreshToken, false),
	})
	if err != nil {
		return nil, err
	}

	return &SocialAuthResult{
		Provider: persist.SocialProviderTwitter,
		ID:       ids.ID,
		Metadata: map[string]interface{}{
			"username":          ids.Username,
			"name":              ids.Name,
			"profile_image_url": ids.ProfileImageURL,
		},
	}, nil
}
