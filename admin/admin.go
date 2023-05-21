package admin

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/SplitFi/go-splitfi/env"
	"github.com/SplitFi/go-splitfi/util"

	"cloud.google.com/go/storage"
	"github.com/SplitFi/go-splitfi/middleware"
	"github.com/SplitFi/go-splitfi/service/persist/postgres"
	"github.com/SplitFi/go-splitfi/service/rpc"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/api/option"
)

// Init initializes the server
func Init() {
	setDefaults()

	router := CoreInit(postgres.MustCreateClient())

	http.Handle("/", router)
}

// CoreInit initializes core server functionality. This is abstracted
// so the test server can also utilize it
func CoreInit(pqClient *sql.DB) *gin.Engine {
	log.Info("initializing server...")

	log.SetReportCaller(true)

	if env.GetString("ENV") != "production" {
		log.SetLevel(log.DebugLevel)
		gin.SetMode(gin.DebugMode)
	}

	router := gin.Default()
	router.Use(middleware.ErrLogger())

	var s *storage.Client
	var err error
	if env.GetString("ENV") != "local" {
		s, err = storage.NewClient(context.Background())
	} else {
		s, err = storage.NewClient(context.Background(), option.WithCredentialsJSON(util.LoadEncryptedServiceKey("./secrets/prod/service-key.json")))
	}
	if err != nil {
		panic(err)
	}

	return handlersInit(router, pqClient, newStatements(pqClient), rpc.NewEthClient(), s)
}

func setDefaults() {
	viper.SetDefault("ENV", "local")
	viper.SetDefault("ALLOWED_ORIGINS", "http://localhost:3000")
	viper.SetDefault("PORT", 4000)
	viper.SetDefault("POSTGRES_HOST", "0.0.0.0")
	viper.SetDefault("POSTGRES_PORT", 5432)
	viper.SetDefault("POSTGRES_USER", "splitfi_backend")
	viper.SetDefault("POSTGRES_PASSWORD", "")
	viper.SetDefault("POSTGRES_DB", "postgres")
	viper.SetDefault("CONTRACT_ADDRESSES", "0x93eC9b03a9C14a530F582aef24a21d7FC88aaC46=[0,1,2,3,4,5,6,7,8]")
	viper.SetDefault("GENERAL_ADDRESS", "0xe3d0fe9b7e0b951663267a3ed1e6577f6f79757e")
	viper.SetDefault("RPC_URL", "https://eth-rinkeby.alchemyapi.io/v2/_2u--i79yarLYdOT4Bgydqa0dBceVRLD")
	viper.SetDefault("OPENSEA_API_KEY", "")
	viper.SetDefault("GCLOUD_SERVICE_KEY", "")
	viper.SetDefault("SNAPSHOT_BUCKET", "splitfi-dev-322005.appspot.com")

	viper.AutomaticEnv()

}
