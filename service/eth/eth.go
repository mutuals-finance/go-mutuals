package eth

import (
	"context"
	"strings"

	"github.com/SplitFi/go-splitfi/contracts"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const ensContractAddress = "0xFaC7BEA255a6990f749363002136aF6556b31e04"

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
