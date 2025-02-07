package admin

import (
	"errors"
	"github.com/mutuals/go-mutuals/service/persist/postgres"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mutuals/go-mutuals/service/persist"
	"github.com/mutuals/go-mutuals/util"
)

var errGetPoolsInput = errors.New("id or user_id must be provided")

type getPoolsInput struct {
	ID     persist.DBID `form:"id"`
	UserID persist.DBID `form:"user_id"`
}

func getPools(poolRepo postgres.PoolRepository) gin.HandlerFunc {
	return func(c *gin.Context) {

		var input getPoolsInput
		if err := c.ShouldBindQuery(&input); err != nil {
			util.ErrResponse(c, http.StatusBadRequest, err)
			return
		}

		if input.ID == "" && input.UserID == "" {
			util.ErrResponse(c, http.StatusBadRequest, errGetPoolsInput)
			return
		}

		var pools []persist.Pool
		//var err error
		//
		//if input.ID == "" {
		//	pool, e := poolRepo.GetByID(c, input.ID)
		//	pools = []persist.Pool{pool}
		//	err = e
		//} else {
		//	pools, err = poolRepo.GetByRecipient(c, input.UserID)
		//}
		//if err != nil {
		//	util.ErrResponse(c, http.StatusInternalServerError, err)
		//	return
		//}

		c.JSON(http.StatusOK, pools)
	}
}
