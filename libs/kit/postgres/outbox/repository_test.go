package outbox

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRepositoryClaimPendingUsesAtomicSkipLockedClaim(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewDispatchRepository(db, Config{Table: "outbox_events"})

	now := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)
	occurredAt := now.Add(-time.Minute)
	payload := []byte(`{"commentId":"c1"}`)

	mock.ExpectQuery(regexp.QuoteMeta(repo.claimPendingSQL)).
		WithArgs(now, now.Add(-30*time.Second), "dispatcher:test", 10).
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

	events, err := repo.ClaimPending(context.Background(), ClaimOptions{
		DispatcherID: "dispatcher:test",
		BatchSize:    10,
		StaleAfter:   30 * time.Second,
		Now:          now,
	})
	if err != nil {
		t.Fatalf("ClaimPending() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events len = %d, want 1", len(events))
	}
	if got := events[0]; got.ID != 42 || got.EventID != "evt_comment_liked_1" || got.PayloadVersion != 2 || got.AttemptCount != 2 {
		t.Fatalf("claimed event = %#v", got)
	}
	assertExpectations(t, mock)
}

func TestRepositoryClaimPendingCanReadConfiguredAggregateVersionColumn(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewDispatchRepository(db, Config{
		Table:                  "outbox_events",
		AggregateVersionColumn: "aggregate_version",
	})

	now := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)
	occurredAt := now.Add(-time.Minute)

	mock.ExpectQuery(regexp.QuoteMeta(repo.claimPendingSQL)).
		WithArgs(now, now.Add(-30*time.Second), "dispatcher:test", 10).
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
			[]byte(`{"postId":"post_1"}`),
			occurredAt,
			2,
		))

	events, err := repo.ClaimPending(context.Background(), ClaimOptions{
		DispatcherID: "dispatcher:test",
		BatchSize:    10,
		StaleAfter:   30 * time.Second,
		Now:          now,
	})
	if err != nil {
		t.Fatalf("ClaimPending() error = %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events len = %d, want 1", len(events))
	}
	if events[0].AggregateVersion == nil || *events[0].AggregateVersion != 6 {
		t.Fatalf("aggregate version = %#v, want 6", events[0].AggregateVersion)
	}
	assertExpectations(t, mock)
}

func TestRepositoryMarkPublishedDetectsLostClaim(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewDispatchRepository(db, Config{Table: "outbox_events"})

	mock.ExpectExec(regexp.QuoteMeta(repo.markPublishedSQL)).
		WithArgs(sqlmock.AnyArg(), int64(42), "dispatcher:test").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err := repo.MarkPublished(context.Background(), Published{
		ID:           42,
		DispatcherID: "dispatcher:test",
		PublishedAt:  time.Date(2026, 7, 5, 10, 1, 0, 0, time.UTC),
	})
	if !errors.Is(err, ErrClaimLost) {
		t.Fatalf("MarkPublished() error = %v, want ErrClaimLost", err)
	}
	assertExpectations(t, mock)
}

func TestRepositoryMarkFailedCanDeadLetter(t *testing.T) {
	db, mock := newMockDB(t)
	repo := NewDispatchRepository(db, Config{Table: "outbox_events"})
	failedAt := time.Date(2026, 7, 5, 10, 2, 0, 0, time.UTC)

	mock.ExpectExec(regexp.QuoteMeta(repo.markFailedSQL)).
		WithArgs("DEAD", 5, nil, "confirm nack", failedAt, int64(42), "dispatcher:test").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.MarkFailed(context.Background(), Failure{
		ID:           42,
		DispatcherID: "dispatcher:test",
		AttemptCount: 5,
		Dead:         true,
		LastError:    "confirm nack",
		FailedAt:     failedAt,
	})
	if err != nil {
		t.Fatalf("MarkFailed() error = %v", err)
	}
	assertExpectations(t, mock)
}

func TestDispatchRepositorySQLDocumentsClaimBasedSchema(t *testing.T) {
	db, _ := newMockDB(t)
	repo := NewDispatchRepository(db, Config{Table: "outbox_events"})

	for _, fragment := range []string{
		"FOR UPDATE SKIP LOCKED",
		"status = 'CLAIMING'",
		"claimed_by = $3",
		"claim_started_at < $2",
	} {
		if !strings.Contains(repo.claimPendingSQL, fragment) {
			t.Fatalf("claim SQL missing %q:\n%s", fragment, repo.claimPendingSQL)
		}
	}
	if !strings.Contains(repo.markFailedSQL, "status = $1") {
		t.Fatalf("mark failed SQL must accept FAILED or DEAD status:\n%s", repo.markFailedSQL)
	}
}

func TestInsertPublisherUsesConfiguredTableAndVersionColumn(t *testing.T) {
	db, mock := newMockDB(t)
	publisher := NewInsertPublisher(Config{Table: "auth_outbox_events", VersionColumn: "event_version"}, fixedEventIDGenerator("evt_auth_1"))
	now := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)

	mock.ExpectExec(regexp.QuoteMeta(publisher.insertSQL)).
		WithArgs(
			"evt_auth_1",
			"auth.account.registered",
			1,
			"account",
			"42",
			[]byte(`{"accountId":42}`),
			now,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := publisher.Publish(context.Background(), db, Message{
		EventType:     "auth.account.registered",
		AggregateType: "account",
		AggregateID:   "42",
		Payload:       []byte(`{"accountId":42}`),
		OccurredAt:    now,
	})
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	assertExpectations(t, mock)
}

func TestConstructorsFailFastForInvalidSQLIdentifier(t *testing.T) {
	assertPanicContains(t, func() {
		NewDispatchRepository(nil, Config{Table: "public.outbox_events"})
	}, "invalid SQL identifier")
	assertPanicContains(t, func() {
		NewInsertPublisher(Config{VersionColumn: "event-version"}, fixedEventIDGenerator("evt_1"))
	}, "invalid SQL identifier")
}

func TestNewInsertPublisherRejectsNilEventIDGenerator(t *testing.T) {
	assertPanicContains(t, func() {
		NewInsertPublisher(Config{Table: "outbox_events"}, nil)
	}, "nil event id generator")
}

type fixedEventIDGenerator string

func (g fixedEventIDGenerator) NewEventID() (string, error) { return string(g), nil }

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

func assertPanicContains(t *testing.T, fn func(), want string) {
	t.Helper()
	defer func() {
		got := recover()
		if got == nil {
			t.Fatalf("expected panic containing %q", want)
		}
		if !strings.Contains(got.(string), want) {
			t.Fatalf("panic = %q, want contains %q", got, want)
		}
	}()
	fn()
}
