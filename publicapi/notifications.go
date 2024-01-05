package publicapi

import (
	"context"
	"time"

	db "github.com/SplitFi/go-splitfi/db/gen/coredb"
	"github.com/SplitFi/go-splitfi/graphql/dataloader"
	"github.com/SplitFi/go-splitfi/service/persist"
	"github.com/SplitFi/go-splitfi/validate"
	"github.com/go-playground/validator/v10"
)

type NotificationsAPI struct {
	queries   *db.Queries
	loaders   *dataloader.Loaders
	validator *validator.Validate
}

func (api NotificationsAPI) GetViewerNotifications(ctx context.Context, before, after *string, first *int, last *int) ([]db.Notification, PageInfo, int, error) {
	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return nil, PageInfo{}, 0, err
	}

	// Validate
	if err := validate.ValidateFields(api.validator, validate.ValidationMap{
		"userID": validate.WithTag(userID, "required"),
	}); err != nil {
		return nil, PageInfo{}, 0, err
	}

	if err := validatePaginationParams(api.validator, first, last); err != nil {
		return nil, PageInfo{}, 0, err
	}

	queryFunc := func(params timeIDPagingParams) ([]db.Notification, error) {
		return api.loaders.GetUserNotificationsBatch.Load(db.GetUserNotificationsBatchParams{
			OwnerID:       userID,
			Limit:         params.Limit,
			CurBeforeTime: params.CursorBeforeTime,
			CurBeforeID:   params.CursorBeforeID,
			CurAfterTime:  params.CursorAfterTime,
			CurAfterID:    params.CursorAfterID,
			PagingForward: params.PagingForward,
		})
	}

	countFunc := func() (int, error) {
		total, err := api.queries.CountUserNotifications(ctx, userID)
		return int(total), err
	}

	cursorFunc := func(n db.Notification) (time.Time, persist.DBID, error) {
		return n.CreatedAt, n.ID, nil
	}

	paginator := timeIDPaginator[db.Notification]{
		QueryFunc:  queryFunc,
		CursorFunc: cursorFunc,
		CountFunc:  countFunc,
	}

	notifications, pageInfo, err := paginator.paginate(before, after, first, last)
	if err != nil {
		return nil, PageInfo{}, 0, err
	}

	count, err := api.queries.CountUserUnseenNotifications(ctx, userID)
	if err != nil {
		return nil, PageInfo{}, 0, err
	}

	return notifications, pageInfo, int(count), err
}

func (api NotificationsAPI) GetByID(ctx context.Context, id persist.DBID) (db.Notification, error) {
	return api.loaders.GetNotificationByIDBatch.Load(id)
}

func (api NotificationsAPI) ClearUserNotifications(ctx context.Context) ([]db.Notification, error) {
	userID, err := getAuthenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}
	return api.queries.ClearNotificationsForUser(ctx, userID)
}
