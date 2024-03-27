package multichain

import "github.com/SplitFi/go-splitfi/service/persist"

type ProviderLookup map[persist.Chain][]any

//type ProviderLookup map[persist.Chain]any

type ChainProvider struct {
	Ethereum *EthereumProvider
	Optimism *OptimismProvider
	Arbitrum *ArbitrumProvider
	//Base        *BaseProvider
	//BaseSepolia *BaseSepoliaProvider
	Polygon *PolygonProvider
}

type EthereumProvider struct {
	TokenDescriptorsFetcher
	TokenMetadataFetcher
	TokensIncrementalOwnerFetcher
	Verifier
}

type OptimismProvider struct {
	TokenDescriptorsFetcher
	TokenMetadataFetcher
	TokensIncrementalOwnerFetcher
}

type ArbitrumProvider struct {
	TokenDescriptorsFetcher
	TokenMetadataFetcher
	TokensIncrementalOwnerFetcher
}

type BaseProvider struct {
	TokenDescriptorsFetcher
	TokenMetadataFetcher
	TokensIncrementalOwnerFetcher
}

type BaseSepoliaProvider struct {
	TokenDescriptorsFetcher
	TokenMetadataFetcher
	TokensIncrementalOwnerFetcher
}

type PolygonProvider struct {
	TokenDescriptorsFetcher
	TokenMetadataFetcher
	TokensIncrementalOwnerFetcher
}
