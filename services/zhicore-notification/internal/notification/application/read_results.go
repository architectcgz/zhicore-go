package application

import "time"

type MarkNotificationReadResult struct {
	NotificationID string
	Read           bool
	Changed        bool
	ReadAt         time.Time
}

type MarkAllNotificationsReadResult struct {
	ReadAll       bool
	AffectedCount int64
	ReadAt        time.Time
}

type UnreadCountResult struct {
	UnreadCount int64
}

type UnreadBreakdownResult struct {
	Total       int64
	Interaction int64
	Content     int64
	Social      int64
	System      int64
	Security    int64
}
