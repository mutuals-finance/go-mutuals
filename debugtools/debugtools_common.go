// debugtools_common.go is always compiled and is not dependent on a build tag.
// It contains shared code used by both debugtools_enabled.go and debugtools_disabled.go.

package debugtools

import (
	"fmt"
	"github.com/SplitFi/go-splitfi/env"

	"github.com/SplitFi/go-splitfi/service/auth"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/socialauth"
)

func IsDebugEnv() bool {
	currentEnv := env.GetString("ENV")
	return currentEnv == "local" || currentEnv == "development" || currentEnv == "sandbox"
}

type DebugAuthenticator struct {
	User               *persist.User
	ChainAddresses     []persist.ChainAddress
	DebugToolsPassword string
}

func (d DebugAuthenticator) GetDescription() string {
	return fmt.Sprintf("DebugAuthenticator(user: %+v, addresses: %v)", d.User, d.ChainAddresses)
}

func NewDebugAuthenticator(user *persist.User, chainAddresses []persist.ChainAddress, debugToolsPassword string) auth.Authenticator {
	return DebugAuthenticator{
		User:               user,
		ChainAddresses:     chainAddresses,
		DebugToolsPassword: debugToolsPassword,
	}
}

type DebugSocialAuthenticator struct {
	Provider           persist.SocialProvider
	ID                 string
	Metadata           map[string]interface{}
	DebugToolsPassword string
}

func NewDebugSocialAuthenticator(provider persist.SocialProvider, id string, metadata map[string]interface{}, debugToolsPassword string) socialauth.Authenticator {
	return DebugSocialAuthenticator{
		Provider:           provider,
		ID:                 id,
		Metadata:           metadata,
		DebugToolsPassword: debugToolsPassword,
	}
}
