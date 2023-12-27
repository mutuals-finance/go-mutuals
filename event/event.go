package event

import (
	"context"
	"fmt"
	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/graphql/dataloader"
	"github.com/SplitFi/go-splitfi/service/logger"
	"github.com/SplitFi/go-splitfi/service/notifications"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/service/persist/postgres"
	sentryutil "github.com/SplitFi/go-splitfi/service/sentry"
	"github.com/SplitFi/go-splitfi/service/task"
	"github.com/SplitFi/go-splitfi/service/tracing"
	"github.com/SplitFi/go-splitfi/util"
	"github.com/SplitFi/go-splitfi/validate"
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"golang.org/x/sync/errgroup"
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
func AddTo(ctx *gin.Context, disableDataloaderCaching bool, notif *notifications.NotificationHandlers, queries *db.Queries, taskClient *task.Client) {
	sender := newEventSender(queries)

	notifications := newEventDispatcher()
	notificationHandler := newNotificationHandler(notif, disableDataloaderCaching, queries)
	sender.addDelayedHandler(notifications, persist.ActionUserFollowedUsers, notificationHandler)
	sender.addDelayedHandler(notifications, persist.ActionViewedSplit, notificationHandler)

	sender.notifications = notifications
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

	persistedEvent, err := sender.eventRepo.Add(ctx, event)
	if err != nil {
		return err
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error { return sender.notifications.dispatchDelayed(ctx, *persistedEvent) })
	return eg.Wait()
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

	persistedEvents := make([]db.Event, 0, len(events))
	for _, e := range events {
		persistedEvent, err := sender.eventRepo.Add(ctx, e)
		if err != nil {
			return err
		}

		persistedEvents = append(persistedEvents, *persistedEvent)
	}

	go func() {

		ctx := sentryutil.NewSentryHubGinContext(ctx)
		if _, err := sender.notifications.dispatchImmediate(ctx, persistedEvents); err != nil {
			logger.For(ctx).Error(err)
			sentryutil.ReportError(ctx, err)
		}

	}()

	/*	feedEvent, err := sender.feed.dispatchImmediate(ctx, persistedEvents)
		if err != nil {
			return nil, err
		}

		return feedEvent.(*db.FeedEvent), nil
	*/

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

	go func() {

		ctx := sentryutil.NewSentryHubGinContext(ctx)
		if _, err := sender.notifications.dispatchGroup(ctx, groupID, action); err != nil {
			logger.For(ctx).Error(err)
			sentryutil.ReportError(ctx, err)
		}

	}()

	/*	feedEvent, err := sender.feed.dispatchGroup(ctx, groupID, action)
		if err != nil {
			return nil, err
		}
	*/
	return nil
}

func For(ctx context.Context) *eventSender {
	gc := util.GinContextFromContext(ctx)
	return gc.Value(eventSenderContextKey).(*eventSender)
}

type registedActions map[persist.Action]struct{}

type eventSender struct {
	notifications *eventDispatcher
	registry      map[sendType]registedActions
	queries       *db.Queries
	eventRepo     postgres.EventRepository
	validate      *validator.Validate
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

func (e *eventSender) addDelayedHandler(dispatcher *eventDispatcher, action persist.Action, handler delayedHandler) {
	dispatcher.addDelayed(action, handler)
	e.registry[delayedKey][action] = struct{}{}
}

func (e *eventSender) addImmediateHandler(dispatcher *eventDispatcher, action persist.Action, handler immediateHandler) {
	dispatcher.addImmediate(action, handler)
	e.registry[immediateKey][action] = struct{}{}
}

func (e *eventSender) addGroupHandler(dispatcher *eventDispatcher, action persist.Action, handler groupHandler) {
	dispatcher.addGroup(action, handler)
	e.registry[groupKey][action] = struct{}{}
}

type eventDispatcher struct {
	delayedHandlers   map[persist.Action]delayedHandler
	immediateHandlers map[persist.Action]immediateHandler
	groupHandlers     map[persist.Action]groupHandler
}

func newEventDispatcher() *eventDispatcher {
	return &eventDispatcher{
		delayedHandlers:   map[persist.Action]delayedHandler{},
		immediateHandlers: map[persist.Action]immediateHandler{},
		groupHandlers:     map[persist.Action]groupHandler{},
	}
}

func (d *eventDispatcher) addDelayed(action persist.Action, handler delayedHandler) {
	d.delayedHandlers[action] = handler
}

func (d *eventDispatcher) addImmediate(action persist.Action, handler immediateHandler) {
	d.immediateHandlers[action] = handler
}

func (d *eventDispatcher) addGroup(action persist.Action, handler groupHandler) {
	d.groupHandlers[action] = handler
}

func (d *eventDispatcher) dispatchDelayed(ctx context.Context, event db.Event) error {
	if handler, ok := d.delayedHandlers[event.Action]; ok {
		return handler.handleDelayed(ctx, event)
	}
	return nil
}

// this will run the handler for each event and return the final non-nil result returned by the handler.
// in the case of the feed, immediate events should be grouped such that only one feed event is created
// and one event is returned
func (d *eventDispatcher) dispatchImmediate(ctx context.Context, event []db.Event) (interface{}, error) {

	resultChan := make(chan interface{})
	errChan := make(chan error)
	var handleables int
	for _, e := range event {
		if handler, ok := d.immediateHandlers[e.Action]; ok {
			handleables++
			go func(event db.Event) {
				result, err := handler.handleImmediate(ctx, event)
				if err != nil {
					errChan <- err
					return
				}
				resultChan <- result
			}(e)
		}
	}

	var result interface{}
	for i := 0; i < handleables; i++ {
		select {
		case r := <-resultChan:
			if r != nil {
				result = r
			}
		case err := <-errChan:
			return nil, err
		}
	}

	return result, nil
}

func (d *eventDispatcher) dispatchGroup(ctx context.Context, groupID string, action persist.Action) (interface{}, error) {
	if handler, ok := d.groupHandlers[action]; ok {
		return handler.handleGroup(ctx, groupID, action)
	}
	return nil, nil
}

type delayedHandler interface {
	handleDelayed(context.Context, db.Event) error
}

type immediateHandler interface {
	handleImmediate(context.Context, db.Event) (interface{}, error)
}

type groupHandler interface {
	handleGroup(context.Context, string, persist.Action) (interface{}, error)
}

// notificationHandlers handles events for consumption as notifications.
type notificationHandler struct {
	dataloaders          *dataloader.Loaders
	notificationHandlers *notifications.NotificationHandlers
}

func newNotificationHandler(notifiers *notifications.NotificationHandlers, disableDataloaderCaching bool, queries *db.Queries) *notificationHandler {
	return &notificationHandler{
		notificationHandlers: notifiers,
		dataloaders:          dataloader.NewLoaders(context.Background(), queries, disableDataloaderCaching, tracing.DataloaderPreFetchHook, tracing.DataloaderPostFetchHook),
	}
}

func (h notificationHandler) handleDelayed(ctx context.Context, persistedEvent db.Event) error {
	owner, err := h.findOwnerForNotificationFromEvent(persistedEvent)
	if err != nil {
		return err
	}

	// if no user found to notify, don't notify
	if owner == "" {
		return nil
	}

	// Don't notify the user on self events
	if persist.DBID(persist.NullStrToStr(persistedEvent.ActorID)) == owner {
		return nil
	}

	// Don't notify the user on un-authed views
	if persistedEvent.Action == persist.ActionViewedSplit && persistedEvent.ActorID.String == "" {
		return nil
	}

	return h.notificationHandlers.Notifications.Dispatch(ctx, db.Notification{
		OwnerID:  owner,
		Action:   persistedEvent.Action,
		Data:     h.createNotificationDataForEvent(persistedEvent),
		EventIds: persist.DBIDList{persistedEvent.ID},
		SplitID:  persistedEvent.SplitID,
		TokenID:  persistedEvent.TokenID,
	})
}

func (h notificationHandler) findOwnerForNotificationFromEvent(ctx context.Context, event db.Event) (persist.DBID, error) {
	switch event.ResourceTypeID {
	case persist.ResourceTypeSplit:
		split, err := h.dataloaders.GetSplitByIdBatch.Load(event.SplitID)
		if err != nil {
			return "", err
		}
		return split.OwnerUserID, nil
	case persist.ResourceTypeUser:
		return event.SubjectID, nil
	case persist.ResourceTypeToken:
		return persist.DBID(event.ActorID.String), nil
	}

	return "", fmt.Errorf("no owner found for event: %s", event.Action)
}

func (h notificationHandler) createNotificationDataForEvent(event db.Event) (data persist.NotificationData) {
	switch event.Action {
	case persist.ActionViewedSplit:
		if event.ActorID.String != "" {
			data.AuthedViewerIDs = []persist.DBID{persist.NullStrToDBID(event.ActorID)}
		}
		if event.ExternalID.String != "" {
			data.UnauthedViewerIDs = []string{persist.NullStrToStr(event.ExternalID)}
		}
	case persist.ActionUserFollowedUsers:
		if event.ActorID.String != "" {
			data.FollowerIDs = []persist.DBID{persist.NullStrToDBID(event.ActorID)}
		}
		data.FollowedBack = persist.NullBool(event.Data.UserFollowedBack)
		data.Refollowed = persist.NullBool(event.Data.UserRefollowed)
	case persist.ActionNewTokensReceived:
		data.NewTokenID = event.Data.NewTokenID
		data.NewTokenQuantity = event.Data.NewTokenQuantity
	case persist.ActionTopActivityBadgeReceived:
		data.ActivityBadgeThreshold = event.Data.ActivityBadgeThreshold
		data.NewTopActiveUser = event.Data.NewTopActiveUser
	default:
		logger.For(nil).Debugf("no notification data for event: %s", event.Action)
	}
	return
}

// followerNotificationHandler handles events for consumption as notifications.
type followerNotificationHandler struct {
	notificationHandlers *notifications.NotificationHandlers
}

func newFollowerNotificationHandler(notifiers *notifications.NotificationHandlers) *followerNotificationHandler {
	return &followerNotificationHandler{
		notificationHandlers: notifiers,
	}
}

func (h followerNotificationHandler) handleDelayed(ctx context.Context, persistedEvent db.Event) error {
	return h.notificationHandlers.Notifications.Dispatch(ctx, db.Notification{
		// no owner or data for follower notifications
		Action:   persistedEvent.Action,
		EventIds: persist.DBIDList{persistedEvent.ID},
		SplitID:  persistedEvent.SplitID,
		TokenID:  persistedEvent.TokenID,
	})
}

// global handles events for consumption as global notifications.
type announcementNotificationHandler struct {
	notificationHandlers *notifications.NotificationHandlers
}

func newAnnouncementNotificationHandler(notifiers *notifications.NotificationHandlers) *announcementNotificationHandler {
	return &announcementNotificationHandler{
		notificationHandlers: notifiers,
	}
}

func (h announcementNotificationHandler) handleDelayed(ctx context.Context, persistedEvent db.Event) error {
	return h.notificationHandlers.Notifications.Dispatch(ctx, db.Notification{
		// no owner or data for follower notifications
		Action:   persistedEvent.Action,
		EventIds: persist.DBIDList{persistedEvent.ID},
		Data: persist.NotificationData{
			AnnouncementDetails: persistedEvent.Data.AnnouncementDetails,
		},
	})
}
