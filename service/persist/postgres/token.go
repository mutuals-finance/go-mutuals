package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/SplitFi/go-splitfi/service/logger"

	"github.com/SplitFi/go-splitfi/service/persist"
)

// TokenRepository represents a postgres repository for tokens
type TokenRepository struct {
	db                                *sql.DB
	getByWalletStmt                   *sql.Stmt
	getByWalletPaginateStmt           *sql.Stmt
	getByTokenIdentifiersStmt         *sql.Stmt
	getByTokenIdentifiersPaginateStmt *sql.Stmt
	getByIdentifiersStmt              *sql.Stmt
	getExistsByTokenIdentifiersStmt   *sql.Stmt
	mostRecentBlockStmt               *sql.Stmt
	upsertStmt                        *sql.Stmt
	deleteStmt                        *sql.Stmt
	deleteByIDStmt                    *sql.Stmt
}

// NewTokenRepository creates a new TokenRepository
func NewTokenRepository(db *sql.DB) *TokenRepository {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// TODO getByWalletStmt should return tokens of a wallet
	getByWalletStmt, err := db.PrepareContext(ctx, `SELECT ID,TOKEN_TYPE,CHAIN,NAME,SYMBOL,LOGO,CONTRACT_ADDRESS,BLOCK_NUMBER,VERSION,CREATED_AT,LAST_UPDATED FROM tokens WHERE CONTRACT_ADDRESS = $1 ORDER BY BLOCK_NUMBER DESC;`)
	checkNoErr(err)

	// TODO getByWalletPaginateStmt should return tokens of a wallet
	getByWalletPaginateStmt, err := db.PrepareContext(ctx, `SELECT ID,TOKEN_TYPE,CHAIN,NAME,SYMBOL,LOGO,CONTRACT_ADDRESS,BLOCK_NUMBER,VERSION,CREATED_AT,LAST_UPDATED FROM tokens WHERE CONTRACT_ADDRESS = $1 ORDER BY BLOCK_NUMBER DESC LIMIT $2 OFFSET $3;`)
	checkNoErr(err)

	getByTokenIdentifiersStmt, err := db.PrepareContext(ctx, `SELECT ID,TOKEN_TYPE,CHAIN,NAME,SYMBOL,LOGO,CONTRACT_ADDRESS,BLOCK_NUMBER,VERSION,CREATED_AT,LAST_UPDATED FROM tokens WHERE CONTRACT_ADDRESS = $1 ORDER BY BLOCK_NUMBER DESC;`)
	checkNoErr(err)

	getByTokenIdentifiersPaginateStmt, err := db.PrepareContext(ctx, `SELECT ID,TOKEN_TYPE,CHAIN,NAME,SYMBOL,LOGO,CONTRACT_ADDRESS,BLOCK_NUMBER,VERSION,CREATED_AT,LAST_UPDATED FROM tokens WHERE CONTRACT_ADDRESS = $1 ORDER BY BLOCK_NUMBER DESC LIMIT $3 OFFSET $4;`)
	checkNoErr(err)

	getByIdentifiersStmt, err := db.PrepareContext(ctx, `SELECT ID,TOKEN_TYPE,CHAIN,NAME,SYMBOL,LOGO,CONTRACT_ADDRESS,BLOCK_NUMBER,VERSION,CREATED_AT,LAST_UPDATED FROM tokens WHERE CONTRACT_ADDRESS = $1;`)
	checkNoErr(err)

	getExistsByTokenIdentifiersStmt, err := db.PrepareContext(ctx, `SELECT EXISTS(SELECT 1 FROM tokens WHERE CONTRACT_ADDRESS = $1);`)
	checkNoErr(err)

	mostRecentBlockStmt, err := db.PrepareContext(ctx, `SELECT MAX(BLOCK_NUMBER) FROM tokens;`)
	checkNoErr(err)

	upsertStmt, err := db.PrepareContext(ctx, `INSERT INTO tokens (ID,TOKEN_TYPE,CHAIN,NAME,SYMBOL,LOGO,CONTRACT_ADDRESS,BLOCK_NUMBER,VERSION,CREATED_AT,LAST_UPDATED) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) ON CONFLICT (CONTRACT_ADDRESS) DO UPDATE SET TOKEN_TYPE = EXCLUDED.TOKEN_TYPE,CHAIN = EXCLUDED.CHAIN,NAME = EXCLUDED.NAME,SYMBOL = EXCLUDED.SYMBOL,LOGO = EXCLUDED.LOGO,CONTRACT_ADDRESS = EXCLUDED.CONTRACT_ADDRESS,BLOCK_NUMBER = EXCLUDED.BLOCK_NUMBER,VERSION = EXCLUDED.VERSION,CREATED_AT = EXCLUDED.CREATED_AT,LAST_UPDATED = EXCLUDED.LAST_UPDATED;`)
	checkNoErr(err)

	deleteStmt, err := db.PrepareContext(ctx, `DELETE FROM tokens WHERE CONTRACT_ADDRESS = $1;`)
	checkNoErr(err)

	deleteByIDStmt, err := db.PrepareContext(ctx, `DELETE FROM tokens WHERE ID = $1;`)
	checkNoErr(err)

	return &TokenRepository{
		db:                                db,
		getByWalletStmt:                   getByWalletStmt,
		getByWalletPaginateStmt:           getByWalletPaginateStmt,
		getByTokenIdentifiersStmt:         getByTokenIdentifiersStmt,
		getByTokenIdentifiersPaginateStmt: getByTokenIdentifiersPaginateStmt,
		getByIdentifiersStmt:              getByIdentifiersStmt,
		getExistsByTokenIdentifiersStmt:   getExistsByTokenIdentifiersStmt,
		mostRecentBlockStmt:               mostRecentBlockStmt,
		upsertStmt:                        upsertStmt,
		deleteStmt:                        deleteStmt,
		deleteByIDStmt:                    deleteByIDStmt,
	}

}

