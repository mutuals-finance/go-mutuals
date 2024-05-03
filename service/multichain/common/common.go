package common

import (
	"context"
	"fmt"
	"github.com/SplitFi/go-splitfi/service/persist"
)

// ChainAgnosticIdentifiers identify tokens despite their chain
type ChainAgnosticIdentifiers struct {
	ContractAddress persist.Address `json:"contract_address"`
}

func (t ChainAgnosticIdentifiers) String() string {
	return fmt.Sprintf("token(address=%s)", t.ContractAddress)
}

// Verifier can verify that a signature is signed by a given key
type Verifier interface {
	VerifySignature(ctx context.Context, pubKey persist.PubKey, walletType persist.WalletType, nonce string, sig string) (bool, error)
}

type TokensByTokenIdentifiersFetcher interface {
	GetTokensByTokenIdentifiers(context.Context, persist.Chain, []ChainAgnosticIdentifiers) ([]ChainAgnosticToken, error)
}

// AssetsIncrementalTokenFetcher supports fetching tokens by contract for syncing incrementally
type AssetsIncrementalTokenFetcher interface {
	// GetAssetsIncrementallyByTokenIdentifiers
	// NOTE: implementations MUST close the rec channel
	// maxLimit is not for pagination, it is to make sure we don't fetch a billion tokens from an omnibus contract
	GetAssetsIncrementallyByTokenIdentifiers(ctx context.Context, address persist.Address, tids []persist.TokenChainAddress, maxLimit int) (<-chan ChainAgnosticAssetsAndTokens, <-chan error)
}

// ChainAgnosticAsset is an asset that is agnostic to the chain it is on
type ChainAgnosticAsset struct {
	Balance      persist.HexString   `json:"balance"`
	OwnerAddress persist.Address     `json:"owner_address"`
	TokenAddress persist.Address     `json:"token_address"`
	BlockNumber  persist.BlockNumber `json:"block_number"`
	IsSpam       *bool               `json:"is_spam"`
}

// ChainAgnosticToken is a token that is agnostic to the chain it is on
type ChainAgnosticToken struct {
	Address     persist.Address     `json:"address"`
	Symbol      string              `json:"symbol"`
	Name        string              `json:"name"`
	TokenType   persist.TokenType   `json:"token_type"`
	Logo        persist.TokenLogo   `json:"logo"`
	Decimals    uint8               `json:"decimals"`
	BlockNumber persist.BlockNumber `json:"block_number"`
	IsSpam      *bool               `json:"is_spam"`
}

type ChainAgnosticAssetsAndTokens struct {
	Assets []ChainAgnosticAsset `json:"assets"`
	Tokens []ChainAgnosticToken `json:"tokens"`
}

// ChainAgnosticAssetDescriptors are the fields that describe a token but cannot be used to uniquely identify it
type ChainAgnosticAssetDescriptors struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ChainAgnosticTokenDescriptors struct {
	Symbol          string          `json:"symbol"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	ProfileImageURL string          `json:"profile_image_url"`
	OwnerAddress    persist.Address `json:"creator_address"`
}
