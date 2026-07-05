package postgres

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestStoreCreateDraftRetriesPublicIDCollisionAndInitializesStats(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{
		PublicIDs: &sequenceIDGenerator{"post_collision", "post_ok"},
		EventIDs:  fixedIDGenerator("evt_unused"),
	})
	runner := NewTransactionRunner(db)

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO posts").
		WithArgs(
			"post_collision",
			int64(42),
			"architect",
			"file_avatar",
			int64(7),
			"draft",
			"summary",
			"cover_1",
			"body_1",
			"sha256:body",
			123,
			11,
		).
		WillReturnError(&pq.Error{Code: "23505", Constraint: "ux_posts_public_id"})
	mock.ExpectQuery("INSERT INTO posts").
		WithArgs(
			"post_ok",
			int64(42),
			"architect",
			"file_avatar",
			int64(7),
			"draft",
			"summary",
			"cover_1",
			"body_1",
			"sha256:body",
			123,
			11,
		).
		WillReturnRows(postRows().AddRow(
			int64(10),
			"post_ok",
			int64(42),
			"DRAFT",
			int64(1),
			"draft",
			"summary",
			"cover_1",
			"body_1",
			"sha256:body",
			123,
			11,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
		))
	mock.ExpectExec("INSERT INTO post_stats").
		WithArgs(int64(10), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	var created ports.PostRecord
	err := runner.WithinTx(context.Background(), func(ctx context.Context, tx ports.Tx) error {
		var createErr error
		created, createErr = store.CreateDraft(ctx, tx, ports.CreateDraftPost{
			OwnerID:              42,
			OwnerDisplayName:     "architect",
			OwnerAvatarFileID:    "file_avatar",
			OwnerProfileVersion:  7,
			Title:                "draft",
			Summary:              "summary",
			CoverFileID:          "cover_1",
			DraftBodyID:          "body_1",
			DraftBodyHash:        "sha256:body",
			DraftSizeBytes:       123,
			DraftPlainTextLength: 11,
		})
		return createErr
	})
	if err != nil {
		t.Fatalf("WithinTx(CreateDraft) error = %v", err)
	}
	if created.ID != 10 || created.PublicID != "post_ok" || created.PostVersion != 1 {
		t.Fatalf("created = %+v, want post_ok id 10 version 1", created)
	}
	assertExpectations(t, mock)
}

func TestStoreGetForUpdateMapsMissingPost(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})
	runner := NewTransactionRunner(db)

	mock.ExpectBegin()
	mock.ExpectQuery("FOR UPDATE").
		WithArgs("missing_post").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectRollback()

	err := runner.WithinTx(context.Background(), func(ctx context.Context, tx ports.Tx) error {
		_, err := store.GetForUpdate(ctx, tx, "missing_post")
		return err
	})
	if !errors.Is(err, domain.ErrPostNotFound) {
		t.Fatalf("GetForUpdate error = %v, want ErrPostNotFound", err)
	}
	assertExpectations(t, mock)
}

func TestStoreSaveDraftBodyUsesOwnerAndDraftCAS(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})
	runner := NewTransactionRunner(db)

	mock.ExpectBegin()
	mock.ExpectQuery("UPDATE posts").
		WithArgs(
			"body_new",
			"sha256:new",
			200,
			20,
			"post_pub_1",
			int64(42),
			int64(3),
			"body_old",
			"sha256:old",
		).
		WillReturnRows(postRows().AddRow(
			int64(10),
			"post_pub_1",
			int64(42),
			"DRAFT",
			int64(4),
			"draft title",
			"",
			"",
			"body_new",
			"sha256:new",
			200,
			20,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
		))
	mock.ExpectCommit()

	var saved ports.PostRecord
	err := runner.WithinTx(context.Background(), func(ctx context.Context, tx ports.Tx) error {
		var saveErr error
		saved, saveErr = store.SaveDraftBody(ctx, tx, ports.SaveDraftBodyUpdate{
			PublicID:             "post_pub_1",
			OwnerID:              42,
			BasePostVersion:      3,
			BaseDraftBodyID:      "body_old",
			BaseDraftBodyHash:    "sha256:old",
			NewDraftBodyID:       "body_new",
			NewDraftBodyHash:     "sha256:new",
			NewDraftSizeBytes:    200,
			NewDraftPlainTextLen: 20,
		})
		return saveErr
	})
	if err != nil {
		t.Fatalf("WithinTx(SaveDraftBody) error = %v", err)
	}
	if saved.PostVersion != 4 || saved.DraftBodyID != "body_new" {
		t.Fatalf("saved = %+v, want new draft pointer version 4", saved)
	}
	assertExpectations(t, mock)
}

