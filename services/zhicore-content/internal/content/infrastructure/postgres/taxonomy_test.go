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

func TestStoreListTags(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})

	mock.ExpectQuery(regexp.QuoteMeta(listTagsSQL)).
		WithArgs("go", int64(7), 21).
		WillReturnRows(tagRows().AddRow(int64(8), "tag_go", "Go", "go", int64(12)))

	got, err := store.ListTags(context.Background(), ports.TagListQuery{
		Cursor: ports.TagCursor{Slug: "go", ID: 7},
		Limit:  21,
	})
	if err != nil {
		t.Fatalf("ListTags() error = %v", err)
	}
	if len(got) != 1 || got[0].Slug != "go" || got[0].PostCount != 12 {
		t.Fatalf("tags = %+v", got)
	}
	assertExpectations(t, mock)
}

func TestStoreGetTagBySlugMapsMissingToTaxonomyError(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})

	mock.ExpectQuery(regexp.QuoteMeta(getTagBySlugSQL)).
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)

	_, err := store.GetTagBySlug(context.Background(), "missing")

	if !errors.Is(err, ports.ErrTaxonomyReferenceNotFound) {
		t.Fatalf("GetTagBySlug() error = %v, want ErrTaxonomyReferenceNotFound", err)
	}
	assertExpectations(t, mock)
}

func TestStoreListPublishedPostsByTag(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})
	publishedAt := time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(listPublishedPostsByTagSQL)).
		WithArgs("go", publishedAt, "post_anchor", 11).
		WillReturnRows(postSummaryRows().AddRow(
			"post_1", int64(42), "architect", "file_avatar", "Published", "summary", "file_cover",
			string(domain.PostStatusPublished), int64(3), publishedAt.Add(-time.Hour), publishedAt.Add(-2*time.Hour), publishedAt,
			int64(10), int64(2), int64(1), int64(4),
		))

	got, err := store.ListPublishedPostsByTag(context.Background(), ports.TaggedPostListQuery{
		Slug:   "go",
		Cursor: ports.PublishedPostCursor{PublishedAt: publishedAt, PublicID: "post_anchor"},
		Limit:  11,
	})
	if err != nil {
		t.Fatalf("ListPublishedPostsByTag() error = %v", err)
	}
	if len(got) != 1 || got[0].PostID != "post_1" {
		t.Fatalf("posts = %+v", got)
	}
	assertExpectations(t, mock)
}

func TestStoreListPostTagsMapsMissingPublishedPost(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})

	mock.ExpectQuery(regexp.QuoteMeta(listPostTagsSQL)).
		WithArgs("post_missing").
		WillReturnRows(tagRows())
	mock.ExpectQuery(regexp.QuoteMeta(selectPublishedPostExistsSQL)).
		WithArgs("post_missing").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	_, err := store.ListPostTags(context.Background(), "post_missing")

	if !errors.Is(err, domain.ErrPostNotFound) {
		t.Fatalf("ListPostTags() error = %v, want ErrPostNotFound", err)
	}
	assertExpectations(t, mock)
}

func TestStoreReplacePostTags(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})
	updatedAt := time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(selectPostTagIDsSQL)).
		WithArgs(int64(100)).
		WillReturnRows(sqlmock.NewRows([]string{"tag_id"}).AddRow(int64(9)))
	mock.ExpectQuery(regexp.QuoteMeta(selectTagsBySlugsSQL)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(tagRows().
			AddRow(int64(10), "tag_go", "Go", "go", int64(12)).
			AddRow(int64(11), "tag_ddd", "DDD", "ddd", int64(7)))
	mock.ExpectExec(regexp.QuoteMeta(deletePostTagsSQL)).
		WithArgs(int64(100)).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec(regexp.QuoteMeta(insertPostTagSQL)).
		WithArgs(int64(100), int64(10), 0).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(insertPostTagSQL)).
		WithArgs(int64(100), int64(11), 1).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(refreshTagStatsSQL)).
		WithArgs(sqlmock.AnyArg(), updatedAt).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectQuery(regexp.QuoteMeta(touchPostTagsSQL)).
		WithArgs(updatedAt, "post_1", int64(42), int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"post_version"}).AddRow(int64(4)))

	got, err := store.ReplacePostTags(context.Background(), nil, ports.ReplacePostTagsInput{
		PostInternalID:  100,
		PostPublicID:    "post_1",
		ActorID:         42,
		BasePostVersion: 3,
		Slugs:           []string{"go", "ddd"},
		UpdatedAt:       updatedAt,
	})
	if err != nil {
		t.Fatalf("ReplacePostTags() error = %v", err)
	}
	if got.PostVersion != 4 || len(got.Tags) != 2 || got.Tags[0].Slug != "go" || got.UpdatedAt != updatedAt {
		t.Fatalf("result = %+v", got)
	}
	assertExpectations(t, mock)
}

func tagRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "public_id", "name", "slug", "post_count"})
}
