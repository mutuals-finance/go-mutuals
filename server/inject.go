//go:build wireinject
// +build wireinject

package server

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/google/wire"
	"github.com/jackc/pgx/v4/pgxpool"

	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/service/multichain"
	"github.com/SplitFi/go-splitfi/service/multichain/alchemy"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/persist/postgres"
	"github.com/SplitFi/go-splitfi/service/redis"
	"github.com/SplitFi/go-splitfi/service/rpc"
	"github.com/SplitFi/go-splitfi/util"
)

// envInit is a type returned after setting up the environment
// Adding envInit as a dependency to a provider will ensure that the environment is set up prior
// to calling the provider
type envInit struct{}

type ethProviderList []any
type optimismProviderList []any
type polygonProviderList []any
type arbitrumProviderList []any
type tokenMetadataCache redis.Cache

// NewMultichainProvider is a wire injector that sets up a multichain provider instance
func NewMultichainProvider(ctx context.Context, envFunc func()) (*multichain.Provider, func()) {
	wire.Build(
		setEnv,
		wire.Value(&http.Client{Timeout: 0}), // HTTP client shared between providers
		postgres.NewRepositories,
		dbConnSet,
		wire.Struct(new(multichain.ChainProvider), "*"),
		// Add additional chains here
		multichainProviderInjector,
		ethInjector,
		optimismInjector,
		//baseInjector,
		//baseSepoliaInjector,
		polygonInjector,
		arbitrumInjector,
	)
	return nil, nil
}

// dbConnSet is a wire provider set for initializing a postgres connection
var dbConnSet = wire.NewSet(
	newPqClient,
	newPgxClient,
	newQueries,
)

func setEnv(f func()) envInit {
	f()
	return envInit{}
}

func newPqClient(e envInit) (*sql.DB, func()) {
	pq := postgres.MustCreateClient()
	return pq, func() { pq.Close() }
}

func newPgxClient(envInit) (*pgxpool.Pool, func()) {
	pgx := postgres.NewPgxClient()
	return pgx, func() { pgx.Close() }
}

func newQueries(p *pgxpool.Pool) *db.Queries {
	return db.New(p)
}

func multichainProviderInjector(context.Context, *postgres.Repositories, *db.Queries, *redis.Cache, *multichain.ChainProvider) *multichain.Provider {
	panic(wire.Build(
		wire.Struct(new(multichain.Provider), "*"),
		// submitTokenBatchInjector,
		newProviderLookup,
	))
}

// New chains must be added here
func newProviderLookup(p *multichain.ChainProvider) multichain.ProviderLookup {
	return multichain.ProviderLookup{
		persist.ChainETH:      p.Ethereum,
		persist.ChainOptimism: p.Optimism,
		persist.ChainArbitrum: p.Arbitrum,
		//persist.ChainBase:        p.Base,
		//persist.ChainBaseSepolia: p.BaseSepolia,
		persist.ChainPolygon: p.Polygon,
	}
}

// This is a workaround for wire because wire wouldn't know which value to inject for args of the same type
type (
	tokensContractFetcherA   multichain.TokensContractFetcher
	tokensContractFetcherB   multichain.TokensContractFetcher
	tokenMetadataFetcherA    multichain.TokenMetadataFetcher
	tokenMetadataFetcherB    multichain.TokenMetadataFetcher
	tokenDescriptorsFetcherA multichain.TokenDescriptorsFetcher
	tokenDescriptorsFetcherB multichain.TokenDescriptorsFetcher
	//tokenIdentifierOwnerFetcherA      multichain.TokenIdentifierOwnerFetcher
	//tokenIdentifierOwnerFetcherB      multichain.TokenIdentifierOwnerFetcher
	tokensIncrementalOwnerFetcherA multichain.TokensIncrementalOwnerFetcher
	tokensIncrementalOwnerFetcherB multichain.TokensIncrementalOwnerFetcher
	//tokensIncrementalContractFetcherA multichain.TokensIncrementalContractFetcher
	//tokensIncrementalContractFetcherB multichain.TokensIncrementalContractFetcher
	//tokensByTokenIdentifiersFetcherA  multichain.TokensByTokenIdentifiersFetcher
	//tokensByTokenIdentifiersFetcherB  multichain.TokensByTokenIdentifiersFetcher
)

func multiContractFetcherProvider(a tokensContractFetcherA, b tokensContractFetcherB) multichain.TokensContractFetcher {
	return wrapper.NewMultiProviderWrapper(wrapper.MultiProviderWapperOptions.WithContractFetchers(a, b))
}

func multiTokenMetadataFetcherProvider(a tokenMetadataFetcherA, b tokenMetadataFetcherB) multichain.TokenMetadataFetcher {
	return wrapper.NewMultiProviderWrapper(wrapper.MultiProviderWapperOptions.WithTokenMetadataFetchers(a, b))
}

