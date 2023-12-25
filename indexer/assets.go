package indexer

/*

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"net/http"
	"time"

	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/rpc"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
)

// GetContractOutput is the response for getting a single smart contract
type GetContractOutput struct {
	Contract rpc.Contract `json:"contract"`
}

// GetContractInput is the input to the Get Contract endpoint
type GetContractInput struct {
	Address persist.Address `form:"address,required"`
}

// UpdateTokenMetadataInput is used to refresh metadata for a given token
type UpdateTokenMetadataInput struct {
	Address common.Address `json:"address,required"`
}

func getContract(contractsRepo persist.ContractRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input GetContractInput
		if err := c.ShouldBindQuery(&input); err != nil {
			err = util.ErrInvalidInput{Reason: fmt.Sprintf("must specify 'address' field: %v", err)}
			util.ErrResponse(c, http.StatusBadRequest, err)
			return
		}

		contract, err := contractsRepo.GetByAddress(c, input.Address)
		if err != nil {
			util.ErrResponse(c, http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, GetContractOutput{Contract: contract})
	}
}

func updateContractMetadata(contractsRepo persist.ContractRepository, ethClient *ethclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input UpdateTokenMetadataInput
		if err := c.ShouldBindJSON(&input); err != nil {
			err = util.ErrInvalidInput{Reason: fmt.Sprintf("must specify 'address' field: %v", err)}
			util.ErrResponse(c, http.StatusBadRequest, err)
			return
		}

		err := updateMetadataForToken(c, input, ethClient, contractsRepo)
		if err != nil {
			util.ErrResponse(c, http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, util.SuccessResponse{Success: true})
	}
}

func updateMetadataForToken(c context.Context, input UpdateTokenMetadataInput, ethClient *ethclient.Client, contractsRepo persist.ContractRepository) error {
	newMetadata, err := rpc.GetTokenContractMetadata(c, input.Address, ethClient)
	if err != nil {
		return err
	}

	latestBlock, err := ethClient.BlockNumber(c)
	if err != nil {
		return err
	}

	up := persist.TokenUpdateInput{
		Name:        persist.NullString(newMetadata.Name),
		Symbol:      persist.NullString(newMetadata.Symbol),
		Decimals:    persist.NullString(newMetadata.Symbol),
		Logo:        persist.NullString(newMetadata.Symbol),
		LatestBlock: persist.BlockNumber(latestBlock),
	}

	timedContext, cancel := context.WithTimeout(c, time.Second*10)
	defer cancel()

	creator, err := rpc.GetContractCreator(timedContext, input.Address, ethClient)
	if err != nil {
		logger.For(c).WithError(err).Errorf("error finding creator address")
	} else {
		up.CreatorAddress = creator
	}

	return contractsRepo.UpdateByAddress(c, input.Address, up)
}
*/
