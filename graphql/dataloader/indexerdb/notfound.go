package indexerdb

import "github.com/mutuals/go-mutuals/service/persist"

func (*GetAccountByIDBatch) getNotFoundError(key persist.DBID) error {
	return persist.ErrAccountNotFound{ID: key}
}
