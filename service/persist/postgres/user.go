package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"strings"
	"time"

	db "github.com/mutuals/go-mutuals/db/gen/coredb"

	"github.com/lib/pq"
	"github.com/mutuals/go-mutuals/service/logger"
	"github.com/mutuals/go-mutuals/service/persist"
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

	updateInfoStmt, err := db.PrepareContext(ctx, `UPDATE users SET USERNAME = $2, USERNAME_IDEMPOTENT = $3, UPDATED_AT = $4 WHERE ID = $1;`)
	//checkNoErr(err)

	// TODO update sql schema
	getByIDStmt, err := db.PrepareContext(ctx, `SELECT ID,DELETED,VERSION,USERNAME,USERNAME_IDEMPOTENT,WALLETS,UNIVERSAL,PRIMARY_WALLET_ID,CREATED_AT,UPDATED_AT FROM users WHERE ID = $1 AND DELETED = FALSE;`)
	//checkNoErr(err)

	// TODO update sql schema
	getByIDsStmt, err := db.PrepareContext(ctx, `SELECT ID,DELETED,VERSION,USERNAME,USERNAME_IDEMPOTENT,WALLETS,UNIVERSAL,PRIMARY_WALLET_ID,CREATED_AT,UPDATED_AT FROM users WHERE ID = ANY($1) AND DELETED = FALSE;`)
	//checkNoErr(err)

	// TODO update sql schema
	getByWalletIDStmt, err := db.PrepareContext(ctx, `SELECT ID,DELETED,VERSION,USERNAME,USERNAME_IDEMPOTENT,WALLETS,UNIVERSAL,PRIMARY_WALLET_ID,CREATED_AT,UPDATED_AT FROM users WHERE ARRAY[$1]::varchar[] <@ WALLETS AND DELETED = FALSE;`)
	//checkNoErr(err)

	// TODO update sql schema
	getByUsernameStmt, err := db.PrepareContext(ctx, `SELECT ID,DELETED,VERSION,USERNAME,USERNAME_IDEMPOTENT,WALLETS,UNIVERSAL,PRIMARY_WALLET_ID,CREATED_AT,UPDATED_AT FROM users WHERE USERNAME_IDEMPOTENT = $1 AND DELETED = FALSE;`)
	//checkNoErr(err)

	deleteStmt, err := db.PrepareContext(ctx, `UPDATE users SET DELETED = TRUE WHERE ID = $1;`)
	checkNoErr(err)

	getWalletIDStmt, err := db.PrepareContext(ctx, `SELECT ID FROM wallets WHERE ADDRESS = $1 AND CHAIN = $2 AND DELETED = FALSE;`)
	//checkNoErr(err)

	getWalletStmt, err := db.PrepareContext(ctx, `SELECT ADDRESS,CHAIN,WALLET_TYPE,VERSION,CREATED_AT,UPDATED_AT FROM wallets WHERE ID = $1 AND DELETED = FALSE;`)
	//checkNoErr(err)

	removeWalletFromUserStmt, err := db.PrepareContext(ctx, `UPDATE users SET WALLETS = ARRAY_REMOVE(WALLETS, $1) WHERE ID = $2 AND NOT $1 = PRIMARY_WALLET_ID AND $1 = ANY(WALLETS);`)
	//checkNoErr(err)

	deleteWalletStmt, err := db.PrepareContext(ctx, `UPDATE wallets SET DELETED = TRUE, UPDATED_AT = NOW() WHERE ID = $1;`)
	//checkNoErr(err)

	return &UserRepository{
		db:             db,
		pgx:            pgx,
		queries:        queries,
		updateInfoStmt: updateInfoStmt,

		getByIDStmt:              getByIDStmt,
		getByIDsStmt:             getByIDsStmt,
		getByWalletIDStmt:        getByWalletIDStmt,
		getByUsernameStmt:        getByUsernameStmt,
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

		res, err := u.updateInfoStmt.ExecContext(pCtx, pID, update.Username, strings.ToLower(update.UsernameIdempotent.String()), update.UpdatedAt)
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
	default:
		return fmt.Errorf("unsupported update type: %T", pUpdate)
	}

	return nil
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

	return "", nil
}

// GetByID gets the user with the given ID
func (u *UserRepository) GetByID(pCtx context.Context, pID persist.DBID) (persist.User, error) {

	user := persist.User{}
	walletIDs := []persist.DBID{}
	/*	err := u.getByIDStmt.QueryRowContext(pCtx, pID).Scan(&user.ID, &user.Deleted, &user.Version, &user.Username, &user.UsernameIdempotent, pq.Array(&walletIDs), &user.Universal, &user.PrimaryWalletID, &user.CreationTime, &user.UpdatedAt)
		if err != nil {
			if err == sql.ErrNoRows {
				return persist.User{}, persist.ErrUserNotFound{UserID: pID}
			}
			return persist.User{}, err
		}
	*/wallets := make([]persist.Wallet, len(walletIDs))

	for i, walletID := range walletIDs {
		wallet := persist.Wallet{ID: walletID}
		err := u.getWalletStmt.QueryRowContext(pCtx, walletID).Scan(&wallet.Address, &wallet.Chain, &wallet.WalletType, &wallet.Version, &wallet.CreationTime, &wallet.UpdatedAt)
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

// GetByUsername gets the user with the given username
func (u *UserRepository) GetByUsername(pCtx context.Context, pUsername string) (persist.User, error) {

	var user persist.User
	err := u.getByUsernameStmt.QueryRowContext(pCtx, strings.ToLower(pUsername)).Scan(&user.ID, &user.Deleted, &user.Version, &user.Username, &user.UsernameIdempotent, pq.Array(&user.Wallets), &user.Universal, &user.PrimaryWalletID, &user.CreationTime, &user.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return persist.User{}, persist.ErrUserNotFound{Username: pUsername}
		}
		return persist.User{}, err
	}
	return user, nil

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