func multiTokenDescriptorsFetcherProvider(a tokenDescriptorsFetcherA, b tokenDescriptorsFetcherB) multichain.TokenDescriptorsFetcher {
	return wrapper.NewMultiProviderWrapper(wrapper.MultiProviderWapperOptions.WithTokenDescriptorsFetchers(a, b))
}

func multiTokenIdentifierOwnerFetcherProvider(a tokenIdentifierOwnerFetcherA, b tokenIdentifierOwnerFetcherB) multichain.TokenIdentifierOwnerFetcher {
	return wrapper.NewMultiProviderWrapper(wrapper.MultiProviderWapperOptions.WithTokenIdentifierOwnerFetchers(a, b))
}

func multiTokensIncrementalOwnerFetcherProvider(a tokensIncrementalOwnerFetcherA, b tokensIncrementalOwnerFetcherB) multichain.TokensIncrementalOwnerFetcher {
	return wrapper.NewMultiProviderWrapper(wrapper.MultiProviderWapperOptions.WithTokensIncrementalOwnerFetchers(a, b))
}

func multiTokensIncrementalContractFetcherProvider(a tokensIncrementalContractFetcherA, b tokensIncrementalContractFetcherB) multichain.TokensIncrementalContractFetcher {
	return wrapper.NewMultiProviderWrapper(wrapper.MultiProviderWapperOptions.WithTokensIncrementalContractFetchers(a, b))
}

func multiTokenByTokenIdentifiersFetcherProvider(a tokensByTokenIdentifiersFetcherA, b tokensByTokenIdentifiersFetcherB) multichain.TokensByTokenIdentifiersFetcher {
	return wrapper.NewMultiProviderWrapper(wrapper.MultiProviderWapperOptions.WithTokenByTokenIdentifiersFetchers(a, b))
}

func customMetadataHandlersInjector(alchemyProvider *alchemy.Provider) *multichain.CustomMetadataHandlers {
	panic(wire.Build(
		multichain.NewCustomMetadataHandlers,
		wire.Bind(new(multichain.TokenMetadataFetcher), util.ToPointer(alchemyProvider)),
		rpc.NewEthClient,
		ipfs.NewShell,
		arweave.NewClient,
	))
}

// -----------------------------------------

func ethInjector(
	envInit,
	context.Context,
	*http.Client,
	// *openseaLimiter,
	// *reservoirLimiter
) (*multichain.EthereumProvider, func()) {
	panic(wire.Build(
		rpc.NewEthClient,
		wire.Value(persist.ChainETH),
		indexer.NewProvider,
		alchemy.NewProvider,
		//openseaProviderInjector,
		ethProviderInjector,
		ethSyncPipelineInjector,
		ethTokensContractFetcherInjector,
		ethTokenMetadataFetcherInjector,
		ethTokenDescriptorsFetcherInjector,
	))
}

func ethProviderInjector(
	ctx context.Context,
	indexerProvider *indexer.Provider,
	syncPipeline *wrapper.SyncPipelineWrapper,
	tokensContractFetcher multichain.TokensContractFetcher,
	tokenDescriptorsFetcher multichain.TokenDescriptorsFetcher,
	tokenMetadataFetcher multichain.TokenMetadataFetcher,
) *multichain.EthereumProvider {
	panic(wire.Build(
		wire.Struct(new(multichain.EthereumProvider), "*"),
		wire.Bind(new(multichain.Verifier), util.ToPointer(indexerProvider)),
		wire.Bind(new(multichain.ContractRefresher), util.ToPointer(indexerProvider)),
		wire.Bind(new(multichain.ContractsOwnerFetcher), util.ToPointer(indexerProvider)),
		wire.Bind(new(multichain.TokenIdentifierOwnerFetcher), util.ToPointer(syncPipeline)),
		wire.Bind(new(multichain.TokensIncrementalOwnerFetcher), util.ToPointer(syncPipeline)),
		wire.Bind(new(multichain.TokensIncrementalContractFetcher), util.ToPointer(syncPipeline)),
		wire.Bind(new(multichain.TokenMetadataBatcher), util.ToPointer(syncPipeline)),
		wire.Bind(new(multichain.TokensByTokenIdentifiersFetcher), util.ToPointer(syncPipeline)),
	))
}

func ethSyncPipelineInjector(
	ctx context.Context,
	httpClient *http.Client,
	chain persist.Chain,
	openseaProvider *opensea.Provider,
	alchemyProvider *alchemy.Provider,
	l *reservoirLimiter,
) (*wrapper.SyncPipelineWrapper, func()) {
	panic(wire.Build(
		wire.Struct(new(wrapper.SyncPipelineWrapper), "*"),
		wire.Bind(new(multichain.TokenMetadataBatcher), util.ToPointer(alchemyProvider)),
		wire.Bind(new(retry.Limiter), util.ToPointer(l)),
		ethTokenIdentifierOwnerFetcherInjector,
		ethTokensIncrementalOwnerFetcherInjector,
		ethTokensContractFetcherInjector,
		ethTokenByTokenIdentifiersFetcherInjector,
		wrapper.NewFillInWrapper,
		customMetadataHandlersInjector,
	))
}

