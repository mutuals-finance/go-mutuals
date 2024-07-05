package multichain

import (
	"context"
	"fmt"
	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/env"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/multichain/common"
	op "github.com/SplitFi/go-splitfi/service/multichain/operation"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/persist/postgres"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/SplitFi/go-splitfi/validate"
)

func init() {
	env.RegisterValidation("TOKEN_PROCESSING_URL", "required")
}

type Provider struct {
	Repos   *postgres.Repositories
	Queries *db.Queries
	Chains  ProviderLookup
}

type ErrProviderFailed struct{ Err error }

func (e ErrProviderFailed) Unwrap() error { return e.Err }
func (e ErrProviderFailed) Error() string { return fmt.Sprintf("calling provider failed: %s", e.Err) }

type chainTokensAndMetadatas struct {
	Chain     persist.Chain
	Tokens    []common.ChainAgnosticToken
	Metadatas []common.ChainAgnosticTokenMetadata
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

// UpdateTokensForPoolUnchecked adds tokens to a payment pool with the requested balances. UpdateTokensForPoolUnchecked does not make any effort to validate
// that the pool owns the tokens, only that the tokens exist and are fetchable on chain. This is useful for adding tokens to a pool when it's
// already known beforehand that the pool owns the token via a trusted source, skipping the potentially expensive operation of fetching a token by its owner.
func (p *Provider) UpdateTokensForPoolUnchecked(ctx context.Context, poolID persist.ChainAddress, tokenAddresses []persist.Address, tokenChains []persist.Chain, newBalances []persist.HexString) ([]op.TokenFullDetails, error) {
	// Validate
	err := validate.Validate(validate.ValidationMap{
		"poolID":         validate.WithTag(poolID, "required"),
		"tokenAddresses": validate.WithTag(tokenAddresses, "required,gt=0,unique"),
		"tokenChains":    validate.WithTag(tokenChains, fmt.Sprintf("len=%d,dive,gt=0", len(tokenAddresses))),
		"newBalances":    validate.WithTag(newBalances, fmt.Sprintf("len=%d,dive,gt=0", len(tokenAddresses))),
	})
	if err != nil {
		return nil, err
	}

	pool, err := p.Queries.GetSplitByChainAddress(ctx, db.GetSplitByChainAddressParams{Address: poolID.Address(), Chain: poolID.Chain()})
	if err != nil {
		return nil, err
	}

	// Group missing metadatas by chain
	metadataMap := make(map[persist.TokenChainAddress]common.ChainAgnosticTokenMetadata)
	missingMetadatasByChain := make(map[persist.Chain][]common.ChainAgnosticIdentifiers)

	existingMetadatas, err := p.Queries.GetTokenMetadatasByTokenIdentifiers(ctx, db.GetTokenMetadatasByTokenIdentifiersParams{
		ContractAddress: tokenAddresses,
		Chain:           tokenChains,
	})
	if err != nil {
		return nil, err
	}

	metadatasToAdd := make([]db.TokenMetadata, len(tokenAddresses)-len(existingMetadatas))

	// fill lookup map with missing metadata
	for i, a := range tokenAddresses {
		c := tokenChains[i]
		if _, ok := metadataMap[persist.NewTokenChainAddress(a, c)]; !ok {
			missingMetadatasByChain[c] = append(missingMetadatasByChain[c], common.ChainAgnosticIdentifiers{ContractAddress: a})
		}
	}

	for tChain, tIDs := range missingMetadatasByChain {
		// Validate that the chain is supported
		_, ok := p.Chains[tChain].(common.TokenMetadataFetcher)
		if !ok {
			err = fmt.Errorf("multichain is not configured to fetch unchecked tokens for chain=%d", tChain)
			logger.For(ctx).Error(err)
			return nil, err
		}

		// TODO check if pool exists on at least a single chain
		if pool.Chain.L1Chain() != tChain.L1Chain() {
			// Return an error if the requested owner address is not owned by the pool
			err := fmt.Errorf("token(chain=%d, tokens=%s) requested owner address=%s, but address is not owned by user", tChain, tIDs, pool.Address)
			logger.For(ctx).Error(err)
			return nil, err
		}

		newMetadatas, err := p.Chains[tChain].(common.TokenMetadataFetcher).GetTokenMetadataByTokenIdentifiersBatch(ctx, tIDs)

		// Exit early if a token in the batch is not found
		if err != nil {
			err := fmt.Errorf("failed to fetch token(chain=%d, tokens=%s): %s", tChain, tIDs, err)
			logger.For(ctx).Error(err)
			return nil, err
		}

		if len(newMetadatas) == 0 {
			err := fmt.Errorf("failed to fetch token(chain=%d, identifiers=%s)", tChain, tIDs)
			logger.For(ctx).Error(err)
			return nil, err
		}

		newMetadatasToAdd := chainMetadatasToUpsertableMetadatas(tChain, newMetadatas)
		metadatasToAdd = append(metadatasToAdd, newMetadatasToAdd...)
	}

	tokensToAdd := make([]db.Token, len(tokenAddresses))
	for i, a := range tokenAddresses {
		tokenToAdd := db.Token{
			Chain:        tokenChains[i],
			TokenAddress: a,
			OwnerAddress: poolID.Address(),
			Balance:      newBalances[i],
		}
		tokensToAdd = append(tokensToAdd, tokenToAdd)
	}

	return p.addPoolTokensWithMetadatas(ctx, tokensToAdd, metadatasToAdd)
}

func (p *Provider) addPoolTokensWithMetadatas(ctx context.Context, tokensToAdd []db.Token, metadatasToAdd []db.TokenMetadata) (newTokens []op.TokenFullDetails, err error) {

	// Insert token metadata
	_, _, err = op.InsertTokenMetadatas(ctx, p.Queries, metadatasToAdd)
	if err != nil {
		logger.For(ctx).Errorf("error in bulk upsert of token metadatas: %s", err)
		return nil, err
	}

	// Insert tokens
	addedTokens, err := op.InsertTokens(ctx, p.Queries, tokensToAdd)
	if err != nil {
		logger.For(ctx).Errorf("error in bulk upsert of tokens: %s", err)
		return nil, err
	}

	return addedTokens, err
}

// chainMetadatasToUpsertableMetadatas returns a unique slice of token metadatas that are ready to be upserted into the database.
func chainMetadatasToUpsertableMetadatas(chain persist.Chain, metadatas []common.ChainAgnosticTokenMetadata) []db.TokenMetadata {
	result := make(map[persist.Address]db.TokenMetadata)

	for _, m := range metadatas {
		normalizedAddress := persist.Address(chain.NormalizeAddress(m.ContractAddress))
		result[normalizedAddress] = mergeTokenMetadatas(result[normalizedAddress], db.TokenMetadata{
			Chain:           chain,
			Symbol:          util.ToNullStringEmptyNull(m.Symbol),
			Name:            util.ToNullStringEmptyNull(m.Name),
			ContractAddress: normalizedAddress,
			Logo:            util.ToNullStringEmptyNull(m.LogoURL),
			Thumbnail:       util.ToNullStringEmptyNull(m.ThumbnailURL),
		})
	}

	return util.MapValues(result)
}

func mergeTokenMetadatas(a db.TokenMetadata, b db.TokenMetadata) db.TokenMetadata {
	a.Name = util.ToNullString(util.FirstNonEmptyString(a.Name.String, b.Name.String), true)
	a.Symbol = util.ToNullString(util.FirstNonEmptyString(a.Symbol.String, b.Symbol.String), true)
	a.Logo = util.ToNullString(util.FirstNonEmptyString(a.Logo.String, b.Logo.String), true)
	a.Thumbnail = util.ToNullString(util.FirstNonEmptyString(a.Thumbnail.String, b.Thumbnail.String), true)
	a.ContractAddress = persist.Address(util.FirstNonEmptyString(a.ContractAddress.String(), b.ContractAddress.String()))
	return a
}
