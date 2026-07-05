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

func TestOutboxDispatchRepositoryClaimPendingOutboxUsesAtomicSkipLockedClaim(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewOutboxDispatchRepository(db)

	now := time.Date(2026, 7, 5, 15, 0, 0, 0, time.UTC)
	occurredAt := now.Add(-time.Minute)
	payload := []byte(`{"postId":"post_1"}`)

	mock.ExpectQuery("WITH picked AS").
		WithArgs(now, now.Add(-30*time.Second), "content-outbox:test", 10).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"event_id",
			"event_type",
			"payload_version",
			"aggregate_type",
			"aggregate_id",
			"aggregate_version",
			"payload_json",
			"occurred_at",
			"attempt_count",
		}).AddRow(
			int64(42),
			"evt_post_published_1",
			"content.post.published",
			2,
			"post",
			"post_1",
			int64(6),
			payload,
			occurredAt,
			2,
		))

	events, err := repo.ClaimPendingOutbox(context.Background(), ports.OutboxClaimOptions{
		DispatcherID: "content-outbox:test",
		BatchSize:    10,
		StaleAfter:   30 * time.Second,
		Now:          now,
	})
	if err != nil {
		t.Fatalf("ClaimPendingOutbox() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events len = %d, want 1", len(events))
	}
	if got := events[0]; got.ID != 42 || got.EventID != "evt_post_published_1" || got.EventType != "content.post.published" || got.PayloadVersion != 2 || got.AttemptCount != 2 {
		t.Fatalf("claimed event = %#v", got)
	}
	if events[0].AggregateVersion != 6 {
		t.Fatalf("aggregate version = %d, want 6", events[0].AggregateVersion)
	}
	if string(events[0].PayloadJSON) != string(payload) || !events[0].OccurredAt.Equal(occurredAt) {
		t.Fatalf("claimed payload/occurredAt = %s/%v", events[0].PayloadJSON, events[0].OccurredAt)
	}
	assertExpectations(t, mock)
}

func TestOutboxDispatchRepositoryMarkPublishedAndFailedRequireClaimOwner(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewOutboxDispatchRepository(db)
	publishedAt := time.Date(2026, 7, 5, 15, 1, 0, 0, time.UTC)
	failedAt := time.Date(2026, 7, 5, 15, 2, 0, 0, time.UTC)
	nextRetryAt := failedAt.Add(time.Minute)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE outbox_events")).
		WithArgs(publishedAt, int64(42), "content-outbox:test").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE outbox_events")).
		WithArgs("FAILED", 3, nextRetryAt, "rabbitmq unavailable", failedAt, int64(43), "content-outbox:test").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE outbox_events")).
		WithArgs("DEAD", 5, nil, "confirm nack", failedAt, int64(44), "content-outbox:test").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := repo.MarkOutboxPublished(context.Background(), ports.OutboxPublished{
		ID:           42,
		DispatcherID: "content-outbox:test",
		PublishedAt:  publishedAt,
	}); err != nil {
		t.Fatalf("MarkOutboxPublished() error = %v", err)
	}
	if err := repo.MarkOutboxFailed(context.Background(), ports.OutboxFailure{
		ID:           43,
		DispatcherID: "content-outbox:test",
		AttemptCount: 3,
		NextRetryAt:  &nextRetryAt,
		LastError:    "rabbitmq unavailable",
		FailedAt:     failedAt,
	}); err != nil {
		t.Fatalf("MarkOutboxFailed(retry) error = %v", err)
	}
	if err := repo.MarkOutboxFailed(context.Background(), ports.OutboxFailure{
		ID:           44,
		DispatcherID: "content-outbox:test",
		AttemptCount: 5,
		Dead:         true,
		LastError:    "confirm nack",
		FailedAt:     failedAt,
	}); err != nil {
		t.Fatalf("MarkOutboxFailed(dead) error = %v", err)
	}
	assertExpectations(t, mock)
}

func TestOutboxDispatchRepositoryDetectsLostClaim(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewOutboxDispatchRepository(db)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE outbox_events")).
		WithArgs(sqlmock.AnyArg(), int64(42), "content-outbox:test").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.MarkOutboxPublished(context.Background(), ports.OutboxPublished{
		ID:           42,
		DispatcherID: "content-outbox:test",
		PublishedAt:  time.Date(2026, 7, 5, 15, 1, 0, 0, time.UTC),
	})
	if !errors.Is(err, ErrOutboxClaimLost) {
		t.Fatalf("MarkOutboxPublished() error = %v, want ErrOutboxClaimLost", err)
	}
	assertExpectations(t, mock)
}
