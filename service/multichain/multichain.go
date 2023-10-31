package multichain

import (
	"context"
	"fmt"
	"github.com/SplitFi/go-splitfi/env"
	"github.com/SplitFi/go-splitfi/service/persist/postgres"
	"github.com/SplitFi/go-splitfi/service/redis"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/sirupsen/logrus"
	"github.com/sourcegraph/conc"
	"sort"
	"time"

	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/persist"
)

func init() {
	env.RegisterValidation("TOKEN_PROCESSING_URL", "required")
}

const staleCommunityTime = time.Minute * 30

const maxCommunitySize = 10_000

var contractNameBlacklist = map[string]bool{
	"unidentified contract": true,
	"unknown contract":      true,
	"unknown":               true,
}

type Provider struct {
	Repos   *postgres.Repositories
	Queries *db.Queries
	Cache   *redis.Cache
	Chains  map[persist.Chain][]any
}

// BlockchainInfo retrieves blockchain info from all chains
type BlockchainInfo struct {
	Chain      persist.Chain `json:"chain_name"`
	ChainID    int           `json:"chain_id"`
	ProviderID string        `json:"provider_id"`
}

// ChainAgnosticToken is a token that is agnostic to the chain it is on
type ChainAgnosticToken struct {
	Descriptors ChainAgnosticTokenDescriptors `json:"descriptors"`

	TokenType persist.TokenType `json:"token_type"`

	Quantity         persist.HexString             `json:"quantity"`
	OwnerAddress     persist.Address               `json:"owner_address"`
	OwnershipHistory []ChainAgnosticAddressAtBlock `json:"previous_owners"`
	TokenMetadata    persist.TokenMetadata         `json:"metadata"`
	ContractAddress  persist.Address               `json:"contract_address"`

	ExternalURL string `json:"external_url"`

	BlockNumber persist.BlockNumber `json:"block_number"`
	IsSpam      *bool               `json:"is_spam"`
}

// ChainAgnosticAddressAtBlock is an address at a block
type ChainAgnosticAddressAtBlock struct {
	Address persist.Address     `json:"address"`
	Block   persist.BlockNumber `json:"block"`
}

// ChainAgnosticTokenDescriptors are the fields that describe a token but cannot be used to uniquely identify it
type ChainAgnosticTokenDescriptors struct {
	Name     string            `json:"name"`
	Symbol   string            `json:"symbol"`
	Logo     string            `json:"logo"`
	Decimals persist.NullInt32 `json:"decimals"`
}

type ChainAgnosticCommunityOwner struct {
	Address persist.Address `json:"address"`
}

type TokenHolder struct {
	UserID        persist.DBID    `json:"user_id"`
	DisplayName   string          `json:"display_name"`
	Address       persist.Address `json:"address"`
	WalletIDs     []persist.DBID  `json:"wallet_ids"`
	PreviewTokens []string        `json:"preview_tokens"`
}

type chainAssets struct {
	priority int
	chain    persist.Chain
	assets   []persist.Asset
}

type errWithPriority struct {
	err      error
	priority int
}

func (e errWithPriority) Error() string {
	return fmt.Sprintf("error with priority %d: %s", e.priority, e.err)
}

// Configurer maintains provider settings
type Configurer interface {
	GetBlockchainInfo() BlockchainInfo
}

// NameResolver is able to resolve an address to a friendly display name
type NameResolver interface {
	GetDisplayNameByAddress(context.Context, persist.Address) string
}

// Verifier can verify that a signature is signed by a given key
type Verifier interface {
	VerifySignature(ctx context.Context, pubKey persist.PubKey, walletType persist.WalletType, nonce string, sig string) (bool, error)
}

// TokensOwnerFetcher supports fetching tokens for syncing
type TokensOwnerFetcher interface {
	GetTokensByWalletAddress(ctx context.Context, address persist.Address) ([]persist.Token, error)
	GetTokenByTokenIdentifiersAndOwner(context.Context, persist.TokenChainAddress, persist.Address) (persist.Token, error)
	GetAssetByTokenIdentifiersAndOwner(context.Context, persist.TokenChainAddress, persist.Address) (persist.Asset, error)
}

// TokensIncrementalOwnerFetcher supports fetching tokens for syncing incrementally
type TokensIncrementalOwnerFetcher interface {
	// GetTokensIncrementallyByWalletAddress NOTE: implementation MUST close the rec channel
	GetTokensIncrementallyByWalletAddress(ctx context.Context, address persist.Address) (rec <-chan []persist.Token, errChain <-chan error)
}

