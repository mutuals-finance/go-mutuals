package persist

import "fmt"

type ErrPoolNotFound struct {
	ID DBID
}

func (e ErrPoolNotFound) Error() string {
	return fmt.Sprintf("pool not found with ID: %s", e.ID)
}

// PoolDB represents a pool in the database (legacy struct for admin compatibility)
type PoolDB struct {
	ID          DBID   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Image       string `json:"image"`
	OwnerID     DBID   `json:"owner_id"`
	Deleted     bool   `json:"deleted"`
}
