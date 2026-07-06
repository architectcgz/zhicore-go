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

func TestStoreListAuthorPosts(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})
	updatedAt := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(listAuthorPostsSQL)).
		WithArgs(int64(1001), "DRAFT", updatedAt, "post_anchor", 21).
		WillReturnRows(postSummaryRows().AddRow(
			"post_1", int64(1001), "architect", "file_avatar", "Draft", "summary", "file_cover",
			"DRAFT", int64(3), nil, updatedAt.Add(-time.Hour), updatedAt,
			int64(10), int64(2), int64(1), int64(4),
		))

	got, err := store.ListAuthorPosts(context.Background(), ports.AuthorPostListQuery{
		OwnerID: 1001,
		Status:  string(domain.PostStatusDraft),
		Cursor: ports.AuthorPostCursor{
			UpdatedAt: updatedAt,
			PublicID:  "post_anchor",
		},
		Limit: 21,
	})
	if err != nil {
		t.Fatalf("ListAuthorPosts() error = %v", err)
	}
	if len(got) != 1 || got[0].PostID != "post_1" || got[0].Status != domain.PostStatusDraft {
		t.Fatalf("author posts = %+v", got)
	}
	assertExpectations(t, mock)
}

func TestStoreGetDraftPost(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})
	updatedAt := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(getDraftPostSQL)).
		WithArgs("post_1").
		WillReturnRows(sqlmock.NewRows(append(postRecordColumnNames(), "created_at", "updated_at")).
			AddRow(
				int64(10), "post_1", int64(1001), "DRAFT", int64(5),
				"Draft", "summary", "file_cover", "body_1", "sha256:body", int64(36), int64(11),
				nil, nil, nil, nil, nil, nil, nil,
				updatedAt.Add(-time.Hour), updatedAt,
			))

	got, err := store.GetDraftPost(context.Background(), "post_1")
	if err != nil {
		t.Fatalf("GetDraftPost() error = %v", err)
	}
	if got.Post.ID != 10 || got.Post.DraftBodyID != "body_1" || got.UpdatedAt != updatedAt {
		t.Fatalf("draft = %+v", got)
	}
	assertExpectations(t, mock)
}

func TestStoreUpdateDraftMeta(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})
	title := "Next"
	summary := "summary"
	updatedAt := time.Date(2026, 7, 5, 12, 30, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(updateDraftMetaSQL)).
		WithArgs("Next", "summary", "file_cover", updatedAt, "post_1", int64(1001), int64(5)).
		WillReturnRows(postRecordRows().AddRow(
			int64(10), "post_1", int64(1001), "DRAFT", int64(6),
			"Next", "summary", "file_cover", nil, nil, nil, nil,
			nil, nil, nil, nil, nil, nil, nil,
		))

	got, err := store.UpdateDraftMeta(context.Background(), nil, ports.UpdateDraftMetaUpdate{
		PublicID:        "post_1",
		OwnerID:         1001,
		BasePostVersion: 5,
		Title:           &title,
		Summary:         &summary,
		CoverFileID:     ports.OptionalStringUpdate{Set: true, Value: "file_cover"},
		UpdatedAt:       updatedAt,
	})
	if err != nil {
		t.Fatalf("UpdateDraftMeta() error = %v", err)
	}
	if got.PostVersion != 6 || got.DraftTitle != "Next" {
		t.Fatalf("updated = %+v", got)
	}
	assertExpectations(t, mock)
}

func TestStoreDeleteDraft(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})
	updatedAt := time.Date(2026, 7, 5, 12, 30, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(deleteDraftSQL)).
		WithArgs("post_1", int64(1001), updatedAt).
		WillReturnRows(postRecordRows().AddRow(
			int64(10), "post_1", int64(1001), "DRAFT", int64(6),
			nil, nil, nil, nil, nil, nil, nil,
			nil, nil, nil, nil, nil, nil, nil,
		))

	got, err := store.DeleteDraft(context.Background(), nil, ports.DeleteDraftUpdate{PublicID: "post_1", OwnerID: 1001, UpdatedAt: updatedAt})
	if err != nil {
		t.Fatalf("DeleteDraft() error = %v", err)
	}
	if got.PostVersion != 6 || got.DraftBodyID != "" {
		t.Fatalf("deleted draft = %+v", got)
	}
	assertExpectations(t, mock)
}

func TestStoreUpdateDraftMetaClassifiesMutationMiss(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})
	updatedAt := time.Date(2026, 7, 5, 12, 30, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(updateDraftMetaSQL)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(regexp.QuoteMeta(classifyPostMutationMissSQL)).
		WithArgs("post_1").
		WillReturnRows(sqlmock.NewRows([]string{"owner_id", "status", "post_version", "draft_body_id", "draft_body_hash"}).
			AddRow(int64(1001), "DRAFT", int64(6), "body_old", "sha256:old"))

	_, err := store.UpdateDraftMeta(context.Background(), nil, ports.UpdateDraftMetaUpdate{
		PublicID:        "post_1",
		OwnerID:         1001,
		BasePostVersion: 5,
		UpdatedAt:       updatedAt,
	})
	if !errors.Is(err, domain.ErrDraftConflict) {
		t.Fatalf("UpdateDraftMeta() error = %v, want ErrDraftConflict", err)
	}
	assertExpectations(t, mock)
}

func TestStoreDeleteDraftClassifiesDeletedMutationMiss(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})
	updatedAt := time.Date(2026, 7, 5, 12, 30, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(deleteDraftSQL)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(regexp.QuoteMeta(classifyPostMutationMissSQL)).
		WithArgs("post_1").
		WillReturnRows(sqlmock.NewRows([]string{"owner_id", "status", "post_version", "draft_body_id", "draft_body_hash"}).
			AddRow(int64(1001), "DELETED", int64(6), nil, nil))

	_, err := store.DeleteDraft(context.Background(), nil, ports.DeleteDraftUpdate{PublicID: "post_1", OwnerID: 1001, UpdatedAt: updatedAt})
	if !errors.Is(err, domain.ErrPostDeleted) {
		t.Fatalf("DeleteDraft() error = %v, want ErrPostDeleted", err)
	}
	assertExpectations(t, mock)
}

func TestDraftMutationSQLGuardsScheduledStatus(t *testing.T) {
	for name, sqlText := range map[string]string{
		"update draft body": updateDraftBodySQL,
		"update draft meta": updateDraftMetaSQL,
		"delete draft":      deleteDraftSQL,
	} {
		if !strings.Contains(sqlText, "'SCHEDULED'") {
			t.Fatalf("%s SQL must reject scheduled posts so queued publish content cannot drift; got:\n%s", name, sqlText)
		}
	}
}

func postRecordRows() *sqlmock.Rows {
	return sqlmock.NewRows(postRecordColumnNames())
}

func postRecordColumnNames() []string {
	return []string{
		"id",
		"public_id",
		"owner_id",
		"status",
		"post_version",
		"draft_title",
		"draft_summary",
		"draft_cover_file_id",
		"draft_body_id",
		"draft_body_hash",
		"draft_size_bytes",
		"draft_plain_text_length",
		"published_title",
		"published_summary",
		"published_cover_file_id",
		"published_body_id",
		"published_body_hash",
		"published_plain_text_length",
		"published_at",
	}
}
