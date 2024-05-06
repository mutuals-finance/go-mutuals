package common

import (
	"context"
	"github.com/SplitFi/go-splitfi/service/persist"
)

// Verifier can verify that a signature is signed by a given key
type Verifier interface {
	VerifySignature(ctx context.Context, pubKey persist.PubKey, walletType persist.WalletType, nonce string, sig string) (bool, error)
}
