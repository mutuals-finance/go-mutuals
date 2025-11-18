package user

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/service/logger"
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
	UserID   persist.DBID    `json:"user_id" form:"user_id"`
	Address  persist.Address `json:"address" form:"address"`
	Chain    persist.Chain   `json:"chain" form:"chain"`
	Username string          `json:"username" form:"username"`
}

// GetUserOutput is the output of the user get pipeline
type GetUserOutput struct {
	UserID    persist.DBID     `json:"id"`
	Username  string           `json:"username"`
	BioStr    string           `json:"bio"`
	Addresses []persist.Wallet `json:"addresses"`
	CreatedAt time.Time        `json:"created_at"`
}

// AddUserAddressesInput is the input for the user add addresses pipeline and also user creation pipeline given that they have the same requirements
type AddUserAddressesInput struct {

	// needed because this is a new user that cant be logged into, and the client creating
	// the user still needs to prove ownership of their address.
	Signature  string             `json:"signature" binding:"signature"`
	Nonce      string             `json:"nonce"`
	Address    persist.Address    `json:"address"   binding:"required"`
	Chain      persist.Chain      `json:"chain"`
	WalletType persist.WalletType `json:"wallet_type"`
}

// AddUserAddressOutput is the output of the user add address pipeline
type AddUserAddressOutput struct {
	SignatureValid bool `json:"signature_valid"`
}

// RemoveUserAddressesInput is the input for the user remove addresses pipeline
type RemoveUserAddressesInput struct {
	Addresses []persist.Address `json:"addresses"   binding:"required"`
	Chains    []persist.Chain   `json:"chains"      binding:"required"`
}

// CreateUserOutput is the output of the user create pipeline
type CreateUserOutput struct {
	SignatureValid bool         `json:"signature_valid"`
	JWTtoken       string       `json:"jwt_token"` // JWT token is sent back to user to use to continue onboarding
	UserID         persist.DBID `json:"user_id"`
	PoolID         persist.DBID `json:"pool_id"`
}

// MergeUsersInput is the input for the user merge pipeline
type MergeUsersInput struct {
	SecondUserID persist.DBID       `json:"second_user_id" binding:"required"`
	Signature    string             `json:"signature" binding:"signature"`
	Nonce        string             `json:"nonce"`
	Address      persist.Address    `json:"address"   binding:"required"`
	Chain        persist.Chain      `json:"chain"`
	WalletType   persist.WalletType `json:"wallet_type"`
}

type CreateUserInput struct {
	DID string
}

// CreateUser creates a new user
func CreateUser(ctx context.Context, in CreateUserInput, repos *postgres.Repositories, queries *coredb.Queries) (user coredb.User, err error) {
	gc := util.MustGetGinContext(ctx)
	tx, err := repos.BeginTx(ctx)
	if err != nil {
		return coredb.User{}, err
	}

	txQueries := queries.WithTx(tx)
	defer tx.Rollback(ctx)

	user, err = queries.CreateUser(ctx, coredb.CreateUserParams{
		ID:  persist.GenerateID(),
		Did: in.DID,
	})

	err = txQueries.AddPiiAccountCreationInfo(ctx, coredb.AddPiiAccountCreationInfoParams{
		UserID:    user.ID,
		IpAddress: gc.ClientIP(),
	})

	if err != nil {
		logger.For(ctx).Warnf("failed to get IP address for userID %s: %s\n", user.ID, err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return coredb.User{}, err
	}

	return user, nil
}

// UpdateUserInfo updates a user by ID and ensures that if they are using an ENS name as a username that their address resolves to that ENS
func UpdateUserInfo(pCtx context.Context, userID persist.DBID, username string, userRepository *postgres.UserRepository, ethClient *ethclient.Client) error {
	if strings.HasSuffix(strings.ToLower(username), ".eth") {
		/*		user, err := userRepository.GetByID(pCtx, userID)
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
		userID,
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
