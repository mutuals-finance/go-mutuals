package common

import (
	"context"
	"github.com/SplitFi/go-splitfi/service/persist"
)

// ChainAgnosticIdentifiers identify tokens despite their chain
type ChainAgnosticIdentifiers struct {
	ContractAddress persist.Address `json:"contract_address"`
}

// Verifier can verify that a signature is signed by a given key
type Verifier interface {
	VerifySignature(ctx context.Context, pubKey persist.PubKey, walletType persist.WalletType, nonce string, sig string) (bool, error)
}

type TokenMetadataFetcher interface {
	GetTokenMetadataByTokenIdentifiersBatch(context.Context, []ChainAgnosticIdentifiers) ([]ChainAgnosticTokenMetadata, error)
}

// ChainAgnosticTokenMetadata is a token metadata that is agnostic to the chain it is on
type ChainAgnosticTokenMetadata struct {
	Symbol          string              `json:"symbol"`
	Name            string              `json:"name"`
	ThumbnailURL    string              `json:"thumbnail_url"`
	LogoURL         string              `json:"logo_url"`
	ContractAddress persist.Address     `json:"contract_address"`
	BlockNumber     persist.BlockNumber `json:"block_number"`
	IsSpam          *bool               `json:"is_spam"`
}

// ChainAgnosticToken is a token balance that is agnostic to the chain it is on
type ChainAgnosticToken struct {
	OwnerAddress persist.Address     `json:"owner_address"`
	TokenAddress persist.Address     `json:"token_address"`
	Balance      persist.HexString   `json:"balance"`
	IsOwnerSpam  *bool               `json:"is_owner_spam"`
	LatestBlock  persist.BlockNumber `json:"latest_block"`
}

type ChainAgnosticTokensWithMetadatas struct {
	Tokens    []ChainAgnosticToken         `json:"tokens"`
	Metadatas []ChainAgnosticTokenMetadata `json:"metadatas"`
}
