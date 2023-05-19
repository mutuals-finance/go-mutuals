package admin

import (
	"context"
	"database/sql"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mikeydub/go-gallery/service/persist/postgres"
	"github.com/mikeydub/go-gallery/util"
)

type statements struct {
	getUserByIDStmt       *sql.Stmt
	getUserByUsernameStmt *sql.Stmt
	getUserByAddressStmt  *sql.Stmt
	deleteUserStmt        *sql.Stmt
	getSplitsRawStmt      *sql.Stmt
	deleteSplitStmt       *sql.Stmt
	deleteCollectionStmt  *sql.Stmt
	updateUserStmt        *sql.Stmt
	updateSplitStmt       *sql.Stmt
	createUserStmt        *sql.Stmt
	createSplitStmt       *sql.Stmt
	createNonceStmt       *sql.Stmt
	getCollectionsStmt    *sql.Stmt
	updateCollectionStmt  *sql.Stmt

	galleryRepo postgres.SplitRepository
	// nftRepo     persist.NFTRepository
	userRepo postgres.UserRepository
	collRepo postgres.CollectionTokenRepository
}

func newStatements(db *sql.DB) *statements {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	getUserByIDStmt, err := db.PrepareContext(ctx, `SELECT ID, ADDRESSES, BIO, USERNAME, USERNAME_IDEMPOTENT, LAST_UPDATED, CREATED_AT FROM USERS WHERE ID = $1 AND DELETED = false;`)
	checkNoErr(err)

	getUserByUsernameStmt, err := db.PrepareContext(ctx, `SELECT ID, ADDRESSES, BIO, USERNAME, USERNAME_IDEMPOTENT, LAST_UPDATED, CREATED_AT FROM USERS WHERE USERNAME_IDEMPOTENT = $1 AND DELETED = false;`)
	checkNoErr(err)

	getUserByAddressStmt, err := db.PrepareContext(ctx, `SELECT ID, ADDRESSES, BIO, USERNAME, USERNAME_IDEMPOTENT, LAST_UPDATED, CREATED_AT FROM users WHERE ADDRESSES @> ARRAY[$1]:: varchar[] AND DELETED = false;`)

	deleteUserStmt, err := db.PrepareContext(ctx, `UPDATE users SET DELETED = true WHERE ID = $1;`)
	checkNoErr(err)

	getSplitsRawStmt, err := db.PrepareContext(ctx, `SELECT ID, COLLECTIONS FROM splits WHERE OWNER_USER_ID = $1;`)
	checkNoErr(err)

	deleteSplitStmt, err := db.PrepareContext(ctx, `UPDATE splits SET DELETED = true WHERE ID = $1;`)
	checkNoErr(err)

	deleteCollectionStmt, err := db.PrepareContext(ctx, `UPDATE collections SET DELETED = true WHERE ID = $1;`)
	checkNoErr(err)

	updateUserStmt, err := db.PrepareContext(ctx, `UPDATE users SET ADDRESSES = $1, BIO = $2, USERNAME = $3, USERNAME_IDEMPOTENT = $4, LAST_UPDATED = $5 WHERE ID = $6;`)
	checkNoErr(err)

	updateSplitStmt, err := db.PrepareContext(ctx, `UPDATE splits SET COLLECTIONS = $1, LAST_UPDATED = $2 WHERE ID = $3;`)
	checkNoErr(err)

	createUserStmt, err := db.PrepareContext(ctx, `INSERT INTO users (ID, ADDRESSES, USERNAME, USERNAME_IDEMPOTENT, BIO) VALUES ($1, $2, $3, $4, $5) RETURNING ID;`)
	checkNoErr(err)

	createSplitStmt, err := db.PrepareContext(ctx, `INSERT INTO splits (ID,OWNER_USER_ID, COLLECTIONS) VALUES ($1, $2, $3) RETURNING ID;`)
	checkNoErr(err)

	createNonceStmt, err := db.PrepareContext(ctx, `INSERT INTO nonces (ID,USER_ID, ADDRESS, VALUE) VALUES ($1, $2, $3, $4);`)
	checkNoErr(err)

	getCollectionsStmt, err := db.PrepareContext(ctx, `SELECT ID,OWNER_USER_ID,NFTS,NAME,COLLECTORS_NOTE,LAYOUT,HIDDEN,VERSION,CREATED_AT,LAST_UPDATED FROM collections WHERE OWNER_USER_ID = $1 AND DELETED = false;`)
	checkNoErr(err)

	updateCollectionStmt, err := db.PrepareContext(ctx, `UPDATE collections SET NFTS = $1, NAME = $2, COLLECTORS_NOTE = $3, LAYOUT = $4, HIDDEN = $5, LAST_UPDATED = $6 WHERE ID = $7;`)
	checkNoErr(err)

	//galleryRepo := postgres.NewSplitRepository(db, nil)
	return &statements{
		getUserByIDStmt:       getUserByIDStmt,
		getUserByUsernameStmt: getUserByUsernameStmt,
		getUserByAddressStmt:  getUserByAddressStmt,
		deleteUserStmt:        deleteUserStmt,
		getSplitsRawStmt:      getSplitsRawStmt,
		deleteSplitStmt:       deleteSplitStmt,
		deleteCollectionStmt:  deleteCollectionStmt,
		updateUserStmt:        updateUserStmt,
		updateSplitStmt:       updateSplitStmt,
		createUserStmt:        createUserStmt,
		createSplitStmt:       createSplitStmt,
		createNonceStmt:       createNonceStmt,
		getCollectionsStmt:    getCollectionsStmt,
		updateCollectionStmt:  updateCollectionStmt,

		//galleryRepo: galleryRepo,
		//// nftRepo:     postgres.NewNFTRepository(db, galleryRepo),
		//userRepo: postgres.NewUserRepository(db),
		//// collRepo:    postgres.NewCollectionRepository(db, galleryRepo),
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
