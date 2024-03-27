package wrapper

import (
	"context"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/multichain"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/util"
	"net/http"
	"sync"
	"time"
)

var (
	MultiProviderWapperOptions MultiProviderWrapperOpts
)

// SyncPipelineWrapper makes a best effort to fetch tokens requested by a sync.
// Specifically, SyncPipelineWrapper searches every applicable provider to find tokens and
// fills missing token fields with data from a supplemental provider.
type SyncPipelineWrapper struct {
	Chain persist.Chain
	// TODO for all fetchers: check if needed + check for more
	TokensOwnerFetcher            multichain.TokensOwnerFetcher
	TokensIncrementalOwnerFetcher multichain.TokensIncrementalOwnerFetcher
	TokensContractFetcher         multichain.TokensContractFetcher
	TokenDescriptorsFetcher       multichain.TokenDescriptorsFetcher
	TokenMetadataFetcher          multichain.TokenMetadataFetcher
	FillInWrapper                 *FillInWrapper
}

func NewSyncPipelineWrapper(
	ctx context.Context,
	chain persist.Chain,
	tokensOwnerFetcher multichain.TokensOwnerFetcher,
	tokensIncrementalOwnerFetcher multichain.TokensIncrementalOwnerFetcher,
	tokensContractFetcher multichain.TokensContractFetcher,
	tokenDescriptorsFetcher multichain.TokenDescriptorsFetcher,
	tokenMetadataFetcher multichain.TokenMetadataFetcher,
	fillInWrapper *FillInWrapper,
) *SyncPipelineWrapper {
	return &SyncPipelineWrapper{
		Chain:                         chain,
		TokensOwnerFetcher:            tokensOwnerFetcher,
		TokensIncrementalOwnerFetcher: tokensIncrementalOwnerFetcher,
		TokensContractFetcher:         tokensContractFetcher,
		TokenDescriptorsFetcher:       tokenDescriptorsFetcher,
		TokenMetadataFetcher:          tokenMetadataFetcher,
		FillInWrapper:                 fillInWrapper,
	}
}

func (w SyncPipelineWrapper) GetTokensByWalletAddress(ctx context.Context, address persist.Address) ([]persist.Token, error) {
	t, err := w.TokensOwnerFetcher.GetTokensByWalletAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	t = w.FillInWrapper.LoadAll(t)
	return t, nil
}

func (w SyncPipelineWrapper) GetTokenByTokenIdentifiersAndOwner(ctx context.Context, ti persist.TokenChainAddress, address persist.Address) (t persist.Token, err error) {
	t, err = w.TokensOwnerFetcher.GetTokenByTokenIdentifiersAndOwner(ctx, ti, address)
	t = w.FillInWrapper.AddToToken(ctx, t)
	return t, err
}

func (w SyncPipelineWrapper) GetTokensIncrementallyByWalletAddress(ctx context.Context, address persist.Address) (<-chan multichain.ChainAgnosticTokensAndContracts, <-chan error) {
	recCh, errCh := w.TokensIncrementalOwnerFetcher.GetTokensIncrementallyByWalletAddress(ctx, address)
	recCh, errCh = w.FillInWrapper.AddToPage(ctx, recCh, errCh)
	return recCh, errCh
}

func (w SyncPipelineWrapper) GetTokensByContractAddress(ctx context.Context, address persist.Address, maxLimit int) (<-chan multichain.ChainAgnosticTokensAndContracts, <-chan error) {
	// TODO change to GetTokensIncrementallyByContractAddress and add offset
	recCh, errCh := w.TokensContractFetcher.GetTokensByContractAddress(ctx, address, maxLimit, 0)
	recCh, errCh = w.FillInWrapper.AddToPage(ctx, recCh, errCh)
	return recCh, errCh
}

/*
	func (w SyncPipelineWrapper) GetTokensByTokenIdentifiers(ctx context.Context, ti multichain.ChainAgnosticIdentifiers) ([]multichain.ChainAgnosticToken, multichain.ChainAgnosticContract, error) {
		t, c, err := w.TokensOwnerFetcher.GetTokensByTokenIdentifiers(ctx, ti)
		t = w.CustomMetadataWrapper.LoadAll(ctx, w.Chain, t)
		t = w.FillInWrapper.LoadAll(t)
		return t, c, err
	}
*/

