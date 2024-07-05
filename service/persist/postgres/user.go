package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"strings"
	"time"

	db "github.com/SplitFi/go-splitfi/db/gen/coredb"

	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/lib/pq"
)

// UserRepository represents a user repository in the postgres database
type UserRepository struct {
	db                       *sql.DB
	pgx                      *pgxpool.Pool
	queries                  *db.Queries
	updateInfoStmt           *sql.Stmt
	getByIDStmt              *sql.Stmt
	getByIDsStmt             *sql.Stmt
	getByWalletIDStmt        *sql.Stmt
	getByUsernameStmt        *sql.Stmt
	getByVerifiedEmailStmt   *sql.Stmt
	deleteStmt               *sql.Stmt
	getGalleriesStmt         *sql.Stmt
	updateCollectionsStmt    *sql.Stmt
	deleteGalleryStmt        *sql.Stmt
	getWalletIDStmt          *sql.Stmt
	getWalletStmt            *sql.Stmt
	removeWalletFromUserStmt *sql.Stmt
	deleteWalletStmt         *sql.Stmt
	addFollowerStmt          *sql.Stmt
	removeFollowerStmt       *sql.Stmt
}

// NewUserRepository creates a new postgres repository for interacting with users
// TODO joins for users to wallets and wallets to addresses
func NewUserRepository(db *sql.DB, queries *db.Queries, pgx *pgxpool.Pool) *UserRepository {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	updateInfoStmt, err := db.PrepareContext(ctx, `UPDATE users SET USERNAME = $2, USERNAME_IDEMPOTENT = $3, LAST_UPDATED = $4 WHERE ID = $1;`)
	checkNoErr(err)

	// TODO update sql schema
	getByIDStmt, err := db.PrepareContext(ctx, `SELECT ID,DELETED,VERSION,USERNAME,USERNAME_IDEMPOTENT,WALLETS,UNIVERSAL,PRIMARY_WALLET_ID,CREATED_AT,LAST_UPDATED FROM users WHERE ID = $1 AND DELETED = FALSE;`)
	checkNoErr(err)

	// TODO update sql schema
	getByIDsStmt, err := db.PrepareContext(ctx, `SELECT ID,DELETED,VERSION,USERNAME,USERNAME_IDEMPOTENT,WALLETS,UNIVERSAL,PRIMARY_WALLET_ID,CREATED_AT,LAST_UPDATED FROM users WHERE ID = ANY($1) AND DELETED = FALSE;`)
	checkNoErr(err)

	// TODO update sql schema
	getByWalletIDStmt, err := db.PrepareContext(ctx, `SELECT ID,DELETED,VERSION,USERNAME,USERNAME_IDEMPOTENT,WALLETS,UNIVERSAL,PRIMARY_WALLET_ID,CREATED_AT,LAST_UPDATED FROM users WHERE ARRAY[$1]::varchar[] <@ WALLETS AND DELETED = FALSE;`)
	checkNoErr(err)

	// TODO update sql schema
	getByUsernameStmt, err := db.PrepareContext(ctx, `SELECT ID,DELETED,VERSION,USERNAME,USERNAME_IDEMPOTENT,WALLETS,UNIVERSAL,PRIMARY_WALLET_ID,CREATED_AT,LAST_UPDATED FROM users WHERE USERNAME_IDEMPOTENT = $1 AND DELETED = FALSE;`)
	checkNoErr(err)

	// TODO update sql schema
	getByVerifiedEmailStmt, err := db.PrepareContext(ctx, `SELECT ID,DELETED,VERSION,USERNAME,USERNAME_IDEMPOTENT,WALLETS,UNIVERSAL,PRIMARY_WALLET_ID,CREATED_AT,LAST_UPDATED FROM pii.user_view WHERE PII_VERIFIED_EMAIL_ADDRESS = $1 AND DELETED = FALSE;`)
	checkNoErr(err)

	deleteStmt, err := db.PrepareContext(ctx, `UPDATE users SET DELETED = TRUE WHERE ID = $1;`)
	checkNoErr(err)

	getWalletIDStmt, err := db.PrepareContext(ctx, `SELECT ID FROM wallets WHERE ADDRESS = $1 AND CHAIN = $2 AND DELETED = FALSE;`)
	checkNoErr(err)

	getWalletStmt, err := db.PrepareContext(ctx, `SELECT ADDRESS,CHAIN,WALLET_TYPE,VERSION,CREATED_AT,LAST_UPDATED FROM wallets WHERE ID = $1 AND DELETED = FALSE;`)
	checkNoErr(err)

	removeWalletFromUserStmt, err := db.PrepareContext(ctx, `UPDATE users SET WALLETS = ARRAY_REMOVE(WALLETS, $1) WHERE ID = $2 AND NOT $1 = PRIMARY_WALLET_ID AND $1 = ANY(WALLETS);`)
	checkNoErr(err)

	deleteWalletStmt, err := db.PrepareContext(ctx, `UPDATE wallets SET DELETED = TRUE, LAST_UPDATED = NOW() WHERE ID = $1;`)
	checkNoErr(err)

	return &UserRepository{
		db:             db,
		pgx:            pgx,
		queries:        queries,
		updateInfoStmt: updateInfoStmt,

		getByIDStmt:              getByIDStmt,
		getByIDsStmt:             getByIDsStmt,
		getByWalletIDStmt:        getByWalletIDStmt,
		getByUsernameStmt:        getByUsernameStmt,
		getByVerifiedEmailStmt:   getByVerifiedEmailStmt,
		deleteStmt:               deleteStmt,
		getWalletIDStmt:          getWalletIDStmt,
		getWalletStmt:            getWalletStmt,
		removeWalletFromUserStmt: removeWalletFromUserStmt,
		deleteWalletStmt:         deleteWalletStmt,
	}
}

