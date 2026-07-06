package application

import (
	"errors"
	"fmt"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

var (
	ErrInvalidRequest        = errors.New("invalid request")
	ErrLoginRequired         = errors.New("login required")
	ErrNotificationNotFound  = ports.ErrNotificationNotFound
	ErrDependencyUnavailable = ports.ErrDependencyUnavailable
)

type Actor struct {
	UserID int64
	Roles  []string
}

type Dependencies struct {
	Commands   ports.NotificationCommandRepository
	Queries    ports.NotificationQueryRepository
	Unread     ports.UnreadCountCacheStore
	IDs        ports.NotificationPublicIDCodec
	Settings   ports.NotificationSettingsRepository
	Deliveries ports.DeliveryRepository
	Clock      ports.Clock
}

type Service struct {
	commands   ports.NotificationCommandRepository
	queries    ports.NotificationQueryRepository
	unread     ports.UnreadCountCacheStore
	ids        ports.NotificationPublicIDCodec
	settings   ports.NotificationSettingsRepository
	deliveries ports.DeliveryRepository
	clock      ports.Clock
}

func NewService(deps Dependencies) (*Service, error) {
	if deps.Commands == nil {
		return nil, fmt.Errorf("notification command repository is required")
	}
	if deps.Queries == nil {
		return nil, fmt.Errorf("notification query repository is required")
	}
	if deps.Unread == nil {
		return nil, fmt.Errorf("notification unread cache is required")
	}
	if deps.IDs == nil {
		return nil, fmt.Errorf("notification public id codec is required")
	}
	if deps.Clock == nil {
		deps.Clock = systemClock{}
	}
	return &Service{
		commands:   deps.Commands,
		queries:    deps.Queries,
		unread:     deps.Unread,
		ids:        deps.IDs,
		settings:   deps.Settings,
		deliveries: deps.Deliveries,
		clock:      deps.Clock,
	}, nil
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now().UTC()
}

func requireActor(actor Actor) error {
	if actor.UserID <= 0 {
		return ErrLoginRequired
	}
	return nil
}

func unreadCacheKeys(userID int64) []string {
	return []string{
		fmt.Sprintf("notification:%d:unread", userID),
		fmt.Sprintf("notification:%d:aggregation", userID),
	}
}

func preferenceCacheKey(userID int64) string {
	return fmt.Sprintf("notification:%d:preferences", userID)
}

func dndCacheKey(userID int64) string {
	return fmt.Sprintf("notification:%d:dnd", userID)
}

func authorSubscriptionCacheKey(userID, authorID int64) string {
	return fmt.Sprintf("notification:%d:author:%d:subscription", userID, authorID)
}

func hasAdminRole(actor Actor) bool {
	for _, role := range actor.Roles {
		if role == "admin" || role == "ADMIN" {
			return true
		}
	}
	return false
}
