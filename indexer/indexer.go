package indexer

import (
	gcptasks "cloud.google.com/go/cloudtasks/apiv2"
	"context"
	"encoding/json"
	"fmt"
	"github.com/SplitFi/go-splitfi/contracts"
	"github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/db/gen/indexerdb"
	"github.com/sourcegraph/conc/iter"
	"math/big"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"cloud.google.com/go/storage"
	"github.com/SplitFi/go-splitfi/env"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/rpc"
	sentryutil "github.com/SplitFi/go-splitfi/service/sentry"
	"github.com/SplitFi/go-splitfi/service/tracing"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/everFinance/goar"
	"github.com/gammazero/workerpool"
	"github.com/getsentry/sentry-go"
	shell "github.com/ipfs/go-ipfs-api"
	"github.com/sirupsen/logrus"
)

func init() {
	env.RegisterValidation("GCLOUD_TOKEN_CONTENT_BUCKET", "required")
	env.RegisterValidation("ALCHEMY_API_URL", "required")
}

const (
	// transferEventHash represents the keccak256 hash of Transfer(address,address,uint256)
	transferEventHash eventHash = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

	defaultWorkerPoolSize     = 3
	defaultWorkerPoolWaitSize = 10
	blocksPerLogsCall         = 50
)

var (
	rpcEnabled            = false // Enables external RPC calls
	erc20ABI, _           = contracts.IERC20MetaData.GetAbi()
	defaultTransferEvents = []eventHash{
		transferEventHash,
	}
)

type errForTokenAtBlockAndIndex struct {
	err error
	boi blockchainOrderInfo
	ti  persist.EthereumTokenIdentifiers
}

func (e errForTokenAtBlockAndIndex) TokenIdentifiers() persist.EthereumTokenIdentifiers {
	return e.ti
}

func (e errForTokenAtBlockAndIndex) OrderInfo() blockchainOrderInfo {
	return e.boi
}

// eventHash represents an event keccak256 hash
type eventHash string

type transfersAtBlock struct {
	block     persist.BlockNumber
	transfers []rpc.Transfer
}

type tokenAtBlock struct {
	ti    persist.TokenChainAddress
	boi   blockchainOrderInfo
	token persist.Token
}

func (o tokenAtBlock) TokenIdentifiers() persist.TokenChainAddress {
	return o.ti
}

func (o tokenAtBlock) OrderInfo() blockchainOrderInfo {
	return o.boi
}

type assetAtBlock struct {
	ti    persist.TokenChainAddress
	boi   blockchainOrderInfo
	asset persist.Asset
}

func (a assetAtBlock) TokenIdentifiers() persist.TokenChainAddress {
	return a.ti
}

func (a assetAtBlock) OrderInfo() blockchainOrderInfo {
	return a.boi
}

type getLogsFunc func(ctx context.Context, curBlock, nextBlock *big.Int, topics [][]common.Hash) ([]types.Log, error)

// indexer is the indexer for the blockchain that uses JSON RPC to scan through logs and process them
// into a format used by the application
type indexer struct {
	ethClient     *ethclient.Client
	httpClient    *http.Client
	ipfsClient    *shell.Shell
	arweaveClient *goar.Client
	storageClient *storage.Client
	queries       *indexerdb.Queries
	splitRepo     persist.SplitRepository
	tokenRepo     persist.TokenRepository
	assetRepo     persist.AssetRepository
	dbMu          *sync.Mutex // Manages writes to the db
	stateMu       *sync.Mutex // Manages updates to the indexer's state
	memoryMu      *sync.Mutex // Manages large memory operations

	tokenBucket string

	chain persist.Chain

	eventHashes []eventHash

	mostRecentBlock uint64  // Current height of the blockchain
	lastSyncedChunk uint64  // Start block of the last chunk handled by the indexer
	maxBlock        *uint64 // If provided, the indexer will only index up to maxBlock

	isListening bool // Indicates if the indexer is waiting for new blocks

	getLogsFunc getLogsFunc

	assetDBHooks []DBHook[persist.Asset]
	tokenDBHooks []DBHook[persist.Token]
}

