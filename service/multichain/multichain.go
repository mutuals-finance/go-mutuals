package multichain

import (
	"context"
	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/env"
	"github.com/SplitFi/go-splitfi/service/multichain/common"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/persist/postgres"
)

func init() {
	env.RegisterValidation("TOKEN_PROCESSING_URL", "required")
}

type Provider struct {
	Repos   *postgres.Repositories
	Queries *db.Queries
	Chains  ProviderLookup
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