// GetByWallet retrieves all tokens associated with a wallet
func (t *TokenRepository) GetByWallet(pCtx context.Context, pAddress persist.EthereumAddress, limit int64, offset int64) ([]persist.Token, error) {
	var rows *sql.Rows
	var err error
	if limit > 0 {
		rows, err = t.getByWalletPaginateStmt.QueryContext(pCtx, pAddress, limit, offset)
	} else {
		rows, err = t.getByWalletStmt.QueryContext(pCtx, pAddress)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokens := make([]persist.Token, 0, 10)
	for rows.Next() {
		token := persist.Token{}
		if err := rows.Scan(&token.ID, &token.TokenType, &token.Chain, &token.Name, &token.Symbol, &token.Logo, &token.ContractAddress, &token.BlockNumber, &token.Version, &token.CreationTime, &token.LastUpdated); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tokens, nil

}

// GetByTokenIdentifiers gets a token by its token ID and contract address
func (t *TokenRepository) GetByTokenIdentifiers(pCtx context.Context, pContractAddress persist.EthereumAddress, limit int64, offset int64) ([]persist.Token, error) {
	var rows *sql.Rows
	var err error
	if limit > 0 {
		rows, err = t.getByTokenIdentifiersPaginateStmt.QueryContext(pCtx, pContractAddress, limit, offset)
	} else {
		rows, err = t.getByTokenIdentifiersStmt.QueryContext(pCtx, pContractAddress)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokens := make([]persist.Token, 0, 10)
	for rows.Next() {
		token := persist.Token{}
		if err := rows.Scan(&token.ID, &token.TokenType, &token.Chain, &token.Name, &token.Symbol, &token.Logo, &token.ContractAddress, &token.BlockNumber, &token.Version, &token.CreationTime, &token.LastUpdated); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(tokens) == 0 {
		return nil, persist.ErrTokenNotFoundByTokenIdentifiers{ContractAddress: pContractAddress}
	}

	return tokens, nil
}

// GetByIdentifiers gets a token by its contract address
func (t *TokenRepository) GetByIdentifiers(pCtx context.Context, pContractAddress persist.EthereumAddress) (persist.Token, error) {
	var token persist.Token
	err := t.getByIdentifiersStmt.QueryRowContext(pCtx, pContractAddress).Scan(&token.ID, &token.TokenType, &token.Chain, &token.Name, &token.Symbol, &token.Logo, &token.ContractAddress, &token.BlockNumber, &token.Version, &token.CreationTime, &token.LastUpdated)
	if err != nil {
		if err == sql.ErrNoRows {
			return token, persist.ErrTokenNotFoundByIdentifiers{ContractAddress: pContractAddress}
		}
		return persist.Token{}, err
	}
	return token, nil
}

// TokenExistsByTokenIdentifiers gets a token by its token ID and contract address and owner address
func (t *TokenRepository) TokenExistsByTokenIdentifiers(pCtx context.Context, pContractAddress persist.EthereumAddress) (bool, error) {
	var exists bool
	err := t.getExistsByTokenIdentifiersStmt.QueryRowContext(pCtx, pContractAddress).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func (t *TokenRepository) BulkUpsert(pCtx context.Context, pTokens []persist.Token) error {
	if len(pTokens) == 0 {
		return nil
	}
	// Postgres only allows 65535 parameters at a time.
	// TODO: Consider trying this implementation at some point instead of chunking:
	//       https://klotzandrew.com/blog/postgres-passing-65535-parameter-limit
	paramsPerRow := 20
	rowsPerQuery := 65535 / paramsPerRow

	if len(pTokens) > rowsPerQuery {
		logger.For(pCtx).Debugf("Chunking %d tokens recursively into %d queries", len(pTokens), len(pTokens)/rowsPerQuery)
		next := pTokens[rowsPerQuery:]
		current := pTokens[:rowsPerQuery]
		if err := t.BulkUpsert(pCtx, next); err != nil {
			return fmt.Errorf("error with tokens upsert: %w", err)
		}
		pTokens = current
	}

	sqlStr := `INSERT INTO tokens (ID,TOKEN_TYPE,CHAIN,NAME,SYMBOL,LOGO,CONTRACT_ADDRESS,BLOCK_NUMBER,VERSION,CREATED_AT,LAST_UPDATED,DELETED) VALUES `
	vals := make([]interface{}, 0, len(pTokens)*paramsPerRow)
	for i, token := range pTokens {
		sqlStr += generateValuesPlaceholders(paramsPerRow, i*paramsPerRow, nil) + ","
		vals = append(vals, persist.GenerateID(), token.TokenType, token.Chain, token.Name, token.Symbol, token.Logo, token.ContractAddress, token.BlockNumber, token.Version, token.CreationTime, token.LastUpdated, token.Deleted)
	}

	sqlStr = sqlStr[:len(sqlStr)-1]
	sqlStr += ` ON CONFLICT (CONTRACT_ADDRESS) WHERE TOKEN_TYPE = 'ERC-20' DO UPDATE SET TOKEN_TYPE = EXCLUDED.TOKEN_TYPE,CHAIN = EXCLUDED.CHAIN,NAME = EXCLUDED.NAME,SYMBOL = EXCLUDED.SYMBOL,LOGO = EXCLUDED.LOGO,CONTRACT_ADDRESS = EXCLUDED.CONTRACT_ADDRESS,BLOCK_NUMBER = EXCLUDED.BLOCK_NUMBER,VERSION = EXCLUDED.VERSION,CREATED_AT = EXCLUDED.CREATED_AT,LAST_UPDATED = EXCLUDED.LAST_UPDATED,DELETED = EXCLUDED.DELETED WHERE EXCLUDED.BLOCK_NUMBER > tokens.BLOCK_NUMBER;`

	_, err := t.db.ExecContext(pCtx, sqlStr, vals...)
	if err != nil {
		logger.For(pCtx).Errorf("SQL: %s", sqlStr)
		return fmt.Errorf("failed to upsert erc20 tokens: %w", err)
	}
	return nil
}

// Upsert adds a token by its token ID and contract address and if its token type is ERC-1155 it also adds using the owner address
func (t *TokenRepository) Upsert(pCtx context.Context, pToken persist.Token) error {
	var err error
	_, err = t.upsertStmt.ExecContext(pCtx, persist.GenerateID(), pToken.TokenType, pToken.Chain, pToken.Name, pToken.Symbol, pToken.Logo, pToken.ContractAddress, pToken.BlockNumber, pToken.Version, pToken.CreationTime, pToken.LastUpdated)
	return err
}

// UpdateByID updates a token by its ID
func (t *TokenRepository) UpdateByID(pCtx context.Context, pID persist.DBID, pUpdate interface{}) error {

	var res sql.Result
	var err error
	switch pUpdate.(type) {
	//case persist.TokenUpdateTotalSupplyInput:
	//	update := pUpdate.(persist.TokenUpdateTotalSupplyInput)
	//	res, err = t.updateTotalSupplyUnsafeStmt.ExecContext(pCtx, update.TotalSupply, update.BlockNumber, persist.LastUpdatedTime{}, pID)
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
		return persist.ErrTokenNotFoundByID{ID: pID}
	}
	return nil
}

func (t *TokenRepository) MostRecentBlock(pCtx context.Context) (persist.BlockNumber, error) {
	var blockNumber persist.BlockNumber
	err := t.mostRecentBlockStmt.QueryRowContext(pCtx).Scan(&blockNumber)
	if err != nil {
		return 0, err
	}
	return blockNumber, nil
}

func (t *TokenRepository) DeleteByID(pCtx context.Context, pID persist.DBID) error {
	_, err := t.deleteByIDStmt.ExecContext(pCtx, pID)
	return err
}

func (t *TokenRepository) deleteTokenUnsafe(pCtx context.Context, pContractAddress persist.EthereumAddress) error {
	_, err := t.deleteStmt.ExecContext(pCtx, pContractAddress)
	return err
}
