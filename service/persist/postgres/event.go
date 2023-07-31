package postgres

import (
	"context"
	"time"

	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/service/persist"
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
	case persist.ResourceTypeToken:
		return r.AddTokenEvent(ctx, event)
	case persist.ResourceTypeSplit:
		return r.AddSplitEvent(ctx, event)
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
		SplitID:        event.SplitID,
		CollectionID:   event.CollectionID,
	})
	return &event, err
}

func (r *EventRepository) AddSplitEvent(ctx context.Context, event db.Event) (*db.Event, error) {
	event, err := r.Queries.CreateSplitEvent(ctx, db.CreateSplitEventParams{
		ID:             persist.GenerateID(),
		ActorID:        event.ActorID,
		Action:         event.Action,
		ResourceTypeID: event.ResourceTypeID,
		SplitID:        event.SplitID,
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

func (r *EventRepository) IsActorSplitActive(ctx context.Context, event db.Event, windowSize time.Duration) (bool, error) {
	return r.Queries.IsActorSplitActive(ctx, db.IsActorSplitActiveParams{
		ActorID:     event.ActorID,
		SplitID:     event.SplitID,
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

// EventsInWindowForSplit returns events belonging to the same window of activity as the given eventID.
func (r *EventRepository) EventsInWindowForSplit(ctx context.Context, eventID, splitID persist.DBID, windowSeconds int, actions persist.ActionList, includeSubject bool) ([]db.Event, error) {
	return r.Queries.GetSplitEventsInWindow(ctx, db.GetSplitEventsInWindowParams{
		ID:             eventID,
		Secs:           float64(windowSeconds),
		Actions:        actions,
		IncludeSubject: includeSubject,
		SplitID:        splitID,
	})
}
