package wrapper

import (
	"context"
	"github.com/SplitFi/go-splitfi/service/multichain/common"
	"github.com/SplitFi/go-splitfi/service/persist"
)

// SyncPipelineWrapper makes a best effort to fetch tokens requested by a sync.
// Specifically, SyncPipelineWrapper searches every applicable provider to find tokens and
// fills missing token fields with data from a supplemental provider.
type SyncPipelineWrapper struct {
	Chain                           persist.Chain
	AssetsIncrementalTokenFetcher   common.AssetsIncrementalTokenFetcher
	TokensByTokenIdentifiersFetcher common.TokensByTokenIdentifiersFetcher
}

func (w SyncPipelineWrapper) GetTokensByTokenIdentifiers(ctx context.Context, chain persist.Chain, tis []common.ChainAgnosticIdentifiers) ([]common.ChainAgnosticToken, error) {
	t, err := w.TokensByTokenIdentifiersFetcher.GetTokensByTokenIdentifiers(ctx, chain, tis)
	return t, err
}

func (w SyncPipelineWrapper) GetAssetsIncrementallyByTokenIdentifiers(ctx context.Context, owner persist.Address, tids []persist.TokenChainAddress, maxLimit int) (<-chan common.ChainAgnosticAssetsAndTokens, <-chan error) {
	recCh, errCh := w.AssetsIncrementalTokenFetcher.GetAssetsIncrementallyByTokenIdentifiers(ctx, owner, tids, maxLimit)
	return recCh, errCh
}