// newIndexer sets up an indexer for retrieving the specified events that will process tokens
func newIndexer(ethClient *ethclient.Client, httpClient *http.Client, ipfsClient *shell.Shell, arweaveClient *goar.Client, storageClient *storage.Client, iQueries *indexerdb.Queries, bQueries *coredb.Queries, taskClient *gcptasks.Client, splitRepo persist.SplitRepository, tokenRepo persist.TokenRepository, assetRepo persist.AssetRepository, pChain persist.Chain, pEvents []eventHash, getLogsFunc getLogsFunc, startingBlock, maxBlock *uint64) *indexer {

	if rpcEnabled && ethClient == nil {
		panic("RPC is enabled but an ethClient wasn't provided!")
	}

	i := &indexer{
		ethClient:     ethClient,
		ipfsClient:    ipfsClient,
		arweaveClient: arweaveClient,
		storageClient: storageClient,
		httpClient:    httpClient,
		splitRepo:     splitRepo,
		tokenRepo:     tokenRepo,
		assetRepo:     assetRepo,
		queries:       iQueries,
		dbMu:          &sync.Mutex{},
		stateMu:       &sync.Mutex{},
		memoryMu:      &sync.Mutex{},

		tokenBucket: env.GetString("GCLOUD_TOKEN_CONTENT_BUCKET"),

		chain: pChain,

		maxBlock: maxBlock,

		eventHashes: pEvents,

		getLogsFunc: getLogsFunc,

		tokenDBHooks: newTokenHooks(iQueries, tokenRepo, ethClient, httpClient),
		assetDBHooks: newAssetHooks(taskClient, bQueries),

		mostRecentBlock: 0,
		lastSyncedChunk: 0,
		isListening:     false,
	}

	if startingBlock != nil {
		i.lastSyncedChunk = *startingBlock
		i.lastSyncedChunk -= i.lastSyncedChunk % blocksPerLogsCall
	} else {
		recentDBBlock, err := tokenRepo.MostRecentBlock(context.Background())
		if err != nil {
			panic(err)
		}
		i.lastSyncedChunk = recentDBBlock.Uint64()

		safeSub, overflowed := math.SafeSub(i.lastSyncedChunk, (i.lastSyncedChunk%blocksPerLogsCall)+(blocksPerLogsCall*defaultWorkerPoolSize))

		if overflowed {
			i.lastSyncedChunk = 0
		} else {
			i.lastSyncedChunk = safeSub
		}

	}

	if maxBlock != nil {
		i.mostRecentBlock = *maxBlock
	} else if rpcEnabled {
		mostRecentBlock, err := ethClient.BlockNumber(context.Background())
		if err != nil {
			panic(err)
		}
		i.mostRecentBlock = mostRecentBlock
	}

	if i.lastSyncedChunk > i.mostRecentBlock {
		panic(fmt.Sprintf("last handled chunk=%d is greater than the height=%d!", i.lastSyncedChunk, i.mostRecentBlock))
	}

	if i.getLogsFunc == nil {
		i.getLogsFunc = i.defaultGetLogs
	}

	logger.For(nil).Infof("starting indexer at block=%d until block=%d with rpc enabled: %t", i.lastSyncedChunk, i.mostRecentBlock, rpcEnabled)
	return i
}

// INITIALIZATION FUNCS ---------------------------------------------------------

// Start begins indexing events from the blockchain
func (i *indexer) Start(ctx context.Context) {
	if rpcEnabled && i.maxBlock == nil {
		go i.listenForNewBlocks(sentryutil.NewSentryHubContext(ctx))
	}

	topics := eventsToTopics(i.eventHashes)

	logger.For(ctx).Info("Catching up to latest block")
	i.isListening = false
	i.catchUp(ctx, topics)

	if !rpcEnabled {
		logger.For(ctx).Info("Running in cached logs only mode, not listening for new logs")
		return
	}

	logger.For(ctx).Info("Subscribing to new logs")
	i.isListening = true
	i.waitForBlocks(ctx, topics)
}

