package postgres

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/ports"
)

func TestOutboxDispatchRepositoryClaimPendingOutboxUsesAtomicSkipLockedClaim(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewOutboxDispatchRepository(db)

	now := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)
	occurredAt := now.Add(-time.Minute)
	payload := []byte(`{"commentId":"c1"}`)

	mock.ExpectQuery("WITH picked AS").
		WithArgs(now, now.Add(-30*time.Second), "zhicore-comment:outbox-dispatcher:test", 10).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"event_id",
			"event_type",
			"payload_version",
			"aggregate_type",
			"aggregate_id",
			"payload_json",
			"occurred_at",
			"attempt_count",
		}).AddRow(
			int64(42),
			"evt_comment_liked_1",
			"comment.liked",
			2,
			"comment",
			"c1",
			payload,
			occurredAt,
			2,
		))

	events, err := repo.ClaimPendingOutbox(context.Background(), ports.OutboxClaimOptions{
		DispatcherID: "zhicore-comment:outbox-dispatcher:test",
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
	if got := events[0]; got.ID != 42 || got.EventID != "evt_comment_liked_1" || got.EventType != "comment.liked" || got.PayloadVersion != 2 || got.AttemptCount != 2 {
		t.Fatalf("claimed event = %#v", got)
	}
	if string(events[0].Payload) != string(payload) || !events[0].OccurredAt.Equal(occurredAt) {
		t.Fatalf("claimed payload/occurredAt = %s/%v", events[0].Payload, events[0].OccurredAt)
	}
	assertExpectations(t, mock)
}

func TestOutboxDispatchRepositoryMarkOutboxPublishedRequiresClaimOwner(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewOutboxDispatchRepository(db)
	publishedAt := time.Date(2026, 7, 5, 10, 1, 0, 0, time.UTC)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE outbox_events")).
		WithArgs(publishedAt, int64(42), "zhicore-comment:outbox-dispatcher:test").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.MarkOutboxPublished(context.Background(), ports.OutboxPublished{
		ID:           42,
		DispatcherID: "zhicore-comment:outbox-dispatcher:test",
		PublishedAt:  publishedAt,
	})
	if err != nil {
		t.Fatalf("MarkOutboxPublished() error = %v", err)
	}
	assertExpectations(t, mock)
}

func TestOutboxDispatchRepositoryMarkOutboxPublishedDetectsLostClaim(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewOutboxDispatchRepository(db)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE outbox_events")).
		WithArgs(sqlmock.AnyArg(), int64(42), "zhicore-comment:outbox-dispatcher:test").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.MarkOutboxPublished(context.Background(), ports.OutboxPublished{
		ID:           42,
		DispatcherID: "zhicore-comment:outbox-dispatcher:test",
		PublishedAt:  time.Date(2026, 7, 5, 10, 1, 0, 0, time.UTC),
	})
	if !errors.Is(err, ErrOutboxClaimLost) {
		t.Fatalf("MarkOutboxPublished() error = %v, want ErrOutboxClaimLost", err)
	}
	assertExpectations(t, mock)
}

func TestOutboxDispatchRepositoryMarkOutboxFailedRequiresClaimOwner(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewOutboxDispatchRepository(db)
	failedAt := time.Date(2026, 7, 5, 10, 2, 0, 0, time.UTC)
	nextRetryAt := failedAt.Add(time.Minute)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE outbox_events")).
		WithArgs("FAILED", 3, nextRetryAt, "rabbitmq unavailable", failedAt, int64(42), "zhicore-comment:outbox-dispatcher:test").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.MarkOutboxFailed(context.Background(), ports.OutboxFailure{
		ID:           42,
		DispatcherID: "zhicore-comment:outbox-dispatcher:test",
		AttemptCount: 3,
		NextRetryAt:  &nextRetryAt,
		LastError:    "rabbitmq unavailable",
		FailedAt:     failedAt,
	})
	if err != nil {
		t.Fatalf("MarkOutboxFailed() error = %v", err)
	}
	assertExpectations(t, mock)
}

func TestOutboxDispatchRepositoryMarkOutboxFailedCanDeadLetter(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewOutboxDispatchRepository(db)
	failedAt := time.Date(2026, 7, 5, 10, 2, 0, 0, time.UTC)

	mock.ExpectExec(regexp.QuoteMeta("UPDATE outbox_events")).
		WithArgs("DEAD", 5, nil, "confirm nack", failedAt, int64(42), "zhicore-comment:outbox-dispatcher:test").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.MarkOutboxFailed(context.Background(), ports.OutboxFailure{
		ID:           42,
		DispatcherID: "zhicore-comment:outbox-dispatcher:test",
		AttemptCount: 5,
		Dead:         true,
		LastError:    "confirm nack",
		FailedAt:     failedAt,
	})
	if err != nil {
		t.Fatalf("MarkOutboxFailed() error = %v", err)
	}
	assertExpectations(t, mock)
}

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, mock
}

func assertExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
