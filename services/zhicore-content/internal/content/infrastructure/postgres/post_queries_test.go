package postgres

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestStoreListPublishedPosts(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})
	publishedAt := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(listPublishedPostsSQL)).
		WithArgs(int64(42), publishedAt, "post_anchor", 21).
		WillReturnRows(postSummaryRows().AddRow(
			"post_1", int64(42), "architect", "file_avatar", "Published", "summary", "file_cover",
			"PUBLISHED", int64(3), publishedAt, publishedAt.Add(-time.Hour), publishedAt,
			int64(10), int64(2), int64(1), int64(4),
		))

	got, err := store.ListPublishedPosts(context.Background(), ports.PostListQuery{
		AuthorID: 42,
		Cursor: ports.PublishedPostCursor{
			PublishedAt: publishedAt,
			PublicID:    "post_anchor",
		},
		Limit: 21,
	})
	if err != nil {
		t.Fatalf("ListPublishedPosts() error = %v", err)
	}
	if len(got) != 1 || got[0].PostID != "post_1" || got[0].LikeCount != 2 {
		t.Fatalf("summaries = %+v", got)
	}
	assertExpectations(t, mock)
}

func TestStoreGetPublishedPostDetail(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})
	publishedAt := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(getPublishedPostDetailSQL)).
		WithArgs("post_1").
		WillReturnRows(postDetailRows().AddRow(
			int64(10), "post_1", int64(42), "architect", "file_avatar", "Published", "summary", "file_cover",
			"PUBLISHED", int64(3), publishedAt, publishedAt.Add(-time.Hour), publishedAt,
			int64(10), int64(2), int64(1), int64(4), "body_1", "sha256:body",
		))

	got, err := store.GetPublishedPostDetail(context.Background(), "post_1")
	if err != nil {
		t.Fatalf("GetPublishedPostDetail() error = %v", err)
	}
	if got.InternalPostID != 10 || got.Summary.PostID != "post_1" || got.PublishedBodyID != "body_1" || got.PublishedHash != "sha256:body" {
		t.Fatalf("detail = %+v", got)
	}
	assertExpectations(t, mock)
}

func TestStoreBatchGetPublishedPostSummaries(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})
	publishedAt := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(batchGetPublishedPostSummariesSQL)).
		WithArgs(pq.Array([]string{"post_2", "post_1"})).
		WillReturnRows(postSummaryRows().AddRow(
			"post_1", int64(42), "architect", "file_avatar", "Published", "summary", "file_cover",
			"PUBLISHED", int64(3), publishedAt, publishedAt.Add(-time.Hour), publishedAt,
			int64(10), int64(2), int64(1), int64(4),
		))

	got, err := store.BatchGetPublishedPostSummaries(context.Background(), []string{"post_2", "post_1"})
	if err != nil {
		t.Fatalf("BatchGetPublishedPostSummaries() error = %v", err)
	}
	if len(got) != 1 || got[0].PostID != "post_1" {
		t.Fatalf("summaries = %+v", got)
	}
	assertExpectations(t, mock)
}

func postSummaryRows() *sqlmock.Rows {
	return sqlmock.NewRows(postSummaryColumnNames())
}

func postDetailRows() *sqlmock.Rows {
	return sqlmock.NewRows(append([]string{"id"}, append(postSummaryColumnNames(), "published_body_id", "published_body_hash")...))
}

func postSummaryColumnNames() []string {
	return []string{
		"public_id",
		"owner_id",
		"owner_display_name",
		"owner_avatar_file_id",
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
	}
}
