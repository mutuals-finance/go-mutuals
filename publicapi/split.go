package publicapi

import (
	"context"
	"crypto/sha256"
	"encoding"
	"encoding/base64"
	"fmt"
	"net"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"github.com/mikeydub/go-gallery/env"
	"github.com/mikeydub/go-gallery/service/auth"
	"github.com/mikeydub/go-gallery/service/persist/postgres"
	"github.com/mikeydub/go-gallery/util"
	"github.com/mikeydub/go-gallery/validate"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-playground/validator/v10"
	"github.com/mikeydub/go-gallery/db/gen/coredb"
	db "github.com/mikeydub/go-gallery/db/gen/coredb"
	"github.com/mikeydub/go-gallery/graphql/dataloader"
	"github.com/mikeydub/go-gallery/graphql/model"
	"github.com/mikeydub/go-gallery/service/persist"
)

const maxCollectionsPerSplit = 1000

type SplitAPI struct {
	repos     *postgres.Repositories
	queries   *db.Queries
	loaders   *dataloader.Loaders
	validator *validator.Validate
	ethClient *ethclient.Client
}

func (api SplitAPI) CreateSplit(ctx context.Context, name, description *string, position string) (db.Split, error) {

	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"name":        {name, "max=200"},
		"description": {description, "max=600"},
		"position":    {position, "required"},
	}); err != nil {
		return db.Split{}, err
	}

	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return db.Split{}, err
	}

	gallery, err := api.repos.SplitRepository.Create(ctx, db.SplitRepoCreateParams{
		SplitID:     persist.GenerateID(),
		Name:        util.FromPointer(name),
		Description: util.FromPointer(description),
		Position:    position,
		OwnerUserID: userID,
	})
	if err != nil {
		return db.Split{}, err
	}

	return gallery, nil
}

func (api SplitAPI) UpdateSplit(ctx context.Context, update model.UpdateSplitInput) (db.Split, error) {

	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"galleryID":           {update.SplitID, "required"},
		"name":                {update.Name, "omitempty,max=200"},
		"description":         {update.Description, "omitempty,max=600"},
		"deleted_collections": {update.DeletedCollections, "omitempty,unique"},
		"created_collections": {update.CreatedCollections, "omitempty,created_collections"},
	}); err != nil {
		return db.Split{}, err
	}

	events := make([]db.Event, 0, len(update.CreatedCollections)+len(update.UpdatedCollections)+1)

	curGal, err := api.loaders.SplitBySplitID.Load(update.SplitID)
	if err != nil {
		return db.Split{}, err
	}

	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return db.Split{}, err
	}

	if curGal.OwnerUserID != userID {
		return db.Split{}, fmt.Errorf("user %s is not the owner of gallery %s", userID, update.SplitID)
	}

	tx, err := api.repos.BeginTx(ctx)
	if err != nil {
		return db.Split{}, err
	}
	defer tx.Rollback(ctx)

	q := api.queries.WithTx(tx)

	// then delete collections
	if len(update.DeletedCollections) > 0 {
		err = q.DeleteCollections(ctx, util.StringersToStrings(update.DeletedCollections))
		if err != nil {
			return db.Split{}, err
		}
	}

	// create collections
	mappedIDs := make(map[persist.DBID]persist.DBID)
	for _, c := range update.CreatedCollections {
		collectionID, err := q.CreateCollection(ctx, db.CreateCollectionParams{
			ID:             persist.GenerateID(),
			Name:           persist.StrPtrToNullStr(&c.Name),
			CollectorsNote: persist.StrPtrToNullStr(&c.CollectorsNote),
			OwnerUserID:    curGal.OwnerUserID,
			SplitID:        update.SplitID,
			Layout:         modelToTokenLayout(c.Layout),
			Hidden:         c.Hidden,
			Nfts:           c.Tokens,
			TokenSettings:  modelToTokenSettings(c.TokenSettings),
		})
		if err != nil {
			return db.Split{}, err
		}

		events = append(events, db.Event{
			ID:             persist.GenerateID(),
			ActorID:        persist.DBIDToNullStr(userID),
			Action:         persist.ActionCollectionCreated,
			ResourceTypeID: persist.ResourceTypeCollection,
			SubjectID:      collectionID,
			CollectionID:   collectionID,
			SplitID:        update.SplitID,
			Data: persist.EventData{
				CollectionTokenIDs:       c.Tokens,
				CollectionCollectorsNote: c.CollectorsNote,
			},
		})

		mappedIDs[c.GivenID] = collectionID
	}

	// update collections

	if len(update.UpdatedCollections) > 0 {
		collEvents, err := updateCollectionsInfoAndTokens(ctx, q, userID, update.SplitID, update.UpdatedCollections)
		if err != nil {
			return db.Split{}, err
		}

		events = append(events, collEvents...)
	}

	// order collections
	for i, c := range update.Order {
		if newID, ok := mappedIDs[c]; ok {
			update.Order[i] = newID
		}
	}

	params := db.UpdateSplitInfoParams{
		ID: update.SplitID,
	}

	util.SetConditionalValue(update.Name, &params.Name, &params.NameSet)
	util.SetConditionalValue(update.Description, &params.Description, &params.DescriptionSet)

	err = q.UpdateSplitInfo(ctx, params)
	if err != nil {
		return db.Split{}, err
	}

	if update.Name != nil || update.Description != nil {
		e := db.Event{
			ID:             persist.GenerateID(),
			ActorID:        persist.DBIDToNullStr(userID),
			Action:         persist.ActionSplitInfoUpdated,
			ResourceTypeID: persist.ResourceTypeSplit,
			SplitID:        update.SplitID,
			SubjectID:      update.SplitID,
		}

		change := false

		if update.Name != nil && *update.Name != curGal.Name {
			e.Data.SplitName = update.Name
			change = true
		}

		if update.Description != nil && *update.Description != curGal.Description {
			e.Data.SplitDescription = update.Description
			change = true
		}

		if change {
			events = append(events, e)
		}

	}

	asList := persist.DBIDList(update.Order)

	if len(asList) > 0 {
		err = q.UpdateSplitCollections(ctx, db.UpdateSplitCollectionsParams{
			SplitID:     update.SplitID,
			Collections: asList,
		})
		if err != nil {
			return db.Split{}, err
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return db.Split{}, err
	}

	newGall, err := api.loaders.SplitBySplitID.Load(update.SplitID)
	if err != nil {
		return db.Split{}, err
	}

	if update.Caption != nil && *update.Caption == "" {
		update.Caption = nil
	}
	err = dispatchEvents(ctx, events, api.validator, update.EditID, nil)
	if err != nil {
		return db.Split{}, err
	}

	return newGall, nil
}