type TokensContractFetcher interface {
	GetTokensByContractAddress(ctx context.Context, contract persist.Address, limit int, offset int) ([]persist.Token, error)
	GetTokensByContractAddressAndOwner(ctx context.Context, owner persist.Address, contract persist.Address, limit int, offset int) ([]persist.Token, error)
}

// TokenMetadataFetcher supports fetching token metadata
type TokenMetadataFetcher interface {
	GetTokenMetadataByTokenIdentifiers(ctx context.Context, ti persist.TokenChainAddress) (persist.TokenMetadata, error)
}

type TokenDescriptorsFetcher interface {
	GetTokenDescriptorsByTokenIdentifiers(ctx context.Context, ti persist.TokenChainAddress) (persist.TokenMetadata, error)
}

type ProviderSupplier interface {
	GetSubproviders() []any
}

// providersMatchingInterface returns providers that adhere to the given interface
func providersMatchingInterface[T any](providers []any) []T {
	matches := make([]T, 0)
	seen := map[string]bool{}
	for _, p := range providers {
		match, ok := p.(T)
		if !ok {
			continue
		}

		if id := p.(Configurer).GetBlockchainInfo().ProviderID; !seen[id] {
			seen[id] = true
			matches = append(matches, match)
		}

		// If the provider has subproviders, make sure we don't add them later
		if ps, ok := p.(ProviderSupplier); ok {
			for _, sp := range ps.GetSubproviders() {
				if id := sp.(Configurer).GetBlockchainInfo().ProviderID; !seen[id] {
					seen[id] = true
				}
			}
		}
	}
	return matches
}

// matchingProvidersByChains returns providers that adhere to the given interface by chain
func matchingProvidersByChains[T any](availableProviders map[persist.Chain][]any, requestedChains ...persist.Chain) map[persist.Chain][]T {
	matches := make(map[persist.Chain][]T, 0)
	for _, chain := range requestedChains {
		matching := providersMatchingInterface[T](availableProviders[chain])
		matches[chain] = matching
	}
	return matches
}

func matchingProvidersForChain[T any](availableProviders map[persist.Chain][]any, chain persist.Chain) []T {
	return matchingProvidersByChains[T](availableProviders, chain)[chain]
}

// matchingWallets returns wallet addresses that belong to any of the passed chains
func (p *Provider) matchingWallets(wallets []persist.Wallet, chains []persist.Chain) map[persist.Chain][]persist.Address {
	matches := make(map[persist.Chain][]persist.Address)
	for _, chain := range chains {
		for _, wallet := range wallets {
			if wallet.Chain == chain {
				matches[chain] = append(matches[chain], wallet.Address)
			} else if overrides := wallet.Chain.L1ChainGroup(); util.Contains(overrides, wallet.Chain) {
				matches[chain] = append(matches[chain], wallet.Address)
			}
		}
	}
	for chain, addresses := range matches {
		matches[chain] = util.Dedupe(addresses, true)
	}
	return matches
}

// matchingWalletsChain returns a list of wallets that match the given chain
func (p *Provider) matchingWalletsChain(wallets []persist.Wallet, chain persist.Chain) []persist.Address {
	return p.matchingWallets(wallets, []persist.Chain{chain})[chain]
}

