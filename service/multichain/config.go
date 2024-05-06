package multichain

import (
	"github.com/SplitFi/go-splitfi/service/multichain/common"
	"github.com/SplitFi/go-splitfi/service/persist"
)

type ProviderLookup map[persist.Chain]any

type ChainProvider struct {
	Ethereum *EthereumProvider
	Optimism *OptimismProvider
	Arbitrum *ArbitrumProvider
	Base     *BaseProvider
	Polygon  *PolygonProvider
}

type EthereumProvider struct {
	common.Verifier
}

type OptimismProvider struct {
}

type ArbitrumProvider struct {
}

type BaseProvider struct {
}

type PolygonProvider struct {
}
