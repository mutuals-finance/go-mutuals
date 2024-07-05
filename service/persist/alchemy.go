package persist

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"time"
)

type AlchemyAddressActivityEventItem struct {
	BlockNumber BlockNumber `json:"blockNum"`
	Hash        HexString   `json:"hash"`
	FromAddress Address     `json:"from_address"`
	ToAddress   Address     `json:"to_address"`
	Value       HexString   `json:"value"`
	Asset       string      `json:"asset"`
	RawContract struct {
		RawValue HexString `json:"rawValue"`
		Address  Address   `json:"address"`
		Decimals int8      `json:"decimals"`
	} `json:"rawContract"`
	Logs []types.Log `json:"logs"`
}

type AlchemyAddressActivityEvent struct {
	Network  Chain                             `json:"network"`
	Activity []AlchemyAddressActivityEventItem `json:"activity"`
}

func (e *AlchemyAddressActivityEvent) MarshalJSON() ([]byte, error) {
	type Alias AlchemyAddressActivityEvent
	return json.Marshal(&struct {
		Network string `json:"fooStatus"`
		*Alias
	}{
		Network: fmt.Sprintf("%d", e.Network),
		Alias:   (*Alias)(e),
	})
}

func (e *AlchemyAddressActivityEvent) UnmarshalJSON(data []byte) error {
	type Alias AlchemyAddressActivityEvent
	aux := &struct {
		Network string `json:"network"`
		*Alias
	}{
		Alias: (*Alias)(e),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	switch aux.Network {
	case "ETH_MAINNET":
		e.Network = ChainETH
	case "MATIC_MAINNET":
		e.Network = ChainPolygon
	case "ARB_MAINNET":
		e.Network = ChainArbitrum
	case "OPT_MAINNET":
		e.Network = ChainOptimism
	case "BASE_MAINNET":
		e.Network = ChainBase
	case "ETH_SEPOLIA":
		e.Network = ChainSepolia
	}

	return nil
}

type AlchemyWebhookInput[T any] struct {
	WebhookId string    `json:"webhookId"`
	Id        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Type      string    `json:"type"`
	Event     T         `json:"event"`
}
