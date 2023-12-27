package indexer

import (
	"context"
	"database/sql"
	"github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/db/gen/indexerdb"
	"github.com/SplitFi/go-splitfi/service/task"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"

	migrate "github.com/SplitFi/go-splitfi/db"
	"github.com/SplitFi/go-splitfi/docker"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/persist/postgres"
	"github.com/SplitFi/go-splitfi/service/rpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/assert"
)

var (
	testBlockFrom = 0
	testBlockTo   = 100
	testAddress   = "0xd8da6bf26964af9d7eed9e03e53415d37aa96045" // vitalik.eth
	ensAddress    = "0x57f1887a8bf19b14fc0df6fd9b2acc9af147ea85"
)

var allLogs = func() []types.Log {
	logs := htmlLogs
	logs = append(logs, ipfsLogs...)
	logs = append(logs, customHandlerLogs...)
	logs = append(logs, svgLogs...)
	logs = append(logs, ensLogs...)
	return logs
}()

func setupTest(t *testing.T) (*assert.Assertions, *sql.DB, *pgxpool.Pool, *pgxpool.Pool) {
	SetDefaults()
	LoadConfigFile("indexer-server", "local")
	ValidateEnv()

	r, err := docker.StartPostgresIndexer()
	if err != nil {
		t.Fatal(err)
	}

	r2, err := docker.StartPostgres()
	if err != nil {
		t.Fatal(err)
	}

	hostAndPort := strings.Split(r.GetHostPort("5432/tcp"), ":")
	t.Setenv("POSTGRES_HOST", hostAndPort[0])
	t.Setenv("POSTGRES_PORT", hostAndPort[1])

	hostAndPort2 := strings.Split(r2.GetHostPort("5432/tcp"), ":")

	db := postgres.MustCreateClient()
	pgx := postgres.NewPgxClient()
	pgx2 := postgres.NewPgxClient(postgres.WithHost(hostAndPort2[0]), postgres.WithPort(5432))

	migrate, err := migrate.RunMigration(db, "./db/migrations/indexer")
	if err != nil {
		t.Fatalf("failed to seed db: %s", err)
	}
	t.Cleanup(func() {
		migrate.Close()
		r.Close()
	})

	return assert.New(t), db, pgx, pgx2
}

func newMockIndexer(db *sql.DB, pool, pool2 *pgxpool.Pool) *indexer {
	start := uint64(testBlockFrom)
	end := uint64(testBlockTo)
	rpcEnabled = true
	ethClient := rpc.NewEthSocketClient()
	iQueries := indexerdb.New(pool)
	bQueries := coredb.New(pool2)

	i := newIndexer(ethClient, &http.Client{Timeout: 10 * time.Minute}, nil, nil, nil, iQueries, bQueries, task.NewClient(context.Background()), postgres.NewSplitRepository(db), postgres.NewTokenRepository(db), postgres.NewAssetRepository(db, bQueries), persist.ChainETH, defaultTransferEvents, func(ctx context.Context, curBlock, nextBlock *big.Int, topics [][]common.Hash) ([]types.Log, error) {
		transferAgainLogs := []types.Log{{
			Address:     common.HexToAddress("0x0c2ee19b2a89943066c2dc7f1bddcc907f614033"),
			Topics:      []common.Hash{common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"), common.HexToHash(testAddress), common.HexToHash("0x0000000000000000000000008914496dc01efcc49a2fa340331fb90969b6f1d2"), common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000000d9")},
			Data:        []byte{},
			BlockNumber: 51,
			TxIndex:     1,
		}}
		if curBlock.Uint64() == 0 {
			return allLogs, nil
		}
		return transferAgainLogs, nil
	}, &start, &end)
	return i
}

