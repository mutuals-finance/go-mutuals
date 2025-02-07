package postgres

import (
	"context"
	"time"

	db "github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/service/persist"
)

type EventRepository struct {
	Queries *db.Queries
}

func (r *EventRepository) Get(ctx context.Context, eventID persist.DBID) (db.Event, error) {
	return r.Queries.GetEvent(ctx, eventID)
}

func (r *EventRepository) Add(ctx context.Context, event db.Event) (*db.Event, error) {
	switch event.ResourceTypeID {
	case persist.ResourceTypeUser:
		return r.AddUserEvent(ctx, event)
		/*	case persist.ResourceTypeToken:
			return r.AddTokenEvent(ctx, event)
		*/
	case persist.ResourceTypePool:
		return r.AddPoolEvent(ctx, event)
	default:
		return nil, persist.ErrUnknownResourceType{ResourceType: event.ResourceTypeID}
	}
}

func (r *EventRepository) AddUserEvent(ctx context.Context, event db.Event) (*db.Event, error) {
	event, err := r.Queries.CreateUserEvent(ctx, db.CreateUserEventParams{
		ID:             persist.GenerateID(),
		ActorID:        event.ActorID,
		Action:         event.Action,
		ResourceTypeID: event.ResourceTypeID,
		UserID:         event.SubjectID,
		Data:           event.Data,
		GroupID:        event.GroupID,
		Caption:        event.Caption,
	})
	return &event, err
}

/*
	func (r *EventRepository) AddTokenEvent(ctx context.Context, event db.Event) (*db.Event, error) {
		event, err := r.Queries.CreateTokenEvent(ctx, db.CreateTokenEventParams{
			ID:             persist.GenerateID(),
			ActorID:        event.ActorID,
			Action:         event.Action,
			ResourceTypeID: event.ResourceTypeID,
			TokenID:        event.SubjectID,
			Data:           event.Data,
			GroupID:        event.GroupID,
			Caption:        event.Caption,
			PoolID:        event.PoolID,
		})
		return &event, err
	}
*/
func (r *EventRepository) AddPoolEvent(ctx context.Context, event db.Event) (*db.Event, error) {
	event, err := r.Queries.CreatePoolEvent(ctx, db.CreatePoolEventParams{
		ID:             persist.GenerateID(),
		ActorID:        event.ActorID,
		Action:         event.Action,
		ResourceTypeID: event.ResourceTypeID,
		PoolID:         event.PoolID,
		Data:           event.Data,
		ExternalID:     event.ExternalID,
		GroupID:        event.GroupID,
		Caption:        event.Caption,
	})
	return &event, err
}

func (r *EventRepository) IsActorActionActive(ctx context.Context, event db.Event, actions persist.ActionList, windowSize time.Duration) (bool, error) {
	return r.Queries.IsActorActionActive(ctx, db.IsActorActionActiveParams{
		ActorID:     event.ActorID,
		Actions:     actions,
		WindowStart: event.CreatedAt,
		WindowEnd:   event.CreatedAt.Add(windowSize),
	})
}

func (r *EventRepository) IsActorSubjectActive(ctx context.Context, event db.Event, windowSize time.Duration) (bool, error) {
	return r.Queries.IsActorSubjectActive(ctx, db.IsActorSubjectActiveParams{
		ActorID:     event.ActorID,
		SubjectID:   event.SubjectID,
		WindowStart: event.CreatedAt,
		WindowEnd:   event.CreatedAt.Add(windowSize),
	})
}

func (r *EventRepository) IsActorPoolActive(ctx context.Context, event db.Event, windowSize time.Duration) (bool, error) {
	return r.Queries.IsActorPoolActive(ctx, db.IsActorPoolActiveParams{
		ActorID:     event.ActorID,
		PoolID:      event.PoolID,
		WindowStart: event.CreatedAt,
		WindowEnd:   event.CreatedAt.Add(windowSize),
	})
}

func (r *EventRepository) IsActorSubjectActionActive(ctx context.Context, event db.Event, actions persist.ActionList, windowSize time.Duration) (bool, error) {
	return r.Queries.IsActorSubjectActionActive(ctx, db.IsActorSubjectActionActiveParams{
		ActorID:     event.ActorID,
		SubjectID:   event.SubjectID,
		Actions:     actions,
		WindowStart: event.CreatedAt,
		WindowEnd:   event.CreatedAt.Add(windowSize),
	})
}

// EventsInWindow returns events belonging to the same window of activity as the given eventID.
func (r *EventRepository) EventsInWindow(ctx context.Context, eventID persist.DBID, windowSeconds int, actions persist.ActionList, includeSubject bool) ([]db.Event, error) {
	return r.Queries.GetEventsInWindow(ctx, db.GetEventsInWindowParams{
		ID:             eventID,
		Secs:           float64(windowSeconds),
		Actions:        actions,
		IncludeSubject: includeSubject,
	})
}

// EventsInWindowForPool returns events belonging to the same window of activity as the given eventID.
func (r *EventRepository) EventsInWindowForPool(ctx context.Context, eventID, poolID persist.DBID, windowSeconds int, actions persist.ActionList, includeSubject bool) ([]db.Event, error) {
	return r.Queries.GetPoolEventsInWindow(ctx, db.GetPoolEventsInWindowParams{
		ID:             eventID,
		Secs:           float64(windowSeconds),
		Actions:        actions,
		IncludeSubject: includeSubject,
		PoolID:         poolID,
	})
}
