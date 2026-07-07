package application

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

const (
	defaultAdminOutboxPage = 1
	defaultAdminOutboxSize = 20
	maxAdminOutboxSize     = 100
	maxAdminOutboxErrorLen = 512
)

var adminOutboxSensitiveURLPattern = regexp.MustCompile(`(?i)\b(?:amqps?|https?|postgres|mongodb(?:\+srv)?)://[^\s]+`)

type ListAdminOutboxEventsQuery struct {
	Actor     *Actor
	Status    string
	EventType string
	Page      int
	Size      int
}

type ListAdminOutboxEventsResult struct {
	Items []AdminOutboxEventItem
	Page  int
	Size  int
	Total int64
}

type AdminOutboxEventItem struct {
	EventID          string
	EventType        string
	AggregateType    string
	AggregateID      string
	AggregateVersion int64
	Status           string
	RetryCount       int
	LastError        string
	OccurredAt       time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type RetryAdminOutboxEventCommand struct {
	Actor   *Actor
	EventID string
	Reason  string
}

type RetryAdminOutboxEventResult struct {
	EventID    string
	Status     string
	RetryCount int
	RetriedAt  time.Time
}

func (s *Service) ListAdminOutboxEvents(ctx context.Context, query ListAdminOutboxEventsQuery) (ListAdminOutboxEventsResult, error) {
	if err := requireAdminActor(query.Actor); err != nil {
		return ListAdminOutboxEventsResult{}, err
	}
	if s.admin == nil {
		return ListAdminOutboxEventsResult{}, ErrDependencyUnavailable
	}
	status, err := normalizeAdminOutboxStatus(query.Status)
	if err != nil {
		return ListAdminOutboxEventsResult{}, err
	}
	page, size := normalizeAdminOutboxPage(query.Page, query.Size)
	if err := s.enforceRateLimit(ctx, actorRateLimitRequest(ports.RateLimitTypeAdminCommand, query.Actor, "outbox_events", "list_admin_outbox_events")); err != nil {
		return ListAdminOutboxEventsResult{}, err
	}
	result, err := s.admin.ListOutboxEvents(ctx, ports.OutboxEventQuery{
		Status:    status,
		EventType: strings.TrimSpace(query.EventType),
		Page:      page,
		Size:      size,
	})
	if err != nil {
		return ListAdminOutboxEventsResult{}, fmt.Errorf("list admin outbox events: %w", err)
	}
	items := make([]AdminOutboxEventItem, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, AdminOutboxEventItem{
			EventID:          item.EventID,
			EventType:        item.EventType,
			AggregateType:    item.AggregateType,
			AggregateID:      item.AggregateID,
			AggregateVersion: item.AggregateVersion,
			Status:           item.Status,
			RetryCount:       item.AttemptCount,
			LastError:        sanitizeAdminOutboxLastError(item.LastError),
			OccurredAt:       item.OccurredAt,
			CreatedAt:        item.CreatedAt,
			UpdatedAt:        item.UpdatedAt,
		})
	}
	return ListAdminOutboxEventsResult{
		Items: items,
		Page:  result.Page,
		Size:  result.Size,
		Total: result.Total,
	}, nil
}

func (s *Service) RetryAdminOutboxEvent(ctx context.Context, command RetryAdminOutboxEventCommand) (RetryAdminOutboxEventResult, error) {
	if err := requireAdminActor(command.Actor); err != nil {
		return RetryAdminOutboxEventResult{}, err
	}
	if s.admin == nil || s.clock == nil {
		return RetryAdminOutboxEventResult{}, ErrDependencyUnavailable
	}
	eventID := strings.TrimSpace(command.EventID)
	reason := strings.TrimSpace(command.Reason)
	if eventID == "" || reason == "" {
		return RetryAdminOutboxEventResult{}, ErrInvalidArgument
	}
	if err := s.enforceRateLimit(ctx, actorRateLimitRequest(ports.RateLimitTypeAdminCommand, command.Actor, eventID, "retry_admin_outbox_event")); err != nil {
		return RetryAdminOutboxEventResult{}, err
	}
	result, err := s.admin.RetryOutboxEvent(ctx, ports.OutboxRetryCommand{
		EventID:     eventID,
		AdminUserID: command.Actor.UserID,
		Reason:      reason,
		RetriedAt:   s.clock.Now(),
	})
	if err != nil {
		return RetryAdminOutboxEventResult{}, fmt.Errorf("retry admin outbox event: %w", err)
	}
	return RetryAdminOutboxEventResult{
		EventID:    result.EventID,
		Status:     result.Status,
		RetryCount: result.RetryCount,
		RetriedAt:  result.RetriedAt,
	}, nil
}

func requireAdminActor(actor *Actor) error {
	if actor == nil || actor.UserID == 0 {
		return ErrLoginRequired
	}
	if !actor.HasRole("admin") && !actor.HasRole("role_admin") {
		return ErrRoleRequired
	}
	return nil
}

func (a *Actor) HasRole(role string) bool {
	if a == nil {
		return false
	}
	want := normalizeRole(role)
	for _, got := range a.Roles {
		if normalizeRole(got) == want {
			return true
		}
	}
	return false
}

func normalizeRole(role string) string {
	return strings.ToLower(strings.TrimSpace(role))
}

func normalizeAdminOutboxStatus(status string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(status))
	switch normalized {
	case "FAILED", "DEAD":
		return normalized, nil
	default:
		return "", ErrInvalidArgument
	}
}

func normalizeAdminOutboxPage(page, size int) (int, int) {
	if page <= 0 {
		page = defaultAdminOutboxPage
	}
	if size <= 0 {
		size = defaultAdminOutboxSize
	}
	if size > maxAdminOutboxSize {
		size = maxAdminOutboxSize
	}
	return page, size
}

func sanitizeAdminOutboxLastError(raw string) string {
	sanitized := strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
	if sanitized == "" {
		return ""
	}
	sanitized = adminOutboxSensitiveURLPattern.ReplaceAllString(sanitized, "<redacted-url>")
	if len(sanitized) <= maxAdminOutboxErrorLen {
		return sanitized
	}
	return sanitized[:maxAdminOutboxErrorLen] + "...truncated"
}
