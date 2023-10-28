package multichain

import (
	"context"
	"sync"

	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/persist"
)

// SyncWithContractEvalFallbackProvider will call its fallback if the primary Provider's token
// response is unsuitable based on Eval
type SyncWithContractEvalFallbackProvider struct {
	Primary  SyncWithContractEvalPrimary
	Fallback SyncWithContractEvalSecondary
	Eval     func(context.Context, persist.Token) bool
}

type SyncWithContractEvalPrimary interface {
	Configurer
	TokensOwnerFetcher
	TokensIncrementalOwnerFetcher
	TokensContractFetcher
	TokenDescriptorsFetcher
	TokenMetadataFetcher
}

type SyncWithContractEvalSecondary interface {
	TokensOwnerFetcher
	TokensIncrementalOwnerFetcher
}

// SyncFailureFallbackProvider will call its fallback if the primary Provider's token
// response fails (returns an error)
type SyncFailureFallbackProvider struct {
	Primary  SyncFailurePrimary
	Fallback SyncFailureSecondary
}

type SyncFailurePrimary interface {
	Configurer
	TokensOwnerFetcher
	TokensIncrementalOwnerFetcher
	TokenDescriptorsFetcher
	TokensContractFetcher
}

type SyncFailureSecondary interface {
	TokensOwnerFetcher
	TokenDescriptorsFetcher
}

func (f SyncWithContractEvalFallbackProvider) GetBlockchainInfo() BlockchainInfo {
	return f.Primary.GetBlockchainInfo()
}

func (f SyncWithContractEvalFallbackProvider) GetTokensByWalletAddress(ctx context.Context, address persist.Address) ([]persist.Token, error) {
	tokens, err := f.Primary.GetTokensByWalletAddress(ctx, address)
	if err != nil {
		return nil, err
	}
	tokens = f.resolveTokens(ctx, tokens)
	return tokens, nil
}

func (f SyncWithContractEvalFallbackProvider) GetTokensIncrementallyByWalletAddress(ctx context.Context, address persist.Address) (<-chan []persist.Token, <-chan error) {
	return getTokensIncrementallyByWalletAddressWithFallback(ctx, address, f.Primary, f.Fallback, f.resolveTokens)
}

func (f SyncWithContractEvalFallbackProvider) GetTokensByContractAddress(ctx context.Context, contract persist.Address, limit int, offset int) ([]persist.Token, error) {
	tokens, err := f.Primary.GetTokensByContractAddress(ctx, contract, limit, offset)
	if err != nil {
		return nil, err
	}
	tokens = f.resolveTokens(ctx, tokens)
	return tokens, nil
}

func (f SyncWithContractEvalFallbackProvider) GetTokenByTokenIdentifiersAndOwner(ctx context.Context, id persist.TokenChainAddress, address persist.Address) (persist.Token, error) {
	token, err := f.Primary.GetTokenByTokenIdentifiersAndOwner(ctx, id, address)
	if err != nil {
		return persist.Token{}, err
	}
	// TODO fill token metadata?
	//if !f.Eval(ctx, token) {
	//	token.TokenMetadata = f.callFallbackIdentifiers(ctx, token).TokenMetadata
	//}
	return token, nil
}

func (f SyncWithContractEvalFallbackProvider) GetTokensByContractAddressAndOwner(ctx context.Context, owner persist.Address, contractAddress persist.Address, limit int, offset int) ([]persist.Token, error) {
	tokens, err := f.Primary.GetTokensByContractAddressAndOwner(ctx, owner, contractAddress, limit, offset)
	if err != nil {
		return nil, err
	}
	tokens = f.resolveTokens(ctx, tokens)
	return tokens, err
}

func (f SyncWithContractEvalFallbackProvider) resolveTokens(ctx context.Context, tokens []persist.Token) []persist.Token {
	usableTokens := make([]persist.Token, len(tokens))
	var wg sync.WaitGroup

	for i, token := range tokens {
		wg.Add(1)
		go func(i int, token persist.Token) {
			defer wg.Done()
			usableTokens[i] = token
		}(i, token)
	}

	wg.Wait()

	return usableTokens
}

func (f *SyncWithContractEvalFallbackProvider) callFallbackIdentifiers(ctx context.Context, primary persist.Token) persist.Token {
	id := persist.TokenChainAddress{Address: persist.Address(primary.ContractAddress), Chain: 1}
	// TODO use OwnerAddress instead of ContractAddress for second parameter of f.Fallback.GetTokenByTokenIdentifiersAndOwner
	backup, err := f.Fallback.GetTokenByTokenIdentifiersAndOwner(ctx, id, persist.Address(primary.ContractAddress))
	if err == nil && f.Eval(ctx, backup) {
		return backup
	}
	logger.For(ctx).WithError(err).Warn("failed to call fallback")
	return primary
}

func (f SyncWithContractEvalFallbackProvider) GetTokenDescriptorsByTokenIdentifiers(ctx context.Context, id persist.TokenChainAddress) (persist.TokenMetadata, error) {
	return f.Primary.GetTokenDescriptorsByTokenIdentifiers(ctx, id)
}

func (f SyncWithContractEvalFallbackProvider) GetTokenMetadataByTokenIdentifiers(ctx context.Context, id persist.TokenChainAddress) (persist.TokenMetadata, error) {
	return f.Primary.GetTokenMetadataByTokenIdentifiers(ctx, id)
}