func (api SplitAPI) PublishSplit(ctx context.Context, update model.PublishSplitInput) error {

	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"galleryID": {update.SplitID, "required"},
		"editID":    {update.EditID, "required"},
	}); err != nil {
		return err
	}

	err := publishEventGroup(ctx, update.EditID, persist.ActionSplitUpdated, update.Caption)
	if err != nil {
		return err
	}

	return nil
}
func updateCollectionsInfoAndTokens(ctx context.Context, q *db.Queries, actor, gallery persist.DBID, update []*model.UpdateCollectionInput) ([]db.Event, error) {

	events := make([]db.Event, 0)

	dbids, err := util.Map(update, func(u *model.UpdateCollectionInput) (string, error) {
		return u.Dbid.String(), nil
	})
	if err != nil {
		return nil, err
	}

	collectorNotes, err := util.Map(update, func(u *model.UpdateCollectionInput) (string, error) {
		return u.CollectorsNote, nil
	})
	if err != nil {
		return nil, err
	}

	layouts, err := util.Map(update, func(u *model.UpdateCollectionInput) (pgtype.JSONB, error) {
		return persist.ToJSONB(modelToTokenLayout(u.Layout))
	})
	if err != nil {
		return nil, err
	}

	tokenSettings, err := util.Map(update, func(u *model.UpdateCollectionInput) (pgtype.JSONB, error) {
		settings := modelToTokenSettings(u.TokenSettings)
		return persist.ToJSONB(settings)
	})
	if err != nil {
		return nil, err
	}

	hiddens, err := util.Map(update, func(u *model.UpdateCollectionInput) (bool, error) {
		return u.Hidden, nil
	})
	if err != nil {
		return nil, err
	}

	names, err := util.Map(update, func(u *model.UpdateCollectionInput) (string, error) {
		return u.Name, nil
	})
	if err != nil {
		return nil, err
	}

	for _, collection := range update {
		curCol, err := q.GetCollectionById(ctx, collection.Dbid)
		if err != nil {
			return nil, err
		}

		// add event if collectors note updated
		if collection.CollectorsNote != "" && collection.CollectorsNote != curCol.CollectorsNote.String {
			events = append(events, db.Event{
				ActorID:        persist.DBIDToNullStr(actor),
				ResourceTypeID: persist.ResourceTypeCollection,
				SubjectID:      collection.Dbid,
				Action:         persist.ActionCollectorsNoteAddedToCollection,
				CollectionID:   collection.Dbid,
				SplitID:        gallery,
				Data: persist.EventData{
					CollectionCollectorsNote: collection.CollectorsNote,
				},
			})
		}
	}

	err = q.UpdateCollectionsInfo(ctx, db.UpdateCollectionsInfoParams{
		Ids:             dbids,
		Names:           names,
		CollectorsNotes: collectorNotes,
		Layouts:         layouts,
		TokenSettings:   tokenSettings,
		Hidden:          hiddens,
	})
	if err != nil {
		return nil, err
	}

	for _, collection := range update {
		curTokens, err := q.GetCollectionTokensByCollectionID(ctx, collection.Dbid)
		if err != nil {
			return nil, err
		}

		err = q.UpdateCollectionTokens(ctx, db.UpdateCollectionTokensParams{
			ID:   collection.Dbid,
			Nfts: collection.Tokens,
		})
		if err != nil {
			return nil, err
		}

		diff := util.Difference(curTokens, collection.Tokens)

		if len(diff) > 0 {
			events = append(events, db.Event{
				ResourceTypeID: persist.ResourceTypeCollection,
				SubjectID:      collection.Dbid,
				Action:         persist.ActionTokensAddedToCollection,
				ActorID:        persist.DBIDToNullStr(actor),
				CollectionID:   collection.Dbid,
				SplitID:        gallery,
				Data: persist.EventData{
					CollectionTokenIDs: diff,
				},
			})
		}
	}
	return events, nil
}

