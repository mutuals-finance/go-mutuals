package persist

import (
	"fmt"
	"github.com/SplitFi/go-splitfi/util"
	"io"
	"strings"
)

type AnnouncementPlatform string

const (
	AnnouncementPlatformWeb    AnnouncementPlatform = "Web"
	AnnouncementPlatformMobile AnnouncementPlatform = "Mobile"
	AnnouncementPlatformAll    AnnouncementPlatform = "All"
)

func (a AnnouncementPlatform) String() string {
	return string(a)
}

func (a AnnouncementPlatform) IsValid() bool {
	switch a {
	case AnnouncementPlatformWeb, AnnouncementPlatformAll, AnnouncementPlatformMobile:
		return true
	default:
		return false
	}
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (c *AnnouncementPlatform) UnmarshalGQL(v interface{}) error {
	n, ok := v.(string)
	if !ok {
		return fmt.Errorf("chain must be a string")
	}

	switch strings.ToLower(n) {
	case "web":
		*c = AnnouncementPlatformWeb
	case "all":
		*c = AnnouncementPlatformAll
	case "mobile":
		*c = AnnouncementPlatformMobile
	default:
		return fmt.Errorf("invalid announcement platform: %s", n)
	}
	return nil
}

// MarshalGQL implements the graphql.Marshaler interface
func (c AnnouncementPlatform) MarshalGQL(w io.Writer) {
	switch c {
	case AnnouncementPlatformWeb:
		w.Write([]byte(`"Web"`))
	case AnnouncementPlatformMobile:
		w.Write([]byte(`"Mobile"`))
	case AnnouncementPlatformAll:
		w.Write([]byte(`"All"`))
	default:
		panic("invalid announcement platform")
	}
}

type AnnouncementDetails struct {
	Platform             AnnouncementPlatform `json:"platform" binding:"required"`
	InternalID           string               `json:"internal_id" binding:"required"`
	ImageURL             string               `json:"image_url,omitempty"`
	Title                string               `json:"title,omitempty"`
	Description          string               `json:"description,omitempty"`
	CTAText              string               `json:"cta_text,omitempty"`
	CTALink              string               `json:"cta_link,omitempty"`
	PushNotificationText string               `json:"push_notification_text,omitempty"`
}

type NotificationData struct {
	AuthedViewerIDs        []DBID               `json:"viewer_ids,omitempty"`
	UnauthedViewerIDs      []string             `json:"unauthed_viewer_ids,omitempty"`
	NewTokenID             DBID                 `json:"new_token_id,omitempty"`
	NewTokenQuantity       HexString            `json:"new_token_quantity,omitempty"`
	ActivityBadgeThreshold int                  `json:"activity_badge_threshold,omitempty"`
	NewTopActiveUser       bool                 `json:"new_top_active_user,omitempty"`
	AnnouncementDetails    *AnnouncementDetails `json:"announcement_details,omitempty"`
}

func (n NotificationData) Validate() NotificationData {
	result := NotificationData{}
	result.AuthedViewerIDs = uniqueDBIDs(n.AuthedViewerIDs)
	result.UnauthedViewerIDs = uniqueStrings(n.UnauthedViewerIDs)
	result.NewTokenID = n.NewTokenID
	result.NewTokenQuantity = n.NewTokenQuantity
	result.ActivityBadgeThreshold = n.ActivityBadgeThreshold
	result.NewTopActiveUser = n.NewTopActiveUser
	result.AnnouncementDetails = n.AnnouncementDetails

	return result
}

func (n NotificationData) Concat(other NotificationData) NotificationData {
	result := NotificationData{}
	result.AuthedViewerIDs = append(other.AuthedViewerIDs, n.AuthedViewerIDs...)
	result.UnauthedViewerIDs = append(other.UnauthedViewerIDs, n.UnauthedViewerIDs...)
	result.NewTokenQuantity = other.NewTokenQuantity.Add(n.NewTokenQuantity)
	result.NewTokenID = DBID(util.FirstNonEmptyString(other.NewTokenID.String(), n.NewTokenID.String()))
	result.ActivityBadgeThreshold, _ = util.FindFirst([]int{other.ActivityBadgeThreshold, n.ActivityBadgeThreshold}, func(i int) bool {
		return i > 0
	})
	result.NewTopActiveUser = other.NewTopActiveUser || n.NewTopActiveUser
	if other.AnnouncementDetails != nil {
		result.AnnouncementDetails = other.AnnouncementDetails
	} else {
		result.AnnouncementDetails = n.AnnouncementDetails
	}

	return result.Validate()
}

func uniqueDBIDs(ids []DBID) []DBID {
	seen := make(map[DBID]bool)
	result := []DBID{}

	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			seen[id] = true
			result = append(result, id)
		}
	}

	return result
}

func uniqueStrings(strs []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, str := range strs {
		if _, ok := seen[str]; !ok {
			seen[str] = true
			result = append(result, str)
		}
	}

	return result
}
