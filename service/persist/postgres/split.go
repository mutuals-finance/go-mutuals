package postgres

import (
	"context"
	"database/sql"
	"fmt"
	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/lib/pq"
	"time"

	"github.com/SplitFi/go-splitfi/service/persist"
)

// SplitRepository is the repository for interacting with splits in a postgres database
type SplitRepository struct {
	db                         *sql.DB
	queries                    *db.Queries
	createStmt                 *sql.Stmt
	getByIDStmt                *sql.Stmt
	getByAddressStmt           *sql.Stmt
	getByRecipientStmt         *sql.Stmt
	getByRecipientPaginateStmt *sql.Stmt
	getRecipientStmt           *sql.Stmt
	getAssetStmt               *sql.Stmt
	upsertStmt                 *sql.Stmt
}

// NewSplitRepository creates a new SplitRepository
func NewSplitRepository(db *sql.DB, queries *db.Queries) *SplitRepository {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	createStmt, err := db.PrepareContext(ctx, `INSERT INTO splits (ID,VERSION,ADDRESS,NAME,DESCRIPTION,LOGO_URL,BANNER_URL,CREATOR_ADDRESS,CHAIN,RECIPIENTS,ASSETS) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING ID;`)
	checkNoErr(err)

	getByIDStmt, err := db.PrepareContext(ctx, `SELECT ID,VERSION,CREATED_AT,LAST_UPDATED,ADDRESS,NAME,DESCRIPTION,LOGO_URL,BANNER_URL,CREATOR_ADDRESS,CHAIN,RECIPIENTS,ASSETS FROM splits WHERE ID = $1;`)
	checkNoErr(err)

	getByAddressStmt, err := db.PrepareContext(ctx, `SELECT ID,VERSION,CREATED_AT,LAST_UPDATED,ADDRESS,NAME,DESCRIPTION,LOGO_URL,BANNER_URL,CREATOR_ADDRESS,CHAIN FROM splits WHERE ADDRESS = $1 AND CHAIN = $2 AND DELETED = false;`)
	checkNoErr(err)

	getByRecipientStmt, err := db.PrepareContext(ctx, `SELECT s.ID,s.VERSION,s.CREATED_AT,s.LAST_UPDATED,s.ADDRESS,s.NAME,s.DESCRIPTION,s.LOGO_URL,s.BANNER_URL,s.CREATOR_ADDRESS,s.CHAIN 
		FROM recipients r
		JOIN splits s ON s.ID = r.SPLIT_ID
		WHERE r.ADDRESS = $1 AND r.CHAIN = $2;`)
	checkNoErr(err)

	getByRecipientPaginateStmt, err := db.PrepareContext(ctx, `SELECT s.ID,s.VERSION,s.CREATED_AT,s.LAST_UPDATED,s.ADDRESS,s.NAME,s.DESCRIPTION,s.LOGO_URL,s.BANNER_URL,s.CREATOR_ADDRESS,s.CHAIN 
		FROM recipients r
		JOIN splits s ON s.ID = r.SPLIT_ID
		WHERE r.ADDRESS = $1 AND r.CHAIN = $2
		ORDER BY s.CREATED_AT DESC LIMIT $3 OFFSET $4`)
	checkNoErr(err)

	getRecipientStmt, err := db.PrepareContext(ctx, `SELECT VERSION,CREATED_AT,LAST_UPDATED,ADDRESS,OWNERSHIP FROM recipients WHERE ADDRESS = $1;`)
	checkNoErr(err)

	getAssetStmt, err := db.PrepareContext(ctx, `SELECT b.ID,b.VERSION,b.CREATED_AT,b.LAST_UPDATED,b.OWNER_ADDRESS,b.BALANCE,b.BLOCK_NUMBER 
		t.ID,t.TOKEN_TYPE,t.CHAIN,t.NAME,t.SYMBOL,t.LOGO,t.DECIMALS,t.TOTAL_SUPPLY,t.CONTRACT_ADDRESS,t.BLOCK_NUMBER,t.VERSION,t.CREATED_AT,t.LAST_UPDATED,t.IS_SPAM		
		FROM balances b
		JOIN tokens t ON t.CONTRACT_ADDRESS = b.TOKEN_ADDRESS 
		WHERE b.ID = $1;`)
	checkNoErr(err)

	upsertStmt, err := db.PrepareContext(ctx, `INSERT INTO splits (ID,VERSION,ADDRESS,NAME,DESCRIPTION,LOGO_URL,BANNER_URL,CREATOR_ADDRESS,CHAIN,CREATED_AT,LAST_UPDATED) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT (ADDRESS,CHAIN) DO UPDATE SET VERSION = EXCLUDED.VERSION, ADDRESS = EXCLUDED.ADDRESS, NAME = EXCLUDED.NAME, DESCRIPTION = EXCLUDED.DESCRIPTION, LOGO_URL = EXCLUDED.LOGO_URL, BANNER_URL = EXCLUDED.BANNER_URL, CREATOR_ADDRESS = EXCLUDED.CREATOR_ADDRESS, CHAIN = EXCLUDED.CHAIN, CREATED_AT = EXCLUDED.CREATED_AT, LAST_UPDATED = EXCLUDED.LAST_UPDATED;`)
	checkNoErr(err)

	return &SplitRepository{db: db, queries: queries, createStmt: createStmt, getByIDStmt: getByIDStmt, getByAddressStmt: getByAddressStmt, upsertStmt: upsertStmt, getByRecipientStmt: getByRecipientStmt, getRecipientStmt: getRecipientStmt, getByRecipientPaginateStmt: getByRecipientPaginateStmt, getAssetStmt: getAssetStmt}
}

