package postgres

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

func TestStoreMarkReadScopesByNotificationAndRecipientAndClampsGroupUnread(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)
	readAt := time.Date(2026, 7, 6, 17, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT public_id, group_key, is_read, read_at").
		WithArgs(int64(1001), int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"public_id", "group_key", "is_read", "read_at"}).
			AddRow("n1abc", "POST_LIKED:post:post_1", false, nil))
	mock.ExpectExec("UPDATE notifications").
		WithArgs(int64(1001), int64(42), readAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("GREATEST(unread_count - 1, 0)")).
		WithArgs(int64(42), "POST_LIKED:post:post_1", readAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	result, err := store.MarkRead(context.Background(), ports.MarkReadInput{NotificationID: 1001, RecipientID: 42, ReadAt: readAt})
	if err != nil {
		t.Fatalf("MarkRead() error = %v", err)
	}
	if result.PublicID != "n1abc" || !result.Changed || result.ReadAt != readAt {
		t.Fatalf("result = %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestStoreMarkReadIsIdempotentForAlreadyReadNotification(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)
	readAt := time.Date(2026, 7, 6, 16, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT public_id, group_key, is_read, read_at").
		WithArgs(int64(1001), int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"public_id", "group_key", "is_read", "read_at"}).
			AddRow("n1abc", "POST_LIKED:post:post_1", true, readAt))
	mock.ExpectCommit()

	result, err := store.MarkRead(context.Background(), ports.MarkReadInput{NotificationID: 1001, RecipientID: 42, ReadAt: readAt.Add(time.Hour)})
	if err != nil {
		t.Fatalf("MarkRead() error = %v", err)
	}
	if result.Changed || result.ReadAt != readAt {
		t.Fatalf("result = %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
