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

func TestStoreListAdminPosts(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})
	updatedAt := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(listAdminPostsSQL)).
		WithArgs("PUBLISHED", int64(42), 20, 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"post_id",
			"author_id",
			"author_name",
			"author_avatar_file_id",
			"title",
			"summary",
			"cover_file_id",
			"status",
			"post_version",
			"published_at",
			"created_at",
			"updated_at",
			"view_count",
			"like_count",
			"favorite_count",
			"comment_count",
			"total_count",
		}).AddRow(
			"post_1", int64(42), "architect", "file_avatar", "Published title", "summary", "file_cover",
			"PUBLISHED", int64(6), updatedAt, updatedAt.Add(-time.Hour), updatedAt,
			int64(10), int64(2), int64(1), int64(3), int64(21),
		))

	page, err := store.ListAdminPosts(context.Background(), ports.AdminPostListQuery{
		Status:   "PUBLISHED",
		AuthorID: 42,
		Page:     2,
		Size:     20,
	})
	if err != nil {
		t.Fatalf("ListAdminPosts() error = %v", err)
	}
	if page.Page != 2 || page.Size != 20 || page.Total != 21 || len(page.Items) != 1 {
		t.Fatalf("page = %+v", page)
	}
	got := page.Items[0]
	if got.PostID != "post_1" || got.AuthorID != 42 || got.Status != domain.PostStatusPublished || got.ViewCount != 10 {
		t.Fatalf("item = %+v", got)
	}
	assertExpectations(t, mock)
}

func TestStoreDeleteAdminPostWritesAuditAndCancelsSchedule(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})
	runner := NewTransactionRunner(db)
	deletedAt := time.Date(2026, 7, 5, 12, 30, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(selectPostForUpdateSQL)).
		WithArgs("post_1").
		WillReturnRows(postRows().AddRow(
			int64(10), "post_1", int64(42), "PUBLISHED", int64(5),
			"draft title", "", "", "body_draft", "sha256:draft", 100, 12,
			"published title", "", "", "body_pub", "sha256:pub", 42, deletedAt.Add(-time.Hour),
		))
	mock.ExpectQuery(regexp.QuoteMeta(adminDeletePostSQL)).
		WithArgs("post_1", deletedAt).
		WillReturnRows(postRows().AddRow(
			int64(10), "post_1", int64(42), "DELETED", int64(6),
			"draft title", "", "", "body_draft", "sha256:draft", 100, 12,
			"published title", "", "", "body_pub", "sha256:pub", 42, deletedAt.Add(-time.Hour),
		))
	mock.ExpectExec(regexp.QuoteMeta(insertAdminPostAuditSQL)).
		WithArgs(int64(10), "post_1", int64(1001), "DELETE", "policy violation", "PUBLISHED", "DELETED", deletedAt).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec(regexp.QuoteMeta(cancelScheduledPublishEventSQL)).
		WithArgs(int64(10), deletedAt).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	var got ports.AdminPostDeleteRecord
	err := runner.WithinTx(context.Background(), func(ctx context.Context, tx ports.Tx) error {
		var deleteErr error
		got, deleteErr = store.DeleteAdminPost(ctx, tx, ports.AdminPostDeleteCommand{
			PublicID:    "post_1",
			AdminUserID: 1001,
			Reason:      "policy violation",
			DeletedAt:   deletedAt,
		})
		return deleteErr
	})
	if err != nil {
		t.Fatalf("WithinTx(DeleteAdminPost) error = %v", err)
	}
	if got.Before.Status != domain.PostStatusPublished || got.After.Status != domain.PostStatusDeleted || got.After.PostVersion != 6 {
		t.Fatalf("delete record = %+v", got)
	}
	assertExpectations(t, mock)
}

func TestStoreDeleteAdminPostMapsMissingAndAlreadyDeleted(t *testing.T) {
	t.Run("missing", func(t *testing.T) {
		db, mock := newMockDB(t)
		store := NewStore(db, StoreConfig{})
		runner := NewTransactionRunner(db)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(selectPostForUpdateSQL)).
			WithArgs("missing").
			WillReturnError(sql.ErrNoRows)
		mock.ExpectRollback()

		err := runner.WithinTx(context.Background(), func(ctx context.Context, tx ports.Tx) error {
			_, err := store.DeleteAdminPost(ctx, tx, ports.AdminPostDeleteCommand{PublicID: "missing"})
			return err
		})
		if !errors.Is(err, domain.ErrPostNotFound) {
			t.Fatalf("DeleteAdminPost() error = %v, want ErrPostNotFound", err)
		}
		assertExpectations(t, mock)
	})

	t.Run("already deleted", func(t *testing.T) {
		db, mock := newMockDB(t)
		store := NewStore(db, StoreConfig{})
		runner := NewTransactionRunner(db)
		deletedAt := time.Date(2026, 7, 5, 12, 30, 0, 0, time.UTC)

		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(selectPostForUpdateSQL)).
			WithArgs("post_1").
			WillReturnRows(postRows().AddRow(
				int64(10), "post_1", int64(42), "DELETED", int64(6),
				"draft title", "", "", "body_draft", "sha256:draft", 100, 12,
				"published title", "", "", "body_pub", "sha256:pub", 42, deletedAt,
			))
		mock.ExpectRollback()

		err := runner.WithinTx(context.Background(), func(ctx context.Context, tx ports.Tx) error {
			_, err := store.DeleteAdminPost(ctx, tx, ports.AdminPostDeleteCommand{PublicID: "post_1"})
			return err
		})
		if !errors.Is(err, domain.ErrPostDeleted) {
			t.Fatalf("DeleteAdminPost() error = %v, want ErrPostDeleted", err)
		}
		assertExpectations(t, mock)
	})
}
