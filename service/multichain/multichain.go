package multichain

import (
	"context"
	"fmt"
	db "github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/env"
	"github.com/mutuals/go-mutuals/service/multichain/common"
	"github.com/mutuals/go-mutuals/service/persist"
	"github.com/mutuals/go-mutuals/service/persist/postgres"
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

// VerifySignature verifies a signature for a wallet address
func (p *Provider) VerifySignature(ctx context.Context, pSig string, pMessage string, pChainAddress persist.ChainPubKey, pWalletType persist.WalletType) (bool, error) {
	if verifier, ok := p.Chains[pChainAddress.Chain()].(common.Verifier); ok {
		if valid, err := verifier.VerifySignature(ctx, pChainAddress.PubKey(), pWalletType, pMessage, pSig); err != nil || !valid {
			return false, err
		}
	}
	return true, nil
}