/*func (w SyncPipelineWrapper) GetTokenMetadataByTokenIdentifiersBatch(ctx context.Context, tIDs []multichain.ChainAgnosticIdentifiers) ([]persist.TokenMetadata, error) {
	ret := make([]persist.TokenMetadata, len(tIDs))
	noCustomHandlerBatch := make([]multichain.ChainAgnosticIdentifiers, 0, len(tIDs))
	noCustomHandlerResultIdxToInputIdx := make(map[int]int)

	// Separate tokens that have custom metadata handlers from those that don't
	for i, tID := range tIDs {
		t := multichain.ChainAgnosticIdentifiers{ContractAddress: tID.ContractAddress, TokenID: tID.TokenID}
		metadata := w.CustomMetadataWrapper.Load(ctx, w.Chain, t)
		if len(metadata) > 0 {
			ret[i] = metadata
			continue
		}
		pos := len(noCustomHandlerBatch)
		noCustomHandlerBatch = append(noCustomHandlerBatch, tID)
		noCustomHandlerResultIdxToInputIdx[pos] = i
	}

	// Fetch metadata for tokens that don't have custom metadata handlers
	if len(noCustomHandlerBatch) > 0 {
		batchMetadata, err := w.TokenMetadataBatcher.GetTokenMetadataByTokenIdentifiersBatch(ctx, noCustomHandlerBatch)
		if err != nil {
			logger.For(ctx).Errorf("error fetching metadata for batch: %s", err)
			sentryutil.ReportError(ctx, err)
		} else {
			if len(batchMetadata) != len(noCustomHandlerBatch) {
				panic(fmt.Sprintf("expected length to the the same; expected=%d; got=%d", len(noCustomHandlerBatch), len(batchMetadata)))
			}
			for i := range batchMetadata {
				ret[noCustomHandlerResultIdxToInputIdx[i]] = batchMetadata[i]
			}
		}
	}

	// Convert metadata to tokens to fill in missing data
	asTokens := make([]multichain.ChainAgnosticToken, len(tIDs))
	for i, tID := range tIDs {
		asTokens[i] = multichain.ChainAgnosticToken{
			ContractAddress: tID.ContractAddress,
			TokenID:         tID.TokenID,
			TokenMetadata:   ret[i],
		}
	}

	ret = w.FillInWrapper.LoadMetadataAll(asTokens)
	return ret, nil
}
*/

// MultiProviderWrapperOpts configures options for the MultiProviderWrapper
type MultiProviderWrapperOpts struct{}

func (o MultiProviderWrapperOpts) WithTokensIncrementalOwnerFetchers(a, b multichain.TokensIncrementalOwnerFetcher) func(*MultiProviderWrapper) {
	return func(m *MultiProviderWrapper) {
		m.TokensIncrementalOwnerFetchers = [2]multichain.TokensIncrementalOwnerFetcher{a, b}
	}
}

func (o MultiProviderWrapperOpts) WithTokensOwnerFetchers(a, b multichain.TokensOwnerFetcher) func(*MultiProviderWrapper) {
	return func(m *MultiProviderWrapper) {
		m.TokenIdentifierOwnerFetchers = [2]multichain.TokensOwnerFetcher{a, b}
	}
}

func (o MultiProviderWrapperOpts) WithTokensContractFetchers(a, b multichain.TokensContractFetcher) func(*MultiProviderWrapper) {
	return func(m *MultiProviderWrapper) { m.ContractFetchers = [2]multichain.TokensContractFetcher{a, b} }
}

func (o MultiProviderWrapperOpts) WithTokenDescriptorsFetchers(a, b multichain.TokenDescriptorsFetcher) func(*MultiProviderWrapper) {
	return func(m *MultiProviderWrapper) {
		m.TokenDescriptorsFetchers = [2]multichain.TokenDescriptorsFetcher{a, b}
	}
}

func (o MultiProviderWrapperOpts) WithTokenMetadataFetchers(a, b multichain.TokenMetadataFetcher) func(*MultiProviderWrapper) {
	return func(m *MultiProviderWrapper) { m.TokenMetadataFetchers = [2]multichain.TokenMetadataFetcher{a, b} }
}

/*func (o MultiProviderWrapperOpts) WithTokensIncrementalContractFetchers(a, b multichain.TokensIncrementalContractFetcher) func(*MultiProviderWrapper) {
	return func(m *MultiProviderWrapper) {
		m.TokensIncrementalContractFetchers = [2]multichain.TokensIncrementalContractFetcher{a, b}
	}
}
*/

