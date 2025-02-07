package admin

import (
	"context"
	"database/sql"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mutuals/go-mutuals/service/persist/postgres"
	"github.com/mutuals/go-mutuals/util"
)

type statements struct {
	getUserByIDStmt       *sql.Stmt
	getUserByUsernameStmt *sql.Stmt
	getUserByAddressStmt  *sql.Stmt
	deleteUserStmt        *sql.Stmt
	getPoolsRawStmt       *sql.Stmt
	deletePoolStmt        *sql.Stmt
	deleteCollectionStmt  *sql.Stmt
	updateUserStmt        *sql.Stmt
	updatePoolStmt        *sql.Stmt
	createUserStmt        *sql.Stmt
	createPoolStmt        *sql.Stmt
	createNonceStmt       *sql.Stmt

	poolRepo postgres.PoolRepository
	userRepo postgres.UserRepository
}

func newStatements(db *sql.DB) *statements {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	getUserByIDStmt, err := db.PrepareContext(ctx, `SELECT ID, ADDRESSES, BIO, USERNAME, USERNAME_IDEMPOTENT, LAST_UPDATED, CREATED_AT FROM USERS WHERE ID = $1 AND DELETED = FALSE;`)
	checkNoErr(err)

	getUserByUsernameStmt, err := db.PrepareContext(ctx, `SELECT ID, ADDRESSES, BIO, USERNAME, USERNAME_IDEMPOTENT, LAST_UPDATED, CREATED_AT FROM USERS WHERE USERNAME_IDEMPOTENT = $1 AND DELETED = FALSE;`)
	checkNoErr(err)

	getUserByAddressStmt, err := db.PrepareContext(ctx, `SELECT ID, ADDRESSES, BIO, USERNAME, USERNAME_IDEMPOTENT, LAST_UPDATED, CREATED_AT FROM users WHERE ADDRESSES @> ARRAY[$1]:: varchar[] AND DELETED = FALSE;`)

	deleteUserStmt, err := db.PrepareContext(ctx, `UPDATE users SET DELETED = TRUE WHERE ID = $1;`)
	checkNoErr(err)

	getPoolsRawStmt, err := db.PrepareContext(ctx, `SELECT ID, COLLECTIONS FROM pools WHERE OWNER_USER_ID = $1;`)
	checkNoErr(err)

	deletePoolStmt, err := db.PrepareContext(ctx, `UPDATE pools SET DELETED = TRUE WHERE ID = $1;`)
	checkNoErr(err)

	deleteCollectionStmt, err := db.PrepareContext(ctx, `UPDATE collections SET DELETED = TRUE WHERE ID = $1;`)
	checkNoErr(err)

	updateUserStmt, err := db.PrepareContext(ctx, `UPDATE users SET ADDRESSES = $1, BIO = $2, USERNAME = $3, USERNAME_IDEMPOTENT = $4, LAST_UPDATED = $5 WHERE ID = $6;`)
	checkNoErr(err)

	updatePoolStmt, err := db.PrepareContext(ctx, `UPDATE pools SET COLLECTIONS = $1, LAST_UPDATED = $2 WHERE ID = $3;`)
	checkNoErr(err)

	createUserStmt, err := db.PrepareContext(ctx, `INSERT INTO users (ID, ADDRESSES, USERNAME, USERNAME_IDEMPOTENT, BIO) VALUES ($1, $2, $3, $4, $5) RETURNING ID;`)
	checkNoErr(err)

	createPoolStmt, err := db.PrepareContext(ctx, `INSERT INTO pools (ID,OWNER_USER_ID, COLLECTIONS) VALUES ($1, $2, $3) RETURNING ID;`)
	checkNoErr(err)

	createNonceStmt, err := db.PrepareContext(ctx, `INSERT INTO nonces (ID,USER_ID, ADDRESS, VALUE) VALUES ($1, $2, $3, $4);`)
	checkNoErr(err)

	//poolRepo := postgres.NewPoolRepository(db, nil)
	return &statements{
		getUserByIDStmt:       getUserByIDStmt,
		getUserByUsernameStmt: getUserByUsernameStmt,
		getUserByAddressStmt:  getUserByAddressStmt,
		deleteUserStmt:        deleteUserStmt,
		getPoolsRawStmt:       getPoolsRawStmt,
		deletePoolStmt:        deletePoolStmt,
		deleteCollectionStmt:  deleteCollectionStmt,
		updateUserStmt:        updateUserStmt,
		updatePoolStmt:        updatePoolStmt,
		createUserStmt:        createUserStmt,
		createPoolStmt:        createPoolStmt,
		createNonceStmt:       createNonceStmt,

		//poolRepo: poolRepo,
		//// nftRepo:     postgres.NewNFTRepository(db, poolRepo),
		//userRepo: postgres.NewUserRepository(db),
		//backupRepo: postgres.NewBackupRepository(db),
	}

}

func rollbackWithErr(c *gin.Context, tx *sql.Tx, status int, err error) {
	util.ErrResponse(c, status, err)
	tx.Rollback()
}

func checkNoErr(err error) {
	if err != nil {
		panic(err)
	}
}