// catchUp processes logs up to the most recent block.
func (i *indexer) catchUp(ctx context.Context, topics [][]common.Hash) {
	wp := workerpool.New(defaultWorkerPoolSize)
	defer wp.StopWait()

	go func() {
		time.Sleep(10 * time.Second)
		for wp.WaitingQueueSize() > 0 {
			logger.For(ctx).Infof("Catching up: waiting for %d workers to finish", wp.WaitingQueueSize())
			time.Sleep(10 * time.Second)
		}
	}()

	from := i.lastSyncedChunk
	for ; from < atomic.LoadUint64(&i.mostRecentBlock); from += blocksPerLogsCall {
		input := from
		toQueue := func() {
			workerCtx := sentryutil.NewSentryHubContext(ctx)
			defer recoverAndWait(workerCtx)
			defer sentryutil.RecoverAndRaise(workerCtx)
			logger.For(workerCtx).Infof("Indexing block range starting at %d", input)
			i.startPipeline(workerCtx, persist.BlockNumber(input), topics)
			i.updateLastSynced(input)
			logger.For(workerCtx).Infof("Finished indexing block range starting at %d", input)
		}
		if wp.WaitingQueueSize() > defaultWorkerPoolWaitSize {
			wp.SubmitWait(toQueue)
		} else {
			wp.Submit(toQueue)
		}
	}
}

func (i *indexer) updateLastSynced(block uint64) {
	i.stateMu.Lock()
	if i.lastSyncedChunk < block {
		i.lastSyncedChunk = block
	}
	i.stateMu.Unlock()
}

// waitForBlocks polls for new blocks.
func (i *indexer) waitForBlocks(ctx context.Context, topics [][]common.Hash) {
	for {
		timeAfterWait := <-time.After(time.Minute * 3)
		i.startNewBlocksPipeline(ctx, topics)
		logger.For(ctx).Infof("Waiting for new blocks... Finished recent blocks in %s", time.Since(timeAfterWait))
	}
}

func (i *indexer) startPipeline(ctx context.Context, start persist.BlockNumber, topics [][]common.Hash) {
	span, ctx := tracing.StartSpan(ctx, "indexer.pipeline", "catchup", sentry.TransactionName("indexer-main:catchup"))
	tracing.AddEventDataToSpan(span, map[string]interface{}{"block": start})
	defer tracing.FinishSpan(span)

	startTime := time.Now()
	transfers := make(chan []transfersAtBlock)
	plugins := NewTransferPlugins(ctx)
	enabledPlugins := []chan<- TransferPluginMsg{plugins.assets.in, plugins.tokens.in}

	statsID := persist.DBID("")
	//statsID, err := i.queries.InsertStatistic(ctx, indexerdb.InsertStatisticParams{ID: persist.GenerateID(), BlockStart: start, BlockEnd: start + blocksPerLogsCall})
	//if err != nil {
	//	panic(err)
	//}

	go func() {
		ctx := sentryutil.NewSentryHubContext(ctx)
		span, ctx := tracing.StartSpan(ctx, "indexer.logs", "processLogs")
		defer tracing.FinishSpan(span)

		logs := i.fetchLogs(ctx, start, topics)
		i.processLogs(ctx, transfers, logs)
	}()
	go i.processAllTransfers(sentryutil.NewSentryHubContext(ctx), transfers, enabledPlugins)
	i.processTokens(ctx, plugins.tokens.out, plugins.assets.out, statsID)

	//err = i.queries.UpdateStatisticSuccess(ctx, indexerdb.UpdateStatisticSuccessParams{ID: statsID, Success: true, ProcessingTimeSeconds: sql.NullInt64{Int64: int64(time.Since(startTime) / time.Second), Valid: true}})
	//if err != nil {
	//	panic(err)
	//}

	logger.For(ctx).Warnf("Finished processing %d blocks from block %d in %s", blocksPerLogsCall, start.Uint64(), time.Since(startTime))
}

