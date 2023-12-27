package postgres

import (
	"context"
	"database/sql"
	"fmt"
	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/service/logger"
	"time"

	"github.com/SplitFi/go-splitfi/service/persist"
)

// AssetRepository represents an asset repository in the postgres database
type AssetRepository struct {
	db                                 *sql.DB
	queries                            *db.Queries
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
func NewAssetRepository(db *sql.DB, queries *db.Queries) *AssetRepository {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	getByOwnerStmt, err := db.PrepareContext(ctx, `SELECT a.ID,a.VERSION,a.CREATED_AT,a.LAST_UPDATED,a.OWNER_ADDRESS,a.BALANCE,a.BLOCK_NUMBER,
		t.ID,t.TOKEN_TYPE,t.CHAIN,t.NAME,t.SYMBOL,t.LOGO,t.DECIMALS,t.TOTAL_SUPPLY,t.CONTRACT_ADDRESS,t.BLOCK_NUMBER,t.VERSION,t.CREATED_AT,t.LAST_UPDATED,t.IS_SPAM		
		FROM assets a
		JOIN tokens t ON t.CONTRACT_ADDRESS = a.TOKEN_ADDRESS
		WHERE a.OWNER_ADDRESS = $1 AND t.DELETED = false;`)
	checkNoErr(err)

	getByOwnerPaginateStmt, err := db.PrepareContext(ctx, `SELECT a.ID,a.VERSION,a.CREATED_AT,a.LAST_UPDATED,a.OWNER_ADDRESS,a.BALANCE,a.BLOCK_NUMBER,
		t.ID,t.TOKEN_TYPE,t.CHAIN,t.NAME,t.SYMBOL,t.LOGO,t.DECIMALS,t.TOTAL_SUPPLY,t.CONTRACT_ADDRESS,t.BLOCK_NUMBER,t.VERSION,t.CREATED_AT,t.LAST_UPDATED,t.IS_SPAM		
		FROM assets a
		JOIN tokens t ON t.CONTRACT_ADDRESS = a.TOKEN_ADDRESS
		WHERE a.OWNER_ADDRESS = $1 AND t.DELETED = false;
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

	return &AssetRepository{db: db, queries: queries, getByOwnerStmt: getByOwnerStmt, getByOwnerPaginateStmt: getByOwnerPaginateStmt, getByTokenStmt: getByTokenStmt, getByTokenPaginateStmt: getByTokenPaginateStmt, getByIdentifiersStmt: getByIdentifiersStmt, upsertByIdentifiersStmt: upsertByIdentifiersStmt, updateAssetUnsafeStmt: updateAssetUnsafeStmt, updateAssetByIdentifiersUnsafeStmt: updateAssetByIdentifiersUnsafeStmt}
}

// GetByOwner retrieves all assets associated with an owner ethereum address
func (a *AssetRepository) GetByOwner(pCtx context.Context, pAddress persist.Address, limit int64, offset int64) ([]persist.AssetDB, error) {
	var rows *sql.Rows
	var err error
	if limit > 0 {
		rows, err = a.getByOwnerPaginateStmt.QueryContext(pCtx, pAddress, limit, offset)
	} else {
		rows, err = a.getByOwnerStmt.QueryContext(pCtx, pAddress)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	assets := make([]persist.AssetDB, 0, 10)
	for rows.Next() {
		asset := persist.AssetDB{}
		if err := rows.Scan(&asset.ID, &asset.Version, &asset.CreationTime, &asset.LastUpdated, &asset.OwnerAddress, &asset.Balance, &asset.BlockNumber, &asset.TokenAddress, &asset.Chain); err != nil {
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
func (a *AssetRepository) GetByToken(pCtx context.Context, pAddress persist.Address, pChain persist.Chain, limit int64, offset int64) ([]persist.Asset, error) {
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
func (a *AssetRepository) GetByIdentifiers(pCtx context.Context, pOwnerAddress, pTokenAddress persist.Address, pChain persist.Chain) (persist.Asset, error) {
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

// UpsertByIdentifiers upserts the asset with the given owner address and token address
func (a *AssetRepository) UpsertByIdentifiers(pCtx context.Context, pOwnerAddress, pTokenAddress persist.Address, pAsset persist.Asset) error {
	_, err := a.upsertByIdentifiersStmt.ExecContext(pCtx, persist.GenerateID(), pAsset.Version, pOwnerAddress, pTokenAddress, pAsset.Balance, pAsset.BlockNumber, pAsset.CreationTime, pAsset.LastUpdated)
	if err != nil {
		return err
	}

	return nil
}

// BulkUpsert upserts the asset with the given owner address and token address
func (a *AssetRepository) BulkUpsert(pCtx context.Context, pAssets []persist.AssetDB) (time.Time, []persist.AssetDB, error) {
	assets, err := a.excludeZeroBalanceAssets(pCtx, pAssets)
	if err != nil {
		return time.Time{}, nil, err
	}

	// If we're not upserting anything, we still need to return the current database time
	// since it may be used by the caller and is assumed valid if err == nil
	if len(assets) == 0 {
		currentTime, err := a.queries.GetCurrentTime(pCtx)
		if err != nil {
			return time.Time{}, nil, err
		}
		return currentTime, []persist.AssetDB{}, nil
	}

	logger.For(pCtx).Infof("Deduping %d assets", len(assets))

	assets = a.dedupeAssets(assets)

	logger.For(pCtx).Infof("Deduped down to %d assets", len(assets))

	logger.For(pCtx).Infof("Starting upsert...")

	var errors []error

	params := db.UpsertAssetsParams{}

	for i := range assets {
		a := &assets[i]
		params.ID = append(params.ID, persist.GenerateID().String())
		params.Version = append(params.Version, a.Version.Int32())
		params.Balance = append(params.Balance, a.Balance.String())
		params.OwnerAddress = append(params.OwnerAddress, a.OwnerAddress.String())
		params.TokenAddress = append(params.TokenAddress, a.TokenAddress.String())
		params.BlockNumber = append(params.BlockNumber, a.BlockNumber.BigInt().Int64())

		// Defer error checking until now to keep the code above from being
		// littered with multiline "if" statements
		if len(errors) > 0 {
			return time.Time{}, nil, errors[0]
		}
	}

	upserted, err := a.queries.UpsertAssets(pCtx, params)
	if err != nil {
		return time.Time{}, nil, err
	}

	// Update tokens with the existing data if the token already exists.
	for i := range assets {
		a := &assets[i]
		(*a).ID = upserted[i].ID
		(*a).CreationTime = persist.CreationTime(upserted[i].CreatedAt)
		(*a).LastUpdated = persist.LastUpdatedTime(upserted[i].LastUpdated)
	}

	return upserted[0].LastUpdated, assets, nil

}

func (a *AssetRepository) excludeZeroBalanceAssets(pCtx context.Context, pAssets []persist.AssetDB) ([]persist.AssetDB, error) {
	newAssets := make([]persist.AssetDB, 0, len(pAssets))
	for _, asset := range pAssets {
		if asset.Balance <= 0 {
			logger.For(pCtx).Warnf("Asset %s from %s has zero balance", asset.TokenAddress, asset.OwnerAddress)
			continue
		}
		newAssets = append(newAssets, asset)
	}
	return newAssets, nil
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
func (a *AssetRepository) UpdateByIdentifiers(pCtx context.Context, pOwnerAddress, pTokenAddress persist.Address, pChain persist.Chain, pUpdate interface{}) error {

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

func (a *AssetRepository) dedupeAssets(pAssets []persist.AssetDB) []persist.AssetDB {
	seen := map[persist.AssetIdentifiers]persist.AssetDB{}
	for _, asset := range pAssets {
		key := persist.NewAssetIdentifiers(asset.TokenAddress, asset.OwnerAddress)
		if seenToken, ok := seen[key]; ok {
			if seenToken.BlockNumber.Uint64() > asset.BlockNumber.Uint64() {
				continue
			}
		}
		seen[key] = asset
	}
	result := make([]persist.AssetDB, 0, len(seen))
	for _, v := range seen {
		result = append(result, v)
	}
	return result
}
