package pool

import (
	"context"

	"github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/service/auth"
	"github.com/mutuals/go-mutuals/service/persist"
)

type CreatePoolInput struct {
	Name        string
	Description string
	Image       string
	Slug        string
	Private     bool
}

// CreatePool creates a new pool
func CreatePool(ctx context.Context, queries *coredb.Queries, input CreatePoolInput) (pool coredb.Pool, err error) {
	userId, err := auth.GetAuthenticatedUserId(ctx)
	if err != nil {
		return coredb.Pool{}, err
	}

	pool, err = queries.CreatePool(ctx, coredb.CreatePoolParams{
		ID:          persist.GenerateID(),
		Name:        input.Name,
		Description: input.Description,
		Image:       input.Image,
		Slug:        input.Slug,
		Private:     input.Private,
		OwnerID:     userId,
	})
	if err != nil {
		return coredb.Pool{}, err
	}

	return pool, nil
}

type UpdatePoolInput struct {
	ID          persist.DBID
	Name        *string
	Description *string
	Image       *string
	Slug        *string
	Private     *bool
}

// UpdatePool updates an existing pool
func UpdatePool(ctx context.Context, queries *coredb.Queries, input UpdatePoolInput) (pool coredb.Pool, err error) {
	// TODO check ownership
	// TODO change owner?
	// TODO change donationBps?

	params := coredb.UpdatePoolParams{
		ID: input.ID,
	}
	if input.Name != nil {
		params.Name = *input.Name
	}
	if input.Description != nil {
		params.Description = *input.Description
	}
	if input.Image != nil {
		params.Image = *input.Image
	}
	if input.Slug != nil {
		params.Slug = *input.Slug
	}
	if input.Private != nil {
		params.Private = *input.Private
	}

	pool, err = queries.UpdatePool(ctx, params)
	if err != nil {
		return coredb.Pool{}, err
	}

	return pool, nil
}

type GetPoolByIdInput struct {
	PoolID persist.DBID
}

// GetPoolById retrieves a pool by its ID
func GetPoolById(ctx context.Context, queries *coredb.Queries, input GetPoolByIdInput) (*coredb.Pool, error) {
	pool, err := queries.GetPoolById(ctx, input.PoolID)
	if err != nil {
		return nil, err
	}
	return &pool, nil
}

type DeletePoolInput struct {
	PoolID persist.DBID
}

// DeletePool soft-deletes a pool
func DeletePool(ctx context.Context, queries *coredb.Queries, input DeletePoolInput) error {
	return queries.DeletePool(ctx, input.PoolID)
}

type PoolBalance struct {
	TotalIncome float64
	Balance     float64
	Withdrawals float64
}

// GetPoolBalance returns aggregated USD balance stats for a pool.
// TODO: implement real calculation from deposits, withdrawals and token balances.
func GetPoolBalance(ctx context.Context, queries *coredb.Queries, poolID persist.DBID) (PoolBalance, error) {
	return PoolBalance{}, nil
}
