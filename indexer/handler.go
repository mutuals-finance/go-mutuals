package indexer

import (
	"cloud.google.com/go/storage"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/everFinance/goar"
	"github.com/gin-gonic/gin"
	shell "github.com/ipfs/go-ipfs-api"
)

const (
	TokensGroupPath              = "/tokens"
	GetTokenMetadataPathRelative = "/metadata"
	GetTokenMetadataPath         = TokensGroupPath + GetTokenMetadataPathRelative
	GetStatusPath                = "/status"
)

func handlersInit(router *gin.Engine, i *indexer, tokenRepository persist.TokenRepository, assetRepository persist.AssetRepository, ethClient *ethclient.Client, ipfsClient *shell.Shell, arweaveClient *goar.Client, storageClient *storage.Client) *gin.Engine {
	router.GET(GetStatusPath, getStatus(i, tokenRepository))

	return router
}

func handlersInitServer(router *gin.Engine, tokenRepository persist.TokenRepository, assetRepository persist.AssetRepository, ethClient *ethclient.Client, ipfsClient *shell.Shell, arweaveClient *goar.Client, storageClient *storage.Client, idxer *indexer) *gin.Engine {

	/*	factoryGroup := router.Group("/factory")
		factoryGroup.GET("/", getContract(contractRepository))
		factoryGroup.POST("/", updateContractMetadata(contractRepository, ethClient))

		splitsGroup := router.Group("/splits")
		splitsGroup.GET("/", getContract(contractRepository))
		splitsGroup.POST("/", updateContractMetadata(contractRepository, ethClient))
	*/
	tokensGroup := router.Group(TokensGroupPath)
	tokensGroup.GET(GetTokenMetadataPathRelative, getTokenMetadata(ipfsClient, ethClient, arweaveClient))
	//tokensGroup.POST("/refresh", updateTokenMetadata(contractRepository, ethClient, httpClient))

	return router
}
