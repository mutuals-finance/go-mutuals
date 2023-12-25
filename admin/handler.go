package admin

import (
	"database/sql"

	"cloud.google.com/go/storage"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
)

func handlersInit(router *gin.Engine, db *sql.DB, stmts *statements, ethcl *ethclient.Client, stg *storage.Client) *gin.Engine {
	api := router.Group("/admin/v1")

	users := api.Group("/users")
	users.GET("/get", getUser(stmts.getUserByIDStmt, stmts.getUserByUsernameStmt, stmts.getUserByAddressStmt))
	users.POST("/merge", mergeUser(db, stmts.getUserByIDStmt, stmts.updateUserStmt, stmts.deleteUserStmt, stmts.getSplitsRawStmt, stmts.deleteSplitStmt, stmts.updateSplitStmt))
	users.POST("/update", updateUser(stmts.updateUserStmt))
	users.POST("/delete", deleteUser(db, stmts.deleteUserStmt, stmts.getSplitsRawStmt, stmts.deleteSplitStmt, stmts.deleteCollectionStmt))
	users.POST("/create", createUser(db, stmts.createUserStmt, stmts.createNonceStmt))

	raw := api.Group("/raw")
	raw.POST("/query", queryRaw(db))

	splits := api.Group("/splits")
	splits.GET("/get", getSplits(stmts.splitRepo))
	//splits.GET("/refresh", refreshCache(stmts.splitRepo))
	//splits.GET("/backup", backupSplits(stmts.splitRepo, stmts.backupRepo))

	snapshot := api.Group("/snapshot")
	snapshot.GET("/get", getSnapshot(stg))
	snapshot.POST("/update", updateSnapshot(stg))

	return router
}