// SyncAssetsByOwnerAndTokenChainAddress updates the token balances for specific tokens for an owner
func (p *Provider) SyncAssetsByOwnerAndTokenChainAddress(ctx context.Context, ownerAddress persist.Address, tokenIdentifiers []persist.TokenChainAddress) ([]persist.Asset, error) {

	ctx = logger.NewContextWithFields(ctx, logrus.Fields{"tids": tokenIdentifiers, "owner_address": ownerAddress})

	chains, _ := util.Map(tokenIdentifiers, func(i persist.TokenChainAddress) (persist.Chain, error) {
		return i.Chain, nil
	})

	chains = util.Dedupe(chains, false)

	errChan := make(chan error)
	incomingAssets := make(chan chainAssets)
	chainsToTokenIdentifiers := make(map[persist.Chain][]persist.TokenChainAddress)
	for _, tid := range tokenIdentifiers {
		chainsToTokenIdentifiers[tid.Chain] = append(chainsToTokenIdentifiers[tid.Chain], tid)
	}

	for c, a := range chainsToTokenIdentifiers {
		chainsToTokenIdentifiers[c] = util.Dedupe(a, false)
	}

	wg := &conc.WaitGroup{}
	for chain, tids := range chainsToTokenIdentifiers {
		logger.For(ctx).Infof("syncing %d chain %d tokens for owner address %s", len(tids), chain, ownerAddress.String())
		tokenFetchers := matchingProvidersForChain[TokensOwnerFetcher](p.Chains, chain)
		wg.Go(func() {
			subWg := &conc.WaitGroup{}
			for i, p := range tokenFetchers {
				innerIncomingAssets := make(chan persist.Asset)
				innerErrChan := make(chan error)
				assets := make([]persist.Asset, 0, len(tids))
				fetcher := p
				priority := i
				for _, tid := range tids {
					subWg.Go(func() {
						// TODO fetch token, but only if its new
						// token, err := fetcher.GetTokenByTokenIdentifiers(ctx, tid)
						asset, err := fetcher.GetAssetByTokenIdentifiersAndOwner(ctx, tid, ownerAddress)
						if err != nil {
							innerErrChan <- err
							return
						}
						innerIncomingAssets <- asset
					})
				}
				for i := 0; i < len(tids)*2; i++ {
					select {
					case asset := <-innerIncomingAssets:
						assets = append(assets, asset)
					case err := <-innerErrChan:
						errChan <- errWithPriority{err: err, priority: priority}
						return
					}
				}
				incomingAssets <- chainAssets{chain: chain, assets: assets, priority: priority}
			}
			subWg.Wait()
		})

	}

	go func() {
		defer close(incomingAssets)
		wg.Wait()
	}()

	return p.receiveSyncedAssetsForOwner(ctx, ownerAddress, chains, incomingAssets, errChan)
}

