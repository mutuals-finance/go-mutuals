package dataloader

import (
	"github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/jackc/pgx/v4"
)

func (*GetSplitByIdBatch) getNotFoundError(key persist.DBID) error {
	return persist.ErrSplitNotFound{ID: key}
}

func (*GetSplitByChainAddressBatch) getNotFoundError(key coredb.GetSplitByChainAddressBatchParams) error {
	return persist.ErrSplitNotFoundByAddress{Address: key.Address}
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

func (*GetUserByChainAddressBatch) getNotFoundError(key coredb.GetUserByChainAddressBatchParams) error {
	return persist.ErrAddressNotOwnedByUser{ChainAddress: persist.NewChainAddress(key.Address, persist.Chain(key.Chain))}
}

func (*GetWalletByIDBatch) getNotFoundError(key persist.DBID) error {
	return pgx.ErrNoRows
}

func (*GetWalletByChainAddressBatch) getNotFoundError(key coredb.GetWalletByChainAddressBatchParams) error {
	return pgx.ErrNoRows
}
