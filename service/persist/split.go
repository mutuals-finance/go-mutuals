package persist

import (
	"fmt"
)

// SplitDB represents a group of collections of NFTs in the database.
// Collections of NFTs will be represented as a list of collection IDs creating
// a join relationship in the database
// This struct will only be used in database operations
type SplitDB struct {
	Version      NullInt32       `json:"version"` // schema version for this model
	ID           DBID            `json:"id" binding:"required"`
	CreationTime CreationTime    `json:"created_at"`
	Deleted      NullBool        `json:"-"`
	LastUpdated  LastUpdatedTime `json:"last_updated"`

	OwnerUserID DBID   `json:"owner_user_id"`
	Collections []DBID `json:"collections"`
}

// Split represents a group of collections of NFTS in the application.
// Collections are represented as structs instead of IDs
// This struct will be decoded from a find database operation and used throughout
// the application where SplitDB is not used
type Split struct {
	Version      NullInt32       `json:"version"` // schema version for this model
	ID           DBID            `json:"id" binding:"required"`
	CreationTime CreationTime    `json:"created_at"`
	Deleted      NullBool        `json:"-"`
	LastUpdated  LastUpdatedTime `json:"last_updated"`

	OwnerUserID DBID         `json:"owner_user_id"`
	Collections []Collection `json:"collections"`
}

// SplitTokenUpdateInput represents a struct that is used to update a splits list of collections in the databse
type SplitTokenUpdateInput struct {
	LastUpdated LastUpdatedTime `json:"last_updated"`

	Collections []DBID `json:"collections"`
}

// ErrSplitNotFound is returned when a split is not found by its ID
type ErrSplitNotFound struct {
	ID           DBID
	CollectionID DBID
}

func (e ErrSplitNotFound) Error() string {
	return fmt.Sprintf("split not found with ID: %v CollectionID: %v", e.ID, e.CollectionID)
}
