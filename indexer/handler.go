package indexer

import (
	"cloud.google.com/go/storage"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/everFinance/goar"
	"github.com/gin-gonic/gin"
	shell "github.com/ipfs/go-ipfs-api"
)

func handlersInit(router *gin.Engine, i *indexer, tokenRepository persist.TokenRepository, contractRepository persist.ContractRepository, ethClient *ethclient.Client, ipfsClient *shell.Shell, arweaveClient *goar.Client, storageClient *storage.Client) *gin.Engine {
	router.GET("/status", getStatus(i, tokenRepository))

	return router
}

func handlersInitServer(router *gin.Engine, queueChan chan processTokensInput, tokenRepository persist.TokenRepository, contractRepository persist.ContractRepository, ethClient *ethclient.Client, ipfsClient *shell.Shell, arweaveClient *goar.Client, storageClient *storage.Client, idxer *indexer) *gin.Engine {

	tokenGroup := router.Group("/tokens")
	tokenGroup.POST("/", updateTokens(tokenRepository, ethClient, ipfsClient, arweaveClient))
	tokenGroup.GET("/", getTokens(queueChan, tokenRepository, contractRepository, ipfsClient, ethClient, arweaveClient))

	factoryGroup := router.Group("/factory")
	factoryGroup.GET("/", getContract(contractRepository))
	factoryGroup.POST("/", updateContractMetadata(contractRepository, ethClient))

	splitsGroup := router.Group("/splits")
	splitsGroup.GET("/", getContract(contractRepository))
	splitsGroup.POST("/", updateContractMetadata(contractRepository, ethClient))

	return router
}
