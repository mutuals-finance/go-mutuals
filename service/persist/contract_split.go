package persist

import (
	"context"
	"database/sql"
	"fmt"
)

// ContractSplit represents a smart contract in the database
type ContractSplit struct {
	Version      NullInt32       `json:"version"` // schema version for this model
	ID           DBID            `json:"id" binding:"required"`
	CreationTime CreationTime    `json:"created_at"`
	Deleted      NullBool        `json:"-"`
	LastUpdated  LastUpdatedTime `json:"last_updated"`

	Chain            Chain          `json:"chain"`
	Address          Address        `json:"address"`
	Name             sql.NullString `json:"name"`
	Description      NullString     `json:"description"`
	CreatorAddress   Address        `json:"creator_address"`
	ProfileImageURL  NullString     `json:"profile_image_url"`
	ProfileBannerURL NullString     `json:"profile_banner_url"`
	BadgeURL         NullString     `json:"badge_url"`
}

// ErrContractNotFoundByAddress is an error type for when a contract is not found by address
type ErrSplitContractNotFound struct {
	Address Address
	Chain   Chain
}

// ContractSplitRepository represents a repository for interacting with persisted contracts
type ContractSplitRepository interface {
	GetByID(ctx context.Context, id DBID) (ContractSplit, error)
	GetByAddress(context.Context, Address, Chain) (ContractSplit, error)
	GetByAddresses(context.Context, []Address, Chain) ([]ContractSplit, error)
	UpsertByAddress(context.Context, Address, Chain, ContractSplit) error
	BulkUpsert(context.Context, []ContractSplit) error
	GetOwnersByAddress(context.Context, Address, Chain, int, int) ([]TokenHolder, error)
}

func (c ContractSplit) ContractIdentifiers() ContractIdentifiers {
	return NewContractIdentifiers(c.Address, c.Chain)
}

func (e ErrSplitContractNotFound) Error() string {
	return fmt.Sprintf("contract not found by address: %s-%d", e.Address, e.Chain)
}