func (p *Provider) receiveSyncedAssetsForOwner(ctx context.Context, ownerAddress persist.Address, chains []persist.Chain, incomingAssets chan chainAssets, errChan chan error) ([]persist.Asset, error) {
	assetsFromProviders := make([]chainAssets, 0, len(chains))

	errs := []error{}
	discrepencyLog := map[int]int{}

outer:
	for {
		select {
		case incomingAssets, ok := <-incomingAssets:
			discrepencyLog[incomingAssets.priority] = len(incomingAssets.assets)
			assetsFromProviders = append(assetsFromProviders, incomingAssets)
			if !ok {
				// TODO check if breaking the loop over here is allowed
				break outer
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-errChan:
			logger.For(ctx).Errorf("error while syncing tokens for owner address %s: %s", ownerAddress, err)
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 && len(assetsFromProviders) == 0 {
		return nil, util.MultiErr(errs)
	}
	if !util.AllEqual(util.MapValues(discrepencyLog)) {
		logger.For(ctx).Debugf("discrepency: %+v", discrepencyLog)
	}

	_, newAssets, err := p.processTokensForUser(ctx, ownerAddress, assetsFromProviders, chains)

	if err != nil {
		return nil, err
	}

	return newAssets, nil
}

// VerifySignature verifies a signature for a wallet address
func (p *Provider) VerifySignature(ctx context.Context, pSig string, pNonce string, pChainAddress persist.ChainPubKey, pWalletType persist.WalletType) (bool, error) {
	verifiers := matchingProvidersForChain[Verifier](p.Chains, pChainAddress.Chain())
	for _, verifier := range verifiers {
		if valid, err := verifier.VerifySignature(ctx, pChainAddress.PubKey(), pWalletType, pNonce, pSig); err != nil || !valid {
			return false, err
		}
	}
	return true, nil
}

// RefreshToken refreshes a token on the given chain using the chain provider for that chain
func (p *Provider) RefreshToken(ctx context.Context, ti persist.TokenChainAddress) error {
	//err := p.processTokenMedia(ctx, ti.TokenID, ti.ContractAddress, ti.Chain)
	//if err != nil {
	//	return err
	//}
	return p.RefreshTokenDescriptorsByTokenIdentifiers(ctx, ti)
}

// RefreshTokenDescriptorsByTokenIdentifiers will refresh the token descriptors for a token by its identifiers.
func (p *Provider) RefreshTokenDescriptorsByTokenIdentifiers(ctx context.Context, ti persist.TokenChainAddress) error {
	finalTokenDescriptors := ChainAgnosticTokenDescriptors{}
	tokenFetchers := matchingProvidersForChain[TokenDescriptorsFetcher](p.Chains, ti.Chain)
	tokenExists := false

	for _, tokenFetcher := range tokenFetchers {

		_, err := tokenFetcher.GetTokenDescriptorsByTokenIdentifiers(ctx, ti)
		// token, err := tokenFetcher.GetTokenDescriptorsByTokenIdentifiers(ctx, ti)
		if err == nil {
			tokenExists = true
			// TODO add token metadata
			//if token.Name != "" && !contractNameBlacklist[strings.ToLower(token.Name)] {
			//	finalTokenDescriptors.Name = token.Name
			//}
			//if token.Symbol != "" {
			//	finalTokenDescriptors.Symbol = token.Symbol
			//}
			//if token.Logo != "" {
			//	finalTokenDescriptors.Logo = token.Logo
			//}
			//if token.Decimals >= 0 {
			//	finalTokenDescriptors.Decimals = token.Decimals
			//}
		} else {
			logger.For(ctx).Infof("token %s not found for refresh (err: %s)", ti.String(), err)
		}
	}

	if !tokenExists {
		return persist.ErrTokenNotFoundByTokenIdentifiers{ContractAddress: persist.EthereumAddress(ti.Address)}
	}

	return p.Queries.UpdateTokenMetadataFieldsByChainAddress(ctx, db.UpdateTokenMetadataFieldsByChainAddressParams{
		Name:   util.ToNullString(finalTokenDescriptors.Name, true),
		Symbol: util.ToNullString(finalTokenDescriptors.Symbol, true),
		Logo:   util.ToNullString(finalTokenDescriptors.Logo, true),
		// Decimals:        finalTokenDescriptors.Decimals,
		ContractAddress: persist.Address(ti.Chain.NormalizeAddress(ti.Address)),
		Chain:           ti.Chain,
	})
}

func (p *Provider) processTokensForUser(ctx context.Context, ownerAddress persist.Address, assetsFromProviders []chainAssets, chains []persist.Chain) ([]persist.Asset, []persist.Asset, error) {
	existingAssets, err := p.Repos.AssetRepository.GetByOwner(ctx, persist.EthereumAddress(ownerAddress), 0, 0)
	if err != nil {
		return nil, nil, err
	}

	wallets := []persist.Address{ownerAddress}
	providerAssetMap := map[persist.Address][]chainAssets{ownerAddress: assetsFromProviders}
	existingAssetMap := map[persist.Address][]persist.Asset{ownerAddress: existingAssets}

	persistedAssets, newAssets, err := p.processTokensForUsers(ctx, wallets, providerAssetMap, existingAssetMap, chains)
	if err != nil {
		return nil, nil, err
	}

	return persistedAssets[ownerAddress], newAssets[ownerAddress], nil
}

func (p *Provider) prepTokensForTokenProcessing(ctx context.Context, assetsFromProviders []chainAssets, existingAssets []persist.Asset, walletAddress persist.Address) ([]persist.Asset, map[persist.TokenChainAddress]bool, error) {
	providerAssets, _ := tokensToNewDedupedTokens(assetsFromProviders, walletAddress)

	// Extract new assets for given owner by their absence in existingAssets
	// assetLookup allows for finding a token in existingAssets in O(1)
	assetLookup := make(map[persist.TokenChainAddress]persist.Asset)
	for _, asset := range existingAssets {
		assetLookup[persist.NewTokenChainAddress(persist.Address(asset.Token.ContractAddress), asset.Token.Chain)] = asset
	}

	newTokensMap := make(map[persist.TokenChainAddress]bool)

	for _, asset := range providerAssets {
		tokenChainAddress := persist.NewTokenChainAddress(persist.Address(asset.Token.ContractAddress), asset.Token.Chain)
		_, exists := assetLookup[tokenChainAddress]

		if !exists {
			newTokensMap[tokenChainAddress] = true
		}
	}

	return providerAssets, newTokensMap, nil
}

func (p *Provider) processTokensForUsers(ctx context.Context, wallets []persist.Address, chainAssetsForOwners map[persist.Address][]chainAssets,
	existingAssetsForUsers map[persist.Address][]persist.Asset, chains []persist.Chain) (map[persist.Address][]persist.Asset, map[persist.Address][]persist.Asset, error) {

	assetsToUpsert := make([]persist.Asset, 0, len(chainAssetsForOwners)*3)
	tokenIsNewForOwner := make(map[persist.Address]map[persist.TokenChainAddress]bool)

	// Find assets to upsert, which are all deduped assets of a wallet address
	// Find all new tokens for a wallet address
	for _, walletAddress := range wallets {
		assets, newTokensMap, err := p.prepTokensForTokenProcessing(ctx, chainAssetsForOwners[walletAddress], existingAssetsForUsers[walletAddress], walletAddress)
		if err != nil {
			return nil, nil, err
		}

		assetsToUpsert = append(assetsToUpsert, assets...)
		tokenIsNewForOwner[walletAddress] = newTokensMap
	}

	// Upsert all assets
	_, persistedAssets, err := p.Repos.AssetRepository.BulkUpsert(ctx, assetsToUpsert)
	if err != nil {
		return nil, nil, err
	}

	persistedAssetsByOwner := make(map[persist.Address][]persist.Asset)
	for _, asset := range persistedAssets {
		ownerAddress := persist.Address(asset.OwnerAddress)
		persistedAssetsByOwner[ownerAddress] = append(persistedAssetsByOwner[ownerAddress], asset)
	}

	// Upsert all assets
	newAssetsForOwner := make(map[persist.Address][]persist.Asset)

	errors := make([]error, 0)

	for _, walletAddress := range wallets {
		newTokensForUser := tokenIsNewForOwner[walletAddress]
		persistedAssetsForOwner := persistedAssetsByOwner[walletAddress]

		newPersistedAssets := make([]persist.Asset, 0, len(persistedAssetsForOwner))
		newPersistedAssetIDs := make([]persist.DBID, 0, len(persistedAssetsForOwner))
		newPersistedAssetIdentifiers := make([]persist.AssetIdentifiers, 0, len(persistedAssetsForOwner))

		for _, asset := range persistedAssetsForOwner {
			if newTokensForUser[persist.NewTokenChainAddress(persist.Address(asset.Token.ContractAddress), asset.Token.Chain)] {
				newPersistedAssets = append(newPersistedAssets, asset)
				newPersistedAssetIDs = append(newPersistedAssetIDs, asset.ID)
				newPersistedAssetIdentifiers = append(newPersistedAssetIdentifiers, persist.NewAssetIdentifiers(asset.Token.ContractAddress, asset.OwnerAddress))
			}
		}

		newAssetsForOwner[walletAddress] = newPersistedAssets
	}

	if len(errors) > 1 {
		return nil, nil, errors[0]
	}

	return persistedAssetsByOwner, newAssetsForOwner, nil
}

func tokensToNewDedupedTokens(assets []chainAssets, ownerWallet persist.Address) ([]persist.Asset, map[persist.DBID]persist.Address) {

	seenAssets := make(map[persist.TokenChainAddress]persist.Asset)

	seenBalances := make(map[persist.TokenChainAddress]persist.NullInt32)
	tokenDBIDToAddress := make(map[persist.DBID]persist.Address)

	sort.SliceStable(assets, func(i int, j int) bool {
		return assets[i].priority < assets[j].priority
	})

	for _, chainAsset := range assets {
		// normalizedAddress := chainAsset.chain.NormalizeAddress(ownerWallet)

		for _, asset := range chainAsset.assets {

			if asset.Balance <= 0 {
				continue
			}

			ti := persist.NewTokenChainAddress(persist.Address(asset.Token.ContractAddress), chainAsset.chain)
			_, seen := seenAssets[ti]

			contractAddress := chainAsset.chain.NormalizeAddress(persist.Address(asset.Token.ContractAddress))
			candidateAsset := persist.Asset{
				// TODO set all asset props
				Token:        persist.Token{Chain: chainAsset.chain, TokenType: asset.Token.TokenType, ContractAddress: persist.EthereumAddress(contractAddress)},
				BlockNumber:  asset.BlockNumber,
				OwnerAddress: persist.EthereumAddress(ownerWallet),
			}

			// If we've never seen the incoming token before, then add it.
			if !seen {
				seenAssets[ti] = candidateAsset
			}

			if q, ok := seenBalances[ti]; ok {
				seenBalances[ti] = q + asset.Balance
			} else {
				seenBalances[ti] = asset.Balance
			}

			seenAsset := seenAssets[ti]
			seenAsset.Balance = seenBalances[ti]
			seenAssets[ti] = seenAsset
			tokenDBIDToAddress[seenAssets[ti].ID] = ti.Address
		}
	}

	res := make([]persist.Asset, len(seenAssets))
	i := 0
	for _, t := range seenAssets {
		res[i] = t
		i++
	}
	return res, tokenDBIDToAddress
}
