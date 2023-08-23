package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/SplitFi/go-splitfi/service/persist"
)

// AssetRepository represents an asset repository in the postgres database
type AssetRepository struct {
	db                                 *sql.DB
	getByOwnerStmt                     *sql.Stmt
	getByOwnerPaginateStmt             *sql.Stmt
	getByTokenStmt                     *sql.Stmt
	getByTokenPaginateStmt             *sql.Stmt
	getByIdentifiersStmt               *sql.Stmt
	upsertByIdentifiersStmt            *sql.Stmt
	updateAssetUnsafeStmt              *sql.Stmt
	updateAssetByIdentifiersUnsafeStmt *sql.Stmt
}

// NewAssetRepository creates a new postgres repository for interacting with assets
func NewAssetRepository(db *sql.DB) *AssetRepository {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	getByOwnerStmt, err := db.PrepareContext(ctx, `SELECT a.ID,a.VERSION,a.CREATED_AT,a.LAST_UPDATED,a.OWNER_ADDRESS,a.BALANCE,a.BLOCK_NUMBER,
		t.ID,t.TOKEN_TYPE,t.CHAIN,t.NAME,t.SYMBOL,t.LOGO,t.DECIMALS,t.TOTAL_SUPPLY,t.CONTRACT_ADDRESS,t.BLOCK_NUMBER,t.VERSION,t.CREATED_AT,t.LAST_UPDATED,t.IS_SPAM		
		FROM assets a
		JOIN tokens t ON t.CONTRACT_ADDRESS = a.TOKEN_ADDRESS
		WHERE a.OWNER_ADDRESS = $1 AND t.CHAIN = $2 AND t.DELETED = false;`)
	checkNoErr(err)

	getByOwnerPaginateStmt, err := db.PrepareContext(ctx, `SELECT a.ID,a.VERSION,a.CREATED_AT,a.LAST_UPDATED,a.OWNER_ADDRESS,a.BALANCE,a.BLOCK_NUMBER,
		t.ID,t.TOKEN_TYPE,t.CHAIN,t.NAME,t.SYMBOL,t.LOGO,t.DECIMALS,t.TOTAL_SUPPLY,t.CONTRACT_ADDRESS,t.BLOCK_NUMBER,t.VERSION,t.CREATED_AT,t.LAST_UPDATED,t.IS_SPAM		
		FROM assets a
		JOIN tokens t ON t.CONTRACT_ADDRESS = a.TOKEN_ADDRESS
		WHERE a.OWNER_ADDRESS = $1 AND t.CHAIN = $2 AND t.DELETED = false;
		ORDER BY a.BALANCE DESC LIMIT $2 OFFSET $3`)
	checkNoErr(err)

	getByTokenStmt, err := db.PrepareContext(ctx, `SELECT a.ID,a.VERSION,a.CREATED_AT,a.LAST_UPDATED,a.OWNER_ADDRESS,a.BALANCE,a.BLOCK_NUMBER,
		t.ID,t.TOKEN_TYPE,t.CHAIN,t.NAME,t.SYMBOL,t.LOGO,t.DECIMALS,t.TOTAL_SUPPLY,t.CONTRACT_ADDRESS,t.BLOCK_NUMBER,t.VERSION,t.CREATED_AT,t.LAST_UPDATED,t.IS_SPAM		
		FROM assets a
		JOIN tokens t ON t.CONTRACT_ADDRESS = a.TOKEN_ADDRESS
		WHERE t.CONTRACT_ADDRESS = $1 AND t.CHAIN = $2 AND t.DELETED = false;`)
	checkNoErr(err)

	getByTokenPaginateStmt, err := db.PrepareContext(ctx, `SELECT a.ID,a.VERSION,a.CREATED_AT,a.LAST_UPDATED,a.OWNER_ADDRESS,a.BALANCE,a.BLOCK_NUMBER,
		t.ID,t.TOKEN_TYPE,t.CHAIN,t.NAME,t.SYMBOL,t.LOGO,t.DECIMALS,t.TOTAL_SUPPLY,t.CONTRACT_ADDRESS,t.BLOCK_NUMBER,t.VERSION,t.CREATED_AT,t.LAST_UPDATED,t.IS_SPAM		
		FROM assets a
		JOIN tokens t ON t.CONTRACT_ADDRESS = a.TOKEN_ADDRESS
		WHERE t.CONTRACT_ADDRESS = $1 AND t.CHAIN = $2 AND t.DELETED = false;
		ORDER BY a.BALANCE DESC LIMIT $3 OFFSET $4`)
	checkNoErr(err)

	getByIdentifiersStmt, err := db.PrepareContext(ctx, `SELECT a.ID,a.VERSION,a.CREATED_AT,a.LAST_UPDATED,a.OWNER_ADDRESS,a.BALANCE,a.BLOCK_NUMBER,
		t.ID,t.TOKEN_TYPE,t.CHAIN,t.NAME,t.SYMBOL,t.LOGO,t.DECIMALS,t.TOTAL_SUPPLY,t.CONTRACT_ADDRESS,t.BLOCK_NUMBER,t.VERSION,t.CREATED_AT,t.LAST_UPDATED,t.IS_SPAM		
		FROM assets a
		JOIN tokens t ON t.CONTRACT_ADDRESS = a.TOKEN_ADDRESS
		WHERE b.OWNER_ADDRESS = $1 AND t.CONTRACT_ADDRESS = $2 AND t.CHAIN = $3 AND t.DELETED = false;`)
	checkNoErr(err)

	upsertByIdentifiersStmt, err := db.PrepareContext(ctx, `INSERT INTO assets (ID,VERSION,OWNER_ADDRESS,TOKEN_ADDRESS,BALANCE,BLOCK_NUMBER,CREATED_AT,LAST_UPDATED) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (OWNER_ADDRESS,TOKEN_ADDRESS) DO UPDATE SET VERSION = EXCLUDED.VERSION, OWNER_ADDRESS = EXCLUDED.OWNER_ADDRESS, TOKEN_ADDRESS = EXCLUDED.TOKEN_ADDRESS, BALANCE = EXCLUDED.BALANCE, BLOCK_NUMBER = EXCLUDED.BLOCK_NUMBER, CREATED_AT = EXCLUDED.CREATED_AT,LAST_UPDATED = EXCLUDED.LAST_UPDATED;`)
	checkNoErr(err)

	updateAssetUnsafeStmt, err := db.PrepareContext(ctx, `UPDATE assets SET BALANCE = $1, BLOCK_NUMBER = $2, LAST_UPDATED = now() WHERE ID = $3;`)
	checkNoErr(err)

	updateAssetByIdentifiersUnsafeStmt, err := db.PrepareContext(ctx, `UPDATE assets SET BALANCE = $1, BLOCK_NUMBER = $2, LAST_UPDATED = now() WHERE OWNER_ADDRESS = $3 AND TOKEN_ADDRESS = $4;`)
	checkNoErr(err)

	return &AssetRepository{db: db, getByOwnerStmt: getByOwnerStmt, getByOwnerPaginateStmt: getByOwnerPaginateStmt, getByTokenStmt: getByTokenStmt, getByTokenPaginateStmt: getByTokenPaginateStmt, getByIdentifiersStmt: getByIdentifiersStmt, upsertByIdentifiersStmt: upsertByIdentifiersStmt, updateAssetUnsafeStmt: updateAssetUnsafeStmt, updateAssetByIdentifiersUnsafeStmt: updateAssetByIdentifiersUnsafeStmt}
}

