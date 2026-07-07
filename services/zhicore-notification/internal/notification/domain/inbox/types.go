package inbox

import "time"

type NotificationID int64
type NotificationPublicID string
type UserID int64
type Category string
type NotificationType string
type TargetType string

const (
	CategoryInteraction Category = "INTERACTION"
	CategoryContent     Category = "CONTENT"
	CategorySocial      Category = "SOCIAL"
	CategorySystem      Category = "SYSTEM"
	CategorySecurity    Category = "SECURITY"
)

type AggregatedNotification struct {
	Type              NotificationType
	Category          Category
	TargetType        TargetType
	TargetID          string
	TotalCount        int64
	UnreadCount       int64
	LatestTime        time.Time
	LatestContent     string
	ActorIDs          []UserID
	AggregatedContent []byte
}
