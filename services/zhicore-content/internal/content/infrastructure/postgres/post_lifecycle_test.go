package postgres

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestStorePostLifecycleMutations(t *testing.T) {
	updatedAt := time.Date(2026, 7, 6, 9, 30, 0, 0, time.UTC)

	t.Run("unpublish uses owner version and published status guard", func(t *testing.T) {
		db, mock := newMockDB(t)
		store := NewStore(db, StoreConfig{})
		runner := NewTransactionRunner(db)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(unpublishPostSQL)).
			WithArgs("post_pub_1", int64(42), int64(3), updatedAt).
			WillReturnRows(postRows().AddRow(
				int64(10), "post_pub_1", int64(42), "DRAFT", int64(4),
				"draft title", "", "", "body_draft", "sha256:draft", 100, 12,
				"published title", "", "", "body_pub", "sha256:pub", 42, updatedAt,
			))
		mock.ExpectCommit()

		var got ports.PostRecord
		err := runner.WithinTx(context.Background(), func(ctx context.Context, tx ports.Tx) error {
			var mutationErr error
			got, mutationErr = store.Unpublish(ctx, tx, ports.PostLifecycleUpdate{
				PublicID:        "post_pub_1",
				OwnerID:         42,
				BasePostVersion: 3,
				UpdatedAt:       updatedAt,
			})
			return mutationErr
		})
		if err != nil {
			t.Fatalf("WithinTx(Unpublish) error = %v", err)
		}
		if got.Status != domain.PostStatusDraft || got.PostVersion != 4 {
			t.Fatalf("record = %+v, want draft version 4", got)
		}
		assertExpectations(t, mock)
	})

	t.Run("delete soft deletes without clearing body pointers", func(t *testing.T) {
		db, mock := newMockDB(t)
		store := NewStore(db, StoreConfig{})
		runner := NewTransactionRunner(db)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(deletePostSQL)).
			WithArgs("post_pub_1", int64(42), int64(3), updatedAt).
			WillReturnRows(postRows().AddRow(
				int64(10), "post_pub_1", int64(42), "DELETED", int64(4),
				"draft title", "", "", "body_draft", "sha256:draft", 100, 12,
				"published title", "", "", "body_pub", "sha256:pub", 42, updatedAt,
			))
		mock.ExpectExec(regexp.QuoteMeta(cancelScheduledPublishEventSQL)).
			WithArgs(int64(10), updatedAt).
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		var got ports.PostRecord
		err := runner.WithinTx(context.Background(), func(ctx context.Context, tx ports.Tx) error {
			var mutationErr error
			got, mutationErr = store.DeletePost(ctx, tx, ports.PostLifecycleUpdate{
				PublicID:        "post_pub_1",
				OwnerID:         42,
				BasePostVersion: 3,
				UpdatedAt:       updatedAt,
			})
			return mutationErr
		})
		if err != nil {
			t.Fatalf("WithinTx(DeletePost) error = %v", err)
		}
		if got.Status != domain.PostStatusDeleted || got.PublishedBodyID != "body_pub" || got.DraftBodyID != "body_draft" {
			t.Fatalf("record = %+v, want deleted with body pointers preserved", got)
		}
		assertExpectations(t, mock)
	})

	t.Run("restore only deleted posts and returns draft", func(t *testing.T) {
		db, mock := newMockDB(t)
		store := NewStore(db, StoreConfig{})
		runner := NewTransactionRunner(db)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(restorePostSQL)).
			WithArgs("post_pub_1", int64(42), int64(3), updatedAt).
			WillReturnRows(postRows().AddRow(
				int64(10), "post_pub_1", int64(42), "DRAFT", int64(4),
				"draft title", "", "", "body_draft", "sha256:draft", 100, 12,
				"published title", "", "", "body_pub", "sha256:pub", 42, updatedAt,
			))
		mock.ExpectCommit()

		var got ports.PostRecord
		err := runner.WithinTx(context.Background(), func(ctx context.Context, tx ports.Tx) error {
			var mutationErr error
			got, mutationErr = store.RestorePost(ctx, tx, ports.PostLifecycleUpdate{
				PublicID:        "post_pub_1",
				OwnerID:         42,
				BasePostVersion: 3,
				UpdatedAt:       updatedAt,
			})
			return mutationErr
		})
		if err != nil {
			t.Fatalf("WithinTx(RestorePost) error = %v", err)
		}
		if got.Status != domain.PostStatusDraft || got.PostVersion != 4 {
			t.Fatalf("record = %+v, want draft version 4", got)
		}
		assertExpectations(t, mock)
	})

	t.Run("classifies lifecycle mutation miss", func(t *testing.T) {
		db, mock := newMockDB(t)
		store := NewStore(db, StoreConfig{})
		runner := NewTransactionRunner(db)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(unpublishPostSQL)).
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(regexp.QuoteMeta(classifyPostMutationMissSQL)).
			WithArgs("post_pub_1").
			WillReturnRows(sqlmock.NewRows([]string{"owner_id", "status", "post_version", "draft_body_id", "draft_body_hash"}).
				AddRow(int64(99), "PUBLISHED", int64(3), "body_draft", "sha256:draft"))
		mock.ExpectRollback()

		err := runner.WithinTx(context.Background(), func(ctx context.Context, tx ports.Tx) error {
			_, err := store.Unpublish(ctx, tx, ports.PostLifecycleUpdate{
				PublicID:        "post_pub_1",
				OwnerID:         42,
				BasePostVersion: 3,
				UpdatedAt:       updatedAt,
			})
			return err
		})
		if !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("Unpublish error = %v, want ErrForbidden", err)
		}
		assertExpectations(t, mock)
	})
}

func TestPostLifecycleSQLMaintainsDeletedAt(t *testing.T) {
	if !strings.Contains(deletePostSQL, "deleted_at = $4") {
		t.Fatalf("delete SQL must set deleted_at from lifecycle timestamp; got:\n%s", deletePostSQL)
	}
	if !strings.Contains(restorePostSQL, "deleted_at = NULL") {
		t.Fatalf("restore SQL must clear deleted_at; got:\n%s", restorePostSQL)
	}
}