//func ethTokensContractFetcherInjector(openseaProvider *opensea.Provider, alchemyProvider *alchemy.Provider) multichain.TokensIncrementalContractFetcher {
//	panic(wire.Build(
//		multiTokensIncrementalContractFetcherProvider,
//		wire.Bind(new(tokensIncrementalContractFetcherA), util.ToPointer(alchemyProvider)),
//		wire.Bind(new(tokensIncrementalContractFetcherB), util.ToPointer(openseaProvider)),
//	))
//}

//func ethTokenIdentifierOwnerFetcherInjector(openseaProvider *opensea.Provider, alchemyProvider *alchemy.Provider) multichain.TokenIdentifierOwnerFetcher {
//	panic(wire.Build(
//		multiTokenIdentifierOwnerFetcherProvider,
//		wire.Bind(new(tokenIdentifierOwnerFetcherA), util.ToPointer(alchemyProvider)),
//		wire.Bind(new(tokenIdentifierOwnerFetcherB), util.ToPointer(openseaProvider)),
//	))
//}

func ethTokensIncrementalOwnerFetcherInjector(openseaProvider *opensea.Provider, alchemyProvider *alchemy.Provider) multichain.TokensIncrementalOwnerFetcher {
	panic(wire.Build(
		multiTokensIncrementalOwnerFetcherProvider,
		wire.Bind(new(tokensIncrementalOwnerFetcherA), util.ToPointer(alchemyProvider)),
		wire.Bind(new(tokensIncrementalOwnerFetcherB), util.ToPointer(openseaProvider)),
	))
}

func ethTokensContractFetcherInjector(openseaProvider *opensea.Provider, alchemyProvider *alchemy.Provider) multichain.TokensContractFetcher {
	panic(wire.Build(
		multiContractFetcherProvider,
		wire.Bind(new(tokensContractFetcherA), util.ToPointer(alchemyProvider)),
		wire.Bind(new(tokensContractFetcherB), util.ToPointer(openseaProvider)),
	))
}

func ethTokenMetadataFetcherInjector(openseaProvider *opensea.Provider, alchemyProvider *alchemy.Provider) multichain.TokenMetadataFetcher {
	panic(wire.Build(
		multiTokenMetadataFetcherProvider,
		wire.Bind(new(tokenMetadataFetcherA), util.ToPointer(alchemyProvider)),
		wire.Bind(new(tokenMetadataFetcherB), util.ToPointer(openseaProvider)),
	))
}

func ethTokenDescriptorsFetcherInjector(openseaProvider *opensea.Provider, alchemyProvider *alchemy.Provider) multichain.TokenDescriptorsFetcher {
	panic(wire.Build(
		multiTokenDescriptorsFetcherProvider,
		wire.Bind(new(tokenDescriptorsFetcherA), util.ToPointer(alchemyProvider)),
		wire.Bind(new(tokenDescriptorsFetcherB), util.ToPointer(openseaProvider)),
	))
}

//func ethTokenByTokenIdentifiersFetcherInjector(openseaProvider *opensea.Provider, alchemyProvider *alchemy.Provider) multichain.TokensByTokenIdentifiersFetcher {
//	panic(wire.Build(
//		multiTokenByTokenIdentifiersFetcherProvider,
//		wire.Bind(new(tokensByTokenIdentifiersFetcherA), util.ToPointer(alchemyProvider)),
//		wire.Bind(new(tokensByTokenIdentifiersFetcherB), util.ToPointer(openseaProvider)),
//	))
//}

// -----------------------------------------

/*// ethProviderSet is a wire injector that creates the set of Ethereum providers
func ethProviderSet(envInit, *task.Client, *http.Client, *tokenMetadataCache) ethProviderList {
	wire.Build(
		rpc.NewEthClient,
		ethProvidersConfig,
		// Add providers for Ethereum here
		eth.NewProvider,
		ethFallbackProvider,
	)
	return ethProviderList{}
}

// ethProvidersConfig is a wire injector that binds multichain interfaces to their concrete Ethereum implementations
func ethProvidersConfig(indexerProvider *eth.Provider, fallbackProvider multichain.SyncFailureFallbackProvider) ethProviderList {
	wire.Build(
		wire.Bind(new(multichain.NameResolver), util.ToPointer(indexerProvider)),
		wire.Bind(new(multichain.Verifier), util.ToPointer(indexerProvider)),
		wire.Bind(new(multichain.TokensOwnerFetcher), util.ToPointer(fallbackProvider)),
		wire.Bind(new(multichain.TokensContractFetcher), util.ToPointer(fallbackProvider)),
		wire.Bind(new(multichain.TokensIncrementalOwnerFetcher), util.ToPointer(fallbackProvider)),
		wire.Bind(new(multichain.TokenMetadataFetcher), util.ToPointer(indexerProvider)),
		wire.Bind(new(multichain.TokenDescriptorsFetcher), util.ToPointer(indexerProvider)),
		ethRequirements,
	)
	return nil
}

// ethRequirements is the set of provider interfaces required for Ethereum
func ethRequirements(
	nr multichain.NameResolver,
	v multichain.Verifier,
	tof multichain.TokensOwnerFetcher,
	toc multichain.TokensContractFetcher,
	tiof multichain.TokensIncrementalOwnerFetcher,
	tmf multichain.TokenMetadataFetcher,
	tdf multichain.TokenDescriptorsFetcher,
) ethProviderList {
	return ethProviderList{nr, v, tof, toc, tiof, tmf, tdf}
}
*/