func (i *indexer) startNewBlocksPipeline(ctx context.Context, topics [][]common.Hash) {
	span, ctx := tracing.StartSpan(ctx, "indexer.pipeline", "polling", sentry.TransactionName("indexer-main:polling"))
	defer tracing.FinishSpan(span)

	transfers := make(chan []transfersAtBlock)
	plugins := NewTransferPlugins(ctx)
	enabledPlugins := []chan<- TransferPluginMsg{plugins.assets.in, plugins.tokens.in}

	mostRecentBlock, err := rpc.RetryGetBlockNumber(ctx, i.ethClient)
	if err != nil {
		panic(err)
	}

	if i.lastSyncedChunk+blocksPerLogsCall > mostRecentBlock {
		logger.For(ctx).Infof("No new blocks to process. Last synced block: %d, most recent block: %d", i.lastSyncedChunk, mostRecentBlock)
		return
	}

	statsID := persist.DBID("")
	//statsID, err := i.queries.InsertStatistic(ctx, indexerdb.InsertStatisticParams{ID: persist.GenerateID(), BlockStart: persist.BlockNumber(i.lastSyncedChunk), BlockEnd: persist.BlockNumber(mostRecentBlock - (i.mostRecentBlock % blocksPerLogsCall))})
	//if err != nil {
	//	panic(err)
	//}

	go i.pollNewLogs(sentryutil.NewSentryHubContext(ctx), transfers, topics, mostRecentBlock)
	go i.processAllTransfers(sentryutil.NewSentryHubContext(ctx), transfers, enabledPlugins)
	i.processTokens(ctx, plugins.tokens.out, plugins.assets.out, statsID)
}

func (i *indexer) listenForNewBlocks(ctx context.Context) {
	defer sentryutil.RecoverAndRaise(ctx)

	for {
		<-time.After(time.Second*12*time.Duration(blocksPerLogsCall) + time.Minute)
		finalBlockUint, err := rpc.RetryGetBlockNumber(ctx, i.ethClient)
		if err != nil {
			panic(fmt.Sprintf("error getting block number: %s", err))
		}
		atomic.StoreUint64(&i.mostRecentBlock, finalBlockUint)
		logger.For(ctx).Debugf("final block number: %v", finalBlockUint)
	}
}

// LOGS FUNCS ---------------------------------------------------------------

func (i *indexer) fetchLogs(ctx context.Context, startingBlock persist.BlockNumber, topics [][]common.Hash) []types.Log {
	curBlock := startingBlock.BigInt()
	nextBlock := new(big.Int).Add(curBlock, big.NewInt(int64(blocksPerLogsCall)))

	logger.For(ctx).Infof("Getting logs from %d to %d", curBlock, nextBlock)

	logsTo, err := i.getLogsFunc(ctx, curBlock, nextBlock, topics)
	if err != nil {
		panic(fmt.Sprintf("error getting logs: %s", err))
	}

	//err = i.queries.UpdateStatisticTotalLogs(ctx, indexerdb.UpdateStatisticTotalLogsParams{
	//	ID:        statsID,
	//	TotalLogs: sql.NullInt64{Int64: int64(len(logsTo)), Valid: true},
	//})
	//if err != nil {
	//	panic(err)
	//}

	logger.For(ctx).Infof("Found %d logs at block %d", len(logsTo), curBlock.Uint64())

	return logsTo
}

func (i *indexer) defaultGetLogs(ctx context.Context, curBlock, nextBlock *big.Int, topics [][]common.Hash) ([]types.Log, error) {
	var logsTo []types.Log
	reader, err := i.storageClient.Bucket(env.GetString("GCLOUD_TOKEN_LOGS_BUCKET")).Object(fmt.Sprintf("%d-%d", curBlock, nextBlock)).NewReader(ctx)
	if err != nil {
		logger.For(ctx).WithError(err).Warn("error getting logs from GCP")
	} else {
		func() {
			logger.For(ctx).Infof("Reading logs from GCP")
			i.memoryMu.Lock()
			defer i.memoryMu.Unlock()
			defer reader.Close()
			err = json.NewDecoder(reader).Decode(&logsTo)
			if err != nil {
				panic(err)
			}
		}()
	}

	if len(logsTo) > 0 {
		lastLog := logsTo[len(logsTo)-1]
		if nextBlock.Uint64()-lastLog.BlockNumber > (blocksPerLogsCall / 5) {
			logger.For(ctx).Warnf("Last log is %d blocks old, skipping", nextBlock.Uint64()-lastLog.BlockNumber)
			logsTo = []types.Log{}
		}
	}

	rpcCtx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	if len(logsTo) == 0 && rpcEnabled {
		logger.For(ctx).Infof("Reading logs from Blockchain")
		logsTo, err = rpc.RetryGetLogs(rpcCtx, i.ethClient, ethereum.FilterQuery{
			FromBlock: curBlock,
			ToBlock:   nextBlock,
			Topics:    topics,
		})
		if err != nil {
			logEntry := logger.For(ctx).WithError(err).WithFields(logrus.Fields{
				"fromBlock": curBlock.String(),
				"toBlock":   nextBlock.String(),
				"rpcCall":   "eth_getFilterLogs",
			})
			if rpcErr, ok := err.(gethrpc.Error); ok {
				logEntry = logEntry.WithFields(logrus.Fields{"rpcErrorCode": strconv.Itoa(rpcErr.ErrorCode())})
			}
			logEntry.Error("failed to fetch logs")
			return []types.Log{}, nil
		}
		go saveLogsInBlockRange(ctx, curBlock.String(), nextBlock.String(), logsTo, i.storageClient, i.memoryMu)
	}
	logger.For(ctx).Infof("Found %d logs at block %d", len(logsTo), curBlock.Uint64())
	return logsTo, nil
}