// UpdateByID updates the user with the given ID
func (u *UserRepository) UpdateByID(pCtx context.Context, pID persist.DBID, pUpdate interface{}) error {

	switch update := pUpdate.(type) {
	case persist.UserUpdateInfoInput:
		aUser, err := u.GetByUsername(pCtx, update.Username.String())
		if err != nil {
			errNotFound := persist.ErrUserNotFound{}
			if !errors.As(err, &errNotFound) {
				return err
			}
		} else {
			if aUser.ID != "" && aUser.ID != pID {
				return persist.ErrUsernameNotAvailable{Username: update.Username.String()}
			}
		}

		res, err := u.updateInfoStmt.ExecContext(pCtx, pID, update.Username, strings.ToLower(update.UsernameIdempotent.String()), update.LastUpdated)
		if err != nil {
			return err
		}
		rows, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if rows == 0 {
			return persist.ErrUserNotFound{UserID: pID}
		}
	case persist.UserUpdateNotificationSettings:
		return u.queries.UpdateNotificationSettingsByID(pCtx, db.UpdateNotificationSettingsByIDParams{
			ID:                   pID,
			NotificationSettings: update.NotificationSettings,
		})
	default:
		return fmt.Errorf("unsupported update type: %T", pUpdate)
	}

	return nil

}

