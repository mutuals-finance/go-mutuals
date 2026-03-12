package persist

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/lib/pq"
)

// Wallet represents an address on any EVM network.
type Wallet struct {
	ID           DBID      `json:"id"`
	Version      NullInt64 `json:"version"`
	CreationTime time.Time `json:"created_at"`
	Deleted      NullBool  `json:"-"`
	UpdatedAt    time.Time `json:"updated_at"`

	Address    Address    `json:"address"`
	Network    Network    `json:"network"`
	WalletType WalletType `json:"wallet_type"`
}

// WalletType is the type of wallet used to sign a message.
type WalletType int

type WalletList []Wallet

// Address represents an EVM-compatible wallet address.
type Address string

// PubKey represents the public key of a wallet.
type PubKey string

// NetworkAddress pairs an address with a network identifier.
// Prefer using Address directly where network distinction is not needed.
type NetworkAddress struct {
	address Address
	network Network
}

// NewNetworkAddress creates a NetworkAddress, normalizing the address to lowercase
// (all supported networks are EVM-compatible).
func NewNetworkAddress(address Address, network Network) NetworkAddress {
	return NetworkAddress{
		address: Address(strings.ToLower(string(address))),
		network: network,
	}
}

func (c NetworkAddress) Address() Address { return c.address }
func (c NetworkAddress) Network() Network { return c.network }

func (c NetworkAddress) String() string {
	return fmt.Sprintf("%s@%d", c.address, c.network)
}

func (c NetworkAddress) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{"address": c.address.String(), "network": int(c.network)})
}

func (c *NetworkAddress) UnmarshalJSON(data []byte) error {
	var v map[string]any
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	if v["address"] != nil {
		c.address = Address(strings.ToLower(v["address"].(string)))
	}
	if v["network"] != nil {
		if n, ok := v["network"].(float64); ok {
			c.network = Network(int(n))
		}
	}
	return nil
}

const (
	// WalletTypeEOA represents an externally owned account (regular wallet address).
	WalletTypeEOA WalletType = iota
	// WalletTypeGnosis represents a smart contract gnosis safe.
	WalletTypeGnosis
)

func (l WalletList) Value() (driver.Value, error) {
	return pq.Array(l).Value()
}

func (l *WalletList) Scan(value interface{}) error {
	return pq.Array(l).Scan(value)
}

func (w *Wallet) Scan(value interface{}) error {
	if value == nil {
		*w = Wallet{}
		return nil
	}
	*w = Wallet{ID: DBID(string(value.([]uint8)))}
	return nil
}

func (w Wallet) Value() (driver.Value, error) {
	if w.ID == "" {
		return "", nil
	}
	return w.ID.String(), nil
}

// UnmarshalGQL implements the graphql.Unmarshaler interface.
func (wa *WalletType) UnmarshalGQL(v interface{}) error {
	n, ok := v.(string)
	if !ok {
		return fmt.Errorf("wrong type for WalletType: %T", v)
	}
	switch n {
	case "EOA":
		*wa = WalletTypeEOA
	case "GnosisSafe":
		*wa = WalletTypeGnosis
	default:
		return fmt.Errorf("unknown WalletType: %s", n)
	}
	return nil
}

// MarshalGQL implements the graphql.Marshaler interface.
func (wa WalletType) MarshalGQL(w io.Writer) {
	switch wa {
	case WalletTypeEOA:
		io.WriteString(w, `"EOA"`)
	case WalletTypeGnosis:
		io.WriteString(w, `"GnosisSafe"`)
	}
}

func (n Address) String() string {
	return strings.ToLower(string(n))
}

func (n Address) Value() (driver.Value, error) {
	if n.String() == "" {
		return "", nil
	}
	return strings.ToValidUTF8(strings.ReplaceAll(n.String(), "\\u0000", ""), ""), nil
}

func (n *Address) Scan(value interface{}) error {
	if value == nil {
		*n = Address("")
		return nil
	}
	asString, ok := value.(string)
	if !ok {
		asUint8Array, ok := value.([]uint8)
		if !ok {
			return fmt.Errorf("Address must be a string or []uint8")
		}
		asString = string(asUint8Array)
	}
	*n = Address(strings.ToLower(asString))
	return nil
}

// EVMAddress converts to a go-ethereum common.Address.
func (n Address) EVMAddress() common.Address {
	return common.HexToAddress(n.String())
}

func (p PubKey) String() string { return string(p) }

// ---------------------------------------------------------------------------
// Wallet errors
// ---------------------------------------------------------------------------

type ErrWalletAlreadyExists struct {
	WalletID DBID
	Address  Address
	OwnerID  DBID
}

func (e ErrWalletAlreadyExists) Error() string {
	return fmt.Sprintf("wallet already exists: wallet ID: %s | address: %s | owner ID: %s", e.WalletID, e.Address, e.OwnerID)
}

var errWalletNotFound ErrWalletNotFound

type ErrWalletNotFound struct{}

func (e ErrWalletNotFound) Unwrap() error { return notFoundError }
func (e ErrWalletNotFound) Error() string { return "wallet not found" }

type ErrWalletNotFoundByID struct{ ID DBID }

func (e ErrWalletNotFoundByID) Unwrap() error { return errWalletNotFound }
func (e ErrWalletNotFoundByID) Error() string { return "wallet not found by id: " + e.ID.String() }

type ErrWalletNotFoundByAddress struct{ Address Address }

func (e ErrWalletNotFoundByAddress) Unwrap() error { return errWalletNotFound }
func (e ErrWalletNotFoundByAddress) Error() string {
	return fmt.Sprintf("wallet not found by address: %s", e.Address)
}