/*func (o MultiProviderWrapperOpts) WithTokenByTokenIdentifiersFetchers(a, b multichain.TokensByTokenIdentifiersFetcher) func(*MultiProviderWrapper) {
	return func(m *MultiProviderWrapper) {
		m.TokensByTokenIdentifiersFetchers = [2]multichain.TokensByTokenIdentifiersFetcher{a, b}
	}
}
*/

// MultiProviderWrapper handles calling into multiple providers. Depending on the calling context, providers are called in parallel or in series.
// In some cases, the first provider to return a result is used, in others, the results are combined.
type MultiProviderWrapper struct {
	TokensOwnerFetchers            [2]multichain.TokensOwnerFetcher
	TokensIncrementalOwnerFetchers [2]multichain.TokensIncrementalOwnerFetcher
	TokensContractFetcher          [2]multichain.TokensContractFetcher
	TokenDescriptorsFetchers       [2]multichain.TokenDescriptorsFetcher
	TokenMetadataFetchers          [2]multichain.TokenMetadataFetcher
}

func NewMultiProviderWrapper(opts ...func(*MultiProviderWrapper)) *MultiProviderWrapper {
	m := &MultiProviderWrapper{}
	for _, o := range opts {
		o(m)
	}
	return m
}

func (m MultiProviderWrapper) GetContractByAddress(ctx context.Context, address persist.Address) (c multichain.ChainAgnosticContract, err error) {
	for _, f := range m.ContractFetchers {
		if c, err = f.GetContractByAddress(ctx, address); err == nil {
			return
		}
	}
	return
}

func (m MultiProviderWrapper) GetTokenByTokenIdentifiersAndOwner(ctx context.Context, ti persist.TokenChainAddress, address persist.Address) (t persist.Token, err error) {
	for _, f := range m.TokensOwnerFetchers {
		if t, err = f.GetTokenByTokenIdentifiersAndOwner(ctx, ti, address); err == nil {
			return
		}
	}
	return
}

// TODO GetAssetByTokenIdentifiersAndOwner

/*
	func (m MultiProviderWrapper) GetAssetByTokenIdentifiersAndOwner(ctx context.Context, address persist.Address, maxLimit int) (<-chan multichain.ChainAgnosticTokensAndContracts, <-chan error) {
		recCh := make(chan multichain.ChainAgnosticTokensAndContracts, 2*10)
		errCh := make(chan error, 2)
		resultA, errA := m.TokensIncrementalContractFetchers[0].GetTokensIncrementallyByContractAddress(ctx, address, maxLimit)
		resultB, errB := m.TokensIncrementalContractFetchers[1].GetTokensIncrementallyByContractAddress(ctx, address, maxLimit)
		go func() { fanIn(ctx, recCh, errCh, resultA, resultB, errA, errB) }()
		return recCh, errCh
	}
*/
/*
func (m MultiProviderWrapper) GetTokenDescriptorsByTokenIdentifiers(ctx context.Context, ti persist.TokenChainAddress) (t persist.TokenMetadata, err error) {
	for _, f := range m.TokenDescriptorsFetchers {
		if t, err = f.GetTokenDescriptorsByTokenIdentifiers(ctx, ti); err == nil {
			return
		}
	}
	return
}
*/
func (m MultiProviderWrapper) GetTokenMetadataByTokenIdentifiers(ctx context.Context, ti persist.TokenChainAddress) (tm persist.TokenMetadata, err error) {
	for _, f := range m.TokenMetadataFetchers {
		if tm, err = f.GetTokenMetadataByTokenIdentifiers(ctx, ti); err == nil {
			return
		}
	}
	return
}

func (m MultiProviderWrapper) GetTokensByWalletAddress(ctx context.Context, address persist.Address) (<-chan []persist.Token, <-chan error) {
	recCh := make(chan []persist.Token)
	errCh := make(chan error, 2)
	resultA, errA := m.TokensIncrementalOwnerFetchers[0].GetTokensIncrementallyByWalletAddress(ctx, address)
	resultB, errB := m.TokensIncrementalOwnerFetchers[1].GetTokensIncrementallyByWalletAddress(ctx, address)
	go func() { fanIn(ctx, recCh, errCh, resultA, resultB, errA, errB) }()
	return recCh, errCh
}

