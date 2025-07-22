package persist

import "fmt"

// ErrAccountNotFound is returned when an account is not found by its ID
type ErrAccountNotFound struct {
	ID DBID
}

func (e ErrAccountNotFound) Error() string {
	return fmt.Sprintf("account not found with ID: %v", e.ID)
}