func (u *UserRepository) createWalletWithTx(ctx context.Context, queries *db.Queries, chainAddress persist.ChainAddress, walletType persist.WalletType, userID persist.DBID) (persist.DBID, error) {
	if queries == nil {
		queries = u.queries
	}

	wallet, err := queries.GetWalletByAddressAndL1Chain(ctx, db.GetWalletByAddressAndL1ChainParams{
		Address: chainAddress.Address(),
		L1Chain: chainAddress.Chain().L1Chain(),
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	walletID := wallet.ID

	if walletID != "" {
		user, err := queries.GetUserByWalletID(ctx, walletID.String())
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return "", err
		}

		if user.ID != "" {
			if user.ID == userID {
				// Wallet already belongs to the user; do nothing
				return walletID, nil
			}
			if user.Universal {
				err = queries.DeleteUserByID(ctx, user.ID)
				if err != nil {
					return "", err
				}
			} else {
				return "", persist.ErrAddressOwnedByUser{ChainAddress: chainAddress, OwnerID: user.ID}
			}
		}

		logger.For(ctx).Infof("wallet %s already exists, but is not owned by a user", walletID)
		// If the wallet exists but doesn't belong to anyone, it should be deleted.

		err = queries.DeleteWalletByID(ctx, walletID)
		if err != nil {
			return "", err
		}

	}

	newWalletID := persist.GenerateID()
	// At this point, we know there's no existing wallet in the database with this ChainAddress, so let's make a new one!
	err = queries.InsertWallet(ctx, db.InsertWalletParams{
		ID:         newWalletID,
		Address:    chainAddress.Address(),
		Chain:      chainAddress.Chain(),
		WalletType: walletType,
		UserID:     userID,
		// TODO L1Chain
		// L1Chain:    chainAddress.Chain().L1Chain(),
	})
	if err != nil {
		return "", persist.ErrWalletCreateFailed{
			ChainAddress: chainAddress,
			WalletID:     walletID,
			Err:          err,
		}
	}

	return newWalletID, nil
}

// Create creates a new user
func (u *UserRepository) Create(pCtx context.Context, pUser persist.CreateUserInput, queries *db.Queries) (persist.DBID, error) {
	if queries == nil {
		tx, err := u.pgx.BeginTx(pCtx, pgx.TxOptions{})
		if err != nil {
			return "", err
		}
		queries = u.queries.WithTx(tx)
		defer tx.Rollback(pCtx)
		defer func() {
			err := tx.Commit(pCtx)
			if err != nil {
				logger.For(pCtx).Errorf("failed to commit transaction: %v", err)
			}
		}()
	}

	if pUser.Username != "" {
		user, err := queries.GetUserByUsername(pCtx, strings.ToLower(pUser.Username))
		if err == nil && user.ID != "" {
			return "", persist.ErrUsernameNotAvailable{Username: pUser.Username}
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return "", err
		}
	}

	userID, err := queries.InsertUser(pCtx, db.InsertUserParams{
		ID:                   persist.GenerateID(),
		Username:             util.ToNullString(pUser.Username, true),
		UsernameIdempotent:   util.ToNullString(strings.ToLower(pUser.Username), true),
		Universal:            pUser.Universal,
		EmailUnsubscriptions: pUser.EmailNotificationsSettings,
	})

	if err != nil {
		return "", err
	}

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
		_, err = u.createWalletWithTx(pCtx, queries, pUser.ChainAddress, pUser.WalletType, userID)
		if err != nil {
			return "", err
		}
	}

	if pUser.Email != nil {
		if pUser.EmailStatus == persist.EmailVerificationStatusVerified {
			err := queries.UpdateUserVerifiedEmail(pCtx, db.UpdateUserVerifiedEmailParams{
				UserID:       userID,
				EmailAddress: *pUser.Email,
			})
			if err != nil {
				logger.For(pCtx).Errorf("failed to insert verified email address when creating new user with userID=%s\n", userID)
			}
		} else if pUser.EmailStatus == persist.EmailVerificationStatusUnverified {
			err := queries.UpdateUserUnverifiedEmail(pCtx, db.UpdateUserUnverifiedEmailParams{
				UserID:       userID,
				EmailAddress: *pUser.Email,
			})
			if err != nil {
				logger.For(pCtx).Errorf("failed to insert unverified email address when creating new user with userID=%s\n", userID)
			}
		}

	}
	return userID, nil
}

// GetByID gets the user with the given ID
func (u *UserRepository) GetByID(pCtx context.Context, pID persist.DBID) (persist.User, error) {

	user := persist.User{}
	walletIDs := []persist.DBID{}
	err := u.getByIDStmt.QueryRowContext(pCtx, pID).Scan(&user.ID, &user.Deleted, &user.Version, &user.Username, &user.UsernameIdempotent, pq.Array(&walletIDs), &user.Universal, &user.PrimaryWalletID, &user.CreationTime, &user.LastUpdated)
	if err != nil {
		if err == sql.ErrNoRows {
			return persist.User{}, persist.ErrUserNotFound{UserID: pID}
		}
		return persist.User{}, err
	}
	wallets := make([]persist.Wallet, len(walletIDs))

	for i, walletID := range walletIDs {
		wallet := persist.Wallet{ID: walletID}
		err = u.getWalletStmt.QueryRowContext(pCtx, walletID).Scan(&wallet.Address, &wallet.Chain, &wallet.WalletType, &wallet.Version, &wallet.CreationTime, &wallet.LastUpdated)
		if err == nil {
			wallets[i] = wallet
		}
		if err != nil && err != sql.ErrNoRows {
			return persist.User{}, fmt.Errorf("failed to get wallet: %w", err)
		}
	}
	user.Wallets = wallets

	return user, nil
}

