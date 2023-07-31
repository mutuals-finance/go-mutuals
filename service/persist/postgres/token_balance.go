package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/service/persist"
)

// TokenBalanceRepository represents a balance repository in the postgres database
type TokenBalanceRepository struct {
	db                                   *sql.DB
	queries                              *db.Queries
	getByOwnerStmt                       *sql.Stmt
	getByOwnerPaginateStmt               *sql.Stmt
	getByTokenStmt                       *sql.Stmt
	getByTokenPaginateStmt               *sql.Stmt
	getByIdentifiersStmt                 *sql.Stmt
	upsertByIdentifiersStmt              *sql.Stmt
	updateBalanceUnsafeStmt              *sql.Stmt
	updateBalanceByIdentifiersUnsafeStmt *sql.Stmt
}

// NewTokenBalanceRepository creates a new postgres repository for interacting with balances
func NewTokenBalanceRepository(db *sql.DB, queries *db.Queries) *TokenBalanceRepository {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	getByOwnerStmt, err := db.PrepareContext(ctx, `SELECT b.ID,b.VERSION,b.CREATED_AT,b.LAST_UPDATED,b.OWNER_ADDRESS,b.BALANCE,b.BLOCK_NUMBER,
		t.ID,t.TOKEN_TYPE,t.CHAIN,t.NAME,t.SYMBOL,t.LOGO,t.DECIMALS,t.TOTAL_SUPPLY,t.CONTRACT_ADDRESS,t.BLOCK_NUMBER,t.VERSION,t.CREATED_AT,t.LAST_UPDATED,t.IS_SPAM		
		FROM balances b
		JOIN tokens t ON t.CONTRACT_ADDRESS = b.TOKEN_ADDRESS
		WHERE b.OWNER_ADDRESS = $1 AND t.CHAIN = $2 AND t.DELETED = false;`)
	checkNoErr(err)

	getByOwnerPaginateStmt, err := db.PrepareContext(ctx, `SELECT b.ID,b.VERSION,b.CREATED_AT,b.LAST_UPDATED,b.OWNER_ADDRESS,b.BALANCE,b.BLOCK_NUMBER,
		t.ID,t.TOKEN_TYPE,t.CHAIN,t.NAME,t.SYMBOL,t.LOGO,t.DECIMALS,t.TOTAL_SUPPLY,t.CONTRACT_ADDRESS,t.BLOCK_NUMBER,t.VERSION,t.CREATED_AT,t.LAST_UPDATED,t.IS_SPAM		
		FROM balances b
		JOIN tokens t ON t.CONTRACT_ADDRESS = b.TOKEN_ADDRESS
		WHERE b.OWNER_ADDRESS = $1 AND t.CHAIN = $2 AND t.DELETED = false;
		ORDER BY BALANCE DESC LIMIT $2 OFFSET $3`)
	checkNoErr(err)

	getByTokenStmt, err := db.PrepareContext(ctx, `SELECT b.ID,b.VERSION,b.CREATED_AT,b.LAST_UPDATED,b.OWNER_ADDRESS,b.BALANCE,b.BLOCK_NUMBER,
		t.ID,t.TOKEN_TYPE,t.CHAIN,t.NAME,t.SYMBOL,t.LOGO,t.DECIMALS,t.TOTAL_SUPPLY,t.CONTRACT_ADDRESS,t.BLOCK_NUMBER,t.VERSION,t.CREATED_AT,t.LAST_UPDATED,t.IS_SPAM		
		FROM balances b
		JOIN tokens t ON t.CONTRACT_ADDRESS = b.TOKEN_ADDRESS
		WHERE t.CONTRACT_ADDRESS = $1 AND t.CHAIN = $2 AND t.DELETED = false;`)
	checkNoErr(err)

	getByTokenPaginateStmt, err := db.PrepareContext(ctx, `SELECT b.ID,b.VERSION,b.CREATED_AT,b.LAST_UPDATED,b.OWNER_ADDRESS,b.BALANCE,b.BLOCK_NUMBER,
		t.ID,t.TOKEN_TYPE,t.CHAIN,t.NAME,t.SYMBOL,t.LOGO,t.DECIMALS,t.TOTAL_SUPPLY,t.CONTRACT_ADDRESS,t.BLOCK_NUMBER,t.VERSION,t.CREATED_AT,t.LAST_UPDATED,t.IS_SPAM		
		FROM balances b
		JOIN tokens t ON t.CONTRACT_ADDRESS = b.TOKEN_ADDRESS
		WHERE t.CONTRACT_ADDRESS = $1 AND t.CHAIN = $2 AND t.DELETED = false;
		ORDER BY BALANCE DESC LIMIT $3 OFFSET $4`)
	checkNoErr(err)

	getByIdentifiersStmt, err := db.PrepareContext(ctx, `SELECT b.ID,b.VERSION,b.CREATED_AT,b.LAST_UPDATED,b.OWNER_ADDRESS,b.BALANCE,b.BLOCK_NUMBER,
		t.ID,t.TOKEN_TYPE,t.CHAIN,t.NAME,t.SYMBOL,t.LOGO,t.DECIMALS,t.TOTAL_SUPPLY,t.CONTRACT_ADDRESS,t.BLOCK_NUMBER,t.VERSION,t.CREATED_AT,t.LAST_UPDATED,t.IS_SPAM		
		FROM balances b
		JOIN tokens t ON t.CONTRACT_ADDRESS = b.TOKEN_ADDRESS
		WHERE b.OWNER_ADDRESS = $1 AND t.CONTRACT_ADDRESS = $2 AND t.CHAIN = $3 AND t.DELETED = false;`)
	checkNoErr(err)

	upsertByIdentifiersStmt, err := db.PrepareContext(ctx, `INSERT INTO balances (ID,VERSION,OWNER_ADDRESS,TOKEN_ADDRESS,BALANCE,BLOCK_NUMBER,CREATED_AT,LAST_UPDATED) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (OWNER_ADDRESS,TOKEN_ADDRESS) DO UPDATE SET VERSION = EXCLUDED.VERSION, OWNER_ADDRESS = EXCLUDED.OWNER_ADDRESS, TOKEN_ADDRESS = EXCLUDED.TOKEN_ADDRESS, BALANCE = EXCLUDED.BALANCE, BLOCK_NUMBER = EXCLUDED.BLOCK_NUMBER, CREATED_AT = EXCLUDED.CREATED_AT,LAST_UPDATED = EXCLUDED.LAST_UPDATED;`)
	checkNoErr(err)

	updateBalanceUnsafeStmt, err := db.PrepareContext(ctx, `UPDATE balances SET BALANCE = $1, BLOCK_NUMBER = $2, LAST_UPDATED = now() WHERE ID = $3;`)
	checkNoErr(err)

	updateBalanceByIdentifiersUnsafeStmt, err := db.PrepareContext(ctx, `UPDATE balances SET BALANCE = $1, BLOCK_NUMBER = $2, LAST_UPDATED = now() WHERE OWNER_ADDRESS = $3 AND TOKEN_ADDRESS = $4;`)
	checkNoErr(err)

	return &TokenBalanceRepository{db: db, queries: queries, getByOwnerStmt: getByOwnerStmt, getByOwnerPaginateStmt: getByOwnerPaginateStmt, getByTokenStmt: getByTokenStmt, getByTokenPaginateStmt: getByTokenPaginateStmt, getByIdentifiersStmt: getByIdentifiersStmt, upsertByIdentifiersStmt: upsertByIdentifiersStmt, updateBalanceUnsafeStmt: updateBalanceUnsafeStmt, updateBalanceByIdentifiersUnsafeStmt: updateBalanceByIdentifiersUnsafeStmt}
}

