package persist

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

// AssetIdentifiers represents a unique identifier for a asset
type AssetIdentifiers string

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
	Balance      NullInt32   `json:"balance"`
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

// AssetUpdateInput represents a struct that is used to update an asset in the database
type AssetUpdateInput struct {
	LastUpdated time.Time   `json:"last_updated"`
	Asset       NullInt32   `json:"asset"`
	BlockNumber BlockNumber `json:"block_number"`
}

// ErrAssetNotFoundByID is an error type for when a asset is not found by id
type ErrAssetNotFoundByID struct {
	ID DBID
}

// ErrAssetNotFoundByIdentifiers is an error that is returned when a asset is not found by its identifiers (owner address and token address)
type ErrAssetNotFoundByIdentifiers struct {
	OwnerAddress Address
	TokenAddress Address
	Chain        Chain
}

// AssetRepository represents a repository for interacting with persisted contracts
type AssetRepository interface {
	GetByOwner(context.Context, Address, int64, int64) ([]Asset, error)
	GetByToken(context.Context, Address, Chain, int64, int64) ([]Asset, error)
	GetByIdentifiers(context.Context, Address, Address, Chain) (Asset, error)
	BulkUpsert(context.Context, []Asset) (time.Time, []Asset, error)
	UpsertByIdentifiers(context.Context, Address, Address, Asset) error
	UpdateByID(context.Context, DBID, interface{}) error
	UpdateByIdentifiers(context.Context, Address, Address, Chain, interface{}) error
}

func (e ErrAssetNotFoundByID) Error() string {
	return fmt.Sprintf("asset not found by ID: %s", e.ID)
}

func (e ErrAssetNotFoundByIdentifiers) Error() string {
	return fmt.Sprintf("asset of %s not found for token address %s-%d", e.OwnerAddress, e.TokenAddress, e.Chain)
}

// NewAssetIdentifiers creates a new asset identifier
func NewAssetIdentifiers(pContractAddress Address, pOwnerAddress Address) AssetIdentifiers {
	return AssetIdentifiers(fmt.Sprintf("%s+%s", pContractAddress, pOwnerAddress))
}

func (b AssetIdentifiers) String() string {
	return string(b)
}

// GetParts returns the parts of the token identifiers
func (b AssetIdentifiers) GetParts() (Address, Address, error) {
	parts := strings.Split(b.String(), "+")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid token identifiers: %s", b)
	}
	return Address(parts[0]), Address(parts[1]), nil
}

// Value implements the driver.Valuer interface
func (b AssetIdentifiers) Value() (driver.Value, error) {
	return b.String(), nil
}

// Scan implements the database/sql Scanner interface for the TokenIdentifiers type
func (b *AssetIdentifiers) Scan(i interface{}) error {
	if i == nil {
		*b = ""
		return nil
	}
	res := strings.Split(i.(string), "+")
	if len(res) != 2 {
		return fmt.Errorf("invalid asset identifiers: %v - %T", i, i)
	}
	*b = AssetIdentifiers(fmt.Sprintf("%s+%s", res[0], res[1]))

	return nil
}