// GetByIDs gets all the users with the given IDs
func (u *UserRepository) GetByIDs(pCtx context.Context, pIDs []persist.DBID) ([]persist.User, error) {

	results := make([]persist.User, 0, len(pIDs))
	rows, err := u.getByIDsStmt.QueryContext(pCtx, pIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		user := persist.User{}
		walletIDs := []persist.DBID{}
		rows.Scan(&user.ID, &user.Deleted, &user.Version, &user.Username, &user.UsernameIdempotent, pq.Array(&walletIDs), &user.Universal, &user.PrimaryWalletID, &user.CreationTime, &user.LastUpdated)
		wallets := make([]persist.Wallet, len(walletIDs))

		for i, walletID := range walletIDs {
			wallet := persist.Wallet{ID: walletID}
			err = u.getWalletStmt.QueryRowContext(pCtx, walletID).Scan(&wallet.Address, &wallet.Chain, &wallet.WalletType, &wallet.Version, &wallet.CreationTime, &wallet.LastUpdated)
			if err != nil && err != sql.ErrNoRows {
				return nil, fmt.Errorf("failed to get wallet: %w", err)
			}
			wallets[i] = wallet
		}
		user.Wallets = wallets
		results = append(results, user)
	}

	return results, nil
}

// GetByChainAddress gets the user who owns the wallet with the specified ChainAddress (if any)
func (u *UserRepository) GetByChainAddress(pCtx context.Context, pChainAddress persist.L1ChainAddress) (persist.User, error) {
	var walletID persist.DBID

	err := u.getWalletIDStmt.QueryRowContext(pCtx, pChainAddress.Address(), pChainAddress.L1Chain()).Scan(&walletID)
	if err != nil {
		if err == sql.ErrNoRows {
			return persist.User{}, persist.ErrWalletNotFoundByAddress{Address: pChainAddress}
		}
		return persist.User{}, err
	}

	return u.GetByWalletID(pCtx, walletID)
}

// GetByWalletID gets the user with the given wallet in their list of addresses
func (u *UserRepository) GetByWalletID(pCtx context.Context, pWalletID persist.DBID) (persist.User, error) {

	var user persist.User
	err := u.getByWalletIDStmt.QueryRowContext(pCtx, pWalletID).Scan(&user.ID, &user.Deleted, &user.Version, &user.Username, &user.UsernameIdempotent, pq.Array(&user.Wallets), &user.Universal, &user.PrimaryWalletID, &user.CreationTime, &user.LastUpdated)
	if err != nil {
		if err == sql.ErrNoRows {
			return persist.User{}, persist.ErrUserNotFound{WalletID: pWalletID}
		}
		return persist.User{}, err
	}

	wallets := make([]persist.Wallet, len(user.Wallets))

	for i, wallet := range user.Wallets {
		err = u.getWalletStmt.QueryRowContext(pCtx, wallet.ID).Scan(&wallet.Address, &wallet.Chain, &wallet.WalletType, &wallet.Version, &wallet.CreationTime, &wallet.LastUpdated)
		if err != nil && err != sql.ErrNoRows {
			return persist.User{}, fmt.Errorf("failed to get wallet: %w", err)
		}
		wallets[i] = wallet
	}

	user.Wallets = wallets

	return user, nil

}

// GetByUsername gets the user with the given username
func (u *UserRepository) GetByUsername(pCtx context.Context, pUsername string) (persist.User, error) {

	var user persist.User
	err := u.getByUsernameStmt.QueryRowContext(pCtx, strings.ToLower(pUsername)).Scan(&user.ID, &user.Deleted, &user.Version, &user.Username, &user.UsernameIdempotent, pq.Array(&user.Wallets), &user.Universal, &user.PrimaryWalletID, &user.CreationTime, &user.LastUpdated)
	if err != nil {
		if err == sql.ErrNoRows {
			return persist.User{}, persist.ErrUserNotFound{Username: pUsername}
		}
		return persist.User{}, err
	}
	return user, nil

}

