package httpapi

import "encoding/json"

type markNotificationReadResp struct {
	NotificationID string `json:"notificationId"`
	Read           bool   `json:"read"`
	Changed        bool   `json:"changed"`
	ReadAt         string `json:"readAt"`
}

type markAllNotificationsReadResp struct {
	ReadAll       bool   `json:"readAll"`
	ReadAt        string `json:"readAt"`
	AffectedCount int64  `json:"affectedCount"`
}

type markNotificationGroupReadResp struct {
	GroupID      string `json:"groupId"`
	Read         bool   `json:"read"`
	ChangedCount int64  `json:"changedCount"`
	UnreadCount  int64  `json:"unreadCount"`
	ReadAt       string `json:"readAt"`
}
type notificationActorSnapshotResp struct {
	PublicID    string  `json:"publicId"`
	DisplayName string  `json:"displayName"`
	AvatarURL   *string `json:"avatarUrl"`
}
type notificationActorResp struct {
	Actor            notificationActorSnapshotResp `json:"actor"`
	EventCount       int64                         `json:"eventCount"`
	LatestOccurredAt string                        `json:"latestOccurredAt"`
}
type notificationActorPageResp struct {
	Items      []notificationActorResp `json:"items"`
	NextCursor string                  `json:"nextCursor,omitempty"`
	HasMore    bool                    `json:"hasMore"`
}

type unreadCountResp struct {
	UnreadCount int64 `json:"unreadCount"`
}

type unreadBreakdownResp struct {
	Total       int64 `json:"total"`
	Interaction int64 `json:"interaction"`
	Content     int64 `json:"content"`
	Social      int64 `json:"social"`
	System      int64 `json:"system"`
	Security    int64 `json:"security"`
}

type notificationPageResp struct {
	Items      []aggregatedNotificationResp `json:"items"`
	NextCursor string                       `json:"nextCursor,omitempty"`
	HasMore    bool                         `json:"hasMore"`
}

type aggregatedNotificationResp struct {
	GroupID          string                          `json:"groupId"`
	Type             string                          `json:"type"`
	Category         string                          `json:"category"`
	TotalCount       int64                           `json:"totalCount"`
	UnreadCount      int64                           `json:"unreadCount"`
	ActorTotalCount  int64                           `json:"actorTotalCount"`
	LatestOccurredAt string                          `json:"latestOccurredAt"`
	Content          notificationContentResp         `json:"content"`
	RecentActors     []notificationActorSnapshotResp `json:"recentActors"`
	Target           *notificationTargetResp         `json:"target"`
}

type notificationContentResp struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}
type notificationTargetResp struct {
	Resource notificationTargetRefResp  `json:"resource"`
	Anchor   *notificationTargetRefResp `json:"anchor,omitempty"`
	Snapshot json.RawMessage            `json:"snapshot"`
}
type notificationTargetRefResp struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type notificationChannelPreferencePayload struct {
	InApp     bool `json:"inApp"`
	Websocket bool `json:"websocket"`
	Email     bool `json:"email"`
	SMS       bool `json:"sms"`
}

type notificationPreferencePayload struct {
	NotificationType string                               `json:"notificationType"`
	Channels         notificationChannelPreferencePayload `json:"channels"`
}

type updateNotificationPreferencesReq struct {
	Preferences []notificationPreferencePayload `json:"preferences"`
}

type notificationPreferencesResp struct {
	UserID      int64                           `json:"userId"`
	Preferences []notificationPreferencePayload `json:"preferences"`
}

type updateNotificationDNDReq struct {
	Enabled    bool     `json:"enabled"`
	StartTime  string   `json:"startTime"`
	EndTime    string   `json:"endTime"`
	Timezone   string   `json:"timezone"`
	Categories []string `json:"categories"`
	Channels   []string `json:"channels"`
}

type notificationDNDResp struct {
	UserID     int64    `json:"userId"`
	Enabled    bool     `json:"enabled"`
	StartTime  string   `json:"startTime"`
	EndTime    string   `json:"endTime"`
	Timezone   string   `json:"timezone"`
	Categories []string `json:"categories"`
	Channels   []string `json:"channels"`
}

type updateAuthorSubscriptionReq struct {
	Level            string `json:"level"`
	InAppEnabled     bool   `json:"inAppEnabled"`
	WebsocketEnabled bool   `json:"websocketEnabled"`
	EmailEnabled     bool   `json:"emailEnabled"`
	DigestEnabled    bool   `json:"digestEnabled"`
}

type authorSubscriptionResp struct {
	UserID           int64  `json:"userId"`
	AuthorID         int64  `json:"authorId"`
	Level            string `json:"level"`
	InAppEnabled     bool   `json:"inAppEnabled"`
	WebsocketEnabled bool   `json:"websocketEnabled"`
	EmailEnabled     bool   `json:"emailEnabled"`
	DigestEnabled    bool   `json:"digestEnabled"`
}

type deliveryPageResp struct {
	Items      []deliveryResp `json:"items"`
	NextCursor string         `json:"nextCursor,omitempty"`
	HasMore    bool           `json:"hasMore"`
}

type deliveryResp struct {
	DeliveryID       string  `json:"deliveryId"`
	RecipientID      int64   `json:"recipientId"`
	NotificationID   *string `json:"notificationId,omitempty"`
	Channel          string  `json:"channel"`
	NotificationType string  `json:"notificationType"`
	Status           string  `json:"status"`
	Provider         string  `json:"provider,omitempty"`
	AttemptCount     int     `json:"attemptCount"`
	LastErrorCode    string  `json:"lastErrorCode,omitempty"`
	NextRetryAt      *string `json:"nextRetryAt,omitempty"`
	CreatedAt        string  `json:"createdAt"`
	UpdatedAt        string  `json:"updatedAt"`
}

type deliveryRetryResp struct {
	DeliveryID string `json:"deliveryId"`
	Status     string `json:"status"`
	Retried    bool   `json:"retried"`
}
