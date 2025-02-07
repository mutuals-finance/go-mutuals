package postgres

import (
	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
)

// SplitRepository is the repository for interacting with splits in a postgres database
type SplitRepository struct {
	queries *db.Queries
}

// NewSplitRepository creates a new SplitRepository
func NewSplitRepository(queries *db.Queries) *SplitRepository {

	return &SplitRepository{queries: queries}
}
