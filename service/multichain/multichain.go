package multichain

import (
	"context"
	"fmt"
	"github.com/SplitFi/go-splitfi/env"
	"github.com/SplitFi/go-splitfi/service/persist/postgres"
	"github.com/SplitFi/go-splitfi/service/redis"
	"github.com/SplitFi/go-splitfi/service/task"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/sirupsen/logrus"
	"github.com/sourcegraph/conc"
	"sort"

	cloudtasks "cloud.google.com/go/cloudtasks/apiv2"
	"github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/persist"
)

func init() {
	env.RegisterValidation("TOKEN_PROCESSING_URL", "required")
}

// SendTokens is called to process a user's batch of tokens
type SendTokens func(context.Context, task.TokenProcessingUserMessage) error

type Provider struct {
	Repos   *postgres.Repositories
	Queries *coredb.Queries
	Cache   *redis.Cache
	Chains  map[persist.Chain][]any
	// some chains use the addresses of other chains, this will map of chain we want tokens from => chain that's address will be used for lookup
	ChainAddressOverrides ChainOverrideMap
	SendTokens            SendTokens
}

// BlockchainInfo retrieves blockchain info from all chains
type BlockchainInfo struct {
	Chain   persist.Chain `json:"chain_name"`
	ChainID int           `json:"chain_id"`
}

type ChainAddress struct {
	Chain   persist.Chain
	Address persist.Address
}

// ChainAgnosticToken is a token that is agnostic to the chain it is on
type ChainAgnosticToken struct {
	TokenType persist.TokenType `json:"token_type"`

	Name        string `json:"name"`
	Description string `json:"description"`

	Quantity         persist.HexString             `json:"quantity"`
	OwnerAddress     persist.Address               `json:"owner_address"`
	OwnershipHistory []ChainAgnosticAddressAtBlock `json:"previous_owners"`
	TokenMetadata    persist.TokenMetadata         `json:"metadata"`
	ContractAddress  persist.Address               `json:"contract_address"`

	ExternalURL string `json:"external_url"`

	BlockNumber persist.BlockNumber `json:"block_number"`
	IsSpam      *bool               `json:"is_spam"`
}

func (c ChainAgnosticToken) hasMetadata() bool {
	return len(c.TokenMetadata) > 0
}

// ChainAgnosticAddressAtBlock is an address at a block
type ChainAgnosticAddressAtBlock struct {
	Address persist.Address     `json:"address"`
	Block   persist.BlockNumber `json:"block"`
}

// ChainAgnosticContract is a contract that is agnostic to the chain it is on
type ChainAgnosticContract struct {
	Address        persist.Address `json:"address"`
	Symbol         string          `json:"symbol"`
	Name           string          `json:"name"`
	Description    string          `json:"description"`
	CreatorAddress persist.Address `json:"creator_address"`

	LatestBlock persist.BlockNumber `json:"latest_block"`
}

