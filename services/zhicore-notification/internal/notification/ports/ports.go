package ports

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotificationNotFound   = errors.New("notification not found")
	ErrDependencyUnavailable  = errors.New("dependency unavailable")
	ErrDuplicateConsumedEvent = errors.New("duplicate consumed event")
)

type NotificationPublicIDCodec interface {
	Encode(id uint64) (string, error)
	Decode(publicID string) (uint64, error)
}

type Clock interface {
	Now() time.Time
}

type NotificationCommandRepository interface {
	MarkRead(ctx context.Context, input MarkReadInput) (MarkReadResult, error)
	MarkAllRead(ctx context.Context, input MarkAllReadInput) (MarkAllReadResult, error)
}

type MarkReadInput struct {
	NotificationID int64
	RecipientID    int64
	ReadAt         time.Time
}

type MarkReadResult struct {
	NotificationID int64
	PublicID       string
	Changed        bool
	ReadAt         time.Time
}

type MarkAllReadInput struct {
	RecipientID int64
	ReadAt      time.Time
}

type MarkAllReadResult struct {
	AffectedCount int64
	ReadAt        time.Time
}

type NotificationQueryRepository interface {
	GetUnreadCount(ctx context.Context, recipientID int64) (int64, error)
	GetUnreadBreakdown(ctx context.Context, recipientID int64) (UnreadBreakdown, error)
	ListAggregated(ctx context.Context, query ListAggregatedQuery) (AggregatedNotificationPage, error)
}

type ListAggregatedQuery struct {
	RecipientID int64
	Cursor      string
	Size        int
	Category    string
	UnreadOnly  bool
}

type AggregatedNotificationPage struct {
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

type UnreadBreakdown struct {
	Total       int64
	Interaction int64
	Content     int64
	Social      int64
	System      int64
	Security    int64
}

type UnreadCountCacheStore interface {
	GetUnreadCount(ctx context.Context, userID int64) (count int64, hit bool, err error)
	SetUnreadCount(ctx context.Context, userID int64, count int64) error
	Delete(ctx context.Context, keys ...string) error
}

type ConsumedEventMetadata struct {
	EventID      string
	EventType    string
	RoutingKey   string
	ConsumerName string
	PayloadHash  string
	OccurredAt   time.Time
	ExpiresAt    time.Time
}

type CreateInteractionNotificationInput struct {
	Event            ConsumedEventMetadata
	RecipientID      int64
	ActorID          *int64
	Category         string
	NotificationType string
	EventCode        string
	Importance       string
	TargetType       string
	TargetID         string
	SourceEventID    string
	DedupeKey        string
	GroupKey         string
	Title            string
	Content          string
	Payload          []byte
	OccurredAt       time.Time
	CreatedAt        time.Time
}

type CreateInteractionNotificationResult struct {
	Created        bool
	NotificationID int64
	PublicID       string
}

type InteractionNotificationStore interface {
	CreateInteractionNotification(ctx context.Context, input CreateInteractionNotificationInput) (CreateInteractionNotificationResult, error)
}
