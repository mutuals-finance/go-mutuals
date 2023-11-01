//go:build wireinject
// +build wireinject

package server

import (
	"context"
	"database/sql"
	"net/http"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	"github.com/google/wire"
	"github.com/jackc/pgx/v4/pgxpool"

	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/service/multichain"
	"github.com/SplitFi/go-splitfi/service/multichain/alchemy"
	"github.com/SplitFi/go-splitfi/service/multichain/eth"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/persist/postgres"
	"github.com/SplitFi/go-splitfi/service/redis"
	"github.com/SplitFi/go-splitfi/service/rpc"
	"github.com/SplitFi/go-splitfi/service/task"
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
		task.NewClient,
		newCommunitiesCache,
		newTokenMetadataCache,
		postgres.NewRepositories,
		dbConnSet,
		wire.Struct(new(multichain.Provider), "*"),
		// Add additional chains here
		newMultichainSet,
		ethProviderSet,
		optimismProviderSet,
		polygonProviderSet,
		arbitrumProviderSet,
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

// ethProviderSet is a wire injector that creates the set of Ethereum providers
func ethProviderSet(envInit, *cloudtasks.Client, *http.Client, *tokenMetadataCache) ethProviderList {
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

// optimismProviderSet is a wire injector that creates the set of Optimism providers
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

// arbitrumProviderSet is a wire injector that creates the set of Arbitrum providers
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

// dedupe removes duplicate providers based on provider ID
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

func newCommunitiesCache() *redis.Cache {
	return redis.NewCache(redis.CommunitiesDB)
}

func newTokenMetadataCache() *tokenMetadataCache {
	cache := redis.NewCache(redis.TokenProcessingThrottleDB)
	return util.ToPointer(tokenMetadataCache(*cache))
}
