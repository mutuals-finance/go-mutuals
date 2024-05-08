package dataloader

import (
	"github.com/SplitFi/go-splitfi/db/gen/coredb"
)

func (*GetUserByUsernameBatch) getKeyForResult(user coredb.User) string {
	return user.Username.String
}
