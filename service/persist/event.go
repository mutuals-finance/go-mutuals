package persist

import (
	"database/sql"
	"fmt"
)

type ResourceType int
type Action string
type ActionList []Action

const (
	ResourceTypeUser ResourceType = iota
	ResourceTypeToken
	ResourceTypeCollection
	ResourceTypeSplit
	ResourceTypeAllUsers
	ActionUserCreated              Action = "UserCreated"
	ActionUserFollowedUsers        Action = "UserFollowedUsers"
	ActionCollectionCreated        Action = "CollectionCreated"
	ActionAdmiredToken             Action = "AdmiredToken"
	ActionViewedSplit              Action = "ViewedSplit"
	ActionViewedToken              Action = "ViewedToken"
	ActionSplitUpdated             Action = "SplitUpdated"
	ActionSplitInfoUpdated         Action = "SplitInfoUpdated"
	ActionNewTokensReceived        Action = "NewTokensReceived"
	ActionTopActivityBadgeReceived Action = "ActivityBadgeReceived"
	ActionAnnouncement             Action = "Announcement"
)

type EventData struct {
	UserBio                string               `json:"user_bio"`
	UserFollowedBack       bool                 `json:"user_followed_back"`
	UserRefollowed         bool                 `json:"user_refollowed"`
	NewTokenID             DBID                 `json:"new_token_id"`
	NewTokenQuantity       HexString            `json:"new_token_quantity"`
	TokenContractID        DBID                 `json:"token_contract_id"`
	TokenDefinitionID      DBID                 `json:"token_definition_id"`
	SplitName              *string              `json:"split_name"`
	SplitDescription       *string              `json:"split_description"`
	SplitNewTokenIDs       map[DBID]DBIDList    `json:"split_new_token_ids"`
	ActivityBadgeThreshold int                  `json:"activity_badge_threshold"`
	NewTopActiveUser       bool                 `json:"new_top_active_user"`
	AnnouncementDetails    *AnnouncementDetails `json:"announcement_details"`
}

type FeedEventData struct {
	UserBio           string            `json:"user_bio"`
	UserFollowedIDs   DBIDList          `json:"user_followed_ids"`
	UserFollowedBack  []bool            `json:"user_followed_back"`
	TokenID           DBID              `json:"token_id"`
	TokenCollectionID DBID              `json:"token_collection_id"`
	TokenSplitID      DBID              `json:"token_split_id"`
	SplitID           DBID              `json:"split_id"`
	SplitName         string            `json:"split_name"`
	SplitDescription  string            `json:"split_description"`
	SplitNewTokenIDs  map[DBID]DBIDList `json:"split_new_token_ids"`
}

type ErrUnknownAction struct {
	Action Action
}

func (e ErrUnknownAction) Error() string {
	return fmt.Sprintf("unknown action: %s", e.Action)
}

type ErrUnknownResourceType struct {
	ResourceType ResourceType
}

func (e ErrUnknownResourceType) Error() string {
	return fmt.Sprintf("unknown resource type: %v", e.ResourceType)
}

func StrPtrToNullStr(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{Valid: true, String: *s}
}

func NullStrToStr(s sql.NullString) string {
	if !s.Valid {
		return ""
	}
	return s.String
}

func DBIDToNullStr(id DBID) sql.NullString {
	s := id.String()
	return StrPtrToNullStr(&s)
}

func NullStrToDBID(s sql.NullString) DBID {
	return DBID(NullStrToStr(s))
}
