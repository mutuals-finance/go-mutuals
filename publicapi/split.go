package publicapi

import (
	"context"
	"crypto/sha256"
	"encoding"
	"encoding/base64"
	"net"

	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/env"
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

func (api SplitAPI) CreateSplit(ctx context.Context, name, description *string, position string) (db.Split, error) {

	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"name":        {name, "max=200"},
		"description": {description, "max=600"},
	}); err != nil {
		return db.Split{}, err
	}

	_, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return db.Split{}, err
	}

	// TODO
	//split, err := api.repos.SplitRepository.Create(ctx, db.SplitRepoCreateParams{
	//	SplitID:     persist.GenerateID(),
	//	Name:        util.FromPointer(name),
	//	Description: util.FromPointer(description),
	//	OwnerUserID: userID,
	//})
	//if err != nil {
	//	return db.Split{}, err
	//}

	return db.Split{}, nil
}

func (api SplitAPI) PublishSplit(ctx context.Context, update model.PublishSplitInput) error {

	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"splitID": {update.SplitID, "required"},
		"editID":  {update.EditID, "required"},
	}); err != nil {
		return err
	}

	err := publishEventGroup(ctx, update.EditID, persist.ActionSplitUpdated, update.Caption)
	if err != nil {
		return err
	}

	return nil
}

func (api SplitAPI) GetSplitById(ctx context.Context, splitID persist.DBID) (*db.Split, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"splitID": {splitID, "required"},
	}); err != nil {
		return nil, err
	}

	split, err := api.loaders.SplitBySplitID.Load(splitID)
	if err != nil {
		return nil, err
	}

	return &split, nil
}

func (api SplitAPI) GetViewerSplitById(ctx context.Context, splitID persist.DBID) (*db.Split, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"splitID": {splitID, "required"},
	}); err != nil {
		return nil, err
	}

	//userID, err := getAuthenticatedUserID(ctx)
	//if err != nil {
	//	return nil, persist.ErrSplitNotFound{ID: splitID}
	//}

	split, err := api.loaders.SplitBySplitID.Load(splitID)
	if err != nil {
		return nil, err
	}

	return &split, nil
}

/*
TODO add missing methods
func (api SplitAPI) UpdateSplitInfo(ctx context.Context, splitID persist.DBID, name, description *string) error {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"splitID":     {splitID, "required"},
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
		ID:          splitID,
		Name:        nullName,
		Description: nullDesc,
	})
	if err != nil {
		return err
	}
	return nil
}
func (api SplitAPI) UpdateSplitHidden(ctx context.Context, splitID persist.DBID, hidden bool) (coredb.Split, error) {
	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"splitID": {splitID, "required"},
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