// -----------------------------------------------

func optimismInjector(
	context.Context,
	*http.Client,
	// *openseaLimiter,
	// *reservoirLimiter
) (*multichain.OptimismProvider, func()) {
	panic(wire.Build(
		wire.Value(persist.ChainOptimism),
		optimismProviderInjector,
		//openseaProviderInjector,
		alchemy.NewProvider,
		optimismSyncPipelineInjector,
		optimisimTokenDescriptorsFetcherInjector,
		optimismTokenMetadataFetcherInjector,
	))
}

func optimismProviderInjector(
	syncPipeline *wrapper.SyncPipelineWrapper,
	tokenDescriptorsFetcher multichain.TokenDescriptorsFetcher,
	tokenMetadataFetcher multichain.TokenMetadataFetcher,
) *multichain.OptimismProvider {
	panic(wire.Build(
		wire.Struct(new(multichain.OptimismProvider), "*"),
		wire.Bind(new(multichain.TokenIdentifierOwnerFetcher), util.ToPointer(syncPipeline)),
		wire.Bind(new(multichain.TokensIncrementalOwnerFetcher), util.ToPointer(syncPipeline)),
		wire.Bind(new(multichain.TokensIncrementalContractFetcher), util.ToPointer(syncPipeline)),
		wire.Bind(new(multichain.TokenMetadataBatcher), util.ToPointer(syncPipeline)),
		wire.Bind(new(multichain.TokensByTokenIdentifiersFetcher), util.ToPointer(syncPipeline)),
	))
}

func optimismSyncPipelineInjector(
	ctx context.Context,
	httpClient *http.Client,
	chain persist.Chain,
	//openseaProvider *opensea.Provider,
	alchemyProvider *alchemy.Provider,
	// l *reservoirLimiter,
) (*wrapper.SyncPipelineWrapper, func()) {
	panic(wire.Build(
		wire.Struct(new(wrapper.SyncPipelineWrapper), "*"),
		wire.Bind(new(multichain.TokenMetadataBatcher), util.ToPointer(alchemyProvider)),
		wire.Bind(new(retry.Limiter), util.ToPointer(l)),
		optimismTokenIdentifierOwnerFetcherInjector,
		optimismTokensIncrementalOwnerFetcherInjector,
		optimismTokensContractFetcherInjector,
		optmismTokenByTokenIdentifiersFetcherInjector,
		wrapper.NewFillInWrapper,
		customMetadataHandlersInjector,
	))
}

func optimismTokensContractFetcherInjector(openseaProvider *opensea.Provider, alchemyProvider *alchemy.Provider) multichain.TokensIncrementalContractFetcher {
	panic(wire.Build(
		multiTokensIncrementalContractFetcherProvider,
		wire.Bind(new(tokensIncrementalContractFetcherA), util.ToPointer(alchemyProvider)),
		wire.Bind(new(tokensIncrementalContractFetcherB), util.ToPointer(openseaProvider)),
	))
}

func optimismTokenIdentifierOwnerFetcherInjector(openseaProvider *opensea.Provider, alchemyProvider *alchemy.Provider) multichain.TokenIdentifierOwnerFetcher {
	panic(wire.Build(
		multiTokenIdentifierOwnerFetcherProvider,
		wire.Bind(new(tokenIdentifierOwnerFetcherA), util.ToPointer(alchemyProvider)),
		wire.Bind(new(tokenIdentifierOwnerFetcherB), util.ToPointer(openseaProvider)),
	))
}

func optimismTokensIncrementalOwnerFetcherInjector(openseaProvider *opensea.Provider, alchemyProvider *alchemy.Provider) multichain.TokensIncrementalOwnerFetcher {
	panic(wire.Build(
		multiTokensIncrementalOwnerFetcherProvider,
		wire.Bind(new(tokensIncrementalOwnerFetcherA), util.ToPointer(alchemyProvider)),
		wire.Bind(new(tokensIncrementalOwnerFetcherB), util.ToPointer(openseaProvider)),
	))
}

