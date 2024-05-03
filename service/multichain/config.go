package multichain

import (
	"github.com/SplitFi/go-splitfi/service/multichain/common"
	"github.com/SplitFi/go-splitfi/service/persist"
)

//type ProviderLookup map[persist.Chain][]any

type ProviderLookup map[persist.Chain]any

type ChainProvider struct {
	Ethereum *EthereumProvider
	Optimism *OptimismProvider
	Arbitrum *ArbitrumProvider
	Base     *BaseProvider
	Polygon  *PolygonProvider
}

type EthereumProvider struct {
	common.AssetsIncrementalTokenFetcher
	common.TokensByTokenIdentifiersFetcher
	common.Verifier
}

type OptimismProvider struct {
	common.AssetsIncrementalTokenFetcher
	common.TokensByTokenIdentifiersFetcher
}

type ArbitrumProvider struct {
	common.AssetsIncrementalTokenFetcher
	common.TokensByTokenIdentifiersFetcher
}

type BaseProvider struct {
	common.AssetsIncrementalTokenFetcher
	common.TokensByTokenIdentifiersFetcher
}

type PolygonProvider struct {
	common.AssetsIncrementalTokenFetcher
	common.TokensByTokenIdentifiersFetcher
}
