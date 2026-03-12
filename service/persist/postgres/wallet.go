package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	db "github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/service/persist"
)

// WalletRepository is a repository for wallets
type WalletRepository struct {
	db      *sql.DB
	queries *db.Queries

	getByIDStmt      *sql.Stmt
	getByAddressStmt *sql.Stmt
	getByUserIDStmt  *sql.Stmt
}

// NewWalletRepository creates a new postgres repository for interacting with wallets
func NewWalletRepository(db *sql.DB, queries *db.Queries) *WalletRepository {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	getByIDStmt, _ := db.PrepareContext(ctx, `SELECT ID,VERSION,CREATED_AT,UPDATED_AT,ADDRESS,WALLET_TYPE,CHAIN FROM wallets WHERE ID = $1 AND DELETED = FALSE;`)
	getByAddressStmt, _ := db.PrepareContext(ctx, `SELECT ID,VERSION,CREATED_AT,UPDATED_AT,ADDRESS,WALLET_TYPE,CHAIN FROM wallets WHERE ADDRESS = $1 AND DELETED = FALSE;`)
	getByUserIDStmt, _ := db.PrepareContext(ctx, `SELECT w.ID,w.VERSION,w.CREATED_AT,w.UPDATED_AT,w.ADDRESS,w.WALLET_TYPE,w.CHAIN FROM users u, UNNEST(u.wallets) WITH ORDINALITY AS uw(wallet_id, wallet_ord) INNER JOIN wallets w ON w.id = uw.wallet_id WHERE u.id = $1 AND u.deleted = FALSE AND w.deleted = FALSE ORDER BY uw.wallet_ord;`)

	return &WalletRepository{
		db:               db,
		queries:          queries,
		getByIDStmt:      getByIDStmt,
		getByAddressStmt: getByAddressStmt,
		getByUserIDStmt:  getByUserIDStmt,
	}
}

func scanWallet(row interface{ Scan(...any) error }, wallet *persist.Wallet) error {
	return row.Scan(&wallet.ID, &wallet.Version, &wallet.CreationTime, &wallet.UpdatedAt, &wallet.Address, &wallet.WalletType, &wallet.Network)
}

// GetByID returns a wallet by its ID
func (w *WalletRepository) GetByID(ctx context.Context, ID persist.DBID) (persist.Wallet, error) {
	var wallet persist.Wallet
	if err := scanWallet(w.getByIDStmt.QueryRowContext(ctx, ID), &wallet); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return wallet, persist.ErrWalletNotFoundByID{ID: ID}
		}
		return wallet, err
	}
	return wallet, nil
}

// GetByAddress returns a wallet by its address
func (w *WalletRepository) GetByAddress(ctx context.Context, address persist.Address) (persist.Wallet, error) {
	var wallet persist.Wallet
	if err := scanWallet(w.getByAddressStmt.QueryRowContext(ctx, address), &wallet); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return wallet, persist.ErrWalletNotFoundByAddress{Address: address}
		}
		return wallet, err
	}
	return wallet, nil
}

// GetByUserID returns all wallets owned by the specified user
func (w *WalletRepository) GetByUserID(ctx context.Context, userID persist.DBID) ([]persist.Wallet, error) {
	rows, err := w.getByUserIDStmt.QueryContext(ctx, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wallets []persist.Wallet
	for rows.Next() {
		var wallet persist.Wallet
		if err := scanWallet(rows, &wallet); err != nil {
			return nil, err
		}
		wallets = append(wallets, wallet)
	}
	return wallets, rows.Err()
}
