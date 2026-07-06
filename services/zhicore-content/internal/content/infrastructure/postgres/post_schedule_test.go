package postgres

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestStorePostScheduleMutations(t *testing.T) {
	now := time.Date(2026, 7, 6, 9, 30, 0, 0, time.UTC)
	scheduledAt := now.Add(time.Hour)

	t.Run("schedule updates post and upserts pending scheduled event", func(t *testing.T) {
		db, mock := newMockDB(t)
		store := NewStore(db, StoreConfig{})
		runner := NewTransactionRunner(db)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(schedulePostSQL)).
			WithArgs("post_1", int64(42), int64(5), "body_draft", "sha256:draft", now).
			WillReturnRows(postRows().AddRow(
				int64(10), "post_1", int64(42), "SCHEDULED", int64(6),
				"draft title", "", "", "body_draft", "sha256:draft", 100, 12,
				nil, nil, nil, nil, nil, nil, nil,
			))
		mock.ExpectExec(regexp.QuoteMeta(upsertScheduledPublishEventSQL)).
			WithArgs(int64(10), "post_1", int64(42), "body_draft", "sha256:draft", scheduledAt, now).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		var got ports.PostRecord
		err := runner.WithinTx(context.Background(), func(ctx context.Context, tx ports.Tx) error {
			var mutationErr error
			got, mutationErr = store.SchedulePost(ctx, tx, ports.SchedulePostUpdate{
				PublicID:        "post_1",
				OwnerID:         42,
				BasePostVersion: 5,
				DraftBodyID:     "body_draft",
				DraftBodyHash:   "sha256:draft",
				ScheduledAt:     scheduledAt,
				UpdatedAt:       now,
			})
			return mutationErr
		})
		if err != nil {
			t.Fatalf("WithinTx(SchedulePost) error = %v", err)
		}
		if got.Status != domain.PostStatusScheduled || got.PostVersion != 6 {
			t.Fatalf("record = %+v, want scheduled version 6", got)
		}
		assertExpectations(t, mock)
	})

	t.Run("cancel schedule updates post and marks pending schedule canceled", func(t *testing.T) {
		db, mock := newMockDB(t)
		store := NewStore(db, StoreConfig{})
		runner := NewTransactionRunner(db)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(cancelScheduleSQL)).
			WithArgs("post_1", int64(42), int64(6), now).
			WillReturnRows(postRows().AddRow(
				int64(10), "post_1", int64(42), "DRAFT", int64(7),
				"draft title", "", "", "body_draft", "sha256:draft", 100, 12,
				nil, nil, nil, nil, nil, nil, nil,
			))
		mock.ExpectExec(regexp.QuoteMeta(cancelScheduledPublishEventSQL)).
			WithArgs(int64(10), now).
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		var got ports.PostRecord
		err := runner.WithinTx(context.Background(), func(ctx context.Context, tx ports.Tx) error {
			var mutationErr error
			got, mutationErr = store.CancelSchedule(ctx, tx, ports.PostLifecycleUpdate{
				PublicID:        "post_1",
				OwnerID:         42,
				BasePostVersion: 6,
				UpdatedAt:       now,
			})
			return mutationErr
		})
		if err != nil {
			t.Fatalf("WithinTx(CancelSchedule) error = %v", err)
		}
		if got.Status != domain.PostStatusDraft || got.PostVersion != 7 {
			t.Fatalf("record = %+v, want draft version 7", got)
		}
		assertExpectations(t, mock)
	})

	t.Run("classifies schedule mutation miss", func(t *testing.T) {
		db, mock := newMockDB(t)
		store := NewStore(db, StoreConfig{})
		runner := NewTransactionRunner(db)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(schedulePostSQL)).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(regexp.QuoteMeta(classifyPostMutationMissSQL)).
			WithArgs("post_1").
			WillReturnRows(sqlmock.NewRows([]string{"owner_id", "status", "post_version", "draft_body_id", "draft_body_hash"}).
				AddRow(int64(99), "DRAFT", int64(5), "body_draft", "sha256:draft"))
		mock.ExpectRollback()

		err := runner.WithinTx(context.Background(), func(ctx context.Context, tx ports.Tx) error {
			_, err := store.SchedulePost(ctx, tx, ports.SchedulePostUpdate{
				PublicID:        "post_1",
				OwnerID:         42,
				BasePostVersion: 5,
				DraftBodyID:     "body_draft",
				DraftBodyHash:   "sha256:draft",
				ScheduledAt:     scheduledAt,
				UpdatedAt:       now,
			})
			return err
		})
		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("SchedulePost error = %v, want ErrForbidden", err)
		}
		assertExpectations(t, mock)
	})
}
