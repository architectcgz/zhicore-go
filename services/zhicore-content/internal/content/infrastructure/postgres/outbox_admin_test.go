package postgres

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestOutboxAdminRepositoryListOutboxEvents(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewOutboxAdminRepository(db)
	occurredAt := time.Date(2026, 7, 5, 15, 0, 0, 0, time.UTC)
	createdAt := occurredAt.Add(time.Second)
	updatedAt := occurredAt.Add(time.Minute)

	mock.ExpectQuery(regexp.QuoteMeta(listAdminOutboxEventsSQL)).
		WithArgs("FAILED", "content.post.published", 20, 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"event_id",
			"event_type",
			"aggregate_type",
			"aggregate_id",
			"aggregate_version",
			"status",
			"attempt_count",
			"last_error",
			"occurred_at",
			"created_at",
			"updated_at",
			"total_count",
		}).AddRow(
			"evt_post_published_1",
			"content.post.published",
			"post",
			"post_1",
			int64(6),
			"FAILED",
			2,
			"rabbitmq publish failed",
			occurredAt,
			createdAt,
			updatedAt,
			int64(21),
		))

	page, err := repo.ListOutboxEvents(context.Background(), ports.OutboxEventQuery{
		Status:    "FAILED",
		EventType: "content.post.published",
		Page:      2,
		Size:      20,
	})
	if err != nil {
		t.Fatalf("ListOutboxEvents() error = %v", err)
	}
	if page.Page != 2 || page.Size != 20 || page.Total != 21 || len(page.Items) != 1 {
		t.Fatalf("page = %+v", page)
	}
	got := page.Items[0]
	if got.EventID != "evt_post_published_1" || got.AggregateVersion != 6 || got.AttemptCount != 2 || got.LastError != "rabbitmq publish failed" {
		t.Fatalf("item = %+v", got)
	}
	assertExpectations(t, mock)
}

func TestOutboxAdminRepositoryRetryOutboxEventWritesAudit(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewOutboxAdminRepository(db)
	retriedAt := time.Date(2026, 7, 5, 16, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(retryAdminOutboxEventSQL)).
		WithArgs("evt_post_published_1", int64(1001), "manual replay", retriedAt).
		WillReturnRows(sqlmock.NewRows([]string{"event_id", "status", "attempt_count"}).
			AddRow("evt_post_published_1", "PENDING", 2))
	mock.ExpectCommit()

	result, err := repo.RetryOutboxEvent(context.Background(), ports.OutboxRetryCommand{
		EventID:     "evt_post_published_1",
		AdminUserID: 1001,
		Reason:      "manual replay",
		RetriedAt:   retriedAt,
	})
	if err != nil {
		t.Fatalf("RetryOutboxEvent() error = %v", err)
	}
	if result.EventID != "evt_post_published_1" || result.Status != "PENDING" || result.RetryCount != 2 || !result.RetriedAt.Equal(retriedAt) {
		t.Fatalf("result = %+v", result)
	}
	assertExpectations(t, mock)
}

func TestOutboxAdminRepositoryRetryOutboxEventDetectsMissingOrNotRetryable(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewOutboxAdminRepository(db)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(retryAdminOutboxEventSQL)).
		WithArgs("evt_missing", int64(1001), "manual replay", sqlmock.AnyArg()).
		WillReturnError(ports.ErrOutboxEventNotFound)
	mock.ExpectRollback()

	_, err := repo.RetryOutboxEvent(context.Background(), ports.OutboxRetryCommand{
		EventID:     "evt_missing",
		AdminUserID: 1001,
		Reason:      "manual replay",
		RetriedAt:   time.Date(2026, 7, 5, 16, 0, 0, 0, time.UTC),
	})
	if !errors.Is(err, ports.ErrOutboxEventNotFound) {
		t.Fatalf("RetryOutboxEvent() error = %v, want ErrOutboxEventNotFound", err)
	}
	assertExpectations(t, mock)
}
