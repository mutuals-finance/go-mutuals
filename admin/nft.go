package admin

import (
	"errors"

	"github.com/SplitFi/go-splitfi/service/persist"
)

var errGetNFTsInput = errors.New("address or user_id must be provided")

type getNFTsInput struct {
	Address persist.EthereumAddress `form:"address"`
	UserID  persist.DBID            `form:"user_id"`
}

type ownsGeneralInput struct {
	Address persist.EthereumAddress `form:"address" binding:"required"`
}

type ownsGeneralOutput struct {
	Owns bool `json:"owns"`
}

// func getNFTs(nftRepo persist.NFTRepository) gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		var input getNFTsInput
// 		if err := c.ShouldBindQuery(&input); err != nil {
// 			util.ErrResponse(c, http.StatusBadRequest, err)
// 			return
// 		}

// 		if input.Address == "" && input.UserID == "" {
// 			util.ErrResponse(c, http.StatusBadRequest, errGetNFTsInput)
// 			return
// 		}

// 		var nfts []persist.NFT
// 		var err error

// 		if input.Address == "" {
// 			nfts, err = nftRepo.GetByUserID(c, input.UserID)
// 		} else {
// 			nfts, err = nftRepo.GetByAddresses(c, []persist.EthereumAddress{input.Address})
// 		}
// 		if err != nil {
// 			util.ErrResponse(c, http.StatusInternalServerError, err)
// 			return
// 		}

// 		c.JSON(http.StatusOK, nfts)
// 	}
// }

// func ownsGeneral(ethClient *ethclient.Client) gin.HandlerFunc {
// 	general, err := contracts.NewIERC1155Caller(common.HexToAddress(env.GetString("GENERAL_ADDRESS")), ethClient)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return func(c *gin.Context) {
// 		var input ownsGeneralInput
// 		if err := c.ShouldBindQuery(&input); err != nil {
// 			util.ErrResponse(c, http.StatusBadRequest, err)
// 			return
// 		}
// 		bal, err := general.BalanceOf(&bind.CallOpts{Context: c}, input.Address.Address(), big.NewInt(0))
// 		if err != nil {
// 			util.ErrResponse(c, http.StatusInternalServerError, err)
// 			return
// 		}
// 		c.JSON(http.StatusOK, ownsGeneralOutput{Owns: bal.Uint64() > 0})
// 	}
// }
