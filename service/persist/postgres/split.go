package postgres

import (
	"context"
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

// Create creates a new split in the database
func (s *SplitRepository) Create(pCtx context.Context, pSplit db.SplitRepoCreateParams) (db.Split, error) {
	split, err := s.queries.SplitRepoCreate(pCtx, pSplit)
	if err != nil {
		return db.Split{}, err
	}

	return split, nil
}