func (i *indexer) processLogs(ctx context.Context, transfersChan chan<- []transfersAtBlock, logsTo []types.Log) {
	defer close(transfersChan)
	defer recoverAndWait(ctx)
	defer sentryutil.RecoverAndRaise(ctx)

	transfers := logsToTransfers(ctx, logsTo)

	logger.For(ctx).Infof("Processed %d logs into %d transfers", len(logsTo), len(transfers))

	transfersChan <- transfersToTransfersAtBlock(transfers)
}

func logsToTransfers(ctx context.Context, pLogs []types.Log) []rpc.Transfer {

	result := make([]rpc.Transfer, 0, len(pLogs)*2)
	for _, pLog := range pLogs {
		initial := time.Now()
		switch {
		case strings.EqualFold(pLog.Topics[0].Hex(), string(transferEventHash)):
			if len(pLog.Topics) < 4 {
				continue
			}

			eventData := map[string]interface{}{}
			err := erc20ABI.UnpackIntoMap(eventData, "Transfer", pLog.Data)
			if err != nil {
				logger.For(ctx).WithError(err).Error("Failed to unpack Transfer event")
				panic(err)
			}

			value, ok := eventData["value"].(*big.Int)
			if !ok {
				panic("Failed to unpack Transfer event, value not found")
			}

			result = append(result, rpc.Transfer{
				From:            persist.Address(pLog.Topics[1].Hex()),
				To:              persist.Address(pLog.Topics[2].Hex()),
				Amount:          value.Uint64(),
				BlockNumber:     persist.BlockNumber(pLog.BlockNumber),
				ContractAddress: persist.Address(pLog.Address.Hex()),
				TokenType:       persist.TokenTypeERC20,
				TxHash:          pLog.TxHash,
				BlockHash:       pLog.BlockHash,
				TxIndex:         pLog.TxIndex,
			})
			logger.For(ctx).Debugf("Processed transfer event in %s", time.Since(initial))

		default:
			logger.For(ctx).WithFields(logrus.Fields{
				"address":   pLog.Address,
				"block":     pLog.BlockNumber,
				"eventType": pLog.Topics[0]},
			).Warn("unknown event")
		}
	}
	return result
}

