package eth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/SplitFi/go-splitfi/contracts"
	"github.com/SplitFi/go-splitfi/env"
	"github.com/SplitFi/go-splitfi/indexer"
	"github.com/SplitFi/go-splitfi/service/auth"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/multichain"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/task"
	"github.com/SplitFi/go-splitfi/util"
	ens "github.com/benny-conn/go-ens"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var eip1271MagicValue = [4]byte{0x16, 0x26, 0xBA, 0x7E}

// Provider is an the struct for retrieving data from the Ethereum blockchain
type Provider struct {
	indexerBaseURL string
	httpClient     *http.Client
	ethClient      *ethclient.Client
	taskClient     *task.Client
}

// NewProvider creates a new ethereum Provider
func NewProvider(httpClient *http.Client, ec *ethclient.Client, tc *task.Client) *Provider {
	return &Provider{
		indexerBaseURL: env.GetString("INDEXER_HOST"),
		httpClient:     httpClient,
		ethClient:      ec,
		taskClient:     tc,
	}
}

// GetBlockchainInfo retrieves blockchain info for ETH
func (d *Provider) GetBlockchainInfo(ctx context.Context) (multichain.BlockchainInfo, error) {
	return multichain.BlockchainInfo{
		Chain:   persist.ChainETH,
		ChainID: 0,
	}, nil
}

func (d *Provider) GetTokenDescriptorsByTokenIdentifiers(ctx context.Context, ti persist.TokenChainAddress) (persist.TokenMetadata, error) {
	// TODO
	metadata, err := d.GetTokenMetadataByTokenIdentifiers(ctx, ti)
	if err != nil {
		return persist.TokenMetadata{}, err
	}
	name, _ := metadata["name"].(string)
	symbol, _ := metadata["contract_symbol"].(string)
	return persist.TokenMetadata{
		"Name":   name,
		"Symbol": symbol,
	}, nil
}

// GetTokenMetadataByTokenIdentifiers retrieves a token's metadata for a given contract address and token ID
func (d *Provider) GetTokenMetadataByTokenIdentifiers(ctx context.Context, ti persist.TokenChainAddress) (persist.TokenMetadata, error) {
	url := fmt.Sprintf("%s%s?contract_address=%s&chain=%d", d.indexerBaseURL, indexer.GetTokenMetadataPath, ti.Address, ti.Chain)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	res, err := d.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, util.GetErrFromResp(res)
	}

	var tokens indexer.GetTokenMetadataOutput
	err = json.NewDecoder(res.Body).Decode(&tokens)
	if err != nil {
		return nil, err
	}

	return tokens.Metadata, nil
}

func (d *Provider) GetDisplayNameByAddress(ctx context.Context, addr persist.Address) string {

	resultChan := make(chan string)
	errChan := make(chan error)
	go func() {
		// no context? who do these guys think they are!? I had to add a goroutine to make sure this doesn't block forever
		domain, err := ens.ReverseResolve(d.ethClient, addr.Address())
		if err != nil {
			errChan <- err
			return
		}
		resultChan <- domain
	}()
	select {
	case result := <-resultChan:
		return result
	case err := <-errChan:
		logger.For(ctx).Errorf("error resolving ens domain: %s", err.Error())
		return addr.String()
	case <-ctx.Done():
		logger.For(ctx).Errorf("error resolving ens domain: %s", ctx.Err().Error())
		return addr.String()
	}
}

// WalletCreated runs whenever a new wallet is created
func (d *Provider) WalletCreated(ctx context.Context, userID persist.DBID, wallet persist.Address, walletType persist.WalletType) error {
	if env.GetString("ENV") == "local" {
		return nil
	}
	//input := task.ValidateNFTsMessage{OwnerAddress: wallet}

	return nil // TODO  task.Client{}.(ctx, input, d.taskClient)
}

// VerifySignature will verify a signature using all available methods (eth_sign and personal_sign)
func (d *Provider) VerifySignature(pCtx context.Context,
	pAddressStr persist.PubKey, pWalletType persist.WalletType, pNonce string, pSignatureStr string) (bool, error) {

	nonce := auth.NewNoncePrepend + pNonce
	// personal_sign
	validBool, err := verifySignature(pSignatureStr,
		nonce,
		pAddressStr, pWalletType,
		true, d.ethClient)

	if !validBool || err != nil {
		// eth_sign
		validBool, err = verifySignature(pSignatureStr,
			nonce,
			pAddressStr, pWalletType,
			false, d.ethClient)
		if err != nil || !validBool {
			nonce = auth.NoncePrepend + pNonce
			validBool, err = verifySignature(pSignatureStr,
				nonce,
				pAddressStr, pWalletType,
				true, d.ethClient)
			if err != nil || !validBool {
				validBool, err = verifySignature(pSignatureStr,
					nonce,
					pAddressStr, pWalletType,
					false, d.ethClient)
			}
		}
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
			return false, auth.ErrAddressSignatureMismatch
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
