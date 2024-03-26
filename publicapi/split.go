package publicapi

import (
	"context"
	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/graphql/dataloader"
	"github.com/SplitFi/go-splitfi/graphql/model"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/persist/postgres"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/SplitFi/go-splitfi/validate"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-playground/validator/v10"
)

type SplitAPI struct {
	repos     *postgres.Repositories
	queries   *db.Queries
	loaders   *dataloader.Loaders
	validator *validator.Validate
	ethClient *ethclient.Client
}

func (api SplitAPI) CreateSplit(ctx context.Context, name, description, logoUrl *string) (db.Split, error) {

	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"name":        {name, "max=200"},
		"description": {description, "max=600"},
		"logoUrl":     {logoUrl, "max=200"},
	}); err != nil {
		return db.Split{}, err
	}

	split, err := api.queries.CreateSplit(ctx, db.CreateSplitParams{
		SplitID:     persist.GenerateID(),
		Name:        util.FromPointer(name),
		Description: util.FromPointer(description),
		LogoUrl:     util.ToNullString(util.FromPointer(logoUrl), false),
	})
	if err != nil {
		return db.Split{}, err
	}

	return split, nil
}

func (api SplitAPI) PublishSplit(ctx context.Context, update model.PublishSplitInput) error {

	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"splitID": {update.SplitID, "required"},
		"editID":  {update.EditID, "required"},
	}); err != nil {
		return err
	}

	//err := publishEventGroup(ctx, update.EditID, persist.ActionSplitUpdated, update.Caption)
	//if err != nil {
	//	return err
	//}

	return nil
}

func (api SplitAPI) GetViewerSplitById(ctx context.Context, splitID persist.DBID) (*db.Split, error) {

	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"splitID": validate.WithTag(splitID, "required"),
	}); err != nil {
		return nil, err
	}

	userID, err := getAuthenticatedUserID(ctx)

	if err != nil {
		return nil, persist.ErrSplitNotFound{ID: splitID}
	}

	split, err := api.queries.GetSplitByRecipientUserID(ctx, db.GetSplitByRecipientUserIDParams{
		UserID:  userID,
		SplitID: splitID,
	})
	if err != nil {
		return nil, err
	}

	return &split, nil
}

func (api SplitAPI) GetSplitById(ctx context.Context, splitID persist.DBID) (*db.Split, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"splitID": {splitID, "required"},
	}); err != nil {
		return nil, err
	}

	split, err := api.loaders.GetSplitByIdBatch.Load(splitID)
	if err != nil {
		return nil, err
	}

	return &split, nil
}

func (api SplitAPI) GetSplitsByIds(ctx context.Context, splitIDs []persist.DBID) ([]*db.Split, []error) {
	splitThunk := func(splitID persist.DBID) func() (db.Split, error) {
		if err := validate.ValidateFields(api.validator, validate.ValidationMap{
			"splitIDs": {splitID, "required"},
		}); err != nil {
			return func() (db.Split, error) { return db.Split{}, err }
		}

		return api.loaders.GetSplitByIdBatch.LoadThunk(splitID)
	}

	// A "thunk" will add this request to a batch, and then return a function that will block to fetch
	// data when called. By creating all of the thunks first (without invoking the functions they return),
	// we're setting up a batch that will eventually fetch all of these requests at the same time when
	// their functions are invoked. "LoadAll" would accomplish something similar, but wouldn't let us
	// validate each splitID parameter first.
	thunks := make([]func() (db.Split, error), len(splitIDs))

	for i, splitID := range splitIDs {
		thunks[i] = splitThunk(splitID)
	}

	splits := make([]*db.Split, len(splitIDs))
	errors := make([]error, len(splitIDs))

	for i := range splitIDs {
		split, err := thunks[i]()
		if err == nil {
			splits[i] = &split
		} else {
			errors[i] = err
		}
	}

	return splits, errors
}

func (api SplitAPI) GetSplitByChainAddress(ctx context.Context, chainAddress persist.ChainAddress) (*db.Split, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"chainAddress": {chainAddress, "required"},
	}); err != nil {
		return nil, err
	}

	split, err := api.loaders.GetSplitByChainAddressBatch.Load(db.GetSplitByChainAddressBatchParams{
		Address: chainAddress.Address(),
		Chain:   chainAddress.Chain(),
	})
	if err != nil {
		return nil, err
	}

	return &split, nil
}

func (api SplitAPI) GetSplitsByRecipientAddressBatch(ctx context.Context, recipientAddress persist.Address) ([]db.Split, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"recipientAddress": validate.WithTag(recipientAddress, "required"),
	}); err != nil {
		return nil, err
	}

	splits, err := api.loaders.GetSplitsByRecipientAddressBatch.Load(recipientAddress)
	if err != nil {
		return nil, err
	}

	return splits, nil
}

func (api SplitAPI) UpdateSplitInfo(ctx context.Context, splitID persist.DBID, name, description, logoUrl *string) error {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"splitID":     {splitID, "required"},
		"name":        {name, "max=200"},
		"description": {description, "max=600"},
		"logoUrl":     {logoUrl, "max=200"},
	}); err != nil {
		return err
	}

	var nullName, nullDesc, nullLogoUrl string
	var nameSet, descSet, logoUrlSet bool

	if name != nil {
		nullName = *name
		nameSet = true
	}
	if description != nil {
		nullDesc = *description
		descSet = true
	}
	if logoUrl != nil {
		nullLogoUrl = *logoUrl
		logoUrlSet = true
	}

	err := api.queries.UpdateSplitInfo(ctx, db.UpdateSplitInfoParams{
		ID:             splitID,
		Name:           nullName,
		Description:    nullDesc,
		LogoUrl:        util.ToNullString(nullLogoUrl, false),
		NameSet:        nameSet,
		DescriptionSet: descSet,
		LogoUrlSet:     logoUrlSet,
	})

	if err != nil {
		return err
	}

	return nil
}

/*
	func (api SplitAPI) UpdateSplitHidden(ctx context.Context, splitID persist.DBID, hidden bool) (db.Split, error) {
		// Validate
		if err := validate.ValidateFields(api.validator, validate.ValidationMap{
			"splitID": validate.WithTag(splitID, "required"),
		}); err != nil {
			return db.Split{}, err
		}

		split, err := api.queries.UpdateSplitHidden(ctx, db.UpdateSplitHiddenParams{
			ID:     splitID,
			Hidden: hidden,
		})
		if err != nil {
			return db.Split{}, err
		}

		return split, nil
	}
*/

func (api SplitAPI) UpdateSplitShares(ctx context.Context, shares []*model.SplitShareInput) error {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"shares": validate.WithTag(shares, "required,min=1"),
	}); err != nil {
		return err
	}

	sids := make([]string, len(shares))
	adds := make([]string, len(shares))
	owns := make([]int32, len(shares))

	for i, share := range shares {
		sids[i] = share.SplitID.String()
		adds[i] = share.RecipientAddress.String()
		owns[i] = int32(share.Ownership)
	}

	err := api.queries.UpdateSplitShares(ctx, db.UpdateSplitSharesParams{
		SplitIds:           sids,
		RecipientAddresses: adds,
		Ownerships:         owns,
	})

	if err != nil {
		return err
	}

	return nil
}
