package publicapi

import (
	"context"
	"encoding/json"

	"github.com/SplitFi/go-splitfi/env"
	"github.com/SplitFi/go-splitfi/service/persist/postgres"

	"cloud.google.com/go/storage"
	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/graphql/dataloader"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/go-playground/validator/v10"
)

type MiscAPI struct {
	repos         *postgres.Repositories
	queries       *db.Queries
	loaders       *dataloader.Loaders
	validator     *validator.Validate
	ethClient     *ethclient.Client
	storageClient *storage.Client
}

func (api MiscAPI) GetGeneralAllowlist(ctx context.Context) ([]persist.Address, error) {
	// Nothing to validate

	bucket := env.GetString("SNAPSHOT_BUCKET")
	logger.For(ctx).Infof("Proxying snapshot from bucket %s", bucket)

	obj := api.storageClient.Bucket(env.GetString("SNAPSHOT_BUCKET")).Object("snapshot.json")

	r, err := obj.NewReader(ctx)
	if err != nil {
		return nil, err
	}

	var addresses []persist.Address
	err = json.NewDecoder(r).Decode(&addresses)
	if err != nil {
		return nil, err
	}

	err = r.Close()
	if err != nil {
		return nil, err
	}

	return addresses, nil
}
