package pool

import (
	"context"

	"github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/service/auth"
	"github.com/mutuals/go-mutuals/service/persist"
)

// CreatePool creates a new pool
func CreatePool(ctx context.Context, queries *coredb.Queries, name string, description string, image string, slug string, private bool) (pool coredb.Pool, err error) {
	userId, err := auth.GetAuthenticatedUserId(ctx)
	if err != nil {
		return coredb.Pool{}, err
	}

	pool, err = queries.CreatePool(ctx, coredb.CreatePoolParams{
		ID:          persist.GenerateID(),
		Name:        name,
		Description: description,
		Image:       image,
		Slug:        slug,
		Private:     private,
		OwnerID:     userId,
	})
	if err != nil {
		return coredb.Pool{}, err
	}

	return pool, nil
}

// UpdatePool updates a new pool
func UpdatePool(ctx context.Context, queries *coredb.Queries, id persist.DBID, name *string, description *string, image *string, slug *string, private *bool) (pool coredb.Pool, err error) {
	// TODO check ownership
	// TODO change owner?
	// TODO change donationBps?
	if err != nil {
		return coredb.Pool{}, err
	}

	params := coredb.UpdatePoolParams{
		ID: id,
	}
	if name != nil {
		params.Name = *name
	}
	if description != nil {
		params.Description = *description
	}
	if image != nil {
		params.Image = *image
	}
	if slug != nil {
		params.Slug = *slug
	}
	if private != nil {
		params.Private = *private
	}

	pool, err = queries.UpdatePool(ctx, params)
	if err != nil {
		return coredb.Pool{}, err
	}

	return pool, nil
}
