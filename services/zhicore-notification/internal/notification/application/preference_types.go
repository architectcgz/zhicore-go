package application

import "time"

type NotificationChannelPreferenceInput struct {
	InApp     bool
	Websocket bool
	Email     bool
	SMS       bool
}

type NotificationPreferenceInput struct {
	NotificationType string
	Channels         NotificationChannelPreferenceInput
}

type NotificationPreferenceResult struct {
	NotificationType string
	Channels         NotificationChannelPreferenceInput
}

type GetNotificationPreferencesQuery struct {
	Actor Actor
}

type UpdateNotificationPreferencesCommand struct {
	Actor       Actor
	Preferences []NotificationPreferenceInput
}

type NotificationPreferencesResult struct {
	UserID      int64
	Preferences []NotificationPreferenceResult
}

type GetNotificationDNDQuery struct {
	Actor Actor
}

type UpdateNotificationDNDCommand struct {
	Actor      Actor
	Enabled    bool
	StartTime  string
	EndTime    string
	Timezone   string
	Categories []string
	Channels   []string
}

type NotificationDNDResult struct {
	UserID     int64
	Enabled    bool
	StartTime  string
	EndTime    string
	Timezone   string
	Categories []string
	Channels   []string
}

type GetAuthorSubscriptionQuery struct {
	Actor    Actor
	AuthorID int64
}

type UpdateAuthorSubscriptionCommand struct {
	Actor            Actor
	AuthorID         int64
	Level            string
	InAppEnabled     bool
	WebsocketEnabled bool
	EmailEnabled     bool
	DigestEnabled    bool
}

type AuthorSubscriptionResult struct {
	UserID           int64
	AuthorID         int64
	Level            string
	InAppEnabled     bool
	WebsocketEnabled bool
	EmailEnabled     bool
	DigestEnabled    bool
}

type ListDeliveriesQuery struct {
	Actor       Actor
	RecipientID int64
	Channel     string
	Status      string
	Cursor      string
	Size        int
}

type DeliveryResult struct {
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

type DeliveryPage struct {
	Items      []DeliveryResult
	NextCursor string
	HasMore    bool
}

type RetryDeliveryCommand struct {
	Actor      Actor
	DeliveryID string
}

type DeliveryRetryResult struct {
	DeliveryID string
	Status     string
	Retried    bool
}