func TestStoreSaveDraftBodyClassifiesOwnerMissAsForbidden(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})
	runner := NewTransactionRunner(db)

	mock.ExpectBegin()
	mock.ExpectQuery("UPDATE posts").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery(regexp.QuoteMeta(classifyPostMutationMissSQL)).
		WithArgs("post_pub_1").
		WillReturnRows(sqlmock.NewRows([]string{"owner_id", "status", "post_version", "draft_body_id", "draft_body_hash"}).
			AddRow(int64(99), "DRAFT", int64(3), "body_old", "sha256:old"))
	mock.ExpectRollback()

	err := runner.WithinTx(context.Background(), func(ctx context.Context, tx ports.Tx) error {
		_, err := store.SaveDraftBody(ctx, tx, ports.SaveDraftBodyUpdate{
			PublicID:          "post_pub_1",
			OwnerID:           42,
			BasePostVersion:   3,
			BaseDraftBodyID:   "body_old",
			BaseDraftBodyHash: "sha256:old",
		})
		return err
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("SaveDraftBody error = %v, want ErrForbidden", err)
	}
	assertExpectations(t, mock)
}

func TestStorePublishWritesPublishedPointerAndClearsDraftByCAS(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})
	runner := NewTransactionRunner(db)
	publishedAt := time.Date(2026, 7, 5, 11, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery("UPDATE posts").
		WithArgs(
			"snapshot_1",
			"sha256:published",
			42,
			publishedAt,
			"post_pub_1",
			int64(42),
			int64(3),
			"draft_1",
			"sha256:draft",
		).
		WillReturnRows(postRows().AddRow(
			int64(10),
			"post_pub_1",
			int64(42),
			"PUBLISHED",
			int64(4),
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			"draft title",
			"summary",
			"cover_1",
			"snapshot_1",
			"sha256:published",
			42,
			publishedAt,
		))
	mock.ExpectCommit()

	var published ports.PostRecord
	err := runner.WithinTx(context.Background(), func(ctx context.Context, tx ports.Tx) error {
		var publishErr error
		published, publishErr = store.Publish(ctx, tx, ports.PublishPostUpdate{
			PublicID:                 "post_pub_1",
			OwnerID:                  42,
			BasePostVersion:          3,
			ExpectedDraftBodyID:      "draft_1",
			ExpectedDraftBodyHash:    "sha256:draft",
			NewPublishedBodyID:       "snapshot_1",
			NewPublishedBodyHash:     "sha256:published",
			NewPublishedPlainTextLen: 42,
			PublishedAt:              publishedAt,
		})
		return publishErr
	})
	if err != nil {
		t.Fatalf("WithinTx(Publish) error = %v", err)
	}
	if published.Status != domain.PostStatusPublished || published.PublishedBodyID != "snapshot_1" || published.DraftBodyID != "" {
		t.Fatalf("published = %+v, want published pointer and cleared draft", published)
	}
	assertExpectations(t, mock)
}

func TestStoreGetPublishedBodyPointerReturnsPointer(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})

	mock.ExpectQuery("SELECT").
		WithArgs("post_pub_1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"public_id",
			"status",
			"published_body_id",
			"published_body_hash",
			"published_plain_text_length",
		}).AddRow(int64(10), "post_pub_1", "PUBLISHED", "body_pub", "sha256:pub", 42))

	pointer, err := store.GetPublishedBodyPointer(context.Background(), "post_pub_1")
	if err != nil {
		t.Fatalf("GetPublishedBodyPointer() error = %v", err)
	}
	if pointer.PostID != 10 || pointer.PublishedBodyID != "body_pub" || pointer.PublishedPlainTextLen != 42 {
		t.Fatalf("pointer = %+v, want published body pointer", pointer)
	}
	assertExpectations(t, mock)
}

func TestStoreIsBodyReferencedChecksPublishedAndDraftPointers(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db, StoreConfig{})

	mock.ExpectQuery(regexp.QuoteMeta(selectBodyReferencedSQL)).
		WithArgs("body_live").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	referenced, err := store.IsBodyReferenced(context.Background(), "body_live")
	if err != nil {
		t.Fatalf("IsBodyReferenced() error = %v", err)
	}
	if !referenced {
		t.Fatalf("referenced = false, want true")
	}
	assertExpectations(t, mock)
}

func postRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
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
	})
}

type sequenceIDGenerator []string

func (g *sequenceIDGenerator) NewID() (string, error) {
	if len(*g) == 0 {
		return "", errors.New("sequence exhausted")
	}
	id := (*g)[0]
	*g = (*g)[1:]
	return id, nil
}

type fixedIDGenerator string

func (g fixedIDGenerator) NewID() (string, error) { return string(g), nil }

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
		t.Fatalf("sql expectations were not met: %v", err)
	}
}
