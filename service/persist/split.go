package persist

import (
	"database/sql"
	"fmt"
)

type Ownership = float64

type Recipient struct {
	Version      NullInt32       `json:"version"` // schema version for this model
	ID           DBID            `json:"id" binding:"required"`
	CreationTime CreationTime    `json:"created_at"`
	LastUpdated  LastUpdatedTime `json:"last_updated"`

	SplitID   DBID            `json:"split_id"`
	Address   EthereumAddress `json:"recipient_address"`
	Ownership Ownership       `json:"ownership"`
}

// SplitDB represents a split in the database.
// Assets will be represented as a list of token balance IDs creating
// a join relationship in the database
// This struct will only be used in database operations
type SplitDB struct {
	Version      NullInt32       `json:"version"` // schema version for this model
	ID           DBID            `json:"id" binding:"required"`
	CreationTime CreationTime    `json:"created_at"`
	Deleted      NullBool        `json:"-"`
	LastUpdated  LastUpdatedTime `json:"last_updated"`

	Chain          Chain           `json:"chain"`
	Address        Address         `json:"address"`
	Name           sql.NullString  `json:"name"`
	Description    NullString      `json:"description"`
	CreatorAddress EthereumAddress `json:"creator_address"`
	Recipients     []DBID          `json:"recipients"`
	LogoURL        NullString      `json:"logo_url"`
	BannerURL      NullString      `json:"banner_url"`
	Assets         []DBID          `json:"assets"`
}

// Split represents a group of collections of NFTS in the application.
// Assets are represented as structs instead of IDs
// This struct will be decoded from a find database operation and used throughout
// the application where SplitDB is not used
type Split struct {
	Version      NullInt32       `json:"version"` // schema version for this model
	ID           DBID            `json:"id" binding:"required"`
	CreationTime CreationTime    `json:"created_at"`
	Deleted      NullBool        `json:"-"`
	LastUpdated  LastUpdatedTime `json:"last_updated"`

	Chain          Chain          `json:"chain"`
	Address        Address        `json:"address"`
	Name           sql.NullString `json:"name"`
	Description    NullString     `json:"description"`
	CreatorAddress Address        `json:"creator_address"`
	Recipients     []Recipient    `json:"recipients"`
	LogoURL        NullString     `json:"profile_image_url"`
	BannerURL      NullString     `json:"profile_banner_url"`
	BadgeURL       NullString     `json:"badge_url"`
	Assets         []TokenBalance `json:"assets"`
}

// SplitTokenUpdateInput represents a struct that is used to update a splits list of collections in the databse
type SplitTokenUpdateInput struct {
	LastUpdated LastUpdatedTime `json:"last_updated"`

	Assets []DBID `json:"assets"`
}

// ErrSplitNotFound is returned when a split is not found by its ID
type ErrSplitNotFound struct {
	ID      DBID
	SplitID DBID
}

func (e ErrSplitNotFound) Error() string {
	return fmt.Sprintf("split not found with ID: %v SplitID: %v", e.ID, e.SplitID)
}

// ErrSplitNotFoundByAddress is returned when a split is not found by its address
type ErrSplitNotFoundByAddress struct {
	Address EthereumAddress
	Chain   Chain
}

func (e ErrSplitNotFoundByAddress) Error() string {
	return fmt.Sprintf("split not found with address: %v-%v", e.Address, e.Chain)
}
