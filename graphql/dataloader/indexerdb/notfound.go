package indexerdb

import (
	"github.com/jackc/pgx/v4"
	"github.com/mutuals/go-mutuals/service/persist"
)

func (*GetAccountByIdBatch) getNotFoundError(key persist.DBID) error {
	return persist.ErrAccountNotFound{ID: key}
}

func (*GetPoolContractByIdBatch) getNotFoundError(key persist.DBID) error {
	return persist.ErrPoolNotFound{ID: key}
}

func (*GetTokenByIdBatch) getNotFoundError(key persist.DBID) error {
	return pgx.ErrNoRows
}

func (*GetAccountByAddressBatch) getNotFoundError(key string) error {
	// TODO: Return a specific error type, not pgx.ErrNoRows
	return pgx.ErrNoRows
}
