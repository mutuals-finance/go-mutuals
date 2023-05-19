package postgres

import (
	"context"
	"errors"
	db "github.com/mikeydub/go-gallery/db/gen/coredb"
	"github.com/mikeydub/go-gallery/util"

	"github.com/mikeydub/go-gallery/service/persist"
)

var errCollsNotOwnedByUser = errors.New("collections not owned by user")

// SplitRepository is the repository for interacting with splits in a postgres database
type SplitRepository struct {
	queries *db.Queries
}

// NewSplitRepository creates a new SplitTokenRepository
// TODO another join to addresses
func NewSplitRepository(queries *db.Queries) *SplitRepository {
	return &SplitRepository{queries: queries}
}

// Create creates a new split
func (g *SplitRepository) Create(pCtx context.Context, pSplit db.SplitRepoCreateParams) (db.Split, error) {

	gal, err := g.queries.SplitRepoCreate(pCtx, pSplit)
	if err != nil {
		return db.Split{}, err
	}

	return gal, nil
}

// Delete deletes a split and ensures that the collections of that split are passed on to another split
func (g *SplitRepository) Delete(pCtx context.Context, pSplit db.SplitRepoDeleteParams) error {

	err := g.queries.SplitRepoDelete(pCtx, pSplit)
	if err != nil {
		return err
	}

	return nil
}

// Update updates the gallery with the given ID and ensures that gallery is owned by the given userID
func (g *SplitRepository) Update(pCtx context.Context, pID persist.DBID, pUserID persist.DBID, pUpdate persist.SplitTokenUpdateInput) error {
	err := ensureCollsOwnedByUserToken(pCtx, g, pUpdate.Collections, pUserID)
	if err != nil {
		return err
	}

	rowsAffected, err := g.queries.SplitRepoUpdate(pCtx, db.SplitRepoUpdateParams{
		CollectionIds: pUpdate.Collections,
		SplitID:       pID,
	})

	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return persist.ErrSplitNotFound{ID: pID}
	}

	return nil
}

// AddCollections adds the given collections to the gallery with the given ID
func (g *SplitRepository) AddCollections(pCtx context.Context, pID persist.DBID, pUserID persist.DBID, pCollections []persist.DBID) error {

	err := ensureCollsOwnedByUserToken(pCtx, g, pCollections, pUserID)
	if err != nil {
		return err
	}

	rowsAffected, err := g.queries.SplitRepoAddCollections(pCtx, db.SplitRepoAddCollectionsParams{
		CollectionIds: util.StringersToStrings(pCollections),
		SplitID:       pID,
	})

	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return persist.ErrSplitNotFound{ID: pID}
	}

	return nil
}

func (g *SplitRepository) GetPreviewsURLsByUserID(pCtx context.Context, pUserID persist.DBID, limit int) ([]string, error) {
	return g.queries.SplitRepoGetPreviewsForUserID(pCtx, db.SplitRepoGetPreviewsForUserIDParams{
		OwnerUserID: pUserID,
		Limit:       int32(limit),
	})
}

func ensureCollsOwnedByUserToken(pCtx context.Context, g *SplitRepository, pColls []persist.DBID, pUserID persist.DBID) error {
	numOwned, err := g.queries.SplitRepoCheckOwnCollections(pCtx, db.SplitRepoCheckOwnCollectionsParams{
		CollectionIds: pColls,
		OwnerUserID:   pUserID,
	})

	if err != nil {
		return err
	}

	if numOwned != int64(len(pColls)) {
		return errCollsNotOwnedByUser
	}

	return nil
}
