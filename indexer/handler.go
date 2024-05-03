package indexer

import (
	"cloud.google.com/go/storage"
	"github.com/SplitFi/go-splitfi/db/gen/indexerdb"
	"github.com/SplitFi/go-splitfi/util"
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

func handlersInit(router *gin.Engine, i *indexer, iQueries *indexerdb.Queries, ethClient *ethclient.Client, ipfsClient *shell.Shell, arweaveClient *goar.Client, storageClient *storage.Client) *gin.Engine {
	router.GET(GetStatusPath, getStatus(i, iQueries))
	router.GET("/alive", util.HealthCheckHandler())

	return router
}

func handlersInitServer(router *gin.Engine, ethClient *ethclient.Client, ipfsClient *shell.Shell, arweaveClient *goar.Client, storageClient *storage.Client, idxer *indexer) *gin.Engine {
	return router
}
