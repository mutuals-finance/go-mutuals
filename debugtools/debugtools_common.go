// debugtools_common.go is always compiled and is not dependent on a build tag.
// It contains shared code used by both debugtools_enabled.go and debugtools_disabled.go.

package debugtools

import (
	"fmt"
	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/env"

	"github.com/SplitFi/go-splitfi/service/auth"
	"github.com/SplitFi/go-splitfi/service/persist"
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

func NewDebugAuthenticator(user *db.User, chainAddresses []persist.ChainAddress, debugToolsPassword string) auth.Authenticator {
	return DebugAuthenticator{
		User:               user,
		ChainAddresses:     chainAddresses,
		DebugToolsPassword: debugToolsPassword,
	}
}
