//go:build wireinject
// +build wireinject

package multichain

import "github.com/SplitFi/go-splitfi/service/multichain/trustwallet"

// NewMultichainProvider is a wire injector that sets up a multichain provider instance
// ethClient.Client and task.Client are expensive to initialize, so they're passed as an arg.
func NewMultichainProvider(context.Context, *postgres.Repositories, *db.Queries, *ethclient.Client, *task.Client, *redis.Cache) *Provider {
	panic(wire.Build(
		wire.Value(http.DefaultClient), // HTTP client shared between providers
		wire.Struct(new(ChainProvider), "*"),
		multichainProviderInjector,
		ethInjector,
		optimismInjector,
		baseInjector,
		polygonInjector,
		arbitrumInjector,
	))
}

func multichainProviderInjector(ctx context.Context, repos *postgres.Repositories, q *db.Queries, chainProvider *ChainProvider) *Provider {
	panic(wire.Build(
		wire.Struct(new(Provider), "*"),
		newProviderLookup,
	))
}

// New chains must be added here
func newProviderLookup(p *ChainProvider) ProviderLookup {
	return ProviderLookup{
		persist.ChainETH:      p.Ethereum,
		persist.ChainOptimism: p.Optimism,
		persist.ChainArbitrum: p.Arbitrum,
		persist.ChainBase:     p.Base,
		persist.ChainPolygon:  p.Polygon,
	}
}

func ethInjector(context.Context, *http.Client, *ethclient.Client) *EthereumProvider {
	panic(wire.Build(
		wire.Value(persist.ChainETH),
		ethProviderInjector,
		ethSyncPipelineInjector,
		indexer.NewProvider,
		ethVerifierInjector,
	))
}

func ethVerifierInjector(ethClient *ethclient.Client) *eth.Verifier {
	panic(wire.Build(wire.Struct(new(eth.Verifier), "*")))
}

func ethProviderInjector(
	ctx context.Context,
	syncPipeline *wrapper.SyncPipelineWrapper,
	verifier *eth.Verifier,
) *EthereumProvider {
	panic(wire.Build(
		wire.Struct(new(EthereumProvider), "*"),
		wire.Bind(new(common.TokensByTokenIdentifiersFetcher), util.ToPointer(syncPipeline)),
		wire.Bind(new(common.AssetsIncrementalTokenFetcher), util.ToPointer(syncPipeline)),
		wire.Bind(new(common.Verifier), util.ToPointer(verifier)),
	))
}

func ethSyncPipelineInjector(
	ctx context.Context,
	httpClient *http.Client,
	chain persist.Chain,
	trustwalletProvider *trustwallet.Provider,
	covalentProvider *trustwallet.Provider,
	ethClient *ethclient.Client,
) *wrapper.SyncPipelineWrapper {
	panic(wire.Build(
		wire.Struct(new(wrapper.SyncPipelineWrapper), "*"),
		wire.Bind(new(common.TokensByTokenIdentifiersFetcher), util.ToPointer(trustwalletProvider)),
		wire.Bind(new(common.AssetsIncrementalTokenFetcher), util.ToPointer(covalentProvider)),
	))
}

func optimismInjector(context.Context, *http.Client, *ethclient.Client) *OptimismProvider {
	panic(wire.Build(
		wire.Value(persist.ChainOptimism),
		optimismProviderInjector,
		optimismSyncPipelineInjector,
	))
}

func optimismProviderInjector(
	syncPipeline *wrapper.SyncPipelineWrapper,
) *OptimismProvider {
	panic(wire.Build(
		wire.Struct(new(OptimismProvider), "*"),
		wire.Bind(new(common.TokensByTokenIdentifiersFetcher), util.ToPointer(syncPipeline)),
		wire.Bind(new(common.AssetsIncrementalTokenFetcher), util.ToPointer(syncPipeline)),
	))
}

func optimismSyncPipelineInjector(
	ctx context.Context,
	httpClient *http.Client,
	chain persist.Chain,
	indexerProvider *indexer.Provider,
	ethClient *ethclient.Client,
) *wrapper.SyncPipelineWrapper {
	panic(wire.Build(
		wire.Struct(new(wrapper.SyncPipelineWrapper), "*"),
		wire.Bind(new(common.TokensByTokenIdentifiersFetcher), util.ToPointer(indexerProvider)),
		wire.Bind(new(common.AssetsIncrementalTokenFetcher), util.ToPointer(indexerProvider)),
	))
}

func arbitrumInjector(context.Context, *http.Client, *ethclient.Client) *ArbitrumProvider {
	panic(wire.Build(
		wire.Value(persist.ChainArbitrum),
		indexer.NewProvider,
		arbitrumProviderInjector,
		arbitrumSyncPipelineInjector,
	))
}

func arbitrumProviderInjector(
	syncPipeline *wrapper.SyncPipelineWrapper,
) *ArbitrumProvider {
	panic(wire.Build(
		wire.Struct(new(ArbitrumProvider), "*"),
		wire.Bind(new(common.TokensByTokenIdentifiersFetcher), util.ToPointer(syncPipeline)),
		wire.Bind(new(common.AssetsIncrementalTokenFetcher), util.ToPointer(syncPipeline)),
	))
}

