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
	mock.ExpectQuery("SELECT public_id, group_key, category, is_read, read_at").
		WithArgs(int64(1001), int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"public_id", "group_key", "category", "is_read", "read_at"}).
			AddRow("n1abc", "POST_LIKED:post:post_1", "INTERACTION", false, nil))
	mock.ExpectExec("UPDATE notifications").
		WithArgs(int64(1001), int64(42), readAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("GREATEST(unread_count - 1, 0)")).
		WithArgs(int64(42), "POST_LIKED:post:post_1", readAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE notification_stats").
		WithArgs(int64(42), "INTERACTION", readAt).
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
	mock.ExpectQuery("SELECT public_id, group_key, category, is_read, read_at").
		WithArgs(int64(1001), int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"public_id", "group_key", "category", "is_read", "read_at"}).
			AddRow("n1abc", "POST_LIKED:post:post_1", "INTERACTION", true, readAt))
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

func TestStoreMarkAllReadResetsNotificationStats(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)
	readAt := time.Date(2026, 7, 6, 17, 30, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE notifications").
		WithArgs(int64(42), readAt).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectExec("UPDATE notification_group_state").
		WithArgs(int64(42), readAt).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec("UPDATE notification_stats").
		WithArgs(int64(42), readAt).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	result, err := store.MarkAllRead(context.Background(), ports.MarkAllReadInput{RecipientID: 42, ReadAt: readAt})
	if err != nil {
		t.Fatalf("MarkAllRead() error = %v", err)
	}
	if result.AffectedCount != 3 || result.ReadAt != readAt {
		t.Fatalf("result = %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestStoreGetUnreadCountReadsNotificationStats(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)

	mock.ExpectQuery("SELECT unread_total").
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"unread_total"}).AddRow(int64(9)))

	count, err := store.GetUnreadCount(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetUnreadCount() error = %v", err)
	}
	if count != 9 {
		t.Fatalf("count = %d, want 9", count)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestStoreGetUnreadBreakdownReadsNotificationStats(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)

	mock.ExpectQuery("SELECT unread_total").
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{
			"unread_total",
			"unread_interaction",
			"unread_content",
			"unread_social",
			"unread_system",
			"unread_security",
		}).AddRow(int64(10), int64(4), int64(2), int64(1), int64(2), int64(1)))

	breakdown, err := store.GetUnreadBreakdown(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetUnreadBreakdown() error = %v", err)
	}
	if breakdown.Total != 10 || breakdown.Interaction != 4 || breakdown.Content != 2 || breakdown.Social != 1 || breakdown.System != 2 || breakdown.Security != 1 {
		t.Fatalf("breakdown = %#v", breakdown)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
