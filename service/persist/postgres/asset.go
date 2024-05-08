package postgres

import (
	"context"
	"database/sql"
	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/service/logger"
	"math/big"
	"time"

	"github.com/SplitFi/go-splitfi/service/persist"
)

// AssetRepository represents an asset repository in the postgres database
type AssetRepository struct {
	db      *sql.DB
	queries *db.Queries
}

// NewAssetRepository creates a new postgres repository for interacting with assets
func NewAssetRepository(db *sql.DB, queries *db.Queries) *AssetRepository {

	return &AssetRepository{db: db, queries: queries}
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
		(*a).ID = upserted[i].Asset.ID
		(*a).CreationTime = time.Time(upserted[i].Asset.CreatedAt)
		(*a).LastUpdated = time.Time(upserted[i].Asset.LastUpdated)
	}

	return upserted[0].Asset.LastUpdated, assets, nil

}

func (a *AssetRepository) excludeZeroBalanceAssets(pCtx context.Context, pAssets []persist.AssetDB) ([]persist.AssetDB, error) {
	newAssets := make([]persist.AssetDB, 0, len(pAssets))
	for _, asset := range pAssets {
		if asset.Balance.BigInt().Cmp(new(big.Int)) <= 0 {
			logger.For(pCtx).Warnf("Asset %s from %s has zero balance", asset.TokenAddress, asset.OwnerAddress)
			continue
		}
		newAssets = append(newAssets, asset)
	}
	return newAssets, nil
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