func (api SplitAPI) DeleteSplit(ctx context.Context, galleryID persist.DBID) error {

	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"galleryID": {galleryID, "required"},
	}); err != nil {
		return err
	}

	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return err
	}

	err = api.repos.SplitRepository.Delete(ctx, db.SplitRepoDeleteParams{
		SplitID:     galleryID,
		OwnerUserID: userID,
	})
	if err != nil {
		return err
	}

	return nil
}

func (api SplitAPI) GetSplitById(ctx context.Context, galleryID persist.DBID) (*db.Split, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"galleryID": {galleryID, "required"},
	}); err != nil {
		return nil, err
	}

	gallery, err := api.loaders.SplitBySplitID.Load(galleryID)
	if err != nil {
		return nil, err
	}

	return &gallery, nil
}

func (api SplitAPI) GetViewerSplitById(ctx context.Context, galleryID persist.DBID) (*db.Split, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"galleryID": {galleryID, "required"},
	}); err != nil {
		return nil, err
	}

	userID, err := getAuthenticatedUserID(ctx)

	if err != nil {
		return nil, persist.ErrSplitNotFound{ID: galleryID}
	}

	gallery, err := api.loaders.SplitBySplitID.Load(galleryID)
	if err != nil {
		return nil, err
	}

	if userID != gallery.OwnerUserID {
		return nil, persist.ErrSplitNotFound{ID: galleryID}
	}

	return &gallery, nil
}

func (api SplitAPI) GetSplitByCollectionId(ctx context.Context, collectionID persist.DBID) (*db.Split, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"collectionID": {collectionID, "required"},
	}); err != nil {
		return nil, err
	}

	gallery, err := api.loaders.SplitByCollectionID.Load(collectionID)
	if err != nil {
		return nil, err
	}

	return &gallery, nil
}

func (api SplitAPI) GetSplitsByUserId(ctx context.Context, userID persist.DBID) ([]db.Split, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"userID": {userID, "required"},
	}); err != nil {
		return nil, err
	}

	splits, err := api.loaders.SplitsByUserID.Load(userID)
	if err != nil {
		return nil, err
	}

	return splits, nil
}

func (api SplitAPI) GetTokenPreviewsBySplitID(ctx context.Context, galleryID persist.DBID) ([]persist.Media, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"galleryID": {galleryID, "required"},
	}); err != nil {
		return nil, err
	}

	medias, err := api.queries.GetSplitTokenMediasBySplitID(ctx, db.GetSplitTokenMediasBySplitIDParams{
		ID:    galleryID,
		Limit: 4,
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return medias, nil
}

func (api SplitAPI) UpdateSplitCollections(ctx context.Context, galleryID persist.DBID, collections []persist.DBID) error {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"galleryID":   {galleryID, "required"},
		"collections": {collections, fmt.Sprintf("required,unique,max=%d", maxCollectionsPerSplit)},
	}); err != nil {
		return err
	}

	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return err
	}

	update := persist.SplitTokenUpdateInput{Collections: collections}

	err = api.repos.SplitRepository.Update(ctx, galleryID, userID, update)
	if err != nil {
		return err
	}

	return nil
}

func (api SplitAPI) UpdateSplitInfo(ctx context.Context, galleryID persist.DBID, name, description *string) error {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"galleryID":   {galleryID, "required"},
		"name":        {name, "max=200"},
		"description": {description, "max=600"},
	}); err != nil {
		return err
	}

	var nullName, nullDesc string
	if name != nil {
		nullName = *name
	}
	if description != nil {
		nullDesc = *description
	}

	err := api.queries.UpdateSplitInfo(ctx, db.UpdateSplitInfoParams{
		ID:          galleryID,
		Name:        nullName,
		Description: nullDesc,
	})
	if err != nil {
		return err
	}
	return nil
}