// GetByEmail gets the user with the given email address
func (u *UserRepository) GetByVerifiedEmail(pCtx context.Context, pEmail persist.Email) (persist.User, error) {

	var user persist.User
	err := u.getByVerifiedEmailStmt.QueryRowContext(pCtx, pEmail.String()).Scan(&user.ID, &user.Deleted, &user.Version, &user.Username, &user.UsernameIdempotent, pq.Array(&user.Wallets), &user.Universal, &user.PrimaryWalletID, &user.CreationTime, &user.LastUpdated)
	if err != nil {
		if err == sql.ErrNoRows {
			return persist.User{}, persist.ErrUserNotFound{Email: pEmail}
		}
		return persist.User{}, err
	}
	return user, nil

}

// AddWallet adds an address to user as well as ensures that the wallet and address exists
func (u *UserRepository) AddWallet(pCtx context.Context, pUserID persist.DBID, pChainAddress persist.ChainAddress, pWalletType persist.WalletType, queries *db.Queries) error {
	_, err := u.createWalletWithTx(pCtx, queries, pChainAddress, pWalletType, pUserID)
	return err
}

// RemoveWallet removes the specified wallet from a user. Returns true if the wallet exists and was successfully removed,
// false if the wallet does not exist or an error was encountered.
func (u *UserRepository) RemoveWallet(pCtx context.Context, pUserID persist.DBID, pWalletID persist.DBID) (bool, error) {
	tx, err := u.db.BeginTx(pCtx, nil)
	if err != nil {
		return false, err
	}

	defer tx.Rollback()

	res, err := tx.StmtContext(pCtx, u.removeWalletFromUserStmt).ExecContext(pCtx, pWalletID, pUserID)
	if err != nil {
		return false, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	if rows == 0 {
		return false, nil
	}

	if _, err := tx.StmtContext(pCtx, u.deleteWalletStmt).ExecContext(pCtx, pWalletID); err != nil {
		return false, err
	}

	if err := tx.Commit(); err != nil {
		return false, err
	}

	return true, nil
}

// Delete deletes the user with the given ID
func (u *UserRepository) Delete(pCtx context.Context, pID persist.DBID) error {

	res, err := u.deleteStmt.ExecContext(pCtx, pID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return persist.ErrUserNotFound{UserID: pID}
	}

	return nil
}

// MergeUsers merges the given users into the first user
func (u *UserRepository) MergeUsers(pCtx context.Context, pInitialUser persist.DBID, pSecondUser persist.DBID) error {

	var user persist.User
	if err := u.getByIDStmt.QueryRowContext(pCtx, pInitialUser).Scan(&user.ID, &user.Deleted, &user.Version, &user.Username, &user.UsernameIdempotent, pq.Array(&user.Wallets), &user.Universal, &user.CreationTime, &user.LastUpdated); err != nil {
		return err
	}

	var secondUser persist.User
	if err := u.getByIDStmt.QueryRowContext(pCtx, pSecondUser).Scan(&secondUser.ID, &secondUser.Deleted, &secondUser.Version, &secondUser.Username, &secondUser.UsernameIdempotent, pq.Array(&secondUser.Wallets), &user.Universal, &secondUser.CreationTime, &secondUser.LastUpdated); err != nil {
		return err
	}

	tx, err := u.db.BeginTx(pCtx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.StmtContext(pCtx, u.deleteStmt).ExecContext(pCtx, secondUser.ID); err != nil {
		return err
	}

	return tx.Commit()
}

func (u *UserRepository) FillWalletDataForUser(pCtx context.Context, user *persist.User) error {

	if len(user.Wallets) == 0 {
		return nil
	}

	wallets := make([]persist.Wallet, 0, len(user.Wallets))
	for _, wallet := range user.Wallets {
		wallet := persist.Wallet{ID: wallet.ID}
		if err := u.getWalletStmt.QueryRowContext(pCtx, wallet.ID).Scan(&wallet.Address, &wallet.Chain, &wallet.WalletType, &wallet.Version, &wallet.CreationTime, &wallet.LastUpdated); err != nil {
			return err
		}

		wallets = append(wallets, wallet)
	}

	user.Wallets = wallets

	return nil
}
