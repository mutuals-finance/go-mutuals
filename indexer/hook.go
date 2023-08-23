package indexer

import (
	"context"
	"net/http"

	"github.com/SplitFi/go-splitfi/service/persist"
)

type DBHook[T any] func(ctx context.Context, it []T) error

func newAssetHooks(repo persist.AssetRepository, httpClient *http.Client) []DBHook[persist.Asset] {
	/*	return []DBHook[persist.Asset]{
			func(ctx context.Context, it []persist.Contract) error {
				return repo.BulUpset(ctx, fillContractFields(ctx, it, repo, httpClient))
			},
		}
	*/
	return []DBHook[persist.Asset]{}
}

func newTokenHooks() []DBHook[persist.Token] {
	return []DBHook[persist.Token]{}
}
