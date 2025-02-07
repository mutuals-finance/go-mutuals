package dataloader

import (
	"github.com/mutuals/go-mutuals/db/gen/coredb"
)

func (*GetUserByUsernameBatch) getKeyForResult(user coredb.User) string {
	return user.Username.String
}
