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
	TokenAddress EthereumAddress
	OwnerAddress EthereumAddress
}

// Asset represents an address that owns a balance of tokens
type Asset struct {
	ID           DBID            `json:"id" binding:"required"`
	Version      NullInt32       `json:"version"` // schema version for this model
	LastUpdated  LastUpdatedTime `json:"last_updated"`
	CreationTime CreationTime    `json:"created_at"`
	OwnerAddress EthereumAddress `json:"owner_address"`
	Token        Token           `json:"token"`
	Balance      NullInt32       `json:"balance"`
	BlockNumber  BlockNumber     `json:"block_number"`
	// TODO make asset dependent on chain param?
	// Chain        Chain           `json:"chain"`
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
	OwnerAddress EthereumAddress
	TokenAddress EthereumAddress
	Chain        Chain
}

// AssetRepository represents a repository for interacting with persisted contracts
type AssetRepository interface {
	GetByOwner(context.Context, EthereumAddress, Chain, int64, int64) ([]Asset, error)
	GetByToken(context.Context, EthereumAddress, Chain, int64, int64) ([]Asset, error)
	GetByIdentifiers(context.Context, EthereumAddress, EthereumAddress, Chain) (Asset, error)
	BulkUpsert(context.Context, []Asset) error
	UpsertByIdentifiers(context.Context, EthereumAddress, EthereumAddress, Asset) error
	UpdateByID(context.Context, DBID, interface{}) error
	UpdateByIdentifiers(context.Context, EthereumAddress, EthereumAddress, Chain, interface{}) error
}

func (e ErrAssetNotFoundByID) Error() string {
	return fmt.Sprintf("asset not found by ID: %s", e.ID)
}

func (e ErrAssetNotFoundByIdentifiers) Error() string {
	return fmt.Sprintf("asset of %s not found for token address %s-%d", e.OwnerAddress, e.TokenAddress, e.Chain)
}

// NewAssetIdentifiers creates a new asset identifier
func NewAssetIdentifiers(pContractAddress EthereumAddress, pOwnerAddress EthereumAddress) AssetIdentifiers {
	return AssetIdentifiers(fmt.Sprintf("%s+%s", pContractAddress, pOwnerAddress))
}

func (b AssetIdentifiers) String() string {
	return string(b)
}

// GetParts returns the parts of the token identifiers
func (b AssetIdentifiers) GetParts() (EthereumAddress, EthereumAddress, error) {
	parts := strings.Split(b.String(), "+")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid token identifiers: %s", b)
	}
	return EthereumAddress(EthereumAddress(parts[0]).String()), EthereumAddress(EthereumAddress(parts[1]).String()), nil
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