// GetByOwner retrieves all assets associated with an owner ethereum address
func (a *AssetRepository) GetByOwner(pCtx context.Context, pAddress persist.EthereumAddress, pChain persist.Chain, limit int64, offset int64) ([]persist.Asset, error) {
	var rows *sql.Rows
	var err error
	if limit > 0 {
		rows, err = a.getByOwnerPaginateStmt.QueryContext(pCtx, pAddress, pChain, limit, offset)
	} else {
		rows, err = a.getByOwnerStmt.QueryContext(pCtx, pAddress, pChain)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	assets := make([]persist.Asset, 0, 10)
	for rows.Next() {
		asset := persist.Asset{}
		if err := rows.Scan(&asset.ID, &asset.Version, &asset.CreationTime, &asset.LastUpdated, &asset.OwnerAddress, &asset.Balance, &asset.BlockNumber, &asset.Token); err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return assets, nil

}

// GetByToken retrieves all assets associated with a token ethereum address
func (a *AssetRepository) GetByToken(pCtx context.Context, pAddress persist.EthereumAddress, pChain persist.Chain, limit int64, offset int64) ([]persist.Asset, error) {
	var rows *sql.Rows
	var err error
	if limit > 0 {
		rows, err = a.getByTokenPaginateStmt.QueryContext(pCtx, pAddress, pChain, limit, offset)
	} else {
		rows, err = a.getByTokenStmt.QueryContext(pCtx, pAddress, pChain)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	assets := make([]persist.Asset, 0, 10)
	for rows.Next() {
		asset := persist.Asset{}
		if err := rows.Scan(&asset.ID, &asset.Version, &asset.CreationTime, &asset.LastUpdated, &asset.OwnerAddress, &asset.Balance, &asset.BlockNumber, &asset.Token); err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return assets, nil

}

// GetByIdentifiers gets a token by its owner address, token address and chain
func (a *AssetRepository) GetByIdentifiers(pCtx context.Context, pOwnerAddress, pTokenAddress persist.EthereumAddress, pChain persist.Chain) (persist.Asset, error) {
	var asset persist.Asset
	err := a.getByIdentifiersStmt.QueryRowContext(pCtx, pOwnerAddress, pTokenAddress, pChain).Scan(&asset.ID, &asset.Version, &asset.CreationTime, &asset.LastUpdated, &asset.OwnerAddress, &asset.Balance, &asset.BlockNumber, &asset.Token)
	if err != nil {
		if err == sql.ErrNoRows {
			return asset, persist.ErrAssetNotFoundByIdentifiers{OwnerAddress: pOwnerAddress, TokenAddress: pTokenAddress, Chain: pChain}
		}
		return persist.Asset{}, err
	}
	return asset, nil
}

// UpsertByAddress upserts the asset with the given owner address and token address
func (a *AssetRepository) UpsertByIdentifiers(pCtx context.Context, pOwnerAddress, pTokenAddress persist.EthereumAddress, pAsset persist.Asset) error {
	_, err := a.upsertByIdentifiersStmt.ExecContext(pCtx, persist.GenerateID(), pAsset.Version, pOwnerAddress, pTokenAddress, pAsset.Balance, pAsset.BlockNumber, pAsset.CreationTime, pAsset.LastUpdated)
	if err != nil {
		return err
	}

	return nil
}

// UpdateByID updates an asset by its ID
func (a *AssetRepository) UpdateByID(pCtx context.Context, pID persist.DBID, pUpdate interface{}) error {

	var res sql.Result
	var err error
	switch pUpdate.(type) {
	case persist.AssetUpdateInput:
		update := pUpdate.(persist.AssetUpdateInput)
		res, err = a.updateAssetUnsafeStmt.ExecContext(pCtx, update.Asset, update.BlockNumber, persist.LastUpdatedTime{}, pID)
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
		return persist.ErrAssetNotFoundByID{ID: pID}
	}
	return nil
}

// UpdateByIdentifiers updates an asset by its owner address, token address and chain
func (a *AssetRepository) UpdateByIdentifiers(pCtx context.Context, pOwnerAddress, pTokenAddress persist.EthereumAddress, pChain persist.Chain, pUpdate interface{}) error {

	var res sql.Result
	var err error
	switch pUpdate.(type) {
	case persist.AssetUpdateInput:
		update := pUpdate.(persist.AssetUpdateInput)
		res, err = a.updateAssetByIdentifiersUnsafeStmt.ExecContext(pCtx, update.Asset, update.BlockNumber, persist.LastUpdatedTime{}, pOwnerAddress, pTokenAddress)
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
		return persist.ErrAssetNotFoundByIdentifiers{OwnerAddress: pOwnerAddress, TokenAddress: pTokenAddress, Chain: pChain}
	}
	return nil
}

// BulkUpsert bulk upserts the assets by address
/*func (b *AssetRepository) BulkUpsert(pCtx context.Context, pAssets []persist.Asset) error {
	if len(pAssets) == 0 {
		return nil
	}
	pAssets = removeDuplicateAsset(pAssets)
	sqlStr := `INSERT INTO assets (ID,VERSION,ADDRESS,SYMBOL,NAME,CREATOR_ADDRESS,CHAIN) VALUES `
	vals := make([]interface{}, 0, len(pAssets)*7)
	for i, asset := range pAssets {
		sqlStr += generateValuesPlaceholders(7, i*7, nil)
		vals = append(vals, persist.GenerateID(), asset.Version, asset.Address, asset.Symbol, asset.Name, asset.CreatorAddress, asset.Chain)
		sqlStr += ","
	}
	sqlStr = sqlStr[:len(sqlStr)-1]
	sqlStr += ` ON CONFLICT (ADDRESS, CHAIN) DO UPDATE SET SYMBOL = EXCLUDED.SYMBOL,NAME = EXCLUDED.NAME,CREATOR_ADDRESS = EXCLUDED.CREATOR_ADDRESS,CHAIN = EXCLUDED.CHAIN;`
	_, err := b.db.ExecContext(pCtx, sqlStr, vals...)
	if err != nil {
		return fmt.Errorf("error bulk upserting assets: %v - SQL: %s -- VALS: %+v", err, sqlStr, vals)
	}

	return nil
}

func removeDuplicateAsset(pAssets []persist.Asset) []persist.Asset {
	if len(pAssets) == 0 {
		return pAssets
	}
	unique := map[persist.AssetIdentifiers]bool{}
	result := make([]persist.Asset, 0, len(pAssets))
	for _, b := range pAssets {
		key := persist.NewAssetIdentifiers(b.Token.ContractAddress, b.OwnerAddress)

		if unique[key] {
			continue
		}
		result = append(result, b)
		unique[key] = true
	}
	return result
}
*/
