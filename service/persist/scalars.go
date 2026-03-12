package persist

import (
	"fmt"
	"io"
	"math/big"
	"strings"
)

// UInt256 represents a 256-bit unsigned integer, serialized as a decimal string in GraphQL.
type UInt256 string

// NewUInt256 creates a UInt256 from a *big.Int.
func NewUInt256(n *big.Int) UInt256 {
	if n == nil {
		return UInt256("0")
	}
	return UInt256(n.String())
}

// BigInt parses the UInt256 value into a *big.Int. Returns nil if the value is empty or invalid.
func (u UInt256) BigInt() *big.Int {
	n := new(big.Int)
	s := strings.TrimSpace(string(u))
	if s == "" || s == "0" {
		return big.NewInt(0)
	}
	// Support both decimal and 0x-prefixed hex
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		n.SetString(s[2:], 16)
	} else {
		n.SetString(s, 10)
	}
	return n
}

func (u UInt256) String() string {
	return string(u)
}

// UnmarshalGQL implements the graphql.Unmarshaler interface.
func (u *UInt256) UnmarshalGQL(v interface{}) error {
	switch val := v.(type) {
	case string:
		*u = UInt256(val)
		return nil
	case int:
		*u = UInt256(fmt.Sprintf("%d", val))
		return nil
	case int64:
		*u = UInt256(fmt.Sprintf("%d", val))
		return nil
	case float64:
		*u = UInt256(fmt.Sprintf("%.0f", val))
		return nil
	default:
		return fmt.Errorf("UInt256 must be a string or number, got %T", v)
	}
}

// MarshalGQL implements the graphql.Marshaler interface.
func (u UInt256) MarshalGQL(w io.Writer) {
	io.WriteString(w, `"`+string(u)+`"`)
}

// Currency represents a currency identifier (e.g. "USD", "ETH") in GraphQL.
type Currency string

func (c Currency) String() string {
	return string(c)
}

// UnmarshalGQL implements the graphql.Unmarshaler interface.
func (c *Currency) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("Currency must be a string, got %T", v)
	}
	*c = Currency(str)
	return nil
}

// MarshalGQL implements the graphql.Marshaler interface.
func (c Currency) MarshalGQL(w io.Writer) {
	io.WriteString(w, `"`+string(c)+`"`)
}

