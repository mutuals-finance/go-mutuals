package user

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v4"
	"github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/service/logger"
	"strings"
	"time"

	"github.com/mutuals/go-mutuals/service/multichain"
	"github.com/mutuals/go-mutuals/service/persist/postgres"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/mutuals/go-mutuals/service/auth"
	"github.com/mutuals/go-mutuals/service/persist"
	"github.com/mutuals/go-mutuals/util"
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

// CreateUser creates a new user
func CreateUser(ctx context.Context, pUser persist.CreateUserInput, userRepo *postgres.UserRepository, queries *coredb.Queries) (user coredb.User, err error) {
	gc := util.MustGetGinContext(ctx)

	if pUser.Username != "" {
		user, err := queries.GetUserByUsername(ctx, strings.ToLower(pUser.Username))
		if err == nil && user.ID != "" {
			return coredb.User{}, persist.ErrUsernameNotAvailable{Username: pUser.Username}
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return coredb.User{}, err
		}
	}

	user, err = queries.CreateUser(ctx, coredb.CreateUserParams{
		ID:                   persist.GenerateID(),
		Username:             util.ToNullString(pUser.Username, true),
		UsernameIdempotent:   util.ToNullString(strings.ToLower(pUser.Username), true),
		Universal:            pUser.Universal,
		EmailUnsubscriptions: pUser.EmailNotificationsSettings,
	})

	/* TODO privy
	   if pUser.PrivyDID != nil {
	   		err := queries.SetPrivyDIDForUser(pCtx, db.SetPrivyDIDForUserParams{
	   			ID:       persist.GenerateID(),
	   			UserID:   userID,
	   			PrivyDid: *pUser.PrivyDID,
	   		})
	   		if err != nil {
	   			return "", err
	   		}
	   	}
	*/

	if pUser.ChainAddress.Address() != "" {
		err := queries.InsertWallet(ctx, coredb.InsertWalletParams{
			ID:      persist.GenerateID(),
			UserID:  user.ID,
			Name:    pUser.Username,
			Address: pUser.ChainAddress.Address(),
		})
		if err != nil {
			return coredb.User{}, err
		}
	}

	if pUser.Email != nil {
		if pUser.EmailStatus == persist.EmailVerificationStatusVerified {
			err := queries.UpdateUserVerifiedEmail(ctx, coredb.UpdateUserVerifiedEmailParams{
				UserID:       user.ID,
				EmailAddress: *pUser.Email,
			})
			if err != nil {
				logger.For(ctx).Errorf("failed to insert verified email address when creating new user with userID=%s\n", user.ID)
			}
		} else if pUser.EmailStatus == persist.EmailVerificationStatusUnverified {
			err := queries.UpdateUserUnverifiedEmail(ctx, coredb.UpdateUserUnverifiedEmailParams{
				UserID:       user.ID,
				EmailAddress: *pUser.Email,
			})
			if err != nil {
				logger.For(ctx).Errorf("failed to insert unverified email address when creating new user with userID=%s\n", user.ID)
			}
		}

	}

	_, _, err = auth.StartSession(gc, queries, user.ID)
	if err != nil {
		return coredb.User{}, err
	}

	return user, nil
}

// RemoveWalletsFromUser removes wallets from a user in the DB, and returns the IDs of the wallets that were removed.
// The set of removed IDs is valid even in cases where this function returns an error; it will contain the IDs of wallets
// that were successfully removed before the error occurred.
func RemoveWalletsFromUser(pCtx context.Context, pUserID persist.DBID, pWalletIDs []persist.DBID, userRepo *postgres.UserRepository) ([]persist.DBID, error) {
	removedIDs := make([]persist.DBID, 0, len(pWalletIDs))

	/*	TODO
		user, err := userRepo.GetByID(pCtx, pUserID)
				if err != nil {
					return removedIDs, err
				}

			for _, walletID := range pWalletIDs {
							if user.PrimaryWalletID.String() == walletID.String() {
								return removedIDs, errUserCannotRemovePrimaryWallet
							}
						}

					if len(user.Wallets) <= len(pWalletIDs) {
						return removedIDs, errUserCannotRemoveAllWallets
					}

				for _, walletID := range pWalletIDs {
						removed, err := userRepo.RemoveWallet(pCtx, pUserID, walletID)
						if err != nil {
							return removedIDs, err
						} else if removed {
							removedIDs = append(removedIDs, walletID)
						}
					}
	*/

	return removedIDs, nil
}

// AddWalletToUser adds a single wallet to a user in the DB because a signature needs to be provided and validated per address
func AddWalletToUser(pCtx context.Context, pUserID persist.DBID, pChainAddress persist.ChainAddress, addressAuth auth.Authenticator,
	userRepo *postgres.UserRepository, mp *multichain.Provider) error {

	authResult, err := addressAuth.Authenticate(pCtx)
	if err != nil {
		return err
	}

	if authResult.User != nil && !authResult.User.Universal {
		return persist.ErrAddressOwnedByUser{ChainAddress: pChainAddress, OwnerID: authResult.User.ID}
	}

	authenticatedAddress, ok := authResult.GetAuthenticatedAddress(pChainAddress)
	if !ok {
		return persist.ErrAddressNotOwnedByUser{ChainAddress: pChainAddress, UserID: authResult.User.ID}
	}

	if err := userRepo.AddWallet(pCtx, pUserID, authenticatedAddress.ChainAddress, authenticatedAddress.WalletType, nil); err != nil {
		return err
	}

	return nil
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