func (i *indexer) pollNewLogs(ctx context.Context, transfersChan chan<- []transfersAtBlock, topics [][]common.Hash, mostRecentBlock uint64) {
	span, ctx := tracing.StartSpan(ctx, "indexer.logs", "pollLogs")
	defer tracing.FinishSpan(span)
	defer close(transfersChan)
	defer recoverAndWait(ctx)
	defer sentryutil.RecoverAndRaise(ctx)

	logger.For(ctx).Infof("Subscribing to new logs from block %d starting with block %d", mostRecentBlock, i.lastSyncedChunk)

	totalLogs := &atomic.Int64{}

	wp := workerpool.New(10)
	// starting at the last chunk that we synced, poll for logs in chunks of blocksPerLogsCall
	for j := i.lastSyncedChunk; j+blocksPerLogsCall <= mostRecentBlock; j += blocksPerLogsCall {
		curBlock := j
		wp.Submit(func() {
			ctx := sentryutil.NewSentryHubContext(ctx)
			defer sentryutil.RecoverAndRaise(ctx)

			nextBlock := curBlock + blocksPerLogsCall

			rpcCtx, cancel := context.WithTimeout(ctx, time.Second*30)
			defer cancel()

			logsTo, err := rpc.RetryGetLogs(rpcCtx, i.ethClient, ethereum.FilterQuery{
				FromBlock: persist.BlockNumber(curBlock).BigInt(),
				ToBlock:   persist.BlockNumber(nextBlock).BigInt(),
				Topics:    topics,
			})
			if err != nil {
				errData := map[string]interface{}{
					"from": curBlock,
					"to":   nextBlock,
					"err":  err.Error(),
				}
				logger.For(ctx).WithError(err).Error(errData)
				return
			}

			totalLogs.Add(int64(len(logsTo)))

			go saveLogsInBlockRange(ctx, strconv.Itoa(int(curBlock)), strconv.Itoa(int(nextBlock)), logsTo, i.storageClient, i.memoryMu)

			logger.For(ctx).Infof("Found %d logs at block %d", len(logsTo), curBlock)

			transfers := logsToTransfers(ctx, logsTo)

			logger.For(ctx).Infof("Processed %d logs into %d transfers", len(logsTo), len(transfers))

			logger.For(ctx).Debugf("Sending %d total transfers to transfers channel", len(transfers))
			transfersChan <- transfersToTransfersAtBlock(transfers)

		})
	}

	wp.StopWait()

	total := totalLogs.Load()

	//err := i.queries.UpdateStatisticTotalLogs(ctx, indexerdb.UpdateStatisticTotalLogsParams{
	//	TotalLogs: sql.NullInt64{Int64: total, Valid: true},
	//	ID:        statsID,
	//})
	//if err != nil {
	//	logger.For(ctx).WithError(err).Error("Failed to update total logs")
	//	panic(err)
	//}

	logger.For(ctx).Infof("Processed %d logs from %d to %d.", total, i.lastSyncedChunk, mostRecentBlock)

	i.updateLastSynced(mostRecentBlock - (mostRecentBlock % blocksPerLogsCall))

}

// TRANSFERS FUNCS -------------------------------------------------------------

func (i *indexer) processAllTransfers(ctx context.Context, incomingTransfers <-chan []transfersAtBlock, plugins []chan<- TransferPluginMsg) {
	span, ctx := tracing.StartSpan(ctx, "indexer.transfers", "processTransfers")
	defer tracing.FinishSpan(span)
	defer sentryutil.RecoverAndRaise(ctx)
	for _, plugin := range plugins {
		defer close(plugin)
	}

	wp := workerpool.New(5)

	logger.For(ctx).Info("Starting to process transfers...")
	var totatTransfers int64
	for transfers := range incomingTransfers {
		if len(transfers) == 0 {
			continue
		}

		totatTransfers += int64(len(transfers))

		submit := transfers
		wp.Submit(func() {
			ctx := sentryutil.NewSentryHubContext(ctx)
			timeStart := time.Now()
			logger.For(ctx).Infof("Processing %d transfers", len(submit))
			i.processTransfers(ctx, submit, plugins)
			logger.For(ctx).Infof("Processed %d transfers in %s", len(submit), time.Since(timeStart))
		})
	}

	logger.For(ctx).Info("Waiting for transfers to finish...")
	wp.StopWait()

	//err := i.queries.UpdateStatisticTotalTransfers(ctx, indexerdb.UpdateStatisticTotalTransfersParams{
	//	TotalTransfers: sql.NullInt64{Int64: totatTransfers, Valid: true},
	//	ID:             statsID,
	//})
	//if err != nil {
	//	logger.For(ctx).WithError(err).Error("Failed to update total transfers")
	//	panic(err)
	//}

	logger.For(ctx).Info("Closing field channels...")
}

func (i *indexer) processTransfers(ctx context.Context, transfers []transfersAtBlock, plugins []chan<- TransferPluginMsg) {

	for _, transferAtBlock := range transfers {
		for _, transfer := range transferAtBlock.transfers {

			key := persist.NewTokenChainAddress(transfer.ContractAddress, i.chain)

			RunTransferPlugins(ctx, transfer, key, plugins)

		}

	}

}