func optimismTokenMetadataFetcherInjector(openseaProvider *opensea.Provider, alchemyProvider *alchemy.Provider) multichain.TokenMetadataFetcher {
	panic(wire.Build(
		multiTokenMetadataFetcherProvider,
		wire.Bind(new(tokenMetadataFetcherA), util.ToPointer(alchemyProvider)),
		wire.Bind(new(tokenMetadataFetcherB), util.ToPointer(openseaProvider)),
	))
}

func optimisimTokenDescriptorsFetcherInjector(openseaProvider *opensea.Provider, alchemyProvider *alchemy.Provider) multichain.TokenDescriptorsFetcher {
	panic(wire.Build(
		multiTokenDescriptorsFetcherProvider,
		wire.Bind(new(tokenDescriptorsFetcherA), util.ToPointer(alchemyProvider)),
		wire.Bind(new(tokenDescriptorsFetcherB), util.ToPointer(openseaProvider)),
	))
}

func optmismTokenByTokenIdentifiersFetcherInjector(openseaProvider *opensea.Provider, alchemyProvider *alchemy.Provider) multichain.TokensByTokenIdentifiersFetcher {
	panic(wire.Build(
		multiTokenByTokenIdentifiersFetcherProvider,
		wire.Bind(new(tokensByTokenIdentifiersFetcherA), util.ToPointer(alchemyProvider)),
		wire.Bind(new(tokensByTokenIdentifiersFetcherB), util.ToPointer(openseaProvider)),
	))
}

// -----------------------------------------------

/*// optimismProviderSet is a wire injector that creates the set of Optimism providers
func optimismProviderSet(*http.Client, *tokenMetadataCache) optimismProviderList {
	wire.Build(
		optimismProvidersConfig,
		wire.Value(persist.ChainOptimism),
		// Add providers for Optimism here
		newAlchemyProvider,
	)
	return optimismProviderList{}
}

// optimismProvidersConfig is a wire injector that binds multichain interfaces to their concrete Optimism implementations
func optimismProvidersConfig(alchemyProvider *alchemy.Provider) optimismProviderList {
	wire.Build(
		wire.Bind(new(multichain.TokensOwnerFetcher), util.ToPointer(alchemyProvider)),
		wire.Bind(new(multichain.TokensIncrementalOwnerFetcher), util.ToPointer(alchemyProvider)),
		wire.Bind(new(multichain.TokensContractFetcher), util.ToPointer(alchemyProvider)),
		wire.Bind(new(multichain.TokenMetadataFetcher), util.ToPointer(alchemyProvider)),
		optimismRequirements,
	)
	return nil
}

// optimismRequirements is the set of provider interfaces required for Optimism
func optimismRequirements(
	tof multichain.TokensOwnerFetcher,
	tiof multichain.TokensIncrementalOwnerFetcher,
	toc multichain.TokensContractFetcher,
	tmf multichain.TokenMetadataFetcher,
) optimismProviderList {
	return optimismProviderList{tof, toc, tiof, tmf}
}
*/

// ------------------------------------------------

func arbitrumInjector(context.Context, *http.Client, *openseaLimiter, *reservoirLimiter) (*multichain.ArbitrumProvider, func()) {
	panic(wire.Build(
		arbitrumProviderInjector,
		wire.Value(persist.ChainArbitrum),
		openseaProviderInjector,
		alchemy.NewProvider,
		arbitrumSyncPipelineInjector,
		arbitrumTokenDescriptorsFetcherInjector,
		arbitrumTokenMetadataFetcherInjector,
	))
}

func arbitrumProviderInjector(
	syncPipeline *wrapper.SyncPipelineWrapper,
	tokenDescriptorsFetcher multichain.TokenDescriptorsFetcher,
	tokenMetadataFetcher multichain.TokenMetadataFetcher,
) *multichain.ArbitrumProvider {
	panic(wire.Build(
		wire.Struct(new(multichain.ArbitrumProvider), "*"),
		wire.Bind(new(multichain.TokenIdentifierOwnerFetcher), util.ToPointer(syncPipeline)),
		wire.Bind(new(multichain.TokensIncrementalOwnerFetcher), util.ToPointer(syncPipeline)),
		wire.Bind(new(multichain.TokensIncrementalContractFetcher), util.ToPointer(syncPipeline)),
		wire.Bind(new(multichain.TokenMetadataBatcher), util.ToPointer(syncPipeline)),
		wire.Bind(new(multichain.TokensByTokenIdentifiersFetcher), util.ToPointer(syncPipeline)),
	))
}

