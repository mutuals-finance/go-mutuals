package indexer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/rpc"
	sentryutil "github.com/SplitFi/go-splitfi/service/sentry"
	"github.com/SplitFi/go-splitfi/service/tracing"
	"github.com/gammazero/workerpool"
	"github.com/getsentry/sentry-go"
	"github.com/sirupsen/logrus"
)

const (
	pluginPoolSize = 32
	pluginTimeout  = 2 * time.Minute
)

// TransferPluginMsg is used to communicate to a plugin.
type TransferPluginMsg struct {
	transfer rpc.Transfer
	key      persist.TokenChainAddress
}

// TransferPlugins are plugins that add contextual data to a transfer.
type TransferPlugins struct {
	assets assetTransfersPlugin
	tokens tokenTransfersPlugin
}

type blockchainOrderInfo struct {
	blockNumber persist.BlockNumber
	txIndex     uint
}

// Less returns true if the current block number and tx index are less than the other block number and tx index.
func (b blockchainOrderInfo) Less(other blockchainOrderInfo) bool {
	if b.blockNumber < other.blockNumber {
		return true
	}
	if b.blockNumber > other.blockNumber {
		return false
	}
	return b.txIndex < other.txIndex
}

type orderedBlockChainData interface {
	TokenIdentifiers() persist.TokenChainAddress
	OrderInfo() blockchainOrderInfo
}

// TransferPluginReceiver receives the results of a plugin.
type TransferPluginReceiver[T, V orderedBlockChainData] func(cur V, inc T) V

func startSpan(ctx context.Context, plugin, op string) (*sentry.Span, context.Context) {
	return tracing.StartSpan(ctx, "indexer.plugin", fmt.Sprintf("%s:%s", plugin, op))
}

// NewTransferPlugins returns a set of transfer plugins. Plugins have an `in` and an optional `out` channel that are handles to the service.
// The `in` channel is used to submit a transfer to a plugin, and the `out` channel is used to receive results from a plugin, if any.
// A plugin can be stopped by closing its `in` channel, which finishes the plugin and lets receivers know that its done.
func NewTransferPlugins(ctx context.Context) TransferPlugins {
	ctx = sentryutil.NewSentryHubContext(ctx)
	return TransferPlugins{
		assets: newAssetsPlugin(ctx),
		tokens: newTokensPlugin(ctx),
	}
}

// RunTransferPlugins returns when all plugins have received the message. Every plugin recieves the same message.
func RunTransferPlugins(ctx context.Context, transfer rpc.Transfer, key persist.TokenChainAddress, plugins []chan<- TransferPluginMsg) {
	span, _ := tracing.StartSpan(ctx, "indexer.plugin", "submitMessage")
	defer tracing.FinishSpan(span)

	msg := TransferPluginMsg{
		transfer: transfer,
		key:      key,
	}
	for _, plugin := range plugins {
		plugin <- msg
	}
}

// RunTransferPluginReceiver runs a plugin receiver and will update the out map with the results of the receiver, ensuring that the most recent data is kept.
// If the incoming channel is nil, the function will return immediately.
func RunTransferPluginReceiver[T, V orderedBlockChainData](ctx context.Context, wg *sync.WaitGroup, mu *sync.Mutex, receiver TransferPluginReceiver[T, V], incoming <-chan T, out map[persist.TokenChainAddress]V) {
	span, _ := tracing.StartSpan(ctx, "indexer.plugin", "runPluginReceiver")
	defer tracing.FinishSpan(span)

	if incoming == nil {
		return
	}

	if out == nil {
		panic("out map must not be nil")
	}

	wg.Add(1)

	go func() {
		defer wg.Done()

		for it := range incoming {
			func() {
				processed := receiver(out[it.TokenIdentifiers()], it)
				cur, ok := out[processed.TokenIdentifiers()]
				if !ok || cur.OrderInfo().Less(processed.OrderInfo()) {
					mu.Lock()
					defer mu.Unlock()
					out[processed.TokenIdentifiers()] = processed
				}
			}()
		}

		logger.For(ctx).WithFields(logrus.Fields{"incoming_type": fmt.Sprintf("%T", *new(T)), "outgoing_type": fmt.Sprintf("%T", *new(V))}).Info("plugin finished receiving")
	}()

}

// assetTransfersPlugin retrieves ownership information for a token.
type assetTransfersPlugin struct {
	in  chan TransferPluginMsg
	out chan assetAtBlock
}

func newAssetsPlugin(ctx context.Context) assetTransfersPlugin {
	in := make(chan TransferPluginMsg)
	out := make(chan assetAtBlock)

	go func() {
		span, _ := startSpan(ctx, "assetTransfersPlugin", "handleBatch")
		defer tracing.FinishSpan(span)
		defer close(out)

		wp := workerpool.New(pluginPoolSize)

		seenContracts := map[persist.Address]bool{}

		for msg := range in {

			if seenContracts[msg.transfer.ContractAddress] {
				continue
			}

			msg := msg
			wp.Submit(func() {

				child := span.StartChild("plugin.contractTransfersPlugin")
				child.Description = "handleMessage"

				out <- assetAtBlock{
					ti: msg.key,
					boi: blockchainOrderInfo{
						blockNumber: msg.transfer.BlockNumber,
						txIndex:     msg.transfer.TxIndex,
					},
					asset: persist.Asset{
						ID:           "",
						Version:      0,
						LastUpdated:  persist.LastUpdatedTime{},
						CreationTime: persist.CreationTime{},
						OwnerAddress: "",
						Token:        persist.Token{ContractAddress: msg.transfer.ContractAddress},
						Balance:      0,
						BlockNumber:  0,
					},
				}

				tracing.FinishSpan(child)
			})

			seenContracts[msg.transfer.ContractAddress] = true
		}

		wp.StopWait()
		logger.For(ctx).Info("contracts plugin finished sending")
	}()

	return assetTransfersPlugin{
		in:  in,
		out: out,
	}
}

type tokenTransfersPlugin struct {
	in  chan TransferPluginMsg
	out chan tokenAtBlock
}

func newTokensPlugin(ctx context.Context) tokenTransfersPlugin {
	in := make(chan TransferPluginMsg)
	out := make(chan tokenAtBlock)

	go func() {
		span, _ := startSpan(ctx, "tokenTransfersPlugin", "handleBatch")
		defer tracing.FinishSpan(span)
		defer close(out)

		wp := workerpool.New(pluginPoolSize)

		seenTokens := map[persist.TokenChainAddress]bool{}

		for msg := range in {

			if seenTokens[msg.key] && msg.transfer.TokenType != persist.TokenTypeERC20 {
				continue
			}

			msg := msg
			wp.Submit(func() {

				child := span.StartChild("plugin.tokenTransfersPlugin")
				child.Description = "handleMessage"

				out <- tokenAtBlock{
					ti: msg.key,
					boi: blockchainOrderInfo{
						blockNumber: msg.transfer.BlockNumber,
						txIndex:     msg.transfer.TxIndex,
					},
					token: persist.Token{
						TokenType:       msg.transfer.TokenType,
						Chain:           persist.ChainETH,
						BlockNumber:     msg.transfer.BlockNumber,
						ContractAddress: msg.key.Address,
					},
				}

				tracing.FinishSpan(child)
			})

			seenTokens[msg.key] = true
		}

		wp.StopWait()
		logger.For(ctx).Info("tokens plugin finished sending")
	}()

	return tokenTransfersPlugin{
		in:  in,
		out: out,
	}
}
