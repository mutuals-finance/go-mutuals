package dataloader

import (
	"github.com/jackc/pgx/v4"
	"github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/service/persist"
)

func (*GetPoolByIdBatch) getNotFoundError(key persist.DBID) error {
	return persist.ErrPoolNotFound{ID: key}
}

func (*GetAllocationByIdBatch) getNotFoundError(key persist.DBID) error {
	return persist.ErrAllocationNotFound{ID: key}
}

func (*GetAllocationAggregationByIdBatch) getNotFoundError(key persist.DBID) error {
	return persist.ErrAllocationAggregationNotFound{ID: key}
}

func (*GetPoolByChainAddressBatch) getNotFoundError(key coredb.GetPoolByChainAddressBatchParams) error {
	return persist.ErrPoolNotFoundByAddress{Address: key.Address}
}

func (*GetNotificationByIDBatch) getNotFoundError(key persist.DBID) error {
	return pgx.ErrNoRows
}

func (*GetUserByIdBatch) getNotFoundError(key persist.DBID) error {
	return persist.ErrUserNotFound{UserID: key}
}

func (*GetUserByUsernameBatch) getNotFoundError(key string) error {
	return persist.ErrUserNotFound{Username: key}
}

func (*GetUserByAddressAndL1Batch) getNotFoundError(key coredb.GetUserByAddressAndL1BatchParams) error {
	return persist.ErrUserNotFound{L1ChainAddress: persist.NewL1ChainAddress(key.Address, persist.Chain(key.L1Chain))}
}

func (*GetWalletByIDBatch) getNotFoundError(key persist.DBID) error {
	return pgx.ErrNoRows
}