func arbitrumSyncPipelineInjector(
	ctx context.Context,
	httpClient *http.Client,
	chain persist.Chain,
	openseaProvider *opensea.Provider,
	alchemyProvider *alchemy.Provider,
	l *reservoirLimiter,
) (*wrapper.SyncPipelineWrapper, func()) {
	panic(wire.Build(
		wire.Struct(new(wrapper.SyncPipelineWrapper), "*"),
		wire.Bind(new(multichain.TokenMetadataBatcher), util.ToPointer(alchemyProvider)),
		wire.Bind(new(retry.Limiter), util.ToPointer(l)),
		arbitrumTokenIdentifierOwnerFetcherInjector,
		arbitrumTokensIncrementalOwnerFetcherInjector,
		arbitrumTokensContractFetcherInjector,
		arbitrumTokenByTokenIdentifiersFetcherInjector,
		wrapper.NewFillInWrapper,
		customMetadataHandlersInjector,
	))
}

func arbitrumTokenMetadataFetcherInjector(openseaProvider *opensea.Provider, alchemyProvider *alchemy.Provider) multichain.TokenMetadataFetcher {
	panic(wire.Build(
		multiTokenMetadataFetcherProvider,
		wire.Bind(new(tokenMetadataFetcherA), util.ToPointer(alchemyProvider)),
		wire.Bind(new(tokenMetadataFetcherB), util.ToPointer(openseaProvider)),
	))
}

func arbitrumTokenDescriptorsFetcherInjector(openseaProvider *opensea.Provider, alchemyProvider *alchemy.Provider) multichain.TokenDescriptorsFetcher {
	panic(wire.Build(
		multiTokenDescriptorsFetcherProvider,
		wire.Bind(new(tokenDescriptorsFetcherA), util.ToPointer(alchemyProvider)),
		wire.Bind(new(tokenDescriptorsFetcherB), util.ToPointer(openseaProvider)),
	))
}

func arbitrumTokensContractFetcherInjector(openseaProvider *opensea.Provider, alchemyProvider *alchemy.Provider) multichain.TokensIncrementalContractFetcher {
	panic(wire.Build(
		multiTokensIncrementalContractFetcherProvider,
		wire.Bind(new(tokensIncrementalContractFetcherA), util.ToPointer(alchemyProvider)),
		wire.Bind(new(tokensIncrementalContractFetcherB), util.ToPointer(openseaProvider)),
	))
}

func arbitrumTokenIdentifierOwnerFetcherInjector(openseaProvider *opensea.Provider, alchemyProvider *alchemy.Provider) multichain.TokenIdentifierOwnerFetcher {
	panic(wire.Build(
		multiTokenIdentifierOwnerFetcherProvider,
		wire.Bind(new(tokenIdentifierOwnerFetcherA), util.ToPointer(alchemyProvider)),
		wire.Bind(new(tokenIdentifierOwnerFetcherB), util.ToPointer(openseaProvider)),
	))
}

func arbitrumTokensIncrementalOwnerFetcherInjector(openseaProvider *opensea.Provider, alchemyProvider *alchemy.Provider) multichain.TokensIncrementalOwnerFetcher {
	panic(wire.Build(
		multiTokensIncrementalOwnerFetcherProvider,
		wire.Bind(new(tokensIncrementalOwnerFetcherA), util.ToPointer(alchemyProvider)),
		wire.Bind(new(tokensIncrementalOwnerFetcherB), util.ToPointer(openseaProvider)),
	))
}

func arbitrumTokenByTokenIdentifiersFetcherInjector(openseaProvider *opensea.Provider, alchemyProvider *alchemy.Provider) multichain.TokensByTokenIdentifiersFetcher {
	panic(wire.Build(
		multiTokenByTokenIdentifiersFetcherProvider,
		wire.Bind(new(tokensByTokenIdentifiersFetcherA), util.ToPointer(alchemyProvider)),
		wire.Bind(new(tokensByTokenIdentifiersFetcherB), util.ToPointer(openseaProvider)),
	))
}

// ------------------------------------------------

/*// arbitrumProviderSet is a wire injector that creates the set of Arbitrum providers
func arbitrumProviderSet(*http.Client, *tokenMetadataCache) arbitrumProviderList {
	wire.Build(
		arbitrumProvidersConfig,
		wire.Value(persist.ChainArbitrum),
		// Add providers for Optimism here
		newAlchemyProvider,
	)
	return arbitrumProviderList{}
}

// arbitrumProvidersConfig is a wire injector that binds multichain interfaces to their concrete Arbitrum implementations
func arbitrumProvidersConfig(alchemyProvider *alchemy.Provider) arbitrumProviderList {
	wire.Build(
		wire.Bind(new(multichain.TokensOwnerFetcher), util.ToPointer(alchemyProvider)),
		wire.Bind(new(multichain.TokensIncrementalOwnerFetcher), util.ToPointer(alchemyProvider)),
		wire.Bind(new(multichain.TokensContractFetcher), util.ToPointer(alchemyProvider)),
		wire.Bind(new(multichain.TokenMetadataFetcher), util.ToPointer(alchemyProvider)),
		wire.Bind(new(multichain.TokenDescriptorsFetcher), util.ToPointer(alchemyProvider)),
		arbitrumRequirements,
	)
	return nil
}

// arbitrumRequirements is the set of provider interfaces required for Arbitrum
func arbitrumRequirements(
	tof multichain.TokensOwnerFetcher,
	tiof multichain.TokensIncrementalOwnerFetcher,
	toc multichain.TokensContractFetcher,
	tmf multichain.TokenMetadataFetcher,
	tdf multichain.TokenDescriptorsFetcher,
) arbitrumProviderList {
	return arbitrumProviderList{tof, toc, tiof, tmf, tdf}
}

*/

