package eth

import (
	"context"
	"errors"
	"fmt"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"strings"
	"time"

	"github.com/SplitFi/go-splitfi/contracts"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const ensContractAddress = "0xFaC7BEA255a6990f749363002136aF6556b31e04"

var (
	// ErrAddressSignatureMismatch is returned when the address signature does not match the address cryptographically
	ErrAddressSignatureMismatch = errors.New("address does not match signature")
	eip1271MagicValue           = [4]byte{0x16, 0x26, 0xBA, 0x7E}
)

// ResolvesENS checks if an ENS resolves to a given address
func ResolvesENS(pCtx context.Context, ens string, userAddr persist.Address, ethcl *ethclient.Client) (bool, error) {

	instance, err := contracts.NewIENSCaller(common.HexToAddress(ens), ethcl)
	if err != nil {
		return false, err
	}

	nh := namehash(ens)
	asBytes32 := [32]byte{}
	for i := 0; i < len(nh); i++ {
		asBytes32[i] = nh[i]
	}

	call, err := instance.Resolver(&bind.CallOpts{Context: pCtx}, asBytes32)
	if err != nil {
		return false, err
	}

	return strings.EqualFold(userAddr.String(), call.String()), nil

}

// function that computes the namehash for a given ENS domain
func namehash(name string) common.Hash {
	node := common.Hash{}

	if len(name) > 0 {
		labels := strings.Split(name, ".")

		for i := len(labels) - 1; i >= 0; i-- {
			labelSha := crypto.Keccak256Hash([]byte(labels[i]))
			node = crypto.Keccak256Hash(node.Bytes(), labelSha.Bytes())
		}
	}

	return node
}

type Verifier struct {
	Client *ethclient.Client
}

// VerifySignature will verify a signature using all available methods (eth_sign and personal_sign)
func (p *Verifier) VerifySignature(pCtx context.Context, pAddressStr persist.PubKey, pWalletType persist.WalletType, pMessage string, pSignatureStr string) (bool, error) {

	// personal_sign
	validBool, err := verifySignature(pSignatureStr, pMessage, pAddressStr, pWalletType, true, p.Client)

	if !validBool || err != nil {
		// eth_sign
		validBool, err = verifySignature(pSignatureStr, pMessage, pAddressStr, pWalletType, false, p.Client)
	}

	if err != nil {
		return false, err
	}

	return validBool, nil
}

func verifySignature(pSignatureStr string,
	pData string,
	pAddress persist.PubKey, pWalletType persist.WalletType,
	pUseDataHeaderBool bool, ec *ethclient.Client) (bool, error) {

	// eth_sign:
	// - https://goethereumbook.org/signature-verify/
	// - http://man.hubwiz.com/docset/Ethereum.docset/Contents/Resources/Documents/eth_sign.html
	// - sign(keccak256("\x19Ethereum Signed Message:\n" + len(message) + message)))

	var data string
	if pUseDataHeaderBool {
		data = fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(pData), pData)
	} else {
		data = pData
	}

	switch pWalletType {
	case persist.WalletTypeEOA:
		dataHash := crypto.Keccak256Hash([]byte(data))

		sig, err := hexutil.Decode(pSignatureStr)
		if err != nil {
			return false, err
		}
		// Ledger-produced signatures have v = 0 or 1
		if sig[64] == 0 || sig[64] == 1 {
			sig[64] += 27
		}
		v := sig[64]
		if v != 27 && v != 28 {
			return false, errors.New("invalid signature (V is not 27 or 28)")
		}
		sig[64] -= 27

		sigPublicKeyECDSA, err := crypto.SigToPub(dataHash.Bytes(), sig)
		if err != nil {
			return false, err
		}

		pubkeyAddressHexStr := crypto.PubkeyToAddress(*sigPublicKeyECDSA).Hex()
		logger.For(nil).Infof("pubkeyAddressHexStr: %s", pubkeyAddressHexStr)
		logger.For(nil).Infof("pAddress: %s", pAddress)
		if !strings.EqualFold(pubkeyAddressHexStr, pAddress.String()) {
			return false, ErrAddressSignatureMismatch
		}

		publicKeyBytes := crypto.CompressPubkey(sigPublicKeyECDSA)

		signatureNoRecoverID := sig[:len(sig)-1]

		return crypto.VerifySignature(publicKeyBytes, dataHash.Bytes(), signatureNoRecoverID), nil
	case persist.WalletTypeGnosis:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		sigValidator, err := contracts.NewISignatureValidator(common.HexToAddress(pAddress.String()), ec)
		if err != nil {
			return false, err
		}

		hashedData := crypto.Keccak256([]byte(data))
		var input [32]byte
		copy(input[:], hashedData)

		result, err := sigValidator.IsValidSignature(&bind.CallOpts{Context: ctx}, input, []byte{})
		if err != nil {
			logger.For(nil).WithError(err).Error("IsValidSignature")
			return false, nil
		}

		return result == eip1271MagicValue, nil
	default:
		return false, errors.New("wallet type not supported")
	}
}
