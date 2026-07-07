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
	Type              string
	Category          string
	TargetType        string
	TargetID          string
	TotalCount        int64
	UnreadCount       int64
	LatestTime        time.Time
	LatestContent     string
	ActorIDs          []int64
	AggregatedContent []byte
}
