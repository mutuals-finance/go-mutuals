package dataloader

import (
	"github.com/SplitFi/go-splitfi/db/gen/coredb"
)

func (*GetUserByUsernameBatch) getKeyForResult(user coredb.User) string {
	return user.Username.String
}

func (*GetTokensByIDs) getKeyForResult(token coredb.Token) string {
	return token.ID.String()
}

func (*GetTokenByChainAddressBatch) getKeyForResult(token coredb.Token) coredb.GetTokenByChainAddressBatchParams {
	return coredb.GetTokenByChainAddressBatchParams{ContractAddress: token.ContractAddress, Chain: token.Chain}
}
