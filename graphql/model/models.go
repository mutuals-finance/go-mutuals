package model

import (
	"fmt"
	"io"
	"time"

	"github.com/SplitFi/go-splitfi/service/persist"
)

type GqlID string

func (v *Viewer) GetGqlIDField_UserID() string {
	return string(v.UserId)
}

type HelperViewerData struct {
	UserId persist.DBID
}

type HelperGroupNotificationUsersConnectionData struct {
	UserIDs persist.DBIDList
}

type HelperSplitFiUserData struct {
	UserID persist.DBID
}

type HelperNotificationSettingsData struct {
	UserId persist.DBID
}

type HelperNotificationsConnectionData struct {
	UserId persist.DBID
}

type HelperUserEmailData struct {
	UserId persist.DBID
}

type ErrInvalidIDFormat struct {
	message string
}

func (e ErrInvalidIDFormat) Error() string {
	return fmt.Sprintf("invalid ID format: %s", e.message)
}

type ErrInvalidIDType struct {
	typeName string
}

func (e ErrInvalidIDType) Error() string {
	return fmt.Sprintf("no fetch method found for ID type '%s'", e.typeName)
}

type Window struct {
	time.Duration
	Name string
}

var (
	lastFiveDaysWindow  = Window{5 * 24 * time.Hour, "LAST_5_DAYS"}
	lastSevenDaysWindow = Window{7 * 24 * time.Hour, "LAST_7_DAYS"}
	allTimeWindow       = Window{1<<63 - 1, "ALL_TIME"}
)

func (w *Window) UnmarshalGQL(v interface{}) error {
	window, ok := v.(string)
	if !ok {
		return fmt.Errorf("Window must be a string")
	}
	switch window {
	case lastFiveDaysWindow.Name:
		*w = lastFiveDaysWindow
	case lastSevenDaysWindow.Name:
		*w = lastSevenDaysWindow
	case allTimeWindow.Name:
		*w = allTimeWindow
	default:
		panic(fmt.Sprintf("unknown window: %s", window))
	}
	return nil
}

func (w Window) MarshalGQL(wt io.Writer) {
	switch {
	case w == lastFiveDaysWindow:
		wt.Write([]byte(fmt.Sprintf(`"%s"`, lastFiveDaysWindow.Name)))
	case w == lastSevenDaysWindow:
		wt.Write([]byte(fmt.Sprintf(`"%s"`, lastSevenDaysWindow.Name)))
	case w == allTimeWindow:
		wt.Write([]byte(fmt.Sprintf(`"%s"`, allTimeWindow.Name)))
	default:
		panic(fmt.Sprintf("unknown window: %v", w))
	}
}
