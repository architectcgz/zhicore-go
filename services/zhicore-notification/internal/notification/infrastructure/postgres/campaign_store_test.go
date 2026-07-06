package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

func TestStorePlanPostPublishedCampaignPersistsCampaignAndInitialShard(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)
	now := time.Date(2026, 7, 6, 19, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO consumed_events").
		WithArgs("evt_post_published_1", "content.post.published", "content.post.published", "zhicore-notification:content-post-consumer", "hash_1", now.Add(168*time.Hour)).
		WillReturnRows(sqlmock.NewRows([]string{"event_id"}).AddRow("evt_post_published_1"))
	mock.ExpectQuery("INSERT INTO notification_campaign").
		WithArgs(
			"evt_post_published_1",
			"POST_PUBLISHED",
			int64(1001),
			int64(41),
			"POST",
			int64(41),
			"Hello",
			"Short summary",
			[]byte(`{"internalId":41}`),
			time.Date(2026, 7, 6, 18, 59, 0, 0, time.UTC),
			now,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(7001)))
	mock.ExpectQuery("INSERT INTO notification_campaign_shard").
		WithArgs(int64(7001), now).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(8001)))
	mock.ExpectExec("UPDATE consumed_events").
		WithArgs("evt_post_published_1", now).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	result, err := store.PlanPostPublishedCampaign(context.Background(), ports.PlanPostPublishedCampaignInput{
		Event: ports.ConsumedEventMetadata{
			EventID:      "evt_post_published_1",
			EventType:    "content.post.published",
			RoutingKey:   "content.post.published",
			ConsumerName: "zhicore-notification:content-post-consumer",
			PayloadHash:  "hash_1",
			ExpiresAt:    now.Add(168 * time.Hour),
		},
		SourceEventID: "evt_post_published_1",
		CampaignType:  "POST_PUBLISHED",
		AuthorID:      1001,
		PostID:        41,
		ObjectType:    "POST",
		ObjectID:      41,
		Title:         "Hello",
		Excerpt:       "Short summary",
		Payload:       []byte(`{"internalId":41}`),
		PublishedAt:   time.Date(2026, 7, 6, 18, 59, 0, 0, time.UTC),
		CreatedAt:     now,
	})
	if err != nil {
		t.Fatalf("PlanPostPublishedCampaign() error = %v", err)
	}
	if !result.Created || result.CampaignID != 7001 || result.ShardID != 8001 {
		t.Fatalf("result = %+v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestStorePlanPostPublishedCampaignReturnsDuplicateWhenEventAlreadyConsumed(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)

	mock.ExpectBegin()
	mock.ExpectQuery("INSERT INTO consumed_events").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectCommit()

	_, err = store.PlanPostPublishedCampaign(context.Background(), ports.PlanPostPublishedCampaignInput{
		Event: ports.ConsumedEventMetadata{
			EventID:      "evt_post_published_1",
			EventType:    "content.post.published",
			RoutingKey:   "content.post.published",
			ConsumerName: "zhicore-notification:content-post-consumer",
			PayloadHash:  "hash_1",
			ExpiresAt:    time.Date(2026, 7, 13, 19, 0, 0, 0, time.UTC),
		},
	})
	if err != ports.ErrDuplicateConsumedEvent {
		t.Fatalf("PlanPostPublishedCampaign() error = %v, want duplicate", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
