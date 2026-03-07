package event

import (
	"context"
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	db "github.com/mutuals/go-mutuals/db/gen/coredb"
	"github.com/mutuals/go-mutuals/service/logger"
	"github.com/mutuals/go-mutuals/service/persist"
	"github.com/mutuals/go-mutuals/service/persist/postgres"
	sentryutil "github.com/mutuals/go-mutuals/service/sentry"
	"github.com/mutuals/go-mutuals/service/task"
	"github.com/mutuals/go-mutuals/util"
	"github.com/mutuals/go-mutuals/validate"
)

type sendType int

const (
	eventSenderContextKey           = "event.eventSender"
	sentryEventContextName          = "event context"
	delayedKey             sendType = iota
	immediateKey
	groupKey
)

// AddTo Register specific event handlers
func AddTo(ctx *gin.Context, disableDataloaderCaching bool, queries *db.Queries, taskClient *task.Client) {
	sender := newEventSender(queries)
	ctx.Set(eventSenderContextKey, &sender)
}

func Dispatch(ctx context.Context, evt db.Event) error {
	ctx = sentryutil.NewSentryHubGinContext(ctx)
	go PushEvent(ctx, evt)
	return nil
}

func DispatchCaptioned(ctx context.Context, evt db.Event, caption *string) error {
	ctx = sentryutil.NewSentryHubGinContext(ctx)

	if caption != nil {
		evt.Caption = persist.StrPtrToNullStr(caption)
		return dispatchImmediate(ctx, []db.Event{evt})
	}

	go PushEvent(ctx, evt)
	return nil
}

func DispatchMany(ctx context.Context, evts []db.Event, editID *string) error {
	if len(evts) == 0 {
		return nil
	}

	for i := range evts {
		evts[i].GroupID = persist.StrPtrToNullStr(editID)
	}

	ctx = sentryutil.NewSentryHubGinContext(ctx)

	for _, evt := range evts {
		go PushEvent(ctx, evt)
	}

	return nil
}

func PushEvent(ctx context.Context, evt db.Event) {
	err := dispatchDelayed(ctx, evt)
	if err != nil {
		sentryutil.ReportError(ctx, err, func(scope *sentry.Scope) {
			logger.For(ctx).Error(err)
			setEventContext(scope, persist.NullStrToDBID(evt.ActorID), evt.SubjectID, evt.Action)
		})
	}
}

func setEventContext(scope *sentry.Scope, actorID, subjectID persist.DBID, action persist.Action) {
	scope.SetContext(sentryEventContextName, sentry.Context{
		"ActorID":   actorID,
		"SubjectID": subjectID,
		"Action":    action,
	})
}

// dispatchDelayed sends the event to all of its registered handlers.
func dispatchDelayed(ctx context.Context, event db.Event) error {
	gc := util.MustGetGinContext(ctx)
	sender := For(gc)

	// validate event
	err := sender.validate.Struct(event)
	if err != nil {
		return err
	}

	if _, handable := sender.registry[delayedKey][event.Action]; !handable {
		logger.For(ctx).WithField("action", event.Action).Warn("no delayed handler configured for action")
		return nil
	}

	_, err = sender.eventRepo.Add(ctx, event)
	if err != nil {
		return err
	}

	return nil
}

// dispatchImmediate flushes the event immediately to its registered handlers.
func dispatchImmediate(ctx context.Context, events []db.Event) error {
	gc := util.MustGetGinContext(ctx)
	sender := For(gc)

	for _, e := range events {
		// Vaidate event
		if err := sender.validate.Struct(e); err != nil {
			return err
		}
		if _, handable := sender.registry[immediateKey][e.Action]; !handable {
			logger.For(ctx).WithField("action", e.Action).Warn("no immediate handler configured for action")
			return nil
		}
	}

	for _, e := range events {
		_, err := sender.eventRepo.Add(ctx, e)
		if err != nil {
			return err
		}
	}

	return nil
}

// DispatchGroup flushes the event group immediately to its registered handlers.
func DispatchGroup(ctx context.Context, groupID string, action persist.Action, caption *string) error {
	gc := util.MustGetGinContext(ctx)
	sender := For(gc)

	if _, handable := sender.registry[groupKey][action]; !handable {
		logger.For(ctx).WithField("action", action).Warn("no group handler configured for action")
		return nil
	}

	if caption != nil {
		err := sender.eventRepo.Queries.UpdateEventCaptionByGroup(ctx, db.UpdateEventCaptionByGroupParams{
			Caption: persist.StrPtrToNullStr(caption),
			GroupID: persist.StrPtrToNullStr(&groupID),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func For(ctx context.Context) *eventSender {
	gc := util.MustGetGinContext(ctx)
	return gc.Value(eventSenderContextKey).(*eventSender)
}

type registedActions map[persist.Action]struct{}

type eventSender struct {
	registry  map[sendType]registedActions
	queries   *db.Queries
	eventRepo postgres.EventRepository
	validate  *validator.Validate
}

func newEventSender(queries *db.Queries) eventSender {
	v := validator.New()
	v.RegisterStructValidation(validate.EventValidator, db.Event{})
	return eventSender{
		registry:  map[sendType]registedActions{delayedKey: {}, immediateKey: {}, groupKey: {}},
		queries:   queries,
		eventRepo: postgres.EventRepository{Queries: queries},
		validate:  v,
	}
}
