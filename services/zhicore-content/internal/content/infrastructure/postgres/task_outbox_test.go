package postgres

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestOutboxPublisherAppendsEventWithGeneratedIDAndAggregateVersion(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{EventIDs: fixedIDGenerator("evt_content_1")})
	runner := NewTransactionRunner(db)
	now := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(insertOutboxEventSQL)).
		WithArgs(
			"evt_content_1",
			"content.post.published",
			1,
			"post",
			"post_pub_1",
			int64(4),
			[]byte(`{"postId":"post_pub_1"}`),
			now,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := runner.WithinTx(context.Background(), func(ctx context.Context, tx ports.Tx) error {
		return store.Append(ctx, tx, ports.OutboxEvent{
			EventType:        "content.post.published",
			PayloadVersion:   1,
			AggregateType:    "post",
			AggregateID:      "post_pub_1",
			AggregateVersion: 4,
			PayloadJSON:      []byte(`{"postId":"post_pub_1"}`),
			OccurredAt:       now,
		})
	})
	if err != nil {
		t.Fatalf("Append(outbox) error = %v", err)
	}
	assertExpectations(t, mock)
}

func TestBodyCleanupTaskAppendIsIdempotentInsideAndOutsideTransaction(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})
	cleanup := NewCleanupTaskStore(store)
	runner := NewTransactionRunner(db)
	now := time.Date(2026, 7, 5, 12, 30, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(upsertCleanupTaskSQL)).
		WithArgs(int64(10), "body_old", "OLD_DRAFT", "draft_replaced", now).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectExec(regexp.QuoteMeta(upsertCleanupTaskSQL)).
		WithArgs(nil, "body_orphan", "ORPHAN_SNAPSHOT", "publish_tx_failed", now).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := runner.WithinTx(context.Background(), func(ctx context.Context, tx ports.Tx) error {
		return cleanup.Append(ctx, tx, ports.BodyCleanupTask{
			PostID:    10,
			BodyID:    "body_old",
			TaskType:  "OLD_DRAFT",
			Reason:    "draft_replaced",
			CreatedAt: now,
		})
	})
	if err != nil {
		t.Fatalf("Append(cleanup in tx) error = %v", err)
	}
	if err := cleanup.AppendOutsideTx(context.Background(), ports.BodyCleanupTask{
		BodyID:    "body_orphan",
		TaskType:  "ORPHAN_SNAPSHOT",
		Reason:    "publish_tx_failed",
		CreatedAt: now,
	}); err != nil {
		t.Fatalf("AppendOutsideTx(cleanup) error = %v", err)
	}
	assertExpectations(t, mock)
}

func TestBodyRepairTaskAppendIsIdempotentInsideAndOutsideTransaction(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})
	repair := NewRepairTaskStore(store)
	runner := NewTransactionRunner(db)
	now := time.Date(2026, 7, 5, 12, 45, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(upsertRepairTaskSQL)).
		WithArgs(int64(10), "body_pub", "body_hash_mismatch", "sha256:expected", "sha256:observed", now).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectExec(regexp.QuoteMeta(upsertRepairTaskSQL)).
		WithArgs(int64(10), "body_pub", "mongo_read_error_after_pg_published", "sha256:expected", nil, now).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := runner.WithinTx(context.Background(), func(ctx context.Context, tx ports.Tx) error {
		return repair.Append(ctx, tx, ports.BodyRepairTask{
			PostID:       10,
			BodyID:       "body_pub",
			TaskType:     "body_hash_mismatch",
			ExpectedHash: "sha256:expected",
			ObservedHash: "sha256:observed",
			CreatedAt:    now,
		})
	})
	if err != nil {
		t.Fatalf("Append(repair in tx) error = %v", err)
	}
	if err := repair.AppendOutsideTx(context.Background(), ports.BodyRepairTask{
		PostID:       10,
		BodyID:       "body_pub",
		TaskType:     "mongo_read_error_after_pg_published",
		ExpectedHash: "sha256:expected",
		CreatedAt:    now,
	}); err != nil {
		t.Fatalf("AppendOutsideTx(repair) error = %v", err)
	}
	assertExpectations(t, mock)
}

var _ = (*sql.DB)(nil)
