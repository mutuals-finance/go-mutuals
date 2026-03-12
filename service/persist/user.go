package persist

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/lib/pq"
)

// User represents a user with all of their addresses
type User struct {
	Version            NullInt32  `json:"version"`
	ID                 DBID       `json:"id" binding:"required"`
	CreationTime       time.Time  `json:"created_at"`
	Deleted            NullBool   `json:"-"`
	UpdatedAt          time.Time  `json:"updated_at"`
	Username           NullString `json:"username"`
	UsernameIdempotent NullString `json:"username_idempotent"`
	Wallets            []Wallet   `json:"wallets"`
	Universal          NullBool   `json:"universal"`
	PrimaryWalletID    NullString `json:"primary_wallet_id"`
}

// UserUpdateInfoInput represents the data to be updated when updating a user
type UserUpdateInfoInput struct {
	UpdatedAt          time.Time  `json:"updated_at"`
	Username           NullString `json:"username"`
	UsernameIdempotent NullString `json:"username_idempotent"`
}

type CreateUserInput struct {
	ID string
}

// ErrUserNotFound is returned when a user is not found
type ErrUserNotFound struct {
	UserID         DBID
	WalletID       DBID
	L1ChainAddress L1ChainAddress
	Username       string
	Authenticator  string
}

func (e ErrUserNotFound) Error() string {
	template := "user not found: %s;authMethod=%s"

	if e.UserID != "" {
		method := fmt.Sprintf("method=%s;userID=%s", "byUserID", e.UserID)
		return fmt.Sprintf(template, method, e.Authenticator)
	}

	if e.WalletID != "" {
		method := fmt.Sprintf("method=%s;walletID=%s", "byWalletID", e.WalletID)
		return fmt.Sprintf(template, method, e.Authenticator)
	}

	if e.Username != "" {
		method := fmt.Sprintf("method=%s;username=%s", "byUsername", e.Username)
		return fmt.Sprintf(template, method, e.Authenticator)
	}

	if e.L1ChainAddress != (L1ChainAddress{}) {
		method := fmt.Sprintf("method=%s;chainAddress=%s", "byChainAddress", e.L1ChainAddress)
		return fmt.Sprintf(template, method, e.Authenticator)
	}

	return fmt.Sprintf("user not found:authMethod=%s", e.Authenticator)
}

type ErrUserAlreadyExists struct {
	ChainAddress  ChainAddress
	Authenticator string
	Username      string
}

func (e ErrUserAlreadyExists) Error() string {
	return fmt.Sprintf("user already exists: username: %s, address: %s, authenticator: %s", e.Username, e.ChainAddress, e.Authenticator)
}

type ErrUsernameNotAvailable struct {
	Username string
}

func (e ErrUsernameNotAvailable) Error() string {
	return fmt.Sprintf("username not available: %s", e.Username)
}

type ErrAddressOwnedByUser struct {
	ChainAddress ChainAddress
	OwnerID      DBID
}

func (e ErrAddressOwnedByUser) Error() string {
	return fmt.Sprintf("address is owned by user: address: %s, ownerID: %s", e.ChainAddress, e.OwnerID)
}

type ErrPushTokenBelongsToAnotherUser struct {
	PushToken string
}

func (e ErrPushTokenBelongsToAnotherUser) Error() string {
	return fmt.Sprintf("push token already belongs to another user: pushToken: %s", e.PushToken)
}

type Role string

const (
	RoleAdmin       Role = "ADMIN"
	RoleBetaTester  Role = "BETA_TESTER"
	RoleEarlyAccess Role = "EARLY_ACCESS"
)

// Scan implements the database/sql Scanner interface for the Role type
func (r *Role) Scan(i interface{}) error {
	if i == nil {
		return nil
	}
	if it, ok := i.([]uint8); ok {
		*r = Role(it)
		return nil
	}
	*r = Role(i.(string))
	return nil
}

// Value implements the database/sql driver Valuer interface for the Role type
func (r *Role) Value() (driver.Value, error) {
	return r, nil
}

// RoleList is a slice of Roles, primarily used to implement scanner/valuer interfaces for Postgres Arrays.
type RoleList []Role

// Value implements the database/sql driver Valuer interface for RoleList.
func (l RoleList) Value() (driver.Value, error) {
	return pq.Array(l).Value()
}

// Scan implements the database/sql Scanner interface for RoleList.
func (l *RoleList) Scan(value interface{}) error {
	return pq.Array(l).Scan(value)
}
