package postgres

import (
	db "github.com/mutuals/go-mutuals/db/gen/coredb"
)

// PoolRepository is the repository for interacting with pools in a postgres database
type PoolRepository struct {
	queries *db.Queries
}

// NewPoolRepository creates a new PoolRepository
func NewPoolRepository(queries *db.Queries) *PoolRepository {

	return &PoolRepository{queries: queries}
}