// Create creates a new split in the database
func (s *SplitRepository) Create(pCtx context.Context, pSplit persist.SplitDB) (persist.DBID, error) {
	var id persist.DBID
	err := s.createStmt.QueryRowContext(pCtx, persist.GenerateID(), pSplit.Version, pSplit.Address, pSplit.Name, pSplit.Description, pSplit.LogoURL, pSplit.BannerURL, pSplit.CreatorAddress, pSplit.Chain, pq.Array(pSplit.Recipients), pq.Array(pSplit.Assets)).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

// GetByID returns a split by its ID
func (s *SplitRepository) GetByID(ctx context.Context, ID persist.DBID) (persist.Split, error) {
	split := persist.Split{}
	var recipientIDs []persist.DBID
	var assetIDs []persist.DBID

	err := s.getByIDStmt.QueryRowContext(ctx, ID).Scan(&split.ID, &split.Version, &split.CreationTime, &split.LastUpdated, &split.Address, &split.Name, &split.Description, &split.LogoURL, &split.BannerURL, &split.CreatorAddress, &split.Chain, pq.Array(&recipientIDs), pq.Array(&assetIDs))
	if err != nil {
		if err == sql.ErrNoRows {
			return split, persist.ErrSplitNotFound{SplitID: ID}
		}
		return split, err
	}

	recipients := make([]persist.Recipient, len(assetIDs))
	assets := make([]persist.TokenBalance, len(assetIDs))

	recipients, assets, err = getReceipientsAndAssets(ctx, s, recipientIDs, assetIDs)

	if err != nil {
		return persist.Split{}, fmt.Errorf("failed to get receipients or assets: %r", err)
	}
	split.Assets = assets
	split.Recipients = recipients

	return split, nil
}

// GetByAddress returns a split by its address
func (s *SplitRepository) GetByAddress(ctx context.Context, address persist.EthereumAddress, chain persist.Chain) (persist.Split, error) {
	split := persist.Split{}
	var recipientIDs []persist.DBID
	var assetIDs []persist.DBID

	err := s.getByAddressStmt.QueryRowContext(ctx, address, chain).Scan(&split.ID, &split.Version, &split.CreationTime, &split.LastUpdated, &split.Address, &split.Name, &split.Description, &split.LogoURL, &split.BannerURL, &split.CreatorAddress, &split.Chain, pq.Array(&recipientIDs), pq.Array(&assetIDs))
	if err != nil {
		if err == sql.ErrNoRows {
			return split, persist.ErrSplitNotFoundByAddress{Address: address, Chain: chain}
		}
		return split, err
	}

	recipients := make([]persist.Recipient, len(assetIDs))
	assets := make([]persist.TokenBalance, len(assetIDs))

	recipients, assets, err = getReceipientsAndAssets(ctx, s, recipientIDs, assetIDs)

	if err != nil {
		return persist.Split{}, fmt.Errorf("failed to get receipients or assets: %r", err)
	}
	split.Assets = assets
	split.Recipients = recipients

	return split, nil

}

// GetByRecipient returns splits from a recipient
func (s *SplitRepository) GetByRecipient(pCtx context.Context, pRecipientAddress persist.EthereumAddress, limit int64, offset int64) ([]persist.Split, error) {
	var rows *sql.Rows
	var err error
	if limit > 0 {
		rows, err = s.getByRecipientPaginateStmt.QueryContext(pCtx, pRecipientAddress, limit, offset)
	} else {
		rows, err = s.getByRecipientStmt.QueryContext(pCtx, pRecipientAddress)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	splits := make([]persist.Split, 0, 10)
	for rows.Next() {
		split := persist.Split{}
		var recipientIDs []persist.DBID
		var assetIDs []persist.DBID

		if err := rows.Scan(&split.ID, &split.Version, &split.CreationTime, &split.LastUpdated, &split.Address, &split.Name, &split.Description, &split.LogoURL, &split.BannerURL, &split.CreatorAddress, &split.Chain, pq.Array(&recipientIDs), pq.Array(&assetIDs)); err != nil {
			return nil, err
		}

		recipients := make([]persist.Recipient, len(assetIDs))
		assets := make([]persist.TokenBalance, len(assetIDs))
		recipients, assets, err = getReceipientsAndAssets(pCtx, s, recipientIDs, assetIDs)

		if err != nil {
			return nil, err
		}
		split.Assets = assets
		split.Recipients = recipients

		splits = append(splits, split)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return splits, nil
}

// Upsert upserts a split by its split ID and contract address
func (s *SplitRepository) Upsert(pCtx context.Context, pSplit persist.SplitDB) error {
	var err error
	_, err = s.upsertStmt.ExecContext(pCtx, persist.GenerateID(), pSplit.Version, pSplit.CreationTime, pSplit.LastUpdated, pSplit.Address, pSplit.Name, pSplit.Description, pSplit.LogoURL, pSplit.BannerURL, pSplit.CreatorAddress, pSplit.Chain, pSplit.Recipients, pSplit.Assets)
	return err
}

// UpdateByID updates a split by its ID
/*func (s *SplitRepository) UpdateByID(pCtx context.Context, pID persist.DBID, pUpdate interface{}) error {
	var res sql.Result
	var err error
	switch pUpdate.(type) {
	case persist.TokenBalanceUpdateInput:
		update := pUpdate.(persist.TokenBalanceUpdateInput)
		res, err = s.update.ExecContext(pCtx, update.Balance, update.BlockNumber, persist.LastUpdatedTime{}, pID)
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
*/

func (s *SplitRepository) GetPreviewsURLsByUserID(pCtx context.Context, pUserID persist.DBID, limit int) ([]string, error) {
	return s.queries.SplitRepoGetPreviewsForUserID(pCtx, db.SplitRepoGetPreviewsForUserIDParams{
		OwnerUserID: pUserID,
		Limit:       int32(limit),
	})
}

func getReceipientsAndAssets(pCtx context.Context, s *SplitRepository, recipientIDs, assetIDs []persist.DBID) ([]persist.Recipient, []persist.TokenBalance, error) {
	recipients := make([]persist.Recipient, len(recipientIDs))
	assets := make([]persist.TokenBalance, len(assetIDs))

	for i, recipientID := range recipientIDs {
		recipient := persist.Recipient{ID: recipientID}
		err := s.getRecipientStmt.QueryRowContext(pCtx, recipientID).Scan(&recipient.Version, &recipient.CreationTime, &recipient.LastUpdated, &recipient.Address, &recipient.Ownership)
		if err != nil {
			return []persist.Recipient{}, []persist.TokenBalance{}, fmt.Errorf("failed to get recipient: %r", err)
		}
		recipients[i] = recipient
	}

	for i, assetID := range assetIDs {
		asset := persist.TokenBalance{ID: assetID}
		err := s.getAssetStmt.QueryRowContext(pCtx, assetID).Scan(&asset.ID, &asset.Version, &asset.CreationTime, &asset.LastUpdated, &asset.OwnerAddress, &asset.Balance, &asset.BlockNumber, &asset.Token)
		if err != nil {
			return []persist.Recipient{}, []persist.TokenBalance{}, fmt.Errorf("failed to get assets: %r", err)
		}
		assets[i] = asset
	}

	return recipients, assets, nil
}
