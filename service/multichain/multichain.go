package multichain

import (
	"context"
	"database/sql"
	"fmt"
	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/env"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/multichain/common"
	op "github.com/SplitFi/go-splitfi/service/multichain/operation"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/persist/postgres"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/sourcegraph/conc"
	"math/big"
	"strings"
)

func init() {
	env.RegisterValidation("TOKEN_PROCESSING_URL", "required")
}

var unknownContractNames = map[string]bool{
	"unidentified contract": true,
	"unknown contract":      true,
	"unknown":               true,
}

const maxCommunitySize = 1000

// SubmitTokensF is called to process a batch of tokens
type SubmitTokensF func(ctx context.Context, tDefIDs []persist.DBID) error

type Provider struct {
	Repos        *postgres.Repositories
	Queries      *db.Queries
	SubmitTokens SubmitTokensF
	Chains       ProviderLookup
}

type ErrProviderFailed struct{ Err error }

func (e ErrProviderFailed) Unwrap() error { return e.Err }
func (e ErrProviderFailed) Error() string { return fmt.Sprintf("calling provider failed: %s", e.Err) }

type ErrProviderContractNotFound struct {
	Contract persist.Address
	Chain    persist.Chain
	Err      error
}

func (e ErrProviderContractNotFound) Unwrap() error { return e.Err }
func (e ErrProviderContractNotFound) Error() string {
	return fmt.Sprintf("provider did not find contract: %s", e.Contract.String())
}

type chainAssetsAndTokens struct {
	Chain  persist.Chain
	Owner  persist.Address
	Assets []common.ChainAgnosticAsset
	Tokens []common.ChainAgnosticToken
}

// SyncAssetsBySplitIDAndTokenIdentifiers updates the balance for specific tokens of a split
func (p *Provider) SyncAssetsBySplitIDAndTokenIdentifiers(ctx context.Context, splitID persist.DBID, tokenIdentifiers []persist.TokenChainAddress) ([]op.AssetFullDetails, error) {
	split, err := p.Queries.GetSplitById(ctx, splitID)
	if err != nil {
		return nil, err
	}

	tokenIdentifiersByChain := make(map[persist.Chain][]persist.TokenChainAddress)
	for _, tid := range tokenIdentifiers {
		tokenIdentifiersByChain[tid.Chain] = append(tokenIdentifiersByChain[tid.Chain], tid)
	}

	for chain, tids := range tokenIdentifiersByChain {
		tokenIdentifiersByChain[chain] = util.Dedupe(tids, false)
	}

	owner := split.Address
	recCh := make(chan chainAssetsAndTokens, len(tokenIdentifiersByChain))
	errCh := make(chan error)

	wg := &conc.WaitGroup{}

	for c, t := range tokenIdentifiersByChain {
		chain := c
		tids := t

		f, ok := p.Chains[chain].(common.AssetsIncrementalTokenFetcher)
		if !ok {
			continue
		}

		wg.Go(func() {
			for _, t := range tids {
				logger.For(ctx).Infof("syncing assets for split=%s; chain=%d; token_address=%s", split.ID, t.Chain, t.Address)
			}

			pageCh, pageErrCh := f.GetAssetsIncrementallyByTokenIdentifiers(ctx, owner, tids, maxCommunitySize)
			for {
				select {
				case page, ok := <-pageCh:
					if !ok {
						return
					}
					recCh <- chainAssetsAndTokens{
						Owner:  owner,
						Chain:  chain,
						Assets: page.Assets,
						Tokens: page.Tokens,
					}
				case err, ok := <-pageErrCh:
					if !ok {
						continue
					}
					errCh <- ErrProviderFailed{Err: err}
					return
				}
			}
		})

	}

	go func() {
		defer close(recCh)
		defer close(errCh)
		wg.Wait()
	}()

	newAssets, _, err := p.addAssetsAndTokens(ctx, recCh, errCh)
	return newAssets, err
}

// RefreshTokensByTokenIdentifiers attempts to sync tokens for by their token chain address.
func (p *Provider) RefreshTokensByTokenIdentifiers(ctx context.Context, tIDs []persist.TokenChainAddress) (token []db.Token, err error) {

	chainsToTokenIdentifiers := make(map[persist.Chain][]common.ChainAgnosticIdentifiers)
	for _, tid := range tIDs {
		chainsToTokenIdentifiers[tid.Chain] = append(chainsToTokenIdentifiers[tid.Chain], common.ChainAgnosticIdentifiers{ContractAddress: tid.Address})
	}

	for c, ts := range chainsToTokenIdentifiers {
		chainsToTokenIdentifiers[c] = util.DedupeWithTranslate(ts, false, func(tid common.ChainAgnosticIdentifiers) persist.Address { return tid.ContractAddress })
	}

	recCh := make(chan chainAssetsAndTokens, len(tIDs))
	errCh := make(chan error)

	wg := &conc.WaitGroup{}

	for c, tids := range chainsToTokenIdentifiers {
		chain := c
		tokenIdentifiers := tids

		fetcher, ok := p.Chains[chain].(common.TokensByTokenIdentifiersFetcher)
		if !ok {
			continue
		}

		wg.Go(func() {
			for _, tid := range tokenIdentifiers {
				logger.For(ctx).Infof("syncing chain=%d; token_address=%s", chain, tid.ContractAddress)
			}

			tokens, err := fetcher.GetTokensByTokenIdentifiers(ctx, chain, tokenIdentifiers)
			if err != nil {
				errCh <- ErrProviderFailed{Err: err}
				return
			}
			recCh <- chainAssetsAndTokens{
				Chain:  chain,
				Tokens: tokens,
			}
		})
	}

	go func() {
		defer close(recCh)
		defer close(errCh)
		wg.Wait()
	}()

	_, newTokens, err := p.addTokens(ctx, recCh, errCh)
	return newTokens, err
}

