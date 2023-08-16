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
	ActionUserCreated       Action = "UserCreated"
	ActionUserFollowedUsers Action = "UserFollowedUsers"
	ActionCollectionCreated Action = "CollectionCreated"
	ActionViewedSplit       Action = "ViewedSplit"
	ActionSplitUpdated      Action = "SplitUpdated"
)

type EventData struct {
	UserBio                           string            `json:"user_bio"`
	UserFollowedBack                  bool              `json:"user_followed_back"`
	UserRefollowed                    bool              `json:"user_refollowed"`
	TokenCollectorsNote               string            `json:"token_collectors_note"`
	TokenCollectionID                 DBID              `json:"token_collection_id"`
	CollectionTokenIDs                DBIDList          `json:"collection_token_ids"`
	CollectionCollectorsNote          string            `json:"collection_collectors_note"`
	SplitName                         *string           `json:"split_name"`
	SplitDescription                  *string           `json:"split_description"`
	SplitNewCollectionCollectorsNotes map[DBID]string   `json:"split_new_collection_collectors_notes"`
	SplitNewTokenIDs                  map[DBID]DBIDList `json:"split_new_token_ids"`
	SplitNewCollections               DBIDList          `json:"split_new_collections"`
	SplitNewTokenCollectorsNotes      map[DBID]string   `json:"split_new_token_collectors_notes"`
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
