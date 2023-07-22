package persist

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

// TokenBalanceIdentifiers represents a unique identifier for a token balance
type TokenBalanceIdentifiers string

// TokenBalance represents an address that owns a balance of tokens
type TokenBalance struct {
	ID           DBID            `json:"id" binding:"required"`
	Version      NullInt32       `json:"version"` // schema version for this model
	CreationTime CreationTime    `json:"created_at"`
	LastUpdated  LastUpdatedTime `json:"last_updated"`
	OwnerAddress EthereumAddress `json:"owner_address"`
	Token        Token           `json:"token"`
	Balance      NullInt32       `json:"balance"`
	BlockNumber  BlockNumber     `json:"block_number"`
	// TODO make token balance dependent on chain param
	// Chain        Chain           `json:"chain"`
}

// TokenBalanceUpdateInput represents a struct that is used to update a balance in the database
type TokenBalanceUpdateInput struct {
	LastUpdated time.Time   `json:"last_updated"`
	Balance     NullInt32   `json:"balance"`
	BlockNumber BlockNumber `json:"block_number"`
}

// ErrTokenBalanceNotFoundByID is an error type for when a token balance is not found by id
type ErrTokenBalanceNotFoundByID struct {
	ID DBID
}

// ErrTokenBalanceNotFoundByIdentifiers is an error that is returned when a token balance is not found by its identifiers (owner address and token address)
type ErrTokenBalanceNotFoundByIdentifiers struct {
	OwnerAddress EthereumAddress
	TokenAddress EthereumAddress
	Chain        Chain
}

// TokenBalanceRepository represents a repository for interacting with persisted contracts
type TokenBalanceRepository interface {
	GetByOwner(context.Context, EthereumAddress, Chain, int, int) ([]TokenBalance, error)
	GetByToken(context.Context, EthereumAddress, Chain, int, int) ([]TokenBalance, error)
	GetByIdentifiers(context.Context, EthereumAddress, EthereumAddress, Chain) (TokenBalance, error)
	UpsertByIdentifiers(context.Context, EthereumAddress, EthereumAddress, TokenBalance) error
	UpdateByID(context.Context, DBID, interface{}) error
	UpdateByIdentifiers(context.Context, EthereumAddress, EthereumAddress, Chain, interface{}) error
}

func (e ErrTokenBalanceNotFoundByID) Error() string {
	return fmt.Sprintf("token balance not found by ID: %s", e.ID)
}

func (e ErrTokenBalanceNotFoundByIdentifiers) Error() string {
	return fmt.Sprintf("token balance of %s not found for token address %s-%d", e.OwnerAddress, e.TokenAddress, e.Chain)
}

// NewTokenBalanceIdentifiers creates a new token balance identifier
func NewTokenBalanceIdentifiers(pContractAddress EthereumAddress, pOwnerAddress EthereumAddress) TokenBalanceIdentifiers {
	return TokenBalanceIdentifiers(fmt.Sprintf("%s+%s", pContractAddress, pOwnerAddress))
}

func (b TokenBalanceIdentifiers) String() string {
	return string(b)
}

// GetParts returns the parts of the token identifiers
func (b TokenBalanceIdentifiers) GetParts() (EthereumAddress, EthereumAddress, error) {
	parts := strings.Split(b.String(), "+")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid token identifiers: %s", b)
	}
	return EthereumAddress(EthereumAddress(parts[0]).String()), EthereumAddress(EthereumAddress(parts[1]).String()), nil
}

// Value implements the driver.Valuer interface
func (b TokenBalanceIdentifiers) Value() (driver.Value, error) {
	return b.String(), nil
}

// Scan implements the database/sql Scanner interface for the TokenIdentifiers type
func (b *TokenBalanceIdentifiers) Scan(i interface{}) error {
	if i == nil {
		*b = ""
		return nil
	}
	res := strings.Split(i.(string), "+")
	if len(res) != 2 {
		return fmt.Errorf("invalid token balance identifiers: %v - %T", i, i)
	}
	*b = TokenBalanceIdentifiers(fmt.Sprintf("%s+%s", res[0], res[1]))

	return nil
}
