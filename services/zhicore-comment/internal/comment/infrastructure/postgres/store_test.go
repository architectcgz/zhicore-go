package postgres

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/ports"
)

func TestStoreCreateUsesTransactionAndReturnsIdentityComment(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db)
	runner := NewTransactionRunner(db)
	now := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)
	draft, err := domain.NewTopLevelDraft("post_pub_1", 1001, 42, "hello", domain.CommentMediaInput{ImageFileIDs: []string{"img_1"}}, now)
	if err != nil {
		t.Fatalf("NewTopLevelDraft() error = %v", err)
	}

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(createCommentSQL)).
		WithArgs(
			"post_pub_1",
			int64(1001),
			int64(42),
			nil,
			nil,
			"hello",
			sqlmock.AnyArg(),
			nil,
			nil,
			"NORMAL",
			now,
			now,
		).
		WillReturnRows(commentRows().AddRow(
			int64(10),
			"post_pub_1",
			int64(1001),
			int64(42),
			nil,
			nil,
			"hello",
			`{img_1}`,
			nil,
			nil,
			"NORMAL",
			now,
			now,
		))
	mock.ExpectCommit()

	var created domain.Comment
	err = runner.WithinTransaction(context.Background(), func(ctx context.Context) error {
		var createErr error
		created, createErr = store.Create(ctx, draft)
		return createErr
	})
	if err != nil {
		t.Fatalf("WithinTransaction(Create) error = %v", err)
	}
	if created.ID != 10 || created.PostID != "post_pub_1" || created.Media.ImageFileIDs[0] != "img_1" {
		t.Fatalf("created comment = %#v", created)
	}
	assertExpectations(t, mock)
}

func TestStoreSoftDeleteSubtreeMarksNormalRowsAndReportsAffectedCount(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db)
	now := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(findCommentForMutationSQL)).
		WithArgs("post_pub_1", int64(10)).
		WillReturnRows(commentRows().AddRow(int64(10), "post_pub_1", int64(1001), int64(42), nil, nil, "root", `{}`, nil, nil, "NORMAL", now, now))
	mock.ExpectQuery(regexp.QuoteMeta(softDeleteSubtreeSQL)).
		WithArgs("post_pub_1", int64(10), int64(42), "AUTHOR", "user_request", now).
		WillReturnRows(sqlmock.NewRows([]string{"affected_count"}).AddRow(3))

	result, err := store.SoftDeleteSubtree(context.Background(), ports.DeleteSubtreeInput{
		PostID:        "post_pub_1",
		CommentID:     10,
		DeletedBy:     42,
		DeletedByRole: "AUTHOR",
		DeleteReason:  "user_request",
		DeletedAt:     now,
	})
	if err != nil {
		t.Fatalf("SoftDeleteSubtree() error = %v", err)
	}
	if result.AffectedCount != 3 || result.Entry.ID != 10 || result.RootID != 10 || result.AlreadyDeleted {
		t.Fatalf("delete result = %#v", result)
	}
	assertExpectations(t, mock)
}

func TestStoreListTopLevelCommentsUsesRecommendedRank(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db)
	now := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(listTopLevelRecommendedSQL)).
		WithArgs("post_pub_1", 20, 20).
		WillReturnRows(commentRecordRows().AddRow(
			int64(10),
			"post_pub_1",
			int64(1001),
			int64(42),
			nil,
			nil,
			"hello",
			`{}`,
			nil,
			nil,
			"NORMAL",
			now,
			now,
			int64(7),
			int64(2),
		))

	page, err := store.ListTopLevelComments(context.Background(), ports.TopLevelCommentPageQuery{
		PostID: "post_pub_1",
		Page:   2,
		Size:   20,
		Sort:   domain.CommentSortRecommended,
	})
	if err != nil {
		t.Fatalf("ListTopLevelComments() error = %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].Comment.ID != 10 || page.Items[0].Stats.LikeCount != 7 {
		t.Fatalf("page = %#v", page)
	}
	assertExpectations(t, mock)
}

func TestStoreGetCommentDetailReturnsCommentRecord(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db)
	now := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(getCommentDetailSQL)).
		WithArgs("post_pub_1", int64(10)).
		WillReturnRows(commentRecordRows().AddRow(
			int64(10),
			"post_pub_1",
			int64(1001),
			int64(42),
			nil,
			nil,
			"hello",
			`{}`,
			nil,
			nil,
			"NORMAL",
			now,
			now,
			int64(7),
			int64(2),
		))

	record, err := store.GetCommentDetail(context.Background(), "post_pub_1", 10)
	if err != nil {
		t.Fatalf("GetCommentDetail() error = %v", err)
	}
	if record.Comment.ID != 10 || record.Stats.LikeCount != 7 || record.Stats.ReplyCount != 2 {
		t.Fatalf("record = %#v", record)
	}
	assertExpectations(t, mock)
}

