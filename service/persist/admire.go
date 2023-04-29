package persist

import (
	"context"
	"fmt"
)

type Admire struct {
	ID          DBID            `json:"id"`
	CreatedAt   CreationTime    `json:"created_at"`
	LastUpdated LastUpdatedTime `json:"last_updated"`
	ActorID     DBID            `json:"actor_id"`
	Deleted     bool            `json:"deleted"`
}

type AdmireRepository interface {
	CreateAdmire(ctx context.Context, actorID DBID) (DBID, error)
	RemoveAdmire(ctx context.Context, admireID DBID) error
}

type ErrAdmireNotFound struct {
	AdmireID DBID
	ActorID  DBID
}

func (e ErrAdmireNotFound) Error() string {
	return fmt.Sprintf("admire not found | AdmireID: %s, ActorID: %s", e.AdmireID, e.ActorID)
}

type ErrAdmireAlreadyExists struct {
	AdmireID DBID
	ActorID  DBID
}

func (e ErrAdmireAlreadyExists) Error() string {
	return fmt.Sprintf("admire already exists | AdmireID: %s, ActorID: %s", e.AdmireID, e.ActorID)
}