// TOKENS FUNCS ---------------------------------------------------------------

func (i *indexer) processTokens(ctx context.Context, tokensOut <-chan tokenAtBlock, assetsOut <-chan assetAtBlock, statsID persist.DBID) {

	wg := &sync.WaitGroup{}
	mu := &sync.Mutex{}

	assetsMap := make(map[persist.TokenChainAddress]assetAtBlock)
	tokensMap := make(map[persist.TokenChainAddress]tokenAtBlock)

	RunTransferPluginReceiver(ctx, wg, mu, assetsPluginReceiver, assetsOut, assetsMap)
	RunTransferPluginReceiver(ctx, wg, mu, tokensPluginReceiver, tokensOut, tokensMap)

	wg.Wait()

	assets := assetsAtBlockToAssets(assetsMap)
	tokens := tokensAtBlockToTokens(tokensMap)

	i.runDBHooks(ctx, assets, tokens, statsID)
}

func tokensAtBlockToTokens(tokensAtBlock map[persist.TokenChainAddress]tokenAtBlock) []persist.Token {
	tokens := make([]persist.Token, 0, len(tokensAtBlock))
	seen := make(map[persist.TokenChainAddress]bool)
	for _, tAtB := range tokensAtBlock {
		if seen[tAtB.TokenIdentifiers()] {
			continue
		}
		tokens = append(tokens, tAtB.token)
		seen[tAtB.TokenIdentifiers()] = true
	}
	return tokens
}

func assetsAtBlockToAssets(assetsAtBlock map[persist.TokenChainAddress]assetAtBlock) []persist.Asset {
	assets := make([]persist.Asset, 0, len(assetsAtBlock))
	seen := make(map[persist.TokenChainAddress]bool)
	for _, aAtB := range assetsAtBlock {
		if seen[aAtB.TokenIdentifiers()] {
			continue
		}
		assets = append(assets, aAtB.asset)
		seen[aAtB.TokenIdentifiers()] = true
	}
	return assets
}

func (i *indexer) runDBHooks(ctx context.Context, assets []persist.Asset, tokens []persist.Token, statsID persist.DBID) {
	defer recoverAndWait(ctx)

	wp := workerpool.New(10)

	for _, hook := range i.assetDBHooks {
		hook := hook
		wp.Submit(func() {
			err := hook(ctx, assets, statsID)
			if err != nil {
				logger.For(ctx).WithError(err).Errorf("Failed to run asset db hook %s", err)
			}
		})
	}

	for _, hook := range i.tokenDBHooks {
		hook := hook
		wp.Submit(func() {
			err := hook(ctx, tokens, statsID)
			if err != nil {
				logger.For(ctx).WithError(err).Errorf("Failed to run token db hook %s", err)
			}

		})
	}

	wp.StopWait()
}

func assetsPluginReceiver(cur assetAtBlock, inc assetAtBlock) assetAtBlock {
	inc.asset.Balance += cur.asset.Balance
	return inc
}

func tokensPluginReceiver(cur tokenAtBlock, inc tokenAtBlock) tokenAtBlock {
	return inc
}

func fillTokenFields(ctx context.Context, tokens []persist.Token, queries *indexerdb.Queries, tokenRepo persist.TokenRepository, httpClient *http.Client, ethClient *ethclient.Client, upChan chan<- []persist.Token, statsID persist.DBID) {

	defer close(upChan)

	tokensNotInDB := make(chan persist.Token)

	batched := make(chan []persist.Token)

	go func() {
		defer close(tokensNotInDB)

		iter.ForEach(tokens, func(t *persist.Token) {
			_, err := tokenRepo.GetByIdentifiers(ctx, t.ContractAddress)
			if err == nil {
				return
			}
			tokensNotInDB <- *t
		})
	}()

	go func() {
		defer close(batched)
		var curBatch []persist.Token
		for token := range tokensNotInDB {
			curBatch = append(curBatch, token)
			if len(curBatch) == 100 {
				logger.For(ctx).Infof("Batching %d tokens for metadata", len(curBatch))
				batched <- curBatch
				curBatch = []persist.Token{}
			}
		}
		if len(curBatch) > 0 {
			logger.For(ctx).Infof("Batching %d tokens for metadata", len(curBatch))
			batched <- curBatch
		}
	}()

	// process tokens in batches of 100
	for batch := range batched {
		// get token metadata
		toUp, _ := GetTokenMetadatas(ctx, batch, httpClient, ethClient)

		logger.For(ctx).Infof("Fetched metadata for %d tokens", len(toUp))

		upChan <- toUp
	}

	/*	err = queries.UpdateStatisticContractStats(ctx, indexerdb.UpdateStatisticContractStatsParams{
			ContractStats: pgtype.JSONB{Bytes: marshalled, Status: pgtype.Present},
			ID:            statsID,
		})
		if err != nil {
			logger.For(ctx).WithError(err).Error("Failed to update token stats")
			panic(err)
		}
	*/

	logger.For(ctx).Infof("Fetched metadata for total %d tokens", len(tokens))

}