var htmlLogs = []types.Log{
	{
		Address: common.HexToAddress("0x0c2ee19b2a89943066c2dc7f1bddcc907f614033"),
		Topics: []common.Hash{
			common.HexToHash(string(transferEventHash)),
			common.HexToHash(persist.ZeroAddress.String()),
			common.HexToHash(testAddress),
			common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000000d9"),
		},
		BlockNumber: 1,
	},
	{
		Address: common.HexToAddress("0x059edd72cd353df5106d2b9cc5ab83a52287ac3a"),
		Topics: []common.Hash{
			common.HexToHash(string(transferEventHash)),
			common.HexToHash(persist.ZeroAddress.String()),
			common.HexToHash(testAddress),
			common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"),
		},
		BlockNumber: 1,
	},
}
var ipfsLogs = []types.Log{
	{
		Address: common.HexToAddress("0xbc4ca0eda7647a8ab7c2061c2e118a18a936f13d"),
		Topics: []common.Hash{common.HexToHash(
			string(transferEventHash)),
			common.HexToHash(persist.ZeroAddress.String()),
			common.HexToHash(testAddress),
			common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"),
		},
		BlockNumber: 2,
	},
}
var customHandlerLogs = []types.Log{
	{
		Address: common.HexToAddress("0xd4e4078ca3495de5b1d4db434bebc5a986197782"),
		Topics: []common.Hash{
			common.HexToHash(string(transferEventHash)),
			common.HexToHash(persist.ZeroAddress.String()),
			common.HexToHash(testAddress),
			common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"),
		},
		BlockNumber: 22,
	},
}
var svgLogs = []types.Log{
	{
		Address: common.HexToAddress("0x69c40e500b84660cb2ab09cb9614fa2387f95f64"),
		Topics: []common.Hash{
			common.HexToHash(string(transferEventHash)),
			common.HexToHash(persist.ZeroAddress.String()),
			common.HexToHash(testAddress),
			common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"),
		},
		BlockNumber: 3,
	},
}
var ensLogs = []types.Log{
	{
		Address: common.HexToAddress(ensAddress),
		Topics: []common.Hash{
			common.HexToHash(string(transferEventHash)),
			common.HexToHash(persist.ZeroAddress.String()),
			common.HexToHash(testAddress),
			common.HexToHash("0xc1cb7903f69821967b365cce775cd62d694cd7ae7cfe00efe1917a55fdae2bb7"),
		},
		BlockNumber: 42,
	},
	{
		Address: common.HexToAddress(ensAddress),
		Topics: []common.Hash{
			common.HexToHash(string(transferEventHash)),
			common.HexToHash(persist.ZeroAddress.String()),
			common.HexToHash(testAddress),
			// Leading zero in token ID
			common.HexToHash("0x08c111a4e7c31becd720bde47f538417068e102d45b7732f24cfeda9e2b22a45"),
		},
		BlockNumber: 42,
	},
}

type expectedSplitsResults map[persist.Address]persist.Split
type expectedTokenResults map[persist.TokenChainAddress]persist.Token
type expectedAssetResults map[persist.AssetIdentifiers]persist.Asset

var expectedSplits expectedSplitsResults = expectedSplitsResults{
	persist.ZeroAddress: persist.Split{
		Name:           "Test Name",
		Description:    "Test Description",
		CreatorAddress: persist.Address(testAddress),
		Address:        persist.ZeroAddress,
		Chain:          persist.ChainETH,
		LogoURL:        "https://example.com/logo/1.png",
		BadgeURL:       "https://example.com/badge/1.png",
		BannerURL:      "https://example.com/banner/1.png",
		Recipients:     []persist.Recipient{{Address: persist.Address(testAddress), Ownership: 1}},
	},
}

var tokenIds = map[string]persist.TokenChainAddress{
	"ETH":  persist.NewTokenChainAddress("", persist.ChainETH),
	"WETH": persist.NewTokenChainAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2", persist.ChainETH),
	"USDT": persist.NewTokenChainAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7", persist.ChainETH),
}

var expectedResults expectedTokenResults = expectedTokenResults{
	tokenIds["ETH"]: persist.Token{
		Name:            "Ether",
		Symbol:          "ETH",
		Decimals:        18,
		TokenType:       persist.TokenTypeNative,
		ContractAddress: tokenIds["ETH"].Address,
		Chain:           tokenIds["ETH"].Chain,
		BlockNumber:     1,
	},
	tokenIds["WETH"]: persist.Token{
		Name:            "Wrapped Ether",
		Symbol:          "WETH",
		Decimals:        18,
		TokenType:       persist.TokenTypeERC20,
		ContractAddress: tokenIds["WETH"].Address,
		Chain:           tokenIds["WETH"].Chain,
		BlockNumber:     22,
	},
	/*	tokenIds["USDT"]: {
			Name:            "Tether USD",
			Symbol:          "USDT",
			Decimals:        18,
			TokenType:       persist.TokenTypeERC20,
			ContractAddress: tokenIds["USDT"].Address,
			Chain:           tokenIds["USDT"].Chain,
			BlockNumber:     2,
		},
	*/}

var expectedAssets expectedAssetResults = expectedAssetResults{
	persist.NewAssetIdentifiers(tokenIds["ETH"].Address, persist.Address(testAddress)): {
		OwnerAddress: persist.Address(testAddress),
		Token:        expectedResults[tokenIds["ETH"]],
		Balance:      100,
		BlockNumber:  23,
	},
	persist.NewAssetIdentifiers(tokenIds["WETH"].Address, persist.Address(testAddress)): {
		OwnerAddress: persist.Address(testAddress),
		Token:        expectedResults[tokenIds["WETH"]],
		Balance:      12,
		BlockNumber:  24,
	},
}

func expectedTokensForAddress(address persist.Address) int {
	count := 0
	for _, asset := range expectedAssets {
		if asset.OwnerAddress == address {
			count++
		}
	}
	return count
}

func expectedSplitsForAddress(address persist.Address) int {
	count := 0
	for _, split := range expectedSplits {
		for _, recipient := range split.Recipients {
			if recipient.Address == address {
				count++
			}
		}
	}
	return count
}

func expectedTokens() []persist.Address {
	addresses := make([]persist.Address, 0, len(expectedResults))
	seen := map[persist.TokenChainAddress]struct{}{}
	for _, token := range expectedResults {
		id := persist.NewTokenChainAddress(token.ContractAddress, token.Chain)
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			addresses = append(addresses, token.ContractAddress)
		}
	}
	return addresses
}
