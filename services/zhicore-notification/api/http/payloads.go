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
	Type              string          `json:"type"`
	Category          string          `json:"category"`
	TargetType        string          `json:"targetType"`
	TargetID          string          `json:"targetId"`
	TotalCount        int64           `json:"totalCount"`
	UnreadCount       int64           `json:"unreadCount"`
	LatestTime        string          `json:"latestTime"`
	LatestContent     string          `json:"latestContent"`
	ActorIDs          []int64         `json:"actorIds"`
	AggregatedContent json.RawMessage `json:"aggregatedContent"`
}
