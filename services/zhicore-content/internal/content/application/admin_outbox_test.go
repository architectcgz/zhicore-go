package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestAdminOutboxUseCases(t *testing.T) {
	t.Run("lists failed and dead events for admin", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.outboxAdmin = &fakeOutboxAdminRepository{
			page: ports.OutboxEventPage{
				Items: []ports.OutboxEventRecord{{
					EventID:          "evt_post_published_1",
					EventType:        "content.post.published",
					AggregateType:    "post",
					AggregateID:      "post_1",
					AggregateVersion: 6,
					Status:           "FAILED",
					AttemptCount:     2,
					LastError:        "amqp://content:secret@mq.internal:5672 closed while publishing",
					OccurredAt:       time.Date(2026, 7, 5, 15, 0, 0, 0, time.UTC),
					CreatedAt:        time.Date(2026, 7, 5, 15, 0, 1, 0, time.UTC),
					UpdatedAt:        time.Date(2026, 7, 5, 15, 1, 0, 0, time.UTC),
				}},
				Page:  2,
				Size:  20,
				Total: 21,
			},
		}
		service := NewService(deps.asDeps())

		result, err := service.ListAdminOutboxEvents(context.Background(), ListAdminOutboxEventsQuery{
			Actor:     &Actor{UserID: 1001, Roles: []string{"editor", "admin"}},
			Status:    "failed",
			EventType: "content.post.published",
			Page:      2,
			Size:      20,
		})
		if err != nil {
			t.Fatalf("ListAdminOutboxEvents() error = %v", err)
		}
		if deps.outboxAdmin.listQuery.Status != "FAILED" || deps.outboxAdmin.listQuery.EventType != "content.post.published" || deps.outboxAdmin.listQuery.Page != 2 || deps.outboxAdmin.listQuery.Size != 20 {
			t.Fatalf("list query = %+v", deps.outboxAdmin.listQuery)
		}
		if result.Total != 21 || len(result.Items) != 1 || result.Items[0].RetryCount != 2 || result.Items[0].AggregateVersion != 6 {
			t.Fatalf("result = %+v", result)
		}
		if result.Items[0].LastError != "<redacted-url> closed while publishing" {
			t.Fatalf("last error = %q, want URL redacted", result.Items[0].LastError)
		}
	})

	t.Run("rejects non-admin list and invalid status", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.outboxAdmin = &fakeOutboxAdminRepository{}
		service := NewService(deps.asDeps())

		_, err := service.ListAdminOutboxEvents(context.Background(), ListAdminOutboxEventsQuery{
			Actor:  &Actor{UserID: 1001, Roles: []string{"writer"}},
			Status: "failed",
		})
		if !errors.Is(err, ErrRoleRequired) {
			t.Fatalf("ListAdminOutboxEvents(non-admin) error = %v, want ErrRoleRequired", err)
		}

		_, err = service.ListAdminOutboxEvents(context.Background(), ListAdminOutboxEventsQuery{
			Actor:  &Actor{UserID: 1001, Roles: []string{"admin"}},
			Status: "published",
		})
		if !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("ListAdminOutboxEvents(invalid status) error = %v, want ErrInvalidArgument", err)
		}
		if deps.outboxAdmin.listCalls != 0 {
			t.Fatalf("list calls = %d, want none after validation failures", deps.outboxAdmin.listCalls)
		}
	})

	t.Run("retries event with admin audit context", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.outboxAdmin = &fakeOutboxAdminRepository{
			retryResult: ports.OutboxRetryResult{
				EventID:    "evt_post_published_1",
				Status:     "PENDING",
				RetryCount: 2,
				RetriedAt:  deps.clock.now,
			},
		}
		service := NewService(deps.asDeps())

		result, err := service.RetryAdminOutboxEvent(context.Background(), RetryAdminOutboxEventCommand{
			Actor:   &Actor{UserID: 1001, Roles: []string{"ROLE_ADMIN"}},
			EventID: "evt_post_published_1",
			Reason:  "manual replay after RabbitMQ recovery",
		})
		if err != nil {
			t.Fatalf("RetryAdminOutboxEvent() error = %v", err)
		}
		if deps.outboxAdmin.retryCommand.EventID != "evt_post_published_1" ||
			deps.outboxAdmin.retryCommand.AdminUserID != 1001 ||
			deps.outboxAdmin.retryCommand.Reason != "manual replay after RabbitMQ recovery" ||
			!deps.outboxAdmin.retryCommand.RetriedAt.Equal(deps.clock.now) {
			t.Fatalf("retry command = %+v", deps.outboxAdmin.retryCommand)
		}
		if result.EventID != "evt_post_published_1" || result.Status != "PENDING" || result.RetryCount != 2 {
			t.Fatalf("retry result = %+v", result)
		}
	})

	t.Run("rejects retry without admin role or reason", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.outboxAdmin = &fakeOutboxAdminRepository{}
		service := NewService(deps.asDeps())

		_, err := service.RetryAdminOutboxEvent(context.Background(), RetryAdminOutboxEventCommand{
			Actor:   &Actor{UserID: 1001, Roles: []string{"writer"}},
			EventID: "evt_post_published_1",
			Reason:  "manual replay",
		})
		if !errors.Is(err, ErrRoleRequired) {
			t.Fatalf("RetryAdminOutboxEvent(non-admin) error = %v, want ErrRoleRequired", err)
		}

		_, err = service.RetryAdminOutboxEvent(context.Background(), RetryAdminOutboxEventCommand{
			Actor:   &Actor{UserID: 1001, Roles: []string{"admin"}},
			EventID: "evt_post_published_1",
		})
		if !errors.Is(err, ErrInvalidArgument) {
			t.Fatalf("RetryAdminOutboxEvent(no reason) error = %v, want ErrInvalidArgument", err)
		}
		if deps.outboxAdmin.retryCalls != 0 {
			t.Fatalf("retry calls = %d, want none after validation failures", deps.outboxAdmin.retryCalls)
		}
	})

	t.Run("preserves missing event error for handler mapping", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.outboxAdmin = &fakeOutboxAdminRepository{retryErr: ports.ErrOutboxEventNotFound}
		service := NewService(deps.asDeps())

		_, err := service.RetryAdminOutboxEvent(context.Background(), RetryAdminOutboxEventCommand{
			Actor:   &Actor{UserID: 1001, Roles: []string{"admin"}},
			EventID: "evt_missing",
			Reason:  "manual replay",
		})
		if !errors.Is(err, ErrOutboxEventNotFound) {
			t.Fatalf("RetryAdminOutboxEvent() error = %v, want ErrOutboxEventNotFound", err)
		}
	})

	t.Run("rate limits retry before repository mutation", func(t *testing.T) {
		deps := newCreatePostDeps()
		deps.outboxAdmin = &fakeOutboxAdminRepository{}
		limiter := &recordingRateLimiter{decision: ports.RateLimitDecision{
			Outcome: ports.RateLimitOutcomeDegradedDenyUnavailable,
			Reason:  "redis_unavailable_fail_closed",
		}}
		serviceDeps := deps.asDeps()
		serviceDeps.Limiter = limiter
		service := NewService(serviceDeps)

		_, err := service.RetryAdminOutboxEvent(context.Background(), RetryAdminOutboxEventCommand{
			Actor:   &Actor{UserID: 1001, Roles: []string{"admin"}},
			EventID: "evt_post_published_1",
			Reason:  "manual replay",
		})

		if !errors.Is(err, ErrDependencyUnavailable) {
			t.Fatalf("RetryAdminOutboxEvent() error = %v, want ErrDependencyUnavailable", err)
		}
		if deps.outboxAdmin.retryCalls != 0 {
			t.Fatalf("retry calls = %d, want none", deps.outboxAdmin.retryCalls)
		}
		if len(limiter.requests) != 1 {
			t.Fatalf("rate limit requests = %+v, want one", limiter.requests)
		}
		got := limiter.requests[0]
		if got.LimitType != ports.RateLimitTypeAdminCommand ||
			got.Subject != "actor:1001" ||
			got.Resource != "evt_post_published_1" ||
			got.Operation != "retry_admin_outbox_event" {
			t.Fatalf("rate limit request = %+v, want admin command event retry", got)
		}
	})
}

type fakeOutboxAdminRepository struct {
	listCalls    int
	listQuery    ports.OutboxEventQuery
	page         ports.OutboxEventPage
	retryCalls   int
	retryCommand ports.OutboxRetryCommand
	retryResult  ports.OutboxRetryResult
	retryErr     error
}

func (f *fakeOutboxAdminRepository) ListOutboxEvents(ctx context.Context, query ports.OutboxEventQuery) (ports.OutboxEventPage, error) {
	f.listCalls++
	f.listQuery = query
	return f.page, nil
}

func (f *fakeOutboxAdminRepository) RetryOutboxEvent(ctx context.Context, command ports.OutboxRetryCommand) (ports.OutboxRetryResult, error) {
	f.retryCalls++
	f.retryCommand = command
	if f.retryErr != nil {
		return ports.OutboxRetryResult{}, f.retryErr
	}
	return f.retryResult, nil
}