func TestStoreListRepliesByPageUsesHotRankAndReturnsTotal(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db)
	now := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)

	mock.ExpectQuery(regexp.QuoteMeta(checkRootCommentSQL)).
		WithArgs("post_pub_1", int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(regexp.QuoteMeta(countRepliesSQL)).
		WithArgs("post_pub_1", int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(int64(2)))
	mock.ExpectQuery(regexp.QuoteMeta(listRepliesHotSQL)).
		WithArgs("post_pub_1", int64(10), 20, 20).
		WillReturnRows(commentRecordRows().AddRow(
			int64(12),
			"post_pub_1",
			int64(1001),
			int64(43),
			int64(10),
			int64(10),
			"reply",
			`{}`,
			nil,
			nil,
			"NORMAL",
			now,
			now,
			int64(9),
			int64(0),
		))

	page, err := store.ListRepliesByPage(context.Background(), ports.ReplyCommentPageQuery{
		PostID: "post_pub_1",
		RootID: 10,
		Page:   2,
		Size:   20,
		Sort:   domain.CommentSortHot,
	})
	if err != nil {
		t.Fatalf("ListRepliesByPage() error = %v", err)
	}
	if page.Total != 2 || len(page.Items) != 1 || page.Items[0].Comment.ID != 12 {
		t.Fatalf("page = %#v", page)
	}
	assertExpectations(t, mock)
}

func TestStoreListRepliesByPageReturnsRootNotFoundWhenRootMissing(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db)

	mock.ExpectQuery(regexp.QuoteMeta(checkRootCommentSQL)).
		WithArgs("post_pub_1", int64(10)).
		WillReturnError(sql.ErrNoRows)

	_, err := store.ListRepliesByPage(context.Background(), ports.ReplyCommentPageQuery{
		PostID: "post_pub_1",
		RootID: 10,
		Page:   1,
		Size:   20,
		Sort:   domain.CommentSortHot,
	})
	if !errors.Is(err, domain.ErrRootCommentNotFound) {
		t.Fatalf("ListRepliesByPage() error = %v, want ErrRootCommentNotFound", err)
	}
	assertExpectations(t, mock)
}

func TestStoreBatchGetViewerLikedReturnsRequestedMap(t *testing.T) {
	db, mock := newMockDB(t)
	store := NewStore(db)

	mock.ExpectQuery(regexp.QuoteMeta(batchViewerLikedSQL)).
		WithArgs(int64(42), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"comment_id"}).AddRow(int64(10)))

	liked, err := store.BatchGetViewerLiked(context.Background(), 42, []domain.CommentID{10, 11})
	if err != nil {
		t.Fatalf("BatchGetViewerLiked() error = %v", err)
	}
	if !liked[10] || liked[11] {
		t.Fatalf("liked = %#v", liked)
	}
	assertExpectations(t, mock)
}

func TestOutboxPublisherInsertsEventWithGeneratedID(t *testing.T) {
	db, mock := newMockDB(t)
	publisher := NewOutboxPublisher(db, fixedEventIDGenerator("evt_comment_1"))
	now := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)

	mock.ExpectExec(regexp.QuoteMeta(insertOutboxEventSQL)).
		WithArgs(
			"evt_comment_1",
			"comment.created",
			1,
			"comment",
			"10",
			[]byte(`{"commentId":10}`),
			now,
		).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := publisher.Publish(context.Background(), ports.OutboxMessage{
		EventType:     "comment.created",
		AggregateType: "comment",
		AggregateID:   "10",
		OccurredAt:    now,
		Payload:       []byte(`{"commentId":10}`),
	})
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	assertExpectations(t, mock)
}

type fixedEventIDGenerator string

func (g fixedEventIDGenerator) NewEventID() (string, error) { return string(g), nil }

func commentRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id",
		"post_id",
		"content_internal_id",
		"author_id",
		"root_id",
		"parent_id",
		"content",
		"image_file_ids",
		"voice_file_id",
		"voice_duration",
		"status",
		"created_at",
		"updated_at",
	})
}

func commentRecordRows() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id",
		"post_id",
		"content_internal_id",
		"author_id",
		"root_id",
		"parent_id",
		"content",
		"image_file_ids",
		"voice_file_id",
		"voice_duration",
		"status",
		"created_at",
		"updated_at",
		"like_count",
		"reply_count",
	})
}

var _ = (*sql.DB)(nil)
