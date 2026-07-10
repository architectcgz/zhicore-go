package postgres

import (
	"context"
	"database/sql"
	"errors"
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
	// All read commands lock group-state rows before changing notification rows.
	// Keeping this order prevents a single read and group-wide read from deadlocking.
	mock.ExpectQuery("SELECT group_key").
		WithArgs(int64(42), "POST_LIKED:post:post_1").
		WillReturnRows(sqlmock.NewRows([]string{"group_key"}).AddRow("POST_LIKED:post:post_1"))
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
	mock.ExpectQuery("SELECT group_key").
		WithArgs(int64(42), "POST_LIKED:post:post_1").
		WillReturnRows(sqlmock.NewRows([]string{"group_key"}).AddRow("POST_LIKED:post:post_1"))
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

func TestStoreMarkGroupReadScopesRecipientAndKeepsCommandIdempotent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)
	readAt := time.Date(2026, 7, 10, 6, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT group_key, category, unread_count").WithArgs(int64(42), "ng1abc").WillReturnRows(sqlmock.NewRows([]string{"group_key", "category", "unread_count"}).AddRow("post_liked:1", "INTERACTION", 2))
	mock.ExpectExec("UPDATE notifications").WithArgs(int64(42), "ng1abc", readAt).WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec("UPDATE notification_group_state").WithArgs(int64(42), "post_liked:1", readAt).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE notification_stats").WithArgs(int64(42), "INTERACTION", int64(2), readAt).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	result, err := store.MarkGroupRead(context.Background(), ports.MarkGroupReadInput{RecipientID: 42, GroupID: "ng1abc", ReadAt: readAt})
	if err != nil {
		t.Fatalf("MarkGroupRead() error = %v", err)
	}
	if result.ChangedCount != 2 || result.UnreadCount != 0 || result.GroupID != "ng1abc" {
		t.Fatalf("result = %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestStoreListGroupActorsReturnsNotFoundWhenGroupIsNotOwned(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)
	mock.ExpectQuery("SELECT 1").WithArgs(int64(42), "ng1other").WillReturnError(sql.ErrNoRows)

	_, err = store.ListGroupActors(context.Background(), ports.ListGroupActorsQuery{RecipientID: 42, GroupID: "ng1other", Size: 20})
	if !errors.Is(err, ports.ErrNotificationNotFound) {
		t.Fatalf("ListGroupActors() error = %v, want not found", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestStoreListGroupActorsRejectsMalformedCursorBeforeOwnershipLookup(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = NewStore(db).ListGroupActors(context.Background(), ports.ListGroupActorsQuery{RecipientID: 42, GroupID: "ng1abc", Cursor: "not-a-cursor", Size: 20})
	if !errors.Is(err, ports.ErrInvalidQuery) {
		t.Fatalf("ListGroupActors() error = %v, want invalid query", err)
	}
}

func TestStoreListGroupActorsUsesStableCursorAndReturnsDistinctActorPage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)
	cursorTime := time.Date(2026, 7, 10, 5, 30, 0, 0, time.UTC)
	cursor := encodeNotificationActorCursor(cursorTime, "user_4")
	latest := cursorTime.Add(-time.Minute)

	mock.ExpectQuery("SELECT 1").WithArgs(int64(42), "ng1abc").WillReturnRows(sqlmock.NewRows([]string{"?column?"}).AddRow(1))
	mock.ExpectQuery("WITH actor_events").
		WithArgs(int64(42), "ng1abc", cursorTime.Format(time.RFC3339Nano), "user_4", 3).
		WillReturnRows(sqlmock.NewRows([]string{"actor_public_id", "actor_display_name", "actor_avatar_url", "event_count", "latest_occurred_at"}).
			AddRow("user_3", "阿宋", nil, 2, latest).
			AddRow("user_2", "小郑", "https://cdn.example/u2.png", 1, latest.Add(-time.Minute)).
			AddRow("user_1", "陈立", nil, 1, latest.Add(-2*time.Minute)))

	page, err := store.ListGroupActors(context.Background(), ports.ListGroupActorsQuery{RecipientID: 42, GroupID: "ng1abc", Cursor: cursor, Size: 2})
	if err != nil {
		t.Fatalf("ListGroupActors() error = %v", err)
	}
	if !page.HasMore || len(page.Items) != 2 || page.Items[0].PublicID != "user_3" || page.Items[0].EventCount != 2 || page.NextCursor == "" {
		t.Fatalf("page = %#v", page)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestStoreListAggregatedReturnsPublicActorSnapshotsAndDistinctCount(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)
	latest := time.Date(2026, 7, 10, 5, 30, 0, 0, time.UTC)
	recentActors := []byte(`[{"publicId":"user_3","displayName":"阿宋","avatarUrl":null},{"publicId":"user_2","displayName":"小郑","avatarUrl":"https://cdn.example/u2.png"},{"publicId":"user_1","displayName":"陈立","avatarUrl":null}]`)

	mock.ExpectQuery("SELECT state.group_id").
		WithArgs(int64(42), "", false, "", "", 2).
		WillReturnRows(sqlmock.NewRows([]string{
			"group_id", "notification_type", "category", "target_type", "target_id", "total_count", "unread_count", "latest_time", "latest_content", "aggregated_content", "actor_total_count", "recent_actors",
		}).AddRow("ng1abc", "POST_LIKED", "INTERACTION", "POST", "41", 5, 2, latest, "陈立等人赞了你的文章", []byte(`{"publicId":"post_1"}`), 4, recentActors))

	page, err := store.ListAggregated(context.Background(), ports.ListAggregatedQuery{RecipientID: 42, Size: 1})
	if err != nil {
		t.Fatalf("ListAggregated() error = %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].ActorTotalCount != 4 || len(page.Items[0].RecentActors) != 3 || page.Items[0].RecentActors[0].PublicID != "user_3" || page.Items[0].RecentActors[1].AvatarURL == nil {
		t.Fatalf("page = %#v", page)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
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
	mock.ExpectQuery("SELECT group_key").
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"group_key"}).AddRow("POST_LIKED:post:post_1").AddRow("USER_FOLLOWED:user:user_1"))
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
