package indexer

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/everFinance/goar"
	"github.com/gin-gonic/gin"
	shell "github.com/ipfs/go-ipfs-api"
)

type Activity struct {
	Category        string                  `json:"type"`
	FromAddress     persist.EthereumAddress `json:"fromAddress"`
	ToAddress       persist.EthereumAddress `json:"toAddress"`
	RawContract     string                  `json:"rawContract"`
	RawValue        persist.HexString       `json:"rawValue"`
	ContractAddress persist.EthereumAddress `json:"address"`
	BlockNumber     persist.BlockNumber     `json:"blockNum"`
	TxHash          string                  `json:"hash"`
	Value           int64                   `json:"value,omitempty"`
	TokenSymbol     string                  `json:"asset,omitempty"`
}

type ActivityEvent struct {
	Network  string     `json:"network"`
	Activity []Activity `json:"activity"`
}

// UpdateActivityInput is the input from the alchemy activity notify webhook
type UpdateActivityInput struct {
	WebhookId string        `json:"webhookId"`
	Id        string        `json:"id"`
	CreatedAt time.Time     `json:"createdAt"`
	Type      string        `json:"type"`
	Event     ActivityEvent `json:"event"`
}

func updateActivity(tokenRepository persist.TokenRepository, ethClient *ethclient.Client, ipfsClient *shell.Shell, arweaveClient *goar.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		input := UpdateActivityInput{}
		if err := c.ShouldBindJSON(&input); err != nil {
			// TODO always return 200 to prevent webhook retry
			util.ErrResponse(c, http.StatusBadRequest, err)
			return
		}
		_, err := processActivities(c, input.Event.Activity, tokenRepository, ethClient)
		if err != nil {
			// TODO always return 200 to prevent webhook retry
			util.ErrResponse(c, http.StatusInternalServerError, err)
			return
		}

		c.JSON(http.StatusOK, util.SuccessResponse{Success: true})
	}
}

// filterActivities checks each activity against the input and returns ones that match the criteria.
func processActivities(ctx context.Context, activities []Activity, tokenRepository persist.TokenRepository, ethClient *ethclient.Client) ([]persist.Token, error) {

	tokens := make([]persist.Token, 0, len(activities))

	var t persist.Token
	var err error

	for _, a := range activities {
		switch a.Category {
		case "erc20":
			t, err = processTokenActivity(ctx, a.ContractAddress, tokenRepository, ethClient)
			if err != nil {
				return tokens, nil
			}

		case "internal":
		case "external":
			t, err = processTokenActivity(ctx, "0xF", tokenRepository, ethClient)
			if err != nil {
				return tokens, nil
			}
		}
		tokens = append(tokens, t)

		// TODO create transfer event

	}

	return tokens, nil
}

func processTokenActivity(pCtx context.Context, contractAddress persist.EthereumAddress, tokenRepo persist.TokenRepository, ec *ethclient.Client) (t persist.Token, err error) {
	tokens, err := tokenRepo.GetByTokenIdentifiers(pCtx, contractAddress, 1, 0)
	if err != nil {
		return t, err
	}

	if len(tokens) > 0 {
		return tokens[0], err
	}

	// TODO fill all necessary token info eg by using third party API
	//e20, err := contracts.NewIERC20Caller(contractAddress.Address(), ec)
	//if err != nil {
	//	return t, err
	//}
	//_, err = e20.BalanceOf(&bind.CallOpts{Context: pCtx}, ownerAddress.Address())
	//if err != nil {
	//	return persist.Token{}, fmt.Errorf("failed to get balance of token %s: %s", contractAddress, err)
	//}
	t.TokenType = persist.TokenTypeERC20

	if err := tokenRepo.Upsert(pCtx, t); err != nil {
		return persist.Token{}, fmt.Errorf("failed to upsert token %s: %s", contractAddress, err)
	}

	return t, nil
}