func (p *Provider) processAssetsForOwners(
	ctx context.Context,
	chainAssetsByOwner map[persist.Address][]common.ChainAgnosticAsset,
	contracts []db.Token,
) (newAssetsByOwner map[persist.Address][]op.AssetFullDetails, err error) {

	upsertableTokens := make([]op.UpsertAsset, 0)

	for owner, chainAssets := range chainAssetsByOwner {
		assets := chainAssetsToUpsertableAssets(chainAssets, contracts, owner)
		upsertableTokens = append(upsertableTokens, assets...)
	}

	uniqueTokens := dedupeAssetInstances(upsertableTokens)

	_, upsertedTokens, err := op.InsertAssets(ctx, p.Queries, uniqueTokens)
	if err != nil {
		return nil, err
	}

	// Create a lookup for owner address to persisted token IDs
	newAssetsByOwner = make(map[persist.Address][]op.AssetFullDetails)
	for _, token := range upsertedTokens {
		newAssetsByOwner[token.Instance.OwnerAddress] = append(newAssetsByOwner[token.Instance.OwnerAddress], token)
	}

	return newAssetsByOwner, err
}

type addAssetsFunc func(ctx context.Context, owner persist.Address, chain persist.Chain, assets []common.ChainAgnosticAsset, tokens []db.Token) (newAssets []op.AssetFullDetails, err error)

func (p *Provider) addAssetsToOwner(ctx context.Context, owner persist.Address, chain persist.Chain, assets []common.ChainAgnosticAsset, tokens []db.Token) (newAssets []op.AssetFullDetails, err error) {
	return p.processAssetsForOwner(ctx, owner, chain, assets, tokens)
}

func (p *Provider) processAssetsForOwner(ctx context.Context, owner persist.Address, chain persist.Chain, assets []common.ChainAgnosticAsset, tokens []db.Token) ([]op.AssetFullDetails, error) {
	chainAssetsByOwner := map[persist.Address][]common.ChainAgnosticAsset{owner: assets}

	newAssets, err := p.processAssetsForOwners(ctx, chainAssetsByOwner, tokens)
	if err != nil {
		return nil, err
	}

	return newAssets[owner], nil
}

func (p *Provider) receiveAssetsAndTokensData(ctx context.Context, recCh <-chan chainAssetsAndTokens, errCh <-chan error, addAssetsF addAssetsFunc) ([]op.AssetFullDetails, []db.Token, error) {
	var newAssets []op.AssetFullDetails
	var currentTokens []db.Token
	var err error

	for {
		select {
		case page, ok := <-recCh:
			if !ok {
				return newAssets, currentTokens, nil
			}
			tokens, err := p.processTokens(ctx, page.Chain, page.Tokens)
			if err != nil {
				return newAssets, currentTokens, nil
			}
			addedAssets, err := addAssetsF(ctx, page.Owner, page.Chain, page.Assets, tokens)
			if err != nil {
				return newAssets, currentTokens, nil
			}

			newAssets = append(newAssets, addedAssets...)

		case <-ctx.Done():
			err = ctx.Err()
			return nil, nil, err
		case err, ok := <-errCh:
			if ok {
				return nil, nil, err
			}
		}
	}
}

func (p *Provider) addTokens(ctx context.Context, recCh <-chan chainAssetsAndTokens, errCh <-chan error) ([]op.AssetFullDetails, []db.Token, error) {
	return p.receiveAssetsAndTokensData(ctx, recCh, errCh, nil)
}

func (p *Provider) addAssetsAndTokens(ctx context.Context, recCh <-chan chainAssetsAndTokens, errCh <-chan error) ([]op.AssetFullDetails, []db.Token, error) {
	return p.receiveAssetsAndTokensData(ctx, recCh, errCh, p.addAssetsToOwner)
}

// VerifySignature verifies a signature for a wallet address
func (p *Provider) VerifySignature(ctx context.Context, pSig string, pMessage string, pChainAddress persist.ChainPubKey, pWalletType persist.WalletType) (bool, error) {
	if verifier, ok := p.Chains[pChainAddress.Chain()].(common.Verifier); ok {
		if valid, err := verifier.VerifySignature(ctx, pChainAddress.PubKey(), pWalletType, pMessage, pSig); err != nil || !valid {
			return false, err
		}
	}
	return true, nil
}

