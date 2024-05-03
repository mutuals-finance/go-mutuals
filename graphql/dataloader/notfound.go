package dataloader

import (
	"github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/jackc/pgx/v4"
)

//func (*GetCollectionByIdBatch) getNotFoundError(key persist.DBID) error {
//	return persist.ErrCollectionNotFoundByID{ID: key}
//}

func (*GetTokenByChainAddressBatch) getNotFoundError(key coredb.GetTokenByChainAddressBatchParams) error {
	return persist.ErrTokenNotFoundByTokenChainAddress{Token: persist.NewTokenChainAddress(key.ContractAddress, key.Chain)}
}

//func (*GetEventByIdBatch) getNotFoundError(key persist.DBID) error {
//	return persist.ErrFeedEventNotFoundByID{ID: key}
//}

func (*GetSplitByIdBatch) getNotFoundError(key persist.DBID) error {
	return persist.ErrSplitNotFound{ID: key}
}

func (*GetSplitByChainAddressBatch) getNotFoundError(key coredb.GetSplitByChainAddressBatchParams) error {
	return persist.ErrSplitNotFoundByAddress{Address: key.Address}
}

func (*GetNotificationByIDBatch) getNotFoundError(key persist.DBID) error {
	return pgx.ErrNoRows
}

func (*GetAssetByIdBatch) getNotFoundError(key persist.DBID) error {
	return persist.ErrAssetNotFoundByID{ID: key}
}

//func (*GetProfileImageByIdBatch) getNotFoundError(key coredb.GetProfileImageByIdBatchParams) error {
//	return persist.ErrProfileImageNotFound{Err: pgx.ErrNoRows, ProfileImageID: key.ID}
//}

func (*GetTokenByIdBatch) getNotFoundError(key persist.DBID) error {
	return persist.ErrTokenNotFoundByID{ID: key}
}

//func (*GetUserByAddressAndL1Batch) getNotFoundError(key coredb.GetUserByAddressAndL1BatchParams) error {
//	return persist.ErrUserNotFound{L1ChainAddress: persist.NewL1ChainAddress(key.Address, persist.Chain(key.L1Chain))}
//}

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

func (*GetTokensByIDs) getNotFoundError(key string) error {
	return pgx.ErrNoRows
}