func fanIn(ctx context.Context, recCh chan<- []persist.Token, errCh chan<- error, resultA, resultB <-chan []persist.Token, errA, errB <-chan error) {
	defer close(recCh)
	defer close(errCh)

	var closingA bool
	var closingB bool

	// It's possible for one provider to not have a contract that the other does. We won't
	// stop pulling data unless neither provider has the contract.
	missing := make(map[persist.TokenChainAddress]bool)

	for {
		select {
		case page, ok := <-resultA:
			if ok {
				recCh <- page
				continue
			}
			if closingB {
				return
			}
			closingA = true
		case page, ok := <-resultB:
			if ok {
				recCh <- page
				continue
			}
			if closingA {
				return
			}
			closingB = true
		case err, ok := <-errA:
			if !ok {
				continue
			}

			if err, ok := util.ErrorAs[multichain.ErrProviderContractNotFound](err); ok {
				logger.For(ctx).Warnf("primary provider could not find contract: %s", err)
				c := persist.NewTokenChainAddress(err.Contract, err.Chain)
				if missing[c] {
					errCh <- err
				} else {
					missing[c] = true
				}
				continue
			}

			errCh <- err
		case err, ok := <-errB:
			if !ok {
				continue
			}

			if err, ok := util.ErrorAs[multichain.ErrProviderContractNotFound](err); ok {
				logger.For(ctx).Warnf("secondary provider could not find contract: %s", err)
				c := persist.NewContractIdentifiers(err.Contract, err.Chain)
				if missing[c] {
					errCh <- err
				} else {
					missing[c] = true
				}
				continue
			}

			errCh <- err
		}
	}
}

// FillInWrapper is a service for adding missing data to tokens.
// Batching pattern adapted from dataloaden (https://github.com/vektah/dataloaden)
type FillInWrapper struct {
	chain       persist.Chain
	ctx         context.Context
	mu          sync.Mutex
	batch       *batch
	wait        time.Duration
	maxBatch    int
	resultCache sync.Map
}

func NewFillInWrapper(ctx context.Context, httpClient *http.Client, chain persist.Chain, l retry.Limiter) (*FillInWrapper, func()) {
	r, cleanup := reservoir.NewProvider(ctx, httpClient, chain, l)
	return &FillInWrapper{
		chain:             chain,
		reservoirProvider: r,
		ctx:               ctx,
		wait:              250 * time.Millisecond,
		maxBatch:          10,
	}, cleanup
}

// AddToToken adds missing data to a token.
func (w *FillInWrapper) AddToToken(ctx context.Context, t persist.Token) persist.Token {
	return w.addToken(t)()
}

// AddToPage adds missing data to each token of a provider page.
func (w *FillInWrapper) AddToPage(ctx context.Context, recCh <-chan multichain.ChainAgnosticTokensAndContracts, errIn <-chan error) (<-chan multichain.ChainAgnosticTokensAndContracts, <-chan error) {
	outCh := make(chan multichain.ChainAgnosticTokensAndContracts, 2*10)
	errOut := make(chan error)
	w.resultCache = sync.Map{}
	go func() {
		defer close(outCh)
		defer close(errOut)
		for {
			select {
			case page, ok := <-recCh:
				if !ok {
					return
				}
				outCh <- w.addPage(page)()
			case err, ok := <-errIn:
				if ok {
					errOut <- err
				}
			case <-ctx.Done():
				errOut <- ctx.Err()
				return
			}
		}
	}()
	logger.For(ctx).Info("finished filling in page")
	return outCh, errOut
}

// LoaddAll fills in missing data for a slice of tokens.
func (w *FillInWrapper) LoadAll(tokens []persist.Token) []persist.Token {
	thunks := make([]func() multichain.ChainAgnosticToken, len(tokens))
	for i, t := range tokens {
		t := t
		thunks[i] = w.addTokenToBatch(t)
	}
	result := make([]persist.Token, len(tokens))
	for i, thunk := range thunks {
		result[i] = thunk()
	}
	return result
}

// LoadMetadataAll returns missing metadata for a slice of tokens.
func (w *FillInWrapper) LoadMetadataAll(tokens []multichain.ChainAgnosticToken) []persist.TokenMetadata {
	thunks := make([]func() multichain.ChainAgnosticToken, len(tokens))
	for i, t := range tokens {
		t := t
		if hasMediaURLs(t.TokenMetadata, w.chain) {
			thunks[i] = func() multichain.ChainAgnosticToken {
				w.cacheTokenResult(t)
				return t
			}
		} else {
			thunks[i] = w.addTokenToBatch(t)
		}
	}
	result := make([]persist.TokenMetadata, len(tokens))
	for i, thunk := range thunks {
		r := thunk()
		result[i] = r.TokenMetadata
	}
	return result
}