func GetTokenMetadatas(ctx context.Context, batch []persist.Token, httpClient *http.Client, ethClient *ethclient.Client) ([]persist.Token, error) {
	toUp := make([]persist.Token, 0, 100)

	for _, t := range batch {
		result := persist.Token{}

		tMetadata, err := rpc.GetTokenContractMetadata(ctx, t.ContractAddress.Address(), ethClient)
		if err != nil {
			logger.For(ctx).WithError(err).WithFields(logrus.Fields{
				"tokenAddress": t.ContractAddress,
			}).Error("error getting token metadata")
		} else {
			result.Name = persist.NullString(tMetadata.Name)
			result.Symbol = persist.NullString(tMetadata.Symbol)
			result.Decimals = persist.NullInt32(tMetadata.Decimals)
			// TODO fill Logo metadata
			result.Logo = ""
		}

		toUp = append(toUp, result)
	}
	return toUp, nil
}

// HELPER FUNCS ---------------------------------------------------------------

func transfersToTransfersAtBlock(transfers []rpc.Transfer) []transfersAtBlock {
	transfersMap := map[persist.BlockNumber]transfersAtBlock{}

	for _, transfer := range transfers {
		if tab, ok := transfersMap[transfer.BlockNumber]; !ok {
			transfers := make([]rpc.Transfer, 0, 10)
			transfers = append(transfers, transfer)
			transfersMap[transfer.BlockNumber] = transfersAtBlock{
				block:     transfer.BlockNumber,
				transfers: transfers,
			}
		} else {
			tab.transfers = append(tab.transfers, transfer)
			transfersMap[transfer.BlockNumber] = tab
		}
	}

	allTransfersAtBlock := make([]transfersAtBlock, len(transfersMap))
	i := 0
	for _, transfersAtBlock := range transfersMap {
		allTransfersAtBlock[i] = transfersAtBlock
		i++
	}
	sort.Slice(allTransfersAtBlock, func(i, j int) bool {
		return allTransfersAtBlock[i].block < allTransfersAtBlock[j].block
	})
	return allTransfersAtBlock
}

func saveLogsInBlockRange(ctx context.Context, curBlock, nextBlock string, logsTo []types.Log, storageClient *storage.Client, memoryMu *sync.Mutex) {
	memoryMu.Lock()
	defer memoryMu.Unlock()
	logger.For(ctx).Infof("Saving logs in block range %s to %s", curBlock, nextBlock)
	obj := storageClient.Bucket(env.GetString("GCLOUD_TOKEN_LOGS_BUCKET")).Object(fmt.Sprintf("%s-%s", curBlock, nextBlock))
	obj.Delete(ctx)
	storageWriter := obj.NewWriter(ctx)

	if err := json.NewEncoder(storageWriter).Encode(logsTo); err != nil {
		panic(err)
	}
	if err := storageWriter.Close(); err != nil {
		panic(err)
	}
}

func recoverAndWait(ctx context.Context) {
	if err := recover(); err != nil {
		logger.For(ctx).Errorf("Error in indexer: %v", err)
		time.Sleep(time.Second * 10)
	}
}

func eventsToTopics(hashes []eventHash) [][]common.Hash {
	events := make([]common.Hash, len(hashes))
	for i, event := range hashes {
		events[i] = common.HexToHash(string(event))
	}
	return [][]common.Hash{events}
}