// ChainAgnosticIdentifiers identify tokens despite their chain
type ChainAgnosticIdentifiers struct {
	ContractAddress persist.Address `json:"contract_address"`
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

type errWithPriority struct {
	err      error
	priority int
}

// ErrChainNotFound is an error that occurs when a chain provider for a given chain is not registered in the MultichainProvider
type ErrChainNotFound struct {
	Chain persist.Chain
}

type chainTokens struct {
	priority int
	chain    persist.Chain
	tokens   []persist.Token
}

type chainAssets struct {
	priority int
	chain    persist.Chain
	assets   []persist.Asset
}

// configurer maintains provider settings
type configurer interface {
	GetBlockchainInfo(context.Context) (BlockchainInfo, error)
}

// nameResolver is able to resolve an address to a friendly display name
type nameResolver interface {
	GetDisplayNameByAddress(context.Context, persist.Address) string
}

// verifier can verify that a signature is signed by a given key
type verifier interface {
	VerifySignature(ctx context.Context, pubKey persist.PubKey, walletType persist.WalletType, nonce string, sig string) (bool, error)
}

type walletHooker interface {
	// WalletCreated is called when a wallet is created
	WalletCreated(context.Context, persist.DBID, persist.Address, persist.WalletType) error
}

// tokensOwnerFetcher supports fetching tokens for syncing
type tokensOwnerFetcher interface {
	GetTokensByWalletAddress(ctx context.Context, address persist.Address, limit int, offset int) ([]ChainAgnosticToken, []ChainAgnosticContract, error)
	GetTokenByTokenIdentifiers(context.Context, persist.TokenChainAddress) (persist.Token, error)
	GetAssetByTokenIdentifiersAndOwner(context.Context, persist.TokenChainAddress, persist.Address) (persist.Asset, error)
}

type tokensContractFetcher interface {
	GetTokensByContractAddress(ctx context.Context, contract persist.Address, limit int, offset int) ([]ChainAgnosticToken, ChainAgnosticContract, error)
	GetTokensByContractAddressAndOwner(ctx context.Context, owner persist.Address, contract persist.Address, limit int, offset int) ([]ChainAgnosticToken, ChainAgnosticContract, error)
}

// contractRefresher supports refreshes of a contract
type contractRefresher interface {
	RefreshContract(context.Context, persist.Address) error
}

// deepRefresher supports deep refreshes
type deepRefresher interface {
	DeepRefresh(ctx context.Context, address persist.Address) error
}

// tokenMetadataFetcher supports fetching token metadata
type tokenMetadataFetcher interface {
	GetTokenMetadataByTokenIdentifiers(ctx context.Context, ti ChainAgnosticIdentifiers, ownerAddress persist.Address) (persist.TokenMetadata, error)
}

type providerSupplier interface {
	GetSubproviders() []any
}

type ChainOverrideMap = map[persist.Chain]*persist.Chain

// NewProvider creates a new MultiChainDataRetriever
func NewProvider(ctx context.Context, repos *postgres.Repositories, queries *coredb.Queries, cache *redis.Cache, taskClient *cloudtasks.Client, chainOverrides ChainOverrideMap, providers ...any) *Provider {
	return &Provider{
		Repos:                 repos,
		Cache:                 cache,
		Queries:               queries,
		Chains:                validateProviders(ctx, providers),
		ChainAddressOverrides: chainOverrides,
		SendTokens: func(ctx context.Context, t task.TokenProcessingUserMessage) error {
			return task.CreateTaskForTokenProcessing(ctx, taskClient, t)
		},
	}
}

func getChainProvidersForTask[T any](providers []any) []T {
	result := make([]T, 0, len(providers))
	for _, p := range providers {
		if provider, ok := p.(T); ok {
			result = append(result, provider)
		} else if subproviders, ok := p.(providerSupplier); ok {
			for _, subprovider := range subproviders.GetSubproviders() {
				if provider, ok := subprovider.(T); ok {
					result = append(result, provider)
				}
			}
		}
	}
	return result
}

func hasProvidersForTask[T any](providers []any) bool {
	for _, p := range providers {
		if _, ok := p.(T); ok {
			return true
		} else if subproviders, ok := p.(providerSupplier); ok {
			for _, subprovider := range subproviders.GetSubproviders() {
				if _, ok := subprovider.(T); ok {
					return true
				}
			}
		}
	}
	return false
}

var chainValidation map[persist.Chain]validation = map[persist.Chain]validation{
	persist.ChainETH: {
		nameResolver:          true,
		verifier:              true,
		tokensOwnerFetcher:    true,
		tokensContractFetcher: true,
		contractRefresher:     true,
		tokenMetadataFetcher:  true,
	},
	persist.ChainOptimism: {
		tokensOwnerFetcher:    true,
		tokensContractFetcher: true,
	},
	persist.ChainPolygon: {
		tokensOwnerFetcher:    true,
		tokensContractFetcher: true,
	},
}

type validation struct {
	nameResolver          bool
	verifier              bool
	tokensOwnerFetcher    bool
	tokensContractFetcher bool
	tokenMetadataFetcher  bool
	contractRefresher     bool
}

func validateProviders(ctx context.Context, providers []any) map[persist.Chain][]any {
	chains := map[persist.Chain][]any{}

	configurers := getChainProvidersForTask[configurer](providers)
	for _, cfg := range configurers {
		info, err := cfg.GetBlockchainInfo(ctx)
		if err != nil {
			panic(err)
		}
		chains[info.Chain] = append(chains[info.Chain], cfg)
	}

	for chain, providers := range chains {
		requirements, ok := chainValidation[chain]
		if !ok {
			logger.For(ctx).Warnf("chain=%d has no provider validation", chain)
			continue
		}

		hasImplementor := validation{}

		if hasNameResolver := hasProvidersForTask[nameResolver](providers); hasNameResolver {
			hasImplementor.nameResolver = true
			requirements.nameResolver = true
		}

		if hasVerifier := hasProvidersForTask[verifier](providers); hasVerifier {
			hasImplementor.verifier = true
			requirements.verifier = true
		}

		if hasTokensOwnerFetcher := hasProvidersForTask[tokensOwnerFetcher](providers); hasTokensOwnerFetcher {
			hasImplementor.tokensOwnerFetcher = true
			requirements.tokensOwnerFetcher = true
		}

		if hasTokensContractFetcher := hasProvidersForTask[tokensContractFetcher](providers); hasTokensContractFetcher {
			hasImplementor.tokensContractFetcher = true
			requirements.tokensContractFetcher = true
		}

		if hasContractRefresher := hasProvidersForTask[contractRefresher](providers); hasContractRefresher {
			hasImplementor.contractRefresher = true
			requirements.contractRefresher = true
		}

		if hasTokenMetadataFetcher := hasProvidersForTask[tokenMetadataFetcher](providers); hasTokenMetadataFetcher {
			hasImplementor.tokenMetadataFetcher = true
			requirements.tokenMetadataFetcher = true
		}

		if hasImplementor != requirements {
			panic(fmt.Sprintf("chain=%d;got=%+v;want=%+v", chain, hasImplementor, requirements))
		}
	}

	return chains
}

// providersMatchingInterface returns providers that adhere to the given interface
func providersMatchingInterface[T any](providers []any) []T {
	matches := make([]T, 0)
	seen := map[string]bool{}
	for _, p := range providers {

		if conf, ok := p.(Configurer); ok && seen[conf.GetBlockchainInfo().ProviderID] {
			continue
		} else if ok {
			seen[conf.GetBlockchainInfo().ProviderID] = true
		} else {
			panic(fmt.Sprintf("provider %T does not implement Configurer", p))
		}

		if match, ok := p.(T); ok {
			matches = append(matches, match)

			// if the provider has subproviders, make sure we don't add them later
			if ps, ok := p.(ProviderSupplier); ok {
				for _, sp := range ps.GetSubproviders() {
					if conf, ok := sp.(Configurer); ok {
						seen[conf.GetBlockchainInfo().ProviderID] = true
					} else {
						panic(fmt.Sprintf("subprovider %T does not implement Configurer", sp))
					}
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
			} else if overrides, ok := p.ChainAddressOverrides[chain]; ok && util.Contains(overrides, wallet.Chain) {
				matches[chain] = append(matches[chain], wallet.Address)
			}
		}
	}
	for chain, addresses := range matches {
		matches[chain] = util.Dedupe(addresses, true)
	}
	return matches
}

// SyncTokensByUserIDAndTokenIdentifiers updates the media for specific tokens for an owner
func (p *Provider) SyncTokensByUserIDAndTokenIdentifiers(ctx context.Context, ownerAddress persist.Address, tokenIdentifiers []persist.TokenChainAddress) ([]persist.Asset, error) {

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
		tokenFetchers := matchingProvidersForChain[tokensOwnerFetcher](p.Chains, chain)
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

// GetTokenMetadataByTokenIdentifiers will get the metadata for a given token identifier
func (d *Provider) GetTokenMetadataByTokenIdentifiers(ctx context.Context, contractAddress persist.Address, ownerAddress persist.Address, chain persist.Chain) (persist.TokenMetadata, error) {

	var metadata persist.TokenMetadata
	var err error

	metadataFetchers := getChainProvidersForTask[tokenMetadataFetcher](d.Chains[chain])

	for _, metadataFetcher := range metadataFetchers {
		metadata, err = metadataFetcher.GetTokenMetadataByTokenIdentifiers(ctx, ChainAgnosticIdentifiers{ContractAddress: contractAddress}, ownerAddress)
		if err != nil {
			logger.For(ctx).Errorf("error fetching token metadata %s", err)
		}
		if err == nil && len(metadata) > 0 {
			return metadata, nil
		}
	}

	return metadata, err
}

// DeepRefreshByChain re-indexes a user's wallets.
func (d *Provider) DeepRefreshByChain(ctx context.Context, userID persist.DBID, chain persist.Chain) error {
	if _, ok := d.Chains[chain]; !ok {
		return nil
	}

	// User doesn't exist
	user, err := d.Repos.UserRepository.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	addresses := make([]persist.Address, 0)
	for _, wallet := range user.Wallets {
		if wallet.Chain == chain {
			addresses = append(addresses, wallet.Address)
		}
	}

	deepRefreshers := getChainProvidersForTask[deepRefresher](d.Chains[chain])

	for _, refresher := range deepRefreshers {
		for _, wallet := range addresses {
			if err := refresher.DeepRefresh(ctx, wallet); err != nil {
				return err
			}
		}
	}

	return nil
}

// RunWalletCreationHooks runs hooks for when a wallet is created
func (d *Provider) RunWalletCreationHooks(ctx context.Context, userID persist.DBID, walletAddress persist.Address, walletType persist.WalletType, chain persist.Chain) error {

	// User doesn't exist
	_, err := d.Repos.UserRepository.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	walletHookers := getChainProvidersForTask[walletHooker](d.Chains[chain])

	for _, hooker := range walletHookers {
		if err := hooker.WalletCreated(ctx, userID, walletAddress, walletType); err != nil {
			return err
		}
	}

	return nil
}

// VerifySignature verifies a signature for a wallet address
func (p *Provider) VerifySignature(ctx context.Context, pSig string, pNonce string, pChainAddress persist.ChainPubKey, pWalletType persist.WalletType) (bool, error) {
	providers, err := p.getProvidersForChain(pChainAddress.Chain())
	if err != nil {
		return false, err
	}
	verifiers := getChainProvidersForTask[verifier](providers)
	for _, verifier := range verifiers {
		if valid, err := verifier.VerifySignature(ctx, pChainAddress.PubKey(), pWalletType, pNonce, pSig); err != nil || !valid {
			return false, err
		}
	}
	return true, nil
}

func (d *Provider) getProvidersForChain(chain persist.Chain) ([]any, error) {
	providers, ok := d.Chains[chain]
	if !ok {
		return nil, ErrChainNotFound{Chain: chain}
	}

	return providers, nil
}

type tokenUniqueIdentifiers struct {
	chain   persist.Chain
	address persist.Address
}

type tokenForUser struct {
	userID   persist.DBID
	token    ChainAgnosticToken
	chain    persist.Chain
	priority int
}

func (t ChainAgnosticIdentifiers) String() string {
	return fmt.Sprintf("%s", t.ContractAddress)
}

func (e ErrChainNotFound) Error() string {
	return fmt.Sprintf("chain provider not found for chain: %d", e.Chain)
}

func (e errWithPriority) Error() string {
	return fmt.Sprintf("error with priority %d: %s", e.priority, e.err)
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

		err = p.SubmitAssetsForOwner(ctx, walletAddress, newPersistedAssetIDs, newPersistedAssetIdentifiers)

		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 1 {
		return nil, nil, errors[0]
	}

	return persistedAssetsByOwner, newAssetsForOwner, nil
}

func tokensToNewDedupedTokens(assets []chainAssets, ownerWallet persist.Address) ([]persist.Asset, map[persist.DBID]persist.Address) {

	seenTokens := make(map[persist.TokenChainAddress]persist.Asset)

	seenQuantities := make(map[persist.TokenChainAddress]persist.HexString)
	tokenDBIDToAddress := make(map[persist.DBID]persist.Address)

	sort.SliceStable(assets, func(i int, j int) bool {
		return assets[i].priority < assets[j].priority
	})

	for _, chainAsset := range assets {
		normalizedAddress := chainAsset.chain.NormalizeAddress(ownerWallet)

		for _, asset := range chainAsset.assets {

			if asset.Balance <= 0 {
				continue
			}

			ti := persist.NewTokenChainAddress(persist.Address(asset.Token.ContractAddress), chainAsset.chain)
			existingToken, seen := seenTokens[ti]

			contractAddress := chainAsset.chain.NormalizeAddress(persist.Address(asset.Token.ContractAddress))
			candidateAsset := persist.Asset{
				Token:        persist.Token{Chain: chainAsset.chain, TokenType: asset.Token.TokenType, ContractAddress: contractAddress},
				BlockNumber:  asset.BlockNumber,
				OwnerAddress: persist.EthereumAddress(ownerWallet),
			}

			// If we've never seen the incoming token before, then add it.
			if !seen {
				seenTokens[ti] = candidateAsset
			}

			var found bool
			for _, wallet := range seenWallets[ti] {
				if wallet.Address == token.OwnerAddress {
					found = true
				}
			}
			if !found {
				if q, ok := seenQuantities[ti]; ok {
					seenQuantities[ti] = q.Add(token.Quantity)
				} else {
					seenQuantities[ti] = token.Quantity
				}
			}

			if w, ok := addressToWallets[chainToken.chain.NormalizeAddress(token.OwnerAddress)]; ok {
				seenWallets[ti] = append(seenWallets[ti], w)
				seenWallets[ti] = dedupeWallets(seenWallets[ti])
			}

			seenToken := seenTokens[ti]
			ownership := fromMultichainToAddressAtBlock(token.OwnershipHistory)
			seenToken.OwnershipHistory = ownership
			seenToken.OwnedByWallets = seenWallets[ti]
			seenToken.Quantity = seenQuantities[ti]
			seenTokens[ti] = seenToken
			tokenDBIDToAddress[seenTokens[ti].ID] = ti.ContractAddress
		}
	}

	res := make([]persist.Asset, len(seenTokens))
	i := 0
	for _, t := range seenTokens {
		res[i] = t
		i++
	}
	return res, tokenDBIDToAddress
}
