package admin

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/mutuals/go-mutuals/service/auth"
	"github.com/mutuals/go-mutuals/service/persist"
	"github.com/mutuals/go-mutuals/util"
)

var errMustProvideUserIdentifier = fmt.Errorf("must provide either ID or username")
var errNoPools = errors.New("no pools found for first user")

type getUserInput struct {
	ID       persist.DBID    `form:"id"`
	Username string          `form:"username"`
	Address  persist.Address `form:"address"`
}
type deleteUserInput struct {
	ID persist.DBID `json:"id" binding:"required"`
}

type updateUserInput struct {
	ID        persist.DBID      `json:"id" binding:"required"`
	Username  string            `json:"username" binding:"required"`
	Bio       string            `json:"bio"`
	Addresses []persist.Address `json:"addresses" binding:"required"`
}

type mergeUserInput struct {
	FirstUserId  persist.DBID `json:"first_user" binding:"required"`
	SecondUserId persist.DBID `json:"second_user" binding:"required"`
}

type createUserInput struct {
	Addresses []persist.Address `json:"addresses" binding:"required"`
	Username  string            `json:"username" binding:"required"`
	Bio       string            `json:"bio"`
}

type createUserOutput struct {
	UserId persist.DBID `json:"user_id"`
}

func getUser(getUserByIDStmt, getUserByUsername, getUserByAddress *sql.Stmt) gin.HandlerFunc {

	return func(c *gin.Context) {
		var input getUserInput
		if err := c.ShouldBindQuery(&input); err != nil {
			util.ErrResponse(c, http.StatusBadRequest, err)
			return
		}

		var user persist.User
		var err error
		if input.ID != "" {
			err = getUserByIDStmt.QueryRowContext(c, input.ID).Scan(&user.ID, pq.Array(&user.Wallets), &user.Bio, &user.Username, &user.UsernameIdempotent, &user.UpdatedAt, &user.CreationTime)
		} else if input.Username != "" {
			err = getUserByUsername.QueryRowContext(c, input.Username).Scan(&user.ID, pq.Array(&user.Wallets), &user.Bio, &user.Username, &user.UsernameIdempotent, &user.UpdatedAt, &user.CreationTime)
		} else if input.Address != "" {
			err = getUserByAddress.QueryRowContext(c, input.Address).Scan(&user.ID, pq.Array(&user.Wallets), &user.Bio, &user.Username, &user.UsernameIdempotent, &user.UpdatedAt, &user.CreationTime)
		} else {
			util.ErrResponse(c, http.StatusBadRequest, errMustProvideUserIdentifier)
			return
		}

		if err != nil {
			util.ErrResponse(c, http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, user)
	}
}

func createUser(db *sql.DB, createUserStmt, createNonceStmt *sql.Stmt) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input createUserInput
		if err := c.ShouldBindJSON(&input); err != nil {
			util.ErrResponse(c, http.StatusBadRequest, err)
			return
		}

		tx, err := db.BeginTx(c, nil)
		if err != nil {
			util.ErrResponse(c, http.StatusInternalServerError, err)
			return
		}

		var userId persist.DBID
		if err := tx.StmtContext(c, createUserStmt).QueryRowContext(c, persist.GenerateID(), pq.Array(input.Addresses), input.Username, strings.ToLower(input.Username), input.Bio).Scan(&userID); err != nil {
			rollbackWithErr(c, tx, http.StatusInternalServerError, err)
			return
		}

		if err := tx.Commit(); err != nil {
			util.ErrResponse(c, http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, createUserOutput{
			UserId: userId,
		})
	}
}

func updateUser(updateUserStmt *sql.Stmt) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input updateUserInput
		if err := c.ShouldBindJSON(&input); err != nil {
			util.ErrResponse(c, http.StatusBadRequest, err)
			return
		}
		if _, err := updateUserStmt.ExecContext(c, pq.Array(input.Addresses), input.Bio, input.Username, strings.ToLower(input.Username), persist.UpdatedAtTime{}, input.ID); err != nil {
			util.ErrResponse(c, http.StatusInternalServerError, err)
			return
		}
		c.JSON(http.StatusOK, util.SuccessResponse{Success: true})
	}
}

