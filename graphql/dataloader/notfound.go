package dataloader

import (
	"github.com/jackc/pgx/v4"
	"github.com/mutuals/go-mutuals/service/persist"
)

func (*GetPoolByIdBatch) getNotFoundError(key persist.DBID) error {
	return persist.ErrPoolNotFound{ID: key}
}

func (*GetNotificationByIDBatch) getNotFoundError(key persist.DBID) error {
	return pgx.ErrNoRows
}

func (*GetUserByIdBatch) getNotFoundError(key persist.DBID) error {
	return persist.ErrUserNotFound{UserID: key}
}

func (*GetClaimByIdBatch) getNotFoundError(key persist.DBID) error {
	// TODO: Return a specific error type, not pgx.ErrNoRows
	return pgx.ErrNoRows
}
