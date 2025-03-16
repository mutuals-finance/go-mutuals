//go:build wireinject
// +build wireinject

package multichain

import (
	"context"
	"net/http"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/wire"

	db "github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/db/gen/indexerdb"
	"github.com/mutuals/go-mutuals/service/eth"
	"github.com/mutuals/go-mutuals/service/multichain/common"
	"github.com/mutuals/go-mutuals/service/persist"
	"github.com/mutuals/go-mutuals/service/persist/postgres"
	"github.com/mutuals/go-mutuals/service/task"
	"github.com/mutuals/go-mutuals/util"
)

// NewMultichainProvider is a wire injector that sets up a multichain provider instance
// ethClient.Client and task.Client are expensive to initialize, so they're passed as an arg.

func NewMultichainProvider(context.Context, *postgres.Repositories, *db.Queries, *indexerdb.Queries, *ethclient.Client, *task.Client) *Provider {
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

func multichainProviderInjector(ctx context.Context, repos *postgres.Repositories, coreQueries *db.Queries, indexerQueries *indexerdb.Queries, chainProvider *ChainProvider) *Provider {
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
		ethProviderInjector,
		ethVerifierInjector,
	))
}

func ethVerifierInjector(ethClient *ethclient.Client) *eth.Verifier {
	panic(wire.Build(wire.Struct(new(eth.Verifier), "*")))
}

func ethProviderInjector(
	ctx context.Context,
	verifier *eth.Verifier,
) *EthereumProvider {
	panic(wire.Build(
		wire.Struct(new(EthereumProvider), "*"),
		wire.Bind(new(common.Verifier), util.ToPointer(verifier)),
	))
}

func optimismInjector(context.Context, *http.Client, *ethclient.Client) *OptimismProvider {
	panic(wire.Build(
		optimismProviderInjector,
	))
}

func optimismProviderInjector() *OptimismProvider {
	panic(wire.Build(
		wire.Struct(new(OptimismProvider), "*"),
	))
}

func arbitrumInjector(context.Context, *http.Client, *ethclient.Client) *ArbitrumProvider {
	panic(wire.Build(
		arbitrumProviderInjector,
	))
}

func arbitrumProviderInjector() *ArbitrumProvider {
	panic(wire.Build(
		wire.Struct(new(ArbitrumProvider), "*"),
	))
}

func baseInjector(context.Context, *http.Client, *ethclient.Client) *BaseProvider {
	panic(wire.Build(
		baseProvidersInjector,
	))
}

func baseProvidersInjector(
	ethClient *ethclient.Client,
) *BaseProvider {
	panic(wire.Build(
		wire.Struct(new(BaseProvider), "*"),
	))
}

func polygonInjector(context.Context, *http.Client, *ethclient.Client) *PolygonProvider {
	panic(wire.Build(
		polygonProvidersInjector,
	))
}

func polygonProvidersInjector(
	ethClient *ethclient.Client,
) *PolygonProvider {
	panic(wire.Build(
		wire.Struct(new(PolygonProvider), "*"),
	))
}