func deleteUser(db *sql.DB, deleteUserStmt, getPoolsStmt, deletePoolStmt, deleteCollectionStmt *sql.Stmt) gin.HandlerFunc {
	return func(c *gin.Context) {

		var input deleteUserInput
		if err := c.ShouldBindJSON(&input); err != nil {
			util.ErrResponse(c, http.StatusBadRequest, err)
			return
		}

		tx, err := db.BeginTx(c, nil)
		if err != nil {
			util.ErrResponse(c, http.StatusInternalServerError, err)
			return
		}

		if _, err := tx.StmtContext(c, deleteUserStmt).ExecContext(c, input.ID); err != nil {
			rollbackWithErr(c, tx, http.StatusInternalServerError, err)
			return
		}

		res, err := getPoolsStmt.QueryContext(c, input.ID)
		if err != nil {
			rollbackWithErr(c, tx, http.StatusInternalServerError, err)
			return
		}
		defer res.Close()

		for res.Next() {
			var g persist.PoolDB
			if err := res.Scan(&g.ID); err != nil {
				rollbackWithErr(c, tx, http.StatusInternalServerError, err)
				return
			}
			// TODO delete pool if user is the only recipient
			//if _, err := tx.StmtContext(c, deletePoolStmt).ExecContext(c, g.ID); err != nil {
			//	rollbackWithErr(c, tx, http.StatusInternalServerError, err)
			//	return
			//}
		}

		if err := res.Err(); err != nil {
			rollbackWithErr(c, tx, http.StatusInternalServerError, err)
			return
		}

		if err := tx.Commit(); err != nil {
			util.ErrResponse(c, http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, util.SuccessResponse{Success: true})
	}
}

func mergeUser(db *sql.DB, getUserByIDStmt, updateUserStmt, deleteUserStmt, getPoolsStmt, deletePoolsStmt, updatePoolStmt *sql.Stmt) gin.HandlerFunc {
	return func(c *gin.Context) {
		var input mergeUserInput
		if err := c.ShouldBindJSON(&input); err != nil {
			util.ErrResponse(c, http.StatusBadRequest, err)
			return
		}

		tx, err := db.BeginTx(c, nil)
		if err != nil {
			util.ErrResponse(c, http.StatusInternalServerError, err)
			return
		}

		var firstUser persist.User
		if err := getUserByIDStmt.QueryRowContext(c, input.FirstUserId).Scan(&firstUser.ID, pq.Array(&firstUser.Wallets), &firstUser.Bio, &firstUser.Username, &firstUser.UsernameIdempotent, &firstUser.UpdatedAt, &firstUser.CreationTime); err != nil {
			rollbackWithErr(c, tx, http.StatusInternalServerError, err)
			return
		}

		var secondUser persist.User
		if err := getUserByIDStmt.QueryRowContext(c, input.SecondUserID).Scan(&secondUser.ID, pq.Array(&secondUser.Wallets), &secondUser.Bio, &secondUser.Username, &secondUser.UsernameIdempotent, &secondUser.UpdatedAt, &secondUser.CreationTime); err != nil {
			rollbackWithErr(c, tx, http.StatusInternalServerError, err)
			return
		}

		if _, err := tx.StmtContext(c, updateUserStmt).ExecContext(c, pq.Array(append(firstUser.Wallets, secondUser.Wallets...)), firstUser.Bio, firstUser.Username, firstUser.UsernameIdempotent, persist.UpdatedAtTime{}, firstUser.ID); err != nil {
			rollbackWithErr(c, tx, http.StatusInternalServerError, err)
			return
		}

		res, err := getPoolsStmt.QueryContext(c, input.FirstUserID)
		if err != nil {
			rollbackWithErr(c, tx, http.StatusInternalServerError, err)
			return
		}
		defer res.Close()

		pools := make([]persist.PoolDB, 0, 1)
		for res.Next() {
			var g persist.PoolDB
			if err := res.Scan(&g.ID); err != nil {
				rollbackWithErr(c, tx, http.StatusInternalServerError, err)
				return
			}
			pools = append(pools, g)
		}

		if err := res.Err(); err != nil {
			rollbackWithErr(c, tx, http.StatusInternalServerError, err)
			return
		}

		nextRes, err := getPoolsStmt.QueryContext(c, input.SecondUserID)
		if err != nil {
			rollbackWithErr(c, tx, http.StatusInternalServerError, err)
			return
		}
		defer nextRes.Close()

		secondPools := make([]persist.PoolDB, 0, 1)
		for nextRes.Next() {
			var g persist.PoolDB
			if err := nextRes.Scan(&g.ID); err != nil {
				rollbackWithErr(c, tx, http.StatusInternalServerError, err)
				return
			}
			secondPools = append(secondPools, g)
		}

		if err := nextRes.Err(); err != nil {
			rollbackWithErr(c, tx, http.StatusInternalServerError, err)
			return
		}

		// TODO: delete pools only if user is last recipient of pool
		//if len(secondPools) > 0 {
		//	delStmt := tx.StmtContext(c, deletePoolsStmt)
		//	for _, g := range secondPools {
		//		if _, err := delStmt.ExecContext(c, g.ID); err != nil {
		//			rollbackWithErr(c, tx, http.StatusInternalServerError, err)
		//			return
		//		}
		//	}
		//}
		//
		//if _, err := tx.StmtContext(c, updatePoolStmt).ExecContext(c, persist.UpdatedAtTime{}, pool.ID); err != nil {
		//	rollbackWithErr(c, tx, http.StatusInternalServerError, err)
		//	return
		//}

		if _, err := tx.StmtContext(c, deleteUserStmt).ExecContext(c, input.SecondUserID); err != nil {
			rollbackWithErr(c, tx, http.StatusInternalServerError, err)
			return
		}

		if err := tx.Commit(); err != nil {
			util.ErrResponse(c, http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, util.SuccessResponse{Success: true})
	}
}
