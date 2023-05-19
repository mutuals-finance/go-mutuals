package admin

import (
	"errors"
	"github.com/mikeydub/go-gallery/service/persist/postgres"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mikeydub/go-gallery/service/persist"
	"github.com/mikeydub/go-gallery/util"
)

var errGetSplitsInput = errors.New("id or user_id must be provided")

type getSplitsInput struct {
	ID     persist.DBID `form:"id"`
	UserID persist.DBID `form:"user_id"`
}

type refreshCacheInput struct {
	UserID persist.DBID `form:"user_id" binding:"required"`
}

type backupSplitsInput struct {
	UserID persist.DBID `form:"user_id" binding:"required"`
}

func getSplits(galleryRepo postgres.SplitRepository) gin.HandlerFunc {
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

		//if input.ID == "" {
		//	gallery, e := galleryRepo.GetByID(c, input.ID)
		//	splits = []persist.Split{gallery}
		//	err = e
		//} else {
		//	splits, err = galleryRepo.GetByUserID(c, input.UserID)
		//}
		//if err != nil {
		//	util.ErrResponse(c, http.StatusInternalServerError, err)
		//	return
		//}

		c.JSON(http.StatusOK, splits)
	}
}