func arbitrumSyncPipelineInjector(
	ctx context.Context,
	httpClient *http.Client,
	chain persist.Chain,
	simplehashProvider *simplehash.Provider,
	ethClient *ethclient.Client,
) *wrapper.SyncPipelineWrapper {
	panic(wire.Build(
		wire.Struct(new(wrapper.SyncPipelineWrapper), "*"),
		wire.Bind(new(common.TokenIdentifierOwnerFetcher), util.ToPointer(simplehashProvider)),
		wire.Bind(new(common.TokensIncrementalOwnerFetcher), util.ToPointer(simplehashProvider)),
		wire.Bind(new(common.TokensIncrementalContractFetcher), util.ToPointer(simplehashProvider)),
		wire.Bind(new(common.TokenMetadataBatcher), util.ToPointer(simplehashProvider)),
		wire.Bind(new(common.TokensByTokenIdentifiersFetcher), util.ToPointer(simplehashProvider)),
		wire.Bind(new(common.TokensByContractWalletFetcher), util.ToPointer(simplehashProvider)),
		customMetadataHandlersInjector,
	))
}

func baseInjector(context.Context, *http.Client, *ethclient.Client) *BaseProvider {
	panic(wire.Build(
		wire.Value(persist.ChainBase),
		simplehash.NewProvider,
		baseProvidersInjector,
		baseSyncPipelineInjector,
	))
}

func baseProvidersInjector(
	syncPipeline *wrapper.SyncPipelineWrapper,
	simplehashProvider *simplehash.Provider,
	ethClient *ethclient.Client,
) *BaseProvider {
	panic(wire.Build(
		wire.Struct(new(BaseProvider), "*"),
		wire.Bind(new(common.TokensByTokenIdentifiersFetcher), util.ToPointer(syncPipeline)),
		wire.Bind(new(common.AssetsIncrementalTokenFetcher), util.ToPointer(syncPipeline)),
	))
}

func baseSyncPipelineInjector(
	ctx context.Context,
	httpClient *http.Client,
	chain persist.Chain,
	simplehashProvider *simplehash.Provider,
	ethClient *ethclient.Client,
) *wrapper.SyncPipelineWrapper {
	panic(wire.Build(
		wire.Struct(new(wrapper.SyncPipelineWrapper), "*"),
		wire.Bind(new(common.TokenIdentifierOwnerFetcher), util.ToPointer(simplehashProvider)),
		wire.Bind(new(common.TokensIncrementalOwnerFetcher), util.ToPointer(simplehashProvider)),
		wire.Bind(new(common.TokensIncrementalContractFetcher), util.ToPointer(simplehashProvider)),
		wire.Bind(new(common.TokenMetadataBatcher), util.ToPointer(simplehashProvider)),
		wire.Bind(new(common.TokensByTokenIdentifiersFetcher), util.ToPointer(simplehashProvider)),
		wire.Bind(new(common.TokensByContractWalletFetcher), util.ToPointer(simplehashProvider)),
		customMetadataHandlersInjector,
	))
}

func polygonInjector(context.Context, *http.Client, *ethclient.Client) *PolygonProvider {
	panic(wire.Build(
		wire.Value(persist.ChainPolygon),
		simplehash.NewProvider,
		polygonProvidersInjector,
		polygonSyncPipelineInjector,
	))
}

func polygonProvidersInjector(
	syncPipeline *wrapper.SyncPipelineWrapper,
	simplehashProvider *simplehash.Provider,
	ethClient *ethclient.Client,
) *PolygonProvider {
	panic(wire.Build(
		wire.Struct(new(PolygonProvider), "*"),
		wire.Bind(new(common.TokensByTokenIdentifiersFetcher), util.ToPointer(syncPipeline)),
		wire.Bind(new(common.AssetsIncrementalTokenFetcher), util.ToPointer(syncPipeline)),
	))
}

func polygonSyncPipelineInjector(
	ctx context.Context,
	httpClient *http.Client,
	chain persist.Chain,
	simplehashProvider *simplehash.Provider,
	ethClient *ethclient.Client,
) *wrapper.SyncPipelineWrapper {
	panic(wire.Build(
		wire.Struct(new(wrapper.SyncPipelineWrapper), "*"),
		wire.Bind(new(common.TokenIdentifierOwnerFetcher), util.ToPointer(simplehashProvider)),
		wire.Bind(new(common.TokensIncrementalOwnerFetcher), util.ToPointer(simplehashProvider)),
		wire.Bind(new(common.TokensIncrementalContractFetcher), util.ToPointer(simplehashProvider)),
		wire.Bind(new(common.TokenMetadataBatcher), util.ToPointer(simplehashProvider)),
		wire.Bind(new(common.TokensByContractWalletFetcher), util.ToPointer(simplehashProvider)),
		wire.Bind(new(common.TokensByTokenIdentifiersFetcher), util.ToPointer(simplehashProvider)),
		customMetadataHandlersInjector,
	))
}
