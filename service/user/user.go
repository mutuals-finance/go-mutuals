package user

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/service/auth"
	"github.com/mutuals/go-mutuals/service/persist/postgres"
	"github.com/mutuals/go-mutuals/util"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mutuals/go-mutuals/service/persist"
)

var errUserCannotRemoveAllWallets = errors.New("user does not have enough wallets to remove")
var errUserCannotRemovePrimaryWallet = errors.New("cannot remove primary wallet address")
var errMustResolveENS = errors.New("ENS username must resolve to owner address")

// GetUserInput is the input for the user get pipeline
type GetUserInput struct {
	UserId   persist.DBID    `json:"user_id" form:"user_id"`
	Address  persist.Address `json:"address" form:"address"`
	Chain    persist.Chain   `json:"chain" form:"chain"`
	Username string          `json:"username" form:"username"`
}

// GetUserOutput is the output of the user get pipeline
type GetUserOutput struct {
	UserId    persist.DBID     `json:"id"`
	Username  string           `json:"username"`
	BioStr    string           `json:"bio"`
	Addresses []persist.Wallet `json:"addresses"`
	CreatedAt time.Time        `json:"created_at"`
}

// RemoveUserAddressesInput is the input for the user remove addresses pipeline
type RemoveUserAddressesInput struct {
	Addresses []persist.Address `json:"addresses"   binding:"required"`
	Chains    []persist.Chain   `json:"chains"      binding:"required"`
}

type CreateUserInput struct {
	DID string
}

// CreateUser creates a new user
func CreateUser(ctx context.Context, queries *coredb.Queries) (user coredb.User, err error) {
	gc := util.MustGetGinContext(ctx)
	userId := auth.GetUserIdFromCtx(gc)
	linkedAccounts := auth.GetLinkedAccountsFromCtx(gc)

	params := coredb.CreateUserParams{UserID: userId}
	for _, a := range linkedAccounts {
		params.ID = append(params.ID, a.Id)
		params.Type = append(params.Type, a.Type)
		params.Address = append(params.Address, a.Address)
		params.ChainType = append(params.ChainType, a.ChainType)
		params.WalletClientType = append(params.WalletClientType, a.WalletClientType)
		params.LinkedAt = append(params.LinkedAt, time.Unix(a.Lv, 0))
	}

	user, err = queries.CreateUser(ctx, params)
	if err != nil {
		return coredb.User{}, err
	}

	return user, nil
}

// UpdateUserInfo updates a user by ID and ensures that if they are using an ENS name as a username that their address resolves to that ENS
func UpdateUserInfo(pCtx context.Context, userId persist.DBID, username string, userRepository *postgres.UserRepository, ethClient *ethclient.Client) error {
	if strings.HasSuffix(strings.ToLower(username), ".eth") {
		/*		user, err := userRepository.GetByID(pCtx, userId)
				if err != nil {
					return err
				}
				can := false
				for _, addr := range user.Wallets {
					if resolves, _ := eth.ResolvesENS(pCtx, username, addr.Address, ethClient); resolves {
						can = true
						break
					}
				}
				if !can {
					return errMustResolveENS
				}
		*/
	}

	err := userRepository.UpdateByID(
		pCtx,
		userId,
		persist.UserUpdateInfoInput{
			UsernameIdempotent: persist.NullString(strings.ToLower(username)),
			Username:           persist.NullString(username),
		},
	)
	if err != nil {
		return err
	}
	return nil
}
