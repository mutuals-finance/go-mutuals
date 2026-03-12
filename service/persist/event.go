package persist

import (
	"fmt"
)

type ResourceType int
type Action string
type ActionList []Action

const (
	ResourceTypeUser ResourceType = iota
	ResourceTypeToken
	ResourceTypeCollection
	ResourceTypePool
	ResourceTypeAllUsers
	ActionUserCreated              Action = "UserCreated"
	ActionUserFollowedUsers        Action = "UserFollowedUsers"
	ActionCollectionCreated        Action = "CollectionCreated"
	ActionAdmiredToken             Action = "AdmiredToken"
	ActionViewedPool               Action = "ViewedPool"
	ActionViewedToken              Action = "ViewedToken"
	ActionPoolUpdated              Action = "PoolUpdated"
	ActionPoolInfoUpdated          Action = "PoolInfoUpdated"
	ActionNewTokensReceived        Action = "NewTokensReceived"
	ActionTopActivityBadgeReceived Action = "ActivityBadgeReceived"
	ActionAnnouncement             Action = "Announcement"
)

type EventData struct {
	UserBio                string               `json:"user_bio"`
	UserFollowedBack       bool                 `json:"user_followed_back"`
	UserRefollowed         bool                 `json:"user_refollowed"`
	NewTokenID             DBID                 `json:"new_token_id"`
	NewTokenQuantity       UInt256              `json:"new_token_quantity"`
	TokenContractID        DBID                 `json:"token_contract_id"`
	TokenDefinitionID      DBID                 `json:"token_definition_id"`
	PoolName               *string              `json:"pool_name"`
	PoolDescription        *string              `json:"pool_description"`
	PoolNewTokenIDs        map[DBID]DBIDList    `json:"pool_new_token_ids"`
	ActivityBadgeThreshold int                  `json:"activity_badge_threshold"`
	NewTopActiveUser       bool                 `json:"new_top_active_user"`
	AnnouncementDetails    *AnnouncementDetails `json:"announcement_details"`
}

type AnnouncementDetails struct {
	Platform             string `json:"platform"`
	InternalID           string `json:"internal_id"`
	ImageURL             string `json:"image_url,omitempty"`
	Title                string `json:"title,omitempty"`
	Description          string `json:"description,omitempty"`
	CTAText              string `json:"cta_text,omitempty"`
	CTALink              string `json:"cta_link,omitempty"`
	PushNotificationText string `json:"push_notification_text,omitempty"`
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
