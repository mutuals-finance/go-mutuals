package indexer

import (
	"context"
	"fmt"
	"github.com/SplitFi/go-splitfi/service/rpc"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/everFinance/goar"
	"github.com/gin-gonic/gin"
	shell "github.com/ipfs/go-ipfs-api"
	"github.com/sirupsen/logrus"
)

type errNoMetadataFound struct {
	Contract persist.EthereumAddress `json:"contract"`
}

func (e errNoMetadataFound) Error() string {
	return fmt.Sprintf("no metadata found for contract %s", e.Contract)
}

type getTokenMetadataInput struct {
	ContractAddress persist.EthereumAddress `form:"contract_address" binding:"required"`
}

type GetTokenMetadataOutput struct {
	Metadata persist.TokenMetadata `json:"metadata"`
}

func getTokenMetadata(ipfsClient *shell.Shell, ethClient *ethclient.Client, arweaveClient *goar.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		input := &getTokenMetadataInput{}

		if err := c.ShouldBindQuery(input); err != nil {
			util.ErrResponse(c, http.StatusBadRequest, err)
			return
		}
		ctx := logger.NewContextWithFields(c, logrus.Fields{
			"contractAddress": input.ContractAddress,
		})

		ctx, cancel := context.WithTimeout(ctx, time.Minute*2)
		defer cancel()

		//TODO get token data
		newURI := ""

		newMetadata, err := rpc.GetMetadataFromURI(ctx, newURI, ipfsClient, arweaveClient)

		if err != nil || len(newMetadata) == 0 {
			logger.For(ctx).Errorf("Error getting metadata from URI: %s (%s)", err, util.TruncateWithEllipsis(newURI, 50))
			status := http.StatusNotFound
			if err != nil {
				switch err.(type) {
				//case util.ErrHTTP:
				//	if caught.Status != http.StatusNotFound {
				//		status = http.StatusInternalServerError
				//	}
				case *url.Error, *net.DNSError, *shell.Error:
					// do nothing
				default:
					status = http.StatusInternalServerError
				}
			}
			util.ErrResponse(c, status, errNoMetadataFound{Contract: input.ContractAddress})
			return
		}

		c.JSON(http.StatusOK, GetTokenMetadataOutput{Metadata: newMetadata})
	}
}
