package model

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mutuals/go-mutuals/service/persist"
)

// GqlID represents a Global Graphql compatible ID (Type:ID)
type GqlID string

// DBID extracts the raw database identifier by stripping the prefix
func (v GqlID) DBID() persist.DBID {
	s := string(v)
	parts := strings.Split(s, ":")
	if len(parts) < 2 {
		return persist.DBID(s)
	}
	return persist.DBID(strings.Join(parts[1:], ":"))
}

// DBIDPtr returns a pointer to the DBID
func (v *GqlID) DBIDPtr() *persist.DBID {
	if v == nil {
		return nil
	}
	dbid := v.DBID()
	return &dbid
}

func (v GqlID) String() string {
	return string(v)
}

func (v *GqlID) UnmarshalGQL(i interface{}) error {
	str, ok := i.(string)
	if !ok {
		return fmt.Errorf("IDs must be strings")
	}
	*v = GqlID(str)
	return nil
}

func (v GqlID) MarshalGQL(w io.Writer) {
	io.WriteString(w, fmt.Sprintf(`"%s"`, v))
}

// ToDBIDList converts a slice of GqlIDs to a persist.DBIDList
func ToDBIDList(ids []GqlID) persist.DBIDList {
	res := make(persist.DBIDList, len(ids))
	for i, id := range ids {
		res[i] = id.DBID()
	}
	return res
}

// ToDBIDPtr converts a GqlID pointer to a persist.DBID pointer
func ToDBIDPtr(id *GqlID) *persist.DBID {
	if id == nil {
		return nil
	}
	return id.DBIDPtr()
}

// JSONOrNil returns the JSON value or nil if the pointer is nil
func JSONOrNil(j *persist.JSON) persist.JSON {
	if j == nil {
		return nil
	}
	return *j
}

// ErrInvalidIDFormat indicates an invalid ID format
type ErrInvalidIDFormat struct {
	message string
}

func (e ErrInvalidIDFormat) Error() string {
	return fmt.Sprintf("invalid ID format: %s", e.message)
}

// ErrInvalidIDType indicates an unsupported ID type
type ErrInvalidIDType struct {
	typeName string
}

func (e ErrInvalidIDType) Error() string {
	return fmt.Sprintf("no fetch method found for ID type '%s'", e.typeName)
}

// Window represents a time window for queries
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
		return fmt.Errorf("unknown window: %s", window)
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
