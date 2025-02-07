package persist

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"time"
)

// SplitStatus is the type of status
type SplitStatus int

// CalculationType is the type of calculation
type CalculationType int

// RecipientType is the type of allocation
type RecipientType int

type Ownership = float64

type Recipient struct {
	Version      NullInt32 `json:"version"` // schema version for this model
	ID           DBID      `json:"id" binding:"required"`
	CreationTime time.Time `json:"created_at"`
	LastUpdated  time.Time `json:"last_updated"`

	SplitID   DBID      `json:"split_id"`
	Address   Address   `json:"recipient_address"`
	Ownership Ownership `json:"ownership"`
}

// SplitDB represents a split in the database.
// Assets will be represented as a list of token balance IDs creating
// a join relationship in the database
// This struct will only be used in database operations
type SplitDB struct {
	ID             DBID           `json:"id" binding:"required"`
	Version        NullInt32      `json:"version"` // schema version for this model
	CreationTime   time.Time      `json:"created_at"`
	LastUpdated    time.Time      `json:"last_updated"`
	Deleted        NullBool       `json:"-"`
	Chain          Chain          `json:"chain"`
	Address        Address        `json:"address"`
	Name           sql.NullString `json:"name"`
	Description    NullString     `json:"description"`
	CreatorAddress Address        `json:"creator_address"`
	LogoURL        NullString     `json:"logo_url"`
	BannerURL      NullString     `json:"banner_url"`
	BadgeURL       NullString     `json:"badge_url"`
	Allocations    []DBID         `json:"allocations"`
	Assets         []DBID         `json:"assets"`
}

// Split represents a group of collections of NFTS in the application.
// Assets are represented as structs instead of IDs
// This struct will be decoded from a find database operation and used throughout
// the application where SplitDB is not used
type Split struct {
	ID             DBID        `json:"id" binding:"required"`
	Version        NullInt32   `json:"version"` // schema version for this model
	CreationTime   time.Time   `json:"created_at"`
	LastUpdated    time.Time   `json:"last_updated"`
	Deleted        NullBool    `json:"-"`
	Chain          Chain       `json:"chain"`
	Address        Address     `json:"address"`
	Name           NullString  `json:"name"`
	Description    NullString  `json:"description"`
	CreatorAddress Address     `json:"creator_address"`
	LogoURL        NullString  `json:"logo_url"`
	BannerURL      NullString  `json:"banner_url"`
	BadgeURL       NullString  `json:"badge_url"`
	Allocations    []Recipient `json:"allocations"`
	Assets         []Asset     `json:"assets"`
}

// SplitRepository represents a repository for interacting with persisted splits
type SplitRepository interface {
	Create(context.Context, SplitDB) (DBID, error)
	GetByID(context.Context, DBID) (Split, error)
	GetByAddress(context.Context, Address, Chain) (Split, error)
	GetByRecipient(context.Context, Address, int64, int64) ([]Split, error)
	Upsert(context.Context, SplitDB) error
}

// SplitTokenUpdateInput represents a struct that is used to update a splits list of collections in the databse
type SplitTokenUpdateInput struct {
	LastUpdated time.Time `json:"last_updated"`

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
	Address Address
	Chain   Chain
}

func (e ErrSplitNotFoundByAddress) Error() string {
	return fmt.Sprintf("split not found with address: %v-%v", e.Address, e.Chain)
}

// ErrAllocationNotFound is returned when an allocation is not found by its ID
type ErrAllocationNotFound struct {
	ID DBID
}

func (e ErrAllocationNotFound) Error() string {
	return fmt.Sprintf("allocation not found with ID: %v", e.ID)
}

// ErrAllocationAggregationNotFound is returned when an allocation aggregation is not found by its ID
type ErrAllocationAggregationNotFound struct {
	ID DBID
}

func (e ErrAllocationAggregationNotFound) Error() string {
	return fmt.Sprintf("allocation aggregation not found with ID: %v", e.ID)
}

const (
	// SplitStatusDraft represents an draft split status
	SplitStatusDraft SplitStatus = iota
	// SplitStatusActive represents an active split status
	SplitStatusActive
	// SplitStatusPaused represents an active but paused split status
	SplitStatusPaused
)

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (ss *SplitStatus) UnmarshalGQL(v interface{}) error {
	n, ok := v.(string)
	if !ok {
		return fmt.Errorf("wrong type for SplitStatus: %T", v)
	}
	switch n {
	case "Draft":
		*ss = SplitStatusDraft
	case "Active":
		*ss = SplitStatusActive
	case "Paused":
		*ss = SplitStatusPaused
	default:
		return fmt.Errorf("unknown SplitStatus: %s", n)
	}
	return nil
}

// MarshalGQL implements the graphql.Marshaler interface
func (ss SplitStatus) MarshalGQL(w io.Writer) {
	switch ss {
	case SplitStatusDraft:
		w.Write([]byte(`"Draft"`))
	case SplitStatusActive:
		w.Write([]byte(`"Active"`))
	case SplitStatusPaused:
		w.Write([]byte(`"Paused"`))
	}
}

const (
	// CalculationTypePercentage represents a percentage allocation
	CalculationTypePercentage CalculationType = iota
	// CalculationTypeFixed represents a fixed allocation
	CalculationTypeFixed
)

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (ct *CalculationType) UnmarshalGQL(v interface{}) error {
	n, ok := v.(string)
	if !ok {
		return fmt.Errorf("wrong type for CalculationType: %T", v)
	}
	switch n {
	case "Percentage":
		*ct = CalculationTypePercentage
	case "Fixed":
		*ct = CalculationTypeFixed
	default:
		return fmt.Errorf("unknown CalculationType: %s", n)
	}
	return nil
}

// MarshalGQL implements the graphql.Marshaler interface
func (ct CalculationType) MarshalGQL(w io.Writer) {
	switch ct {
	case CalculationTypePercentage:
		w.Write([]byte(`"Percentage"`))
	case CalculationTypeFixed:
		w.Write([]byte(`"Fixed"`))
	}
}

const (
	// RecipientTypeDefaultItem represents
	RecipientTypeDefaultItem RecipientType = iota
	// RecipientTypeDefaultGroup represents
	RecipientTypeDefaultGroup
	// RecipientTypePrioritizedGroup represents
	RecipientTypePrioritizedGroup
	// RecipientTypeTimedGroup represents
	RecipientTypeTimedGroup
)

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (at *RecipientType) UnmarshalGQL(v interface{}) error {
	n, ok := v.(string)
	if !ok {
		return fmt.Errorf("wrong type for RecipientType: %T", v)
	}
	switch n {
	case "DefaultItem":
		*at = RecipientTypeDefaultItem
	case "DefaultGroup":
		*at = RecipientTypeDefaultGroup
	case "PrioritizedGroup":
		*at = RecipientTypePrioritizedGroup
	case "TimedGroup":
		*at = RecipientTypeTimedGroup
	default:
		return fmt.Errorf("unknown RecipientType: %s", n)
	}
	return nil
}

// MarshalGQL implements the graphql.Marshaler interface
func (at RecipientType) MarshalGQL(w io.Writer) {
	switch at {
	case RecipientTypeDefaultItem:
		w.Write([]byte(`"DefaultRecipient"`))
	case RecipientTypeDefaultGroup:
		w.Write([]byte(`"DefaultGroup"`))
	case RecipientTypePrioritizedGroup:
		w.Write([]byte(`"PrioritizedGroup"`))
	case RecipientTypeTimedGroup:
		w.Write([]byte(`"TimedGroup"`))
	}
}