// processTokens deduplicates tokens and upserts them into the database.
func (p *Provider) processTokens(ctx context.Context, chain persist.Chain, tokens []common.ChainAgnosticToken) (newTokens []db.Token, err error) {
	tokensToAdd := chainTokensToUpsertableTokens(chain, tokens)
	addedTokens, err := p.Repos.TokenRepository.BulkUpsert(ctx, tokensToAdd)
	if err != nil {
		return nil, err
	}

	return addedTokens, nil
}

// chainAssetsToUpsertableAssets returns a unique slice of assets that are ready to be upserted into the database.
func chainAssetsToUpsertableAssets(assets []common.ChainAgnosticAsset, existingTokens []db.Token, owner persist.Address) []op.UpsertAsset {
	addressToToken := make(map[persist.Address]db.Token)

	for _, token := range existingTokens {
		addressToToken[token.ContractAddress] = token
	}

	seenAssets := make(map[persist.AssetChainAddress]op.UpsertAsset)
	seenBalances := make(map[persist.AssetChainAddress]persist.HexString)

	for _, asset := range assets {

		if asset.Balance.BigInt().Cmp(big.NewInt(0)) == 0 {
			continue
		}

		tokenAddress := asset.TokenAddress
		token, ok := addressToToken[tokenAddress]
		if !ok {
			panic(fmt.Sprintf("no persisted token for chain=%s, address=%s", "chain", tokenAddress))
		}

		ai := persist.AssetChainAddress{
			Chain:        token.Chain,
			TokenAddress: asset.TokenAddress,
			OwnerAddress: owner,
		}

		// Duplicate assets will have the same values for these fields, so we only need to set them once
		if _, ok := seenAssets[ai]; !ok {
			seenAssets[ai] = op.UpsertAsset{
				Identifiers: ai,
				Asset: db.Asset{
					TokenAddress: token.ContractAddress,
					Chain:        token.Chain,
					OwnerAddress: owner,
					BlockNumber:  sql.NullInt64{Int64: asset.BlockNumber.BigInt().Int64(), Valid: true},
				},
			}
		}

		if q, ok := seenBalances[ai]; ok {
			seenBalances[ai] = q.Add(asset.Balance)
		} else {
			seenBalances[ai] = asset.Balance
		}

		finalSeenAsset := seenAssets[ai]
		finalSeenAsset.Asset.Balance = seenBalances[ai]
		seenAssets[ai] = finalSeenAsset
	}

	res := make([]op.UpsertAsset, len(seenAssets))

	i := 0
	for _, a := range seenAssets {
		res[i] = a
		i++
	}

	return res
}

// chainTokensToUpsertableTokens returns a unique slice of contracts that are ready to be upserted into the database.
func chainTokensToUpsertableTokens(chain persist.Chain, contracts []common.ChainAgnosticToken) []db.Token {
	result := map[persist.Address]db.Token{}
	for _, c := range contracts {
		result[c.Address] = mergeTokens(result[c.Address], db.Token{
			Name:   util.ToNullStringEmptyNull(c.Name),
			Symbol: util.ToNullStringEmptyNull(c.Symbol),
			Logo:   util.ToNullStringEmptyNull(string(c.Logo)),
		})
	}
	r := make([]db.Token, 0, len(result))
	for address, c := range result {
		r = append(r, db.Token{
			Chain:           chain,
			ContractAddress: address,
			Symbol:          c.Symbol,
			Name:            c.Name,
			Logo:            c.Logo,
		})
	}
	return r
}

func mergeTokens(a db.Token, b db.Token) db.Token {
	a.Name, _ = util.FindFirst([]sql.NullString{a.Name, b.Name}, func(s sql.NullString) bool { return s.String != "" && !unknownContractNames[strings.ToLower(s.String)] })
	a.Symbol = util.ToNullString(util.FirstNonEmptyString(a.Symbol.String, b.Symbol.String), true)
	a.Logo = util.ToNullString(util.FirstNonEmptyString(a.Logo.String, b.Logo.String), true)
	return a
}

func dedupeAssetInstances(assets []op.UpsertAsset) (uniqueAssets []op.UpsertAsset) {
	type Key struct {
		Val op.UpsertAsset
		ID  string
	}

	keys := util.MapWithoutError(assets, func(a op.UpsertAsset) Key {
		return Key{
			Val: a,
			ID:  fmt.Sprintf("%d:%s:%s", a.Identifiers.Chain, a.Identifiers.TokenAddress, a.Identifiers.OwnerAddress),
		}
	})

	a := util.DedupeWithTranslate(keys, false, func(k Key) string { return k.ID })
	return util.MapWithoutError(a, func(k Key) op.UpsertAsset { return k.Val })
}