func (api SplitAPI) UpdateSplitHidden(ctx context.Context, galleryID persist.DBID, hidden bool) (coredb.Split, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"galleryID": {galleryID, "required"},
	}); err != nil {
		return db.Split{}, err
	}

	gallery, err := api.queries.UpdateSplitHidden(ctx, db.UpdateSplitHiddenParams{
		ID:     galleryID,
		Hidden: hidden,
	})
	if err != nil {
		return db.Split{}, err
	}

	return gallery, nil
}

func (api SplitAPI) UpdateSplitPositions(ctx context.Context, positions []*model.SplitPositionInput) error {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"positions": {positions, "required,min=1"},
	}); err != nil {
		return err
	}

	user, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return err
	}

	ids := make([]string, len(positions))
	pos := make([]string, len(positions))
	for i, position := range positions {
		ids[i] = position.SplitID.String()
		pos[i] = position.Position
	}

	tx, err := api.repos.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	q := api.queries.WithTx(tx)

	err = q.UpdateSplitPositions(ctx, db.UpdateSplitPositionsParams{
		SplitIds:    ids,
		Positions:   pos,
		OwnerUserID: user,
	})
	if err != nil {
		return err
	}

	areDuplicates, err := q.UserHasDuplicateSplitPositions(ctx, user)
	if err != nil {
		return err
	}
	if areDuplicates {
		return fmt.Errorf("gallery positions are not unique for user %s", user)
	}

	return tx.Commit(ctx)
}

func (api SplitAPI) ViewSplit(ctx context.Context, galleryID persist.DBID) (db.Split, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"galleryID": {galleryID, "required"},
	}); err != nil {
		return db.Split{}, err
	}

	gallery, err := api.loaders.SplitBySplitID.Load(galleryID)
	if err != nil {
		return db.Split{}, err
	}

	gc := util.GinContextFromContext(ctx)

	if auth.GetUserAuthedFromCtx(gc) {
		userID, err := getAuthenticatedUserID(ctx)
		if err != nil {
			return db.Split{}, err
		}

		if gallery.OwnerUserID != userID {
			// only view gallery if the user hasn't already viewed it in this most recent notification period

			err = dispatchEvent(ctx, db.Event{
				ActorID:        persist.DBIDToNullStr(userID),
				ResourceTypeID: persist.ResourceTypeSplit,
				SubjectID:      galleryID,
				Action:         persist.ActionViewedSplit,
				SplitID:        galleryID,
			}, api.validator, nil)
			if err != nil {
				return db.Split{}, err
			}
		}
	} else {
		err := dispatchEvent(ctx, db.Event{
			ResourceTypeID: persist.ResourceTypeSplit,
			SubjectID:      galleryID,
			Action:         persist.ActionViewedSplit,
			SplitID:        galleryID,
			ExternalID:     persist.StrPtrToNullStr(getExternalID(ctx)),
		}, api.validator, nil)
		if err != nil {
			return db.Split{}, err
		}
	}

	return gallery, nil
}

func getExternalID(ctx context.Context) *string {
	gc := util.GinContextFromContext(ctx)
	if ip := net.ParseIP(gc.ClientIP()); ip != nil && !ip.IsPrivate() {
		hash := sha256.New()
		hash.Write([]byte(env.GetString("BACKEND_SECRET") + ip.String()))
		res, _ := hash.(encoding.BinaryMarshaler).MarshalBinary()
		externalID := base64.StdEncoding.EncodeToString(res)
		return &externalID
	}
	return nil
}

func modelToTokenLayout(u *model.CollectionLayoutInput) persist.TokenLayout {
	sectionLayout := make([]persist.CollectionSectionLayout, len(u.SectionLayout))
	for i, layout := range u.SectionLayout {
		sectionLayout[i] = persist.CollectionSectionLayout{
			Columns:    persist.NullInt32(layout.Columns),
			Whitespace: layout.Whitespace,
		}
	}
	return persist.TokenLayout{
		Sections:      persist.StandardizeCollectionSections(u.Sections),
		SectionLayout: sectionLayout,
	}
}

func modelToTokenSettings(u []*model.CollectionTokenSettingsInput) map[persist.DBID]persist.CollectionTokenSettings {
	settings := make(map[persist.DBID]persist.CollectionTokenSettings)
	for _, tokenSetting := range u {
		settings[tokenSetting.TokenID] = persist.CollectionTokenSettings{RenderLive: tokenSetting.RenderLive}
	}
	return settings
}