// -----------------------------------------------------------

func polygonInjector(context.Context, *http.Client, *openseaLimiter, *reservoirLimiter) (*multichain.PolygonProvider, func()) {
	panic(wire.Build(
		polygonProvidersInjector,
		wire.Value(persist.ChainPolygon),
		openseaProviderInjector,
		alchemy.NewProvider,
		polygonSyncPipelineInjector,
		polygonTokenDescriptorFetcherInjector,
		polygonTokenMetadataFetcherInjector,
	))
}

func polygonSyncPipelineInjector(
	ctx context.Context,
	httpClient *http.Client,
	chain persist.Chain,
	openseaProvider *opensea.Provider,
	alchemyProvider *alchemy.Provider,
	l *reservoirLimiter,
) (*wrapper.SyncPipelineWrapper, func()) {
	panic(wire.Build(
		wire.Struct(new(wrapper.SyncPipelineWrapper), "*"),
		wire.Bind(new(multichain.TokenMetadataBatcher), util.ToPointer(alchemyProvider)),
		wire.Bind(new(retry.Limiter), util.ToPointer(l)),
		polygonTokenIdentifierOwnerFetcherInjector,
		polygonTokensIncrementalOwnerFetcherInjector,
		polygonTokensContractFetcherInjector,
		polygonTokenByTokenIdentifiersFetcherInjector,
		wrapper.NewFillInWrapper,
		customMetadataHandlersInjector,
	))
}

func polygonTokenMetadataFetcherInjector(openseaProvider *opensea.Provider, alchemyProvider *alchemy.Provider) multichain.TokenMetadataFetcher {
	panic(wire.Build(
		multiTokenMetadataFetcherProvider,
		wire.Bind(new(tokenMetadataFetcherA), util.ToPointer(alchemyProvider)),
		wire.Bind(new(tokenMetadataFetcherB), util.ToPointer(openseaProvider)),
	))
}

func polygonTokenDescriptorFetcherInjector(openseaProvider *opensea.Provider, alchemyProvider *alchemy.Provider) multichain.TokenDescriptorsFetcher {
	panic(wire.Build(
		multiTokenDescriptorsFetcherProvider,
		wire.Bind(new(tokenDescriptorsFetcherA), util.ToPointer(alchemyProvider)),
		wire.Bind(new(tokenDescriptorsFetcherB), util.ToPointer(openseaProvider)),
	))
}

func polygonTokensContractFetcherInjector(openseaProvider *opensea.Provider, alchemyProvider *alchemy.Provider) multichain.TokensIncrementalContractFetcher {
	panic(wire.Build(
		multiTokensIncrementalContractFetcherProvider,
		wire.Bind(new(tokensIncrementalContractFetcherA), util.ToPointer(alchemyProvider)),
		wire.Bind(new(tokensIncrementalContractFetcherB), util.ToPointer(openseaProvider)),
	))
}

func polygonTokenIdentifierOwnerFetcherInjector(openseaProvider *opensea.Provider, alchemyProvider *alchemy.Provider) multichain.TokenIdentifierOwnerFetcher {
	panic(wire.Build(
		multiTokenIdentifierOwnerFetcherProvider,
		wire.Bind(new(tokenIdentifierOwnerFetcherA), util.ToPointer(alchemyProvider)),
		wire.Bind(new(tokenIdentifierOwnerFetcherB), util.ToPointer(openseaProvider)),
	))
}

func polygonTokensIncrementalOwnerFetcherInjector(openseaProvider *opensea.Provider, alchemyProvider *alchemy.Provider) multichain.TokensIncrementalOwnerFetcher {
	panic(wire.Build(
		multiTokensIncrementalOwnerFetcherProvider,
		wire.Bind(new(tokensIncrementalOwnerFetcherA), util.ToPointer(alchemyProvider)),
		wire.Bind(new(tokensIncrementalOwnerFetcherB), util.ToPointer(openseaProvider)),
	))
}

func polygonProvidersInjector(
	syncPipeline *wrapper.SyncPipelineWrapper,
	tokenDescriptorsFetcher multichain.TokenDescriptorsFetcher,
	tokenMetadataFetcher multichain.TokenMetadataFetcher,
) *multichain.PolygonProvider {
	panic(wire.Build(
		wire.Struct(new(multichain.PolygonProvider), "*"),
		wire.Bind(new(multichain.TokenIdentifierOwnerFetcher), util.ToPointer(syncPipeline)),
		wire.Bind(new(multichain.TokensIncrementalOwnerFetcher), util.ToPointer(syncPipeline)),
		wire.Bind(new(multichain.TokensIncrementalContractFetcher), util.ToPointer(syncPipeline)),
		wire.Bind(new(multichain.TokenMetadataBatcher), util.ToPointer(syncPipeline)),
		wire.Bind(new(multichain.TokensByTokenIdentifiersFetcher), util.ToPointer(syncPipeline)),
	))
}

