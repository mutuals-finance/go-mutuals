package admin

import (
	"errors"
	"github.com/SplitFi/go-splitfi/service/persist/postgres"
	"net/http"

	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/gin-gonic/gin"
)

var errGetSplitsInput = errors.New("id or user_id must be provided")

type getSplitsInput struct {
	ID     persist.DBID `form:"id"`
	UserID persist.DBID `form:"user_id"`
}

func getSplits(splitRepo postgres.SplitRepository) gin.HandlerFunc {
	return func(c *gin.Context) {

		var input getSplitsInput
		if err := c.ShouldBindQuery(&input); err != nil {
			util.ErrResponse(c, http.StatusBadRequest, err)
			return
		}

		if input.ID == "" && input.UserID == "" {
			util.ErrResponse(c, http.StatusBadRequest, errGetSplitsInput)
			return
		}

		var splits []persist.Split
		//var err error
		//
		//if input.ID == "" {
		//	split, e := splitRepo.GetByID(c, input.ID)
		//	splits = []persist.Split{split}
		//	err = e
		//} else {
		//	splits, err = splitRepo.GetByRecipient(c, input.UserID)
		//}
		//if err != nil {
		//	util.ErrResponse(c, http.StatusInternalServerError, err)
		//	return
		//}

		c.JSON(http.StatusOK, splits)
	}
}