func (f SyncWithContractEvalFallbackProvider) GetSubproviders() []any {
	return []any{f.Primary, f.Fallback}
}

func (f SyncFailureFallbackProvider) GetBlockchainInfo() BlockchainInfo {
	return f.Primary.GetBlockchainInfo()
}

func (f SyncFailureFallbackProvider) GetTokensByWalletAddress(ctx context.Context, address persist.Address) ([]persist.Token, error) {
	tokens, err := f.Primary.GetTokensByWalletAddress(ctx, address)
	if err != nil {
		logger.For(ctx).WithError(err).Warn("failed to get tokens by wallet address from primary in failure fallback")
		return f.Fallback.GetTokensByWalletAddress(ctx, address)
	}

	return tokens, nil
}

func (f SyncFailureFallbackProvider) GetTokensIncrementallyByWalletAddress(ctx context.Context, address persist.Address) (<-chan []persist.Token, <-chan error) {
	return getTokensIncrementallyByWalletAddressWithFallback(ctx, address, f.Primary, f.Fallback, nil)
}

func getTokensIncrementallyByWalletAddressWithFallback(ctx context.Context, address persist.Address, primary TokensIncrementalOwnerFetcher, fallback any, processTokens func(context.Context, []persist.Token) []persist.Token) (<-chan []persist.Token, <-chan error) {
	rec := make(chan []persist.Token)
	errChan := make(chan error)

	go func() {
		defer close(rec)
		// create sub channels so that we can separately keep track of the results and errors and handle them here as opposed to the original receiving channels
		subRec, subErrChan := primary.GetTokensIncrementallyByWalletAddress(ctx, address)
		for {
			select {
			case err := <-subErrChan:
				// FIXIT maybe we return an error from the primary that specifies what tokens failed so we don't have to go through the whole process again on a fallback
				if tiof, ok := fallback.(TokensIncrementalOwnerFetcher); ok {
					logger.For(ctx).Warnf("failed to get tokens incrementally by wallet address from primary in failure fallback: %s", err)
					fallbackRec, fallbackErrChan := tiof.GetTokensIncrementallyByWalletAddress(ctx, address)
					for {
						select {
						case err := <-fallbackErrChan:
							logger.For(ctx).Warnf("failed to get tokens incrementally by wallet address from fallback in failure fallback: %s", err)
							errChan <- err
							return
						case tokens, ok := <-fallbackRec:
							if !ok {
								return
							}
							if processTokens != nil {
								tokens = processTokens(ctx, tokens)
							}
							rec <- tokens
						}
					}
				}
				errChan <- err
				return

			case tokens, ok := <-subRec:
				if !ok {
					return
				}
				if processTokens != nil {
					tokens = processTokens(ctx, tokens)
				}
				rec <- tokens
			}
		}
	}()
	return rec, errChan
}

func (f SyncFailureFallbackProvider) GetTokenByTokenIdentifiersAndOwner(ctx context.Context, id persist.TokenChainAddress, address persist.Address) (persist.Token, error) {
	token, err := f.Primary.GetTokenByTokenIdentifiersAndOwner(ctx, id, address)
	if err != nil {
		logger.For(ctx).WithError(err).Warn("failed to get token by token identifiers and owner from primary in failure fallback")
		return f.Fallback.GetTokenByTokenIdentifiersAndOwner(ctx, id, address)
	}
	return token, nil
}

func (f SyncFailureFallbackProvider) GetTokenDescriptorsByTokenIdentifiers(ctx context.Context, id persist.TokenChainAddress) (persist.TokenMetadata, error) {
	token, err := f.Primary.GetTokenDescriptorsByTokenIdentifiers(ctx, id)
	if err != nil {
		logger.For(ctx).WithError(err).Warn("failed to get token by token identifiers and owner from primary in failure fallback")
		return f.Fallback.GetTokenDescriptorsByTokenIdentifiers(ctx, id)
	}
	return token, nil
}

func (f SyncFailureFallbackProvider) GetTokensByContractAddress(ctx context.Context, contract persist.Address, limit int, offset int) ([]persist.Token, error) {
	ts, err := f.Primary.GetTokensByContractAddress(ctx, contract, limit, offset)
	if err != nil {
		logger.For(ctx).WithError(err).Warn("failed to get token by token identifiers and owner from primary in failure fallback")
		if tcf, ok := f.Fallback.(TokensContractFetcher); ok {
			return tcf.GetTokensByContractAddress(ctx, contract, limit, offset)
		}
		return nil, err
	}
	return ts, nil
}
func (f SyncFailureFallbackProvider) GetTokensByContractAddressAndOwner(ctx context.Context, owner persist.Address, contract persist.Address, limit int, offset int) ([]persist.Token, error) {
	ts, err := f.Primary.GetTokensByContractAddressAndOwner(ctx, owner, contract, limit, offset)
	if err != nil {
		logger.For(ctx).WithError(err).Warn("failed to get token by token identifiers and owner from primary in failure fallback")
		if tcf, ok := f.Fallback.(TokensContractFetcher); ok {
			return tcf.GetTokensByContractAddressAndOwner(ctx, owner, contract, limit, offset)
		}
		return nil, err
	}
	return ts, nil
}

func (f SyncFailureFallbackProvider) GetSubproviders() []any {
	return []any{f.Primary, f.Fallback}
}
