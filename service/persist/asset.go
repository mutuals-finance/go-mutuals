package persist

import (
	"time"
)

// AssetChainAddress represents a unique identifier for an asset
type AssetChainAddress struct {
	Chain        Chain
	TokenAddress Address
	OwnerAddress Address
}

// AssetDB represents an asset in the database.
// This struct will only be used in database operations
type AssetDB struct {
	ID           DBID        `json:"id" binding:"required"`
	Version      NullInt32   `json:"version"` // schema version for this model
	OwnerAddress Address     `json:"owner_address"`
	TokenAddress Address     `json:"token"`
	Chain        Chain       `json:"chain"`
	Balance      HexString   `json:"balance"`
	BlockNumber  BlockNumber `json:"block_number"`
	LastUpdated  time.Time   `json:"last_updated"`
	CreationTime time.Time   `json:"created_at"`
}

// Asset represents an address that owns a balance of tokens
type Asset struct {
	ID           DBID        `json:"id" binding:"required"`
	Version      NullInt32   `json:"version"` // schema version for this model
	LastUpdated  time.Time   `json:"last_updated"`
	CreationTime time.Time   `json:"created_at"`
	OwnerAddress Address     `json:"owner_address"`
	Token        Token       `json:"token"`
	Balance      NullInt32   `json:"balance"`
	BlockNumber  BlockNumber `json:"block_number"`
}
