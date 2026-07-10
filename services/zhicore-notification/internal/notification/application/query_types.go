package application

import "time"

type GetUnreadCountQuery struct {
	Actor Actor
}

type GetUnreadBreakdownQuery struct {
	Actor Actor
}

type ListNotificationsQuery struct {
	Actor      Actor
	Cursor     string
	Size       int
	Category   string
	UnreadOnly bool
}

type NotificationPage struct {
	Items        []AggregatedNotification
	NextCursor   string
	HasMore      bool
	RepairSignal bool
}

type AggregatedNotification struct {
	GroupID           string
	Type              string
	Category          string
	TargetType        string
	TargetID          string
	TotalCount        int64
	UnreadCount       int64
	LatestTime        time.Time
	LatestContent     string
	ActorIDs          []int64
	ActorTotalCount   int64
	RecentActors      []NotificationActorSnapshot
	AggregatedContent []byte
}

type NotificationActorSnapshot struct {
	PublicID    string
	DisplayName string
	AvatarURL   *string
}

type ListNotificationGroupActorsQuery struct {
	Actor   Actor
	GroupID string
	Cursor  string
	Size    int
}

type NotificationActorPage struct {
	Items      []NotificationActor
	NextCursor string
	HasMore    bool
}

type NotificationActor struct {
	PublicID         string
	DisplayName      string
	AvatarURL        *string
	EventCount       int64
	LatestOccurredAt time.Time
}