// GetByOwner retrieves all balances associated with an owner ethereum address
func (b *TokenBalanceRepository) GetByOwner(pCtx context.Context, pAddress persist.EthereumAddress, pChain persist.Chain, limit int64, offset int64) ([]persist.TokenBalance, error) {
	var rows *sql.Rows
	var err error
	if limit > 0 {
		rows, err = b.getByOwnerPaginateStmt.QueryContext(pCtx, pAddress, pChain, limit, offset)
	} else {
		rows, err = b.getByOwnerStmt.QueryContext(pCtx, pAddress, pChain)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	balances := make([]persist.TokenBalance, 0, 10)
	for rows.Next() {
		balance := persist.TokenBalance{}
		if err := rows.Scan(&balance.ID, &balance.Version, &balance.CreationTime, &balance.LastUpdated, &balance.OwnerAddress, &balance.Balance, &balance.BlockNumber, &balance.Token); err != nil {
			return nil, err
		}
		balances = append(balances, balance)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return balances, nil

}

// GetByToken retrieves all balances associated with a token ethereum address
func (b *TokenBalanceRepository) GetByToken(pCtx context.Context, pAddress persist.EthereumAddress, pChain persist.Chain, limit int64, offset int64) ([]persist.TokenBalance, error) {
	var rows *sql.Rows
	var err error
	if limit > 0 {
		rows, err = b.getByTokenPaginateStmt.QueryContext(pCtx, pAddress, pChain, limit, offset)
	} else {
		rows, err = b.getByTokenStmt.QueryContext(pCtx, pAddress, pChain)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	balances := make([]persist.TokenBalance, 0, 10)
	for rows.Next() {
		balance := persist.TokenBalance{}
		if err := rows.Scan(&balance.ID, &balance.Version, &balance.CreationTime, &balance.LastUpdated, &balance.OwnerAddress, &balance.Balance, &balance.BlockNumber, &balance.Token); err != nil {
			return nil, err
		}
		balances = append(balances, balance)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return balances, nil

}

// GetByIdentifiers gets a token by its owner address, token address and chain
func (b *TokenBalanceRepository) GetByIdentifiers(pCtx context.Context, pOwnerAddress, pTokenAddress persist.EthereumAddress, pChain persist.Chain) (persist.TokenBalance, error) {
	var balance persist.TokenBalance
	err := b.getByIdentifiersStmt.QueryRowContext(pCtx, pOwnerAddress, pTokenAddress, pChain).Scan(&balance.ID, &balance.Version, &balance.CreationTime, &balance.LastUpdated, &balance.OwnerAddress, &balance.Balance, &balance.BlockNumber, &balance.Token)
	if err != nil {
		if err == sql.ErrNoRows {
			return balance, persist.ErrTokenBalanceNotFoundByIdentifiers{OwnerAddress: pOwnerAddress, TokenAddress: pTokenAddress, Chain: pChain}
		}
		return persist.TokenBalance{}, err
	}
	return balance, nil
}

// UpsertByAddress upserts the balance with the given owner address and token address
func (b *TokenBalanceRepository) UpsertByIdentifiers(pCtx context.Context, pOwnerAddress, pTokenAddress persist.EthereumAddress, pTokenBalance persist.TokenBalance) error {
	_, err := b.upsertByIdentifiersStmt.ExecContext(pCtx, persist.GenerateID(), pTokenBalance.Version, pOwnerAddress, pTokenAddress, pTokenBalance.Balance, pTokenBalance.BlockNumber, pTokenBalance.CreationTime, pTokenBalance.LastUpdated)
	if err != nil {
		return err
	}

	return nil
}

// UpdateByID updates a token balance by its ID
func (b *TokenBalanceRepository) UpdateByID(pCtx context.Context, pID persist.DBID, pUpdate interface{}) error {

	var res sql.Result
	var err error
	switch pUpdate.(type) {
	case persist.TokenBalanceUpdateInput:
		update := pUpdate.(persist.TokenBalanceUpdateInput)
		res, err = b.updateBalanceUnsafeStmt.ExecContext(pCtx, update.Balance, update.BlockNumber, persist.LastUpdatedTime{}, pID)
	default:
		return fmt.Errorf("unsupported update type: %T", pUpdate)
	}
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return persist.ErrTokenBalanceNotFoundByID{ID: pID}
	}
	return nil
}

// UpdateByIdentifiers updates a token balance by its owner address, token address and chain
func (b *TokenBalanceRepository) UpdateByIdentifiers(pCtx context.Context, pOwnerAddress, pTokenAddress persist.EthereumAddress, pChain persist.Chain, pUpdate interface{}) error {

	var res sql.Result
	var err error
	switch pUpdate.(type) {
	case persist.TokenBalanceUpdateInput:
		update := pUpdate.(persist.TokenBalanceUpdateInput)
		res, err = b.updateBalanceByIdentifiersUnsafeStmt.ExecContext(pCtx, update.Balance, update.BlockNumber, persist.LastUpdatedTime{}, pOwnerAddress, pTokenAddress)
	default:
		return fmt.Errorf("unsupported update type: %T", pUpdate)
	}
	if err != nil {
		return err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return persist.ErrTokenBalanceNotFoundByIdentifiers{OwnerAddress: pOwnerAddress, TokenAddress: pTokenAddress, Chain: pChain}
	}
	return nil
}

// BulkUpsert bulk upserts the balances by address
/*func (b *TokenBalanceRepository) BulkUpsert(pCtx context.Context, pBalances []persist.TokenBalance) error {
	if len(pBalances) == 0 {
		return nil
	}
	pBalances = removeDuplicateTokensBalance(pBalances)
	sqlStr := `INSERT INTO balances (ID,VERSION,ADDRESS,SYMBOL,NAME,CREATOR_ADDRESS,CHAIN) VALUES `
	vals := make([]interface{}, 0, len(pBalances)*7)
	for i, balance := range pBalances {
		sqlStr += generateValuesPlaceholders(7, i*7, nil)
		vals = append(vals, persist.GenerateID(), balance.Version, balance.Address, balance.Symbol, balance.Name, balance.CreatorAddress, balance.Chain)
		sqlStr += ","
	}
	sqlStr = sqlStr[:len(sqlStr)-1]
	sqlStr += ` ON CONFLICT (ADDRESS, CHAIN) DO UPDATE SET SYMBOL = EXCLUDED.SYMBOL,NAME = EXCLUDED.NAME,CREATOR_ADDRESS = EXCLUDED.CREATOR_ADDRESS,CHAIN = EXCLUDED.CHAIN;`
	_, err := b.db.ExecContext(pCtx, sqlStr, vals...)
	if err != nil {
		return fmt.Errorf("error bulk upserting balances: %v - SQL: %s -- VALS: %+v", err, sqlStr, vals)
	}

	return nil
}

func removeDuplicateTokensBalance(pBalances []persist.TokenBalance) []persist.TokenBalance {
	if len(pBalances) == 0 {
		return pBalances
	}
	unique := map[persist.TokenBalanceIdentifiers]bool{}
	result := make([]persist.TokenBalance, 0, len(pBalances))
	for _, b := range pBalances {
		key := persist.NewTokenBalanceIdentifiers(b.Token.ContractAddress, b.OwnerAddress)

		if unique[key] {
			continue
		}
		result = append(result, b)
		unique[key] = true
	}
	return result
}
*/
