// debugtools_common.go is always compiled and is not dependent on a build tag.
// It contains shared code used by both debugtools_enabled.go and debugtools_disabled.go.

package debugtools

import (
	"context"
	"fmt"
	db "github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/env"

	"github.com/mutuals/go-mutuals/service/auth"
	"github.com/mutuals/go-mutuals/service/persist"
)

func IsDebugEnv() bool {
	currentEnv := env.GetString("ENV")
	return currentEnv == "local" || currentEnv == "development" || currentEnv == "sandbox"
}

type DebugAuthenticator struct {
	User               *db.User
	ChainAddresses     []persist.ChainAddress
	DebugToolsPassword string
}

func (d DebugAuthenticator) GetDescription() string {
	return fmt.Sprintf("DebugAuthenticator(user: %+v, addresses: %v)", d.User, d.ChainAddresses)
}

func (d DebugAuthenticator) UserRegistered(context.Context) (bool, error) {
	return false, nil
}

func NewDebugAuthenticator(user *db.User, chainAddresses []persist.ChainAddress, debugToolsPassword string) auth.Authenticator {
	return DebugAuthenticator{
		User:               user,
		ChainAddresses:     chainAddresses,
		DebugToolsPassword: debugToolsPassword,
	}
}