func polygonTokenByTokenIdentifiersFetcherInjector(openseaProvider *opensea.Provider, alchemyProvider *alchemy.Provider) multichain.TokensByTokenIdentifiersFetcher {
	panic(wire.Build(
		multiTokenByTokenIdentifiersFetcherProvider,
		wire.Bind(new(tokensByTokenIdentifiersFetcherA), util.ToPointer(alchemyProvider)),
		wire.Bind(new(tokensByTokenIdentifiersFetcherB), util.ToPointer(openseaProvider)),
	))
}

// -----------------------------------------------------------

/*
// polygonProviderSet is a wire injector that creates the set of polygon providers
func polygonProviderSet(*http.Client, *tokenMetadataCache) polygonProviderList {
	wire.Build(
		polygonProvidersConfig,
		wire.Value(persist.ChainPolygon),
		// Add providers for Polygon here
		newAlchemyProvider,
	)
	return polygonProviderList{}
}

// polygonProvidersConfig is a wire injector that binds multichain interfaces to their concrete Polygon implementations
func polygonProvidersConfig(alchemyProvider *alchemy.Provider) polygonProviderList {
	wire.Build(
		wire.Bind(new(multichain.TokensOwnerFetcher), util.ToPointer(alchemyProvider)),
		wire.Bind(new(multichain.TokensIncrementalOwnerFetcher), util.ToPointer(alchemyProvider)),
		wire.Bind(new(multichain.TokensContractFetcher), util.ToPointer(alchemyProvider)),
		wire.Bind(new(multichain.TokenMetadataFetcher), util.ToPointer(alchemyProvider)),
		polygonRequirements,
	)
	return nil
}

// polygonRequirements is the set of provider interfaces required for Polygon
func polygonRequirements(
	tof multichain.TokensOwnerFetcher,
	tiof multichain.TokensIncrementalOwnerFetcher,
	toc multichain.TokensContractFetcher,
	tmf multichain.TokenMetadataFetcher,
) polygonProviderList {
	return polygonProviderList{tof, tiof, toc, tmf}
}
*/

// ----------------------------------------------

//func newTokenManageCache() *redis.Cache {
//	return redis.NewCache(redis.TokenManageCache)
//}
//
//func submitTokenBatchInjector(context.Context, *redis.Cache) multichain.SubmitTokensF {
//	panic(wire.Build(
//		submitBatch,
//		tokenmanage.New,
//		task.NewClient,
//		tickToken,
//	))
//}
//
//func tickToken() tokenmanage.TickToken { return nil }
//
//func submitBatch(tm *tokenmanage.Manager) multichain.SubmitTokensF {
//	return tm.SubmitBatch
//}

// ----------------------------------------------

/*// dedupe removes duplicate providers based on provider ID
func dedupe(providers []any) []any {
	seen := map[string]bool{}
	deduped := []any{}
	for _, p := range providers {
		if id := p.(multichain.Configurer).GetBlockchainInfo().ProviderID; !seen[id] {
			seen[id] = true
			deduped = append(deduped, p)
		}
	}
	return deduped
}

// newMultichain is a wire provider that creates a multichain provider
func newMultichainSet(
	ethProviders ethProviderList,
	optimismProviders optimismProviderList,
	polygonProviders polygonProviderList,
	arbitrumProviders arbitrumProviderList,
) map[persist.Chain][]any {
	chainToProviders := map[persist.Chain][]any{}
	chainToProviders[persist.ChainETH] = dedupe(ethProviders)
	chainToProviders[persist.ChainOptimism] = dedupe(optimismProviders)
	chainToProviders[persist.ChainPolygon] = dedupe(polygonProviders)
	chainToProviders[persist.ChainArbitrum] = dedupe(arbitrumProviders)
	return chainToProviders
}

func ethFallbackProvider(httpClient *http.Client, cache *tokenMetadataCache) multichain.SyncFailureFallbackProvider {
	wire.Build(
		wire.Value(persist.ChainETH),
		newAlchemyProvider,
		wire.Bind(new(multichain.SyncFailurePrimary), new(*alchemy.Provider)),
		wire.Struct(new(multichain.SyncFailureFallbackProvider), "*"),
	)
	return multichain.SyncFailureFallbackProvider{}
}

func newAlchemyProvider(httpClient *http.Client, chain persist.Chain, cache *tokenMetadataCache) *alchemy.Provider {
	return alchemy.NewProvider(chain, httpClient)
}
*/