// LoadFallbackAll returns missing fallback media for a slice of tokens.
func (w *FillInWrapper) LoadFallbackAll(tokens []multichain.ChainAgnosticToken) []persist.FallbackMedia {
	thunks := make([]func() multichain.ChainAgnosticToken, len(tokens))
	for i, t := range tokens {
		t := t
		if t.FallbackMedia.IsServable() {
			thunks[i] = func() multichain.ChainAgnosticToken {
				w.cacheTokenResult(t)
				return t
			}
		} else {
			thunks[i] = w.addTokenToBatch(t)
		}
	}
	result := make([]persist.FallbackMedia, len(tokens))
	for i, thunk := range thunks {
		r := thunk()
		result[i] = r.FallbackMedia
	}
	return result
}

func (w *FillInWrapper) addPage(p multichain.ChainAgnosticTokensAndContracts) func() multichain.ChainAgnosticTokensAndContracts {
	thunks := make([]func() multichain.ChainAgnosticToken, len(p.Tokens))
	for i, t := range p.Tokens {
		thunks[i] = w.addToken(t)
	}
	return func() multichain.ChainAgnosticTokensAndContracts {
		for i, thunk := range thunks {
			p.Tokens[i] = thunk()
		}
		return p
	}
}

func (w *FillInWrapper) addToken(t persist.Token) func() persist.Token {
	if hasMediaURLs(t.TokenMetadata, w.chain) && t.FallbackMedia.IsServable() {
		return func() multichain.ChainAgnosticToken {
			w.cacheTokenResult(t)
			return t
		}
	}
	return w.addTokenToBatch(t)
}

func (w *FillInWrapper) cacheTokenResult(t multichain.ChainAgnosticToken) {
	tID := persist.NewTokenIdentifiers(t.ContractAddress, t.TokenID, w.chain)
	w.resultCache.Store(tID, t)
}

func (w *FillInWrapper) addTokenToBatch(t persist.Token) func() persist.Token {
	ti := multichain.ChainAgnosticIdentifiers{ContractAddress: t.ContractAddress, TokenID: t.TokenID}

	if v, ok := w.resultCache.Load(ti); ok {
		return func() persist.Token {
			f := v.(persist.Token)
			if !t.FallbackMedia.IsServable() {
				t.FallbackMedia = f.FallbackMedia
			}
			if !hasMediaURLs(t.TokenMetadata, w.chain) {
				t.TokenMetadata = f.TokenMetadata
			}
			return t
		}
	}

	w.mu.Lock()

	if w.batch == nil {
		w.batch = &batch{done: make(chan struct{})}
	}
	b := w.batch
	pos := b.addToBatch(w, ti)

	w.mu.Unlock()

	return func() persist.Token {
		<-b.done
		if b.errors[pos] != nil {
			return t
		}
		if !t.FallbackMedia.IsServable() {
			t.FallbackMedia = b.results[pos].FallbackMedia
		}
		if !hasMediaURLs(t.TokenMetadata, w.chain) {
			t.TokenMetadata = b.results[pos].TokenMetadata
		}
		return t
	}
}

func hasMediaURLs(metadata persist.TokenMetadata, chain persist.Chain) bool {
	_, _, err := media.FindMediaURLsChain(metadata, chain)
	return err == nil
}

type batch struct {
	tokens  []multichain.ChainAgnosticIdentifiers
	errors  []error
	results []multichain.ChainAgnosticToken
	closing bool
	done    chan struct{}
}

func (b *batch) addToBatch(w *FillInWrapper, t multichain.ChainAgnosticIdentifiers) int {
	pos := len(b.tokens)
	b.tokens = append(b.tokens, t)
	if pos == 0 {
		go b.startTimer(w)
	}

	if w.maxBatch != 0 && pos >= w.maxBatch-1 {
		if !b.closing {
			b.closing = true
			w.batch = nil
			go b.end(w)
		}
	}

	return pos
}

func (b *batch) startTimer(w *FillInWrapper) {
	time.Sleep(w.wait)
	w.mu.Lock()

	// we must have hit a batch limit and are already finalizing this batch
	if b.closing {
		w.mu.Unlock()
		return
	}

	w.batch = nil
	w.mu.Unlock()

	b.end(w)
}

func (b *batch) end(w *FillInWrapper) {
	ctx, cancel := context.WithTimeout(w.ctx, 10*time.Second)
	defer cancel()
	b.results, b.errors = w.reservoirProvider.GetTokensByTokenIdentifiersBatch(ctx, b.tokens)
	for i := range b.results {
		if b.errors[i] == nil {
			w.resultCache.Store(b.tokens[i], b.results[i])
		}
	}
	close(b.done)
}
