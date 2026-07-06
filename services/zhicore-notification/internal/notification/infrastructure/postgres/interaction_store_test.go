package postgres

import (
	"context"
	"database/sql"
	"strconv"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

func TestStoreCreateInteractionNotificationPersistsEventAndInboxInOneTransaction(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewStoreWithCodec(db, fakePublicIDCodec{})
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	actorID := int64(2002)

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO consumed_events").
		WithArgs("evt_like_1", "content.post.liked", "content.post.liked", "zhicore-notification:content-post-consumer", "hash_1", now.Add(168*time.Hour)).
		WillReturnRows(sqlmock.NewRows([]string{"event_id"}).AddRow("evt_like_1"))
	mock.ExpectQuery("nextval").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(10001)))
	mock.ExpectQuery("INSERT INTO notifications").
		WithArgs(
			int64(10001),
			"ntf_10001",
			int64(1001),
			sql.NullInt64{Int64: actorID, Valid: true},
			"INTERACTION",
			"POST_LIKED",
			"content.post.liked",
			"NORMAL",
			"POST",
			"41",
			"evt_like_1",
			"post_liked:41:2002",
			"post_liked:41",
			"New like",
			"liked your post",
			[]byte(`{"internalId":41}`),
			now,
			now,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(10001)))
	mock.ExpectExec("INSERT INTO notification_group_state").
		WithArgs(int64(1001), "post_liked:41", "POST_LIKED", "INTERACTION", "POST", "41", int64(10001), now, "liked your post", sql.NullInt64{Int64: actorID, Valid: true}, []byte(`{"internalId":41}`), now).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO notification_stats").
		WithArgs(int64(1001), "INTERACTION", now).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE consumed_events").
		WithArgs("evt_like_1", now).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	result, err := store.CreateInteractionNotification(context.Background(), ports.CreateInteractionNotificationInput{
		Event: ports.ConsumedEventMetadata{
			EventID:      "evt_like_1",
			EventType:    "content.post.liked",
			RoutingKey:   "content.post.liked",
			ConsumerName: "zhicore-notification:content-post-consumer",
			PayloadHash:  "hash_1",
			ExpiresAt:    now.Add(168 * time.Hour),
		},
		RecipientID:      1001,
		ActorID:          &actorID,
		Category:         "INTERACTION",
		NotificationType: "POST_LIKED",
		EventCode:        "content.post.liked",
		Importance:       "NORMAL",
		TargetType:       "POST",
		TargetID:         "41",
		SourceEventID:    "evt_like_1",
		DedupeKey:        "post_liked:41:2002",
		GroupKey:         "post_liked:41",
		Title:            "New like",
		Content:          "liked your post",
		Payload:          []byte(`{"internalId":41}`),
		OccurredAt:       now,
		CreatedAt:        now,
	})
	if err != nil {
		t.Fatalf("CreateInteractionNotification() error = %v", err)
	}
	if !result.Created || result.NotificationID != 10001 || result.PublicID != "ntf_10001" {
		t.Fatalf("result = %+v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestStoreCreateInteractionNotificationReturnsDuplicateWhenEventAlreadyConsumed(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewStoreWithCodec(db, fakePublicIDCodec{})
	now := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO consumed_events").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectCommit()

	_, err = store.CreateInteractionNotification(context.Background(), ports.CreateInteractionNotificationInput{
		Event: ports.ConsumedEventMetadata{
			EventID:      "evt_like_1",
			EventType:    "content.post.liked",
			RoutingKey:   "content.post.liked",
			ConsumerName: "zhicore-notification:content-post-consumer",
			PayloadHash:  "hash_1",
			ExpiresAt:    now.Add(168 * time.Hour),
		},
		RecipientID:      1001,
		NotificationType: "POST_LIKED",
		Category:         "INTERACTION",
		TargetType:       "POST",
		TargetID:         "41",
		DedupeKey:        "post_liked:41:2002",
		GroupKey:         "post_liked:41",
		Title:            "New like",
		Content:          "liked your post",
		OccurredAt:       now,
		CreatedAt:        now,
	})
	if err != ports.ErrDuplicateConsumedEvent {
		t.Fatalf("error = %v, want duplicate consumed event", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

type fakePublicIDCodec struct{}

func (fakePublicIDCodec) Encode(id uint64) (string, error) {
	return "ntf_" + strconv.FormatUint(id, 10), nil
}

func (fakePublicIDCodec) Decode(publicID string) (uint64, error) {
	return 0, nil
}
