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

type PlanPostPublishedCampaignInput struct {
	Event         ConsumedEventMetadata
	SourceEventID string
	CampaignType  string
	AuthorID      int64
	PostID        int64
	ObjectType    string
	ObjectID      int64
	Title         string
	Excerpt       string
	Payload       []byte
	PublishedAt   time.Time
	CreatedAt     time.Time
}

type PlanCampaignResult struct {
	Created    bool
	CampaignID int64
	ShardID    int64
}

type ClaimCampaignShardInput struct {
	WorkerID     string
	Now          time.Time
	ClaimTimeout time.Duration
}

type ClaimedCampaignShard struct {
	Found           bool
	ShardID         int64
	CampaignID      int64
	FollowerCursor  string
	AttemptCount    int
	ClaimDeadlineAt time.Time
}

type CampaignRepository interface {
	PlanPostPublishedCampaign(ctx context.Context, input PlanPostPublishedCampaignInput) (PlanCampaignResult, error)
	ClaimCampaignShard(ctx context.Context, input ClaimCampaignShardInput) (ClaimedCampaignShard, error)
}

type NotificationSettingsRepository interface {
	GetNotificationPreferences(ctx context.Context, userID int64) (NotificationPreferences, error)
	SaveNotificationPreferences(ctx context.Context, input SaveNotificationPreferencesInput) (NotificationPreferences, error)
	GetNotificationDND(ctx context.Context, userID int64) (NotificationDND, error)
	SaveNotificationDND(ctx context.Context, input SaveNotificationDNDInput) (NotificationDND, error)
	GetAuthorSubscription(ctx context.Context, input GetAuthorSubscriptionInput) (AuthorSubscription, error)
	SaveAuthorSubscription(ctx context.Context, input SaveAuthorSubscriptionInput) (AuthorSubscription, error)
}

type NotificationPreferences struct {
	UserID      int64
	Preferences []NotificationPreference
}

type NotificationPreference struct {
	NotificationType string
	Channel          string
	Enabled          bool
}

type SaveNotificationPreferencesInput struct {
	UserID      int64
	Preferences []NotificationPreference
	UpdatedAt   time.Time
}

type NotificationDND struct {
	UserID     int64
	Enabled    bool
	StartTime  string
	EndTime    string
	Timezone   string
	Categories []string
	Channels   []string
}

type SaveNotificationDNDInput struct {
	UserID     int64
	Enabled    bool
	StartTime  string
	EndTime    string
	Timezone   string
	Categories []string
	Channels   []string
	UpdatedAt  time.Time
}

type GetAuthorSubscriptionInput struct {
	UserID   int64
	AuthorID int64
}

type AuthorSubscription struct {
	UserID           int64
	AuthorID         int64
	Level            string
	InAppEnabled     bool
	WebsocketEnabled bool
	EmailEnabled     bool
	DigestEnabled    bool
}

type SaveAuthorSubscriptionInput struct {
	UserID           int64
	AuthorID         int64
	Level            string
	InAppEnabled     bool
	WebsocketEnabled bool
	EmailEnabled     bool
	DigestEnabled    bool
	UpdatedAt        time.Time
}

type DeliveryRepository interface {
	ListDeliveries(ctx context.Context, query ListDeliveriesQuery) (DeliveryPage, error)
	RetryDelivery(ctx context.Context, input RetryDeliveryInput) (DeliveryRetryResult, error)
}

type ListDeliveriesQuery struct {
	RequesterID int64
	IsAdmin     bool
	RecipientID int64
	Channel     string
	Status      string
	Cursor      string
	Size        int
}

type DeliveryPage struct {
	Items      []Delivery
	NextCursor string
	HasMore    bool
}

type Delivery struct {
	DeliveryID       string
	RecipientID      int64
	NotificationID   *string
	Channel          string
	NotificationType string
	Status           string
	Provider         string
	AttemptCount     int
	LastErrorCode    string
	NextRetryAt      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type RetryDeliveryInput struct {
	DeliveryID  int64
	RequesterID int64
	IsAdmin     bool
	RetriedAt   time.Time
}

type DeliveryRetryResult struct {
	PublicID    string
	RecipientID int64
	Status      string
	Retried     bool
}
