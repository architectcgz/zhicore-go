package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/lib/pq"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

func TestPostgresMaterializeCampaignFollowersFanoutIntegration(t *testing.T) {
	dsn := os.Getenv("ZHICORE_NOTIFICATION_PG_INTEGRATION_DSN")
	if dsn == "" {
		t.Skip("set ZHICORE_NOTIFICATION_PG_INTEGRATION_DSN to run PostgreSQL fanout integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}

	now := time.Date(2026, 7, 7, 23, 30, 0, 0, time.UTC)
	authorID := int64(910000001)
	recipients := []int64{910000101, 910000102, 910000103, 910000104, 910000105}
	sourceEventID := fmt.Sprintf("it_fanout_%d", time.Now().UnixNano())
	cleanupIntegrationFanout(t, ctx, db, 0, sourceEventID, recipients, authorID)
	campaignID := insertIntegrationCampaign(t, ctx, db, sourceEventID, authorID, now)
	t.Cleanup(func() {
		cleanupIntegrationFanout(t, context.Background(), db, campaignID, sourceEventID, recipients, authorID)
	})

	insertAuthorSubscription(t, ctx, db, recipients[1], authorID, "MUTED", false, false, false, false, now)
	insertAuthorSubscription(t, ctx, db, recipients[2], authorID, "DIGEST_ONLY", false, false, false, true, now)
	insertNotificationPreference(t, ctx, db, recipients[2], "POST_PUBLISHED_BY_FOLLOWING", "EMAIL", true, now)
	insertNotificationPreference(t, ctx, db, recipients[3], "POST_PUBLISHED_BY_FOLLOWING", "IN_APP", false, now)
	insertNotificationDND(t, ctx, db, recipients[4], now)

	store := NewStoreWithCodec(db, fakePublicIDCodec{})
	result, err := store.MaterializeCampaignFollowers(ctx, ports.MaterializeCampaignFollowersInput{
		ShardID:          810000001,
		CampaignID:       campaignID,
		AuthorID:         authorID,
		PostID:           810000001,
		AudienceClass:    "HOT",
		NotificationType: "POST_PUBLISHED_BY_FOLLOWING",
		Category:         "CONTENT",
		EventCode:        "content.post.published",
		TargetType:       "POST",
		TargetID:         "810000001",
		Title:            "Integration fanout post",
		Content:          "Integration fanout summary",
		Payload:          []byte(`{"postId":810000001}`),
		OccurredAt:       now,
		CreatedAt:        now,
		FollowerIDs:      recipients,
	})
	if err != nil {
		t.Fatalf("MaterializeCampaignFollowers() error = %v", err)
	}
	if result.ProcessedCount != 5 || result.SuccessCount != 2 || result.SkippedCount != 3 || result.FailedCount != 0 {
		t.Fatalf("result = %+v, want processed=5 success=2 skipped=3 failed=0", result)
	}

	assertIntegrationCount(t, ctx, db, "notifications", "recipient_id = ANY($1)", recipients, 2)
	assertIntegrationCount(t, ctx, db, "notification_stats", "recipient_id = ANY($1)", recipients, 2)
	assertIntegrationCount(t, ctx, db, "notification_group_state", "recipient_id = ANY($1)", recipients, 2)
	assertIntegrationCount(t, ctx, db, "notification_delivery", "campaign_id = $1", []int64{campaignID}, 6)
	assertDeliveryStatus(t, ctx, db, campaignID, recipients[0], "IN_APP", "IN_APP")
	assertDeliveryStatus(t, ctx, db, campaignID, recipients[0], "WEBSOCKET", "WEBSOCKET_PENDING")
	assertDeliveryStatus(t, ctx, db, campaignID, recipients[1], "IN_APP", "SKIPPED")
	assertDeliveryStatus(t, ctx, db, campaignID, recipients[2], "EMAIL", "DIGEST_PENDING")
	assertDeliveryStatus(t, ctx, db, campaignID, recipients[3], "IN_APP", "SKIPPED")
	assertDeliveryStatus(t, ctx, db, campaignID, recipients[4], "IN_APP", "IN_APP")
	assertNoDeliveryStatus(t, ctx, db, campaignID, recipients[4], "WEBSOCKET")
}

func TestPostgresClaimCampaignShardMultiWorkerIntegration(t *testing.T) {
	dsn := os.Getenv("ZHICORE_NOTIFICATION_PG_INTEGRATION_DSN")
	if dsn == "" {
		t.Skip("set ZHICORE_NOTIFICATION_PG_INTEGRATION_DSN to run PostgreSQL shard claim integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	db.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = db.Close() })
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}

	now := time.Date(2026, 7, 8, 0, 15, 0, 0, time.UTC)
	authorID := int64(920000001)
	sourceEventID := fmt.Sprintf("it_claim_%d", time.Now().UnixNano())
	cleanupIntegrationCampaignRows(t, ctx, db, sourceEventID, 0)
	campaignID := insertIntegrationCampaign(t, ctx, db, sourceEventID, authorID, now)
	t.Cleanup(func() {
		cleanupIntegrationCampaignRows(t, context.Background(), db, sourceEventID, campaignID)
	})
	for index := 0; index < 6; index++ {
		insertIntegrationCampaignShard(t, ctx, db, campaignID, "HOT", now)
	}

	store := NewStoreWithCodec(db, fakePublicIDCodec{})
	start := make(chan struct{})
	var wg sync.WaitGroup
	claims := make(chan ports.ClaimedCampaignShard, 6)
	errs := make(chan error, 6)
	for index := 0; index < 6; index++ {
		workerID := fmt.Sprintf("integration-worker-%d", index+1)
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			claim, err := store.ClaimCampaignShard(ctx, ports.ClaimCampaignShardInput{
				WorkerID:     workerID,
				Now:          now,
				ClaimTimeout: 30 * time.Second,
			})
			if err != nil {
				errs <- err
				return
			}
			claims <- claim
		}()
	}
	close(start)
	wg.Wait()
	close(claims)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("ClaimCampaignShard() error = %v", err)
		}
	}

	seenShardIDs := map[int64]string{}
	for claim := range claims {
		if !claim.Found {
			t.Fatalf("worker did not claim a shard")
		}
		if previousWorker, ok := seenShardIDs[claim.ShardID]; ok {
			t.Fatalf("shard %d claimed by both %s and %s", claim.ShardID, previousWorker, claim.ClaimedBy)
		}
		seenShardIDs[claim.ShardID] = claim.ClaimedBy
		if err := store.CompleteCampaignShard(ctx, ports.CompleteCampaignShardInput{
			ShardID:         claim.ShardID,
			WorkerID:        claim.ClaimedBy,
			ClaimDeadlineAt: claim.ClaimDeadlineAt,
			ProcessedCount:  1,
			SuccessCount:    1,
			CompletedAt:     now.Add(time.Second),
		}); err != nil {
			t.Fatalf("CompleteCampaignShard(%d) error = %v", claim.ShardID, err)
		}
	}
	if len(seenShardIDs) != 6 {
		t.Fatalf("claimed shard count = %d, want 6", len(seenShardIDs))
	}
	assertIntegrationCampaignShardStatusCount(t, ctx, db, campaignID, "COMPLETED", 6)
	assertIntegrationCampaignShardStatusCount(t, ctx, db, campaignID, "PROCESSING", 0)
}

func insertIntegrationCampaign(t *testing.T, ctx context.Context, db *sql.DB, sourceEventID string, authorID int64, now time.Time) int64 {
	t.Helper()
	var campaignID int64
	err := db.QueryRowContext(ctx, `
INSERT INTO notification_campaign (
    source_event_id, campaign_type, author_id, post_id, object_type, object_id,
    audience_class, title, excerpt, payload, published_at, status, created_at, updated_at
) VALUES ($1, 'POST_PUBLISHED', $2, 810000001, 'POST', 810000001,
    'HOT', 'Integration fanout post', 'Integration fanout summary', '{"postId":810000001}'::jsonb, $3, 'PLANNED', $3, $3)
RETURNING id`, sourceEventID, authorID, now).Scan(&campaignID)
	if err != nil {
		t.Fatalf("insert integration campaign: %v", err)
	}
	return campaignID
}

func insertIntegrationCampaignShard(t *testing.T, ctx context.Context, db *sql.DB, campaignID int64, audienceClass string, now time.Time) {
	t.Helper()
	_, err := db.ExecContext(ctx, `
INSERT INTO notification_campaign_shard (
    campaign_id, audience_class, status, created_at, updated_at
) VALUES ($1, $2, 'PENDING', $3, $3)`, campaignID, audienceClass, now)
	if err != nil {
		t.Fatalf("insert integration campaign shard: %v", err)
	}
}

func insertAuthorSubscription(t *testing.T, ctx context.Context, db *sql.DB, userID int64, authorID int64, level string, inApp bool, websocket bool, email bool, digest bool, now time.Time) {
	t.Helper()
	_, err := db.ExecContext(ctx, `
INSERT INTO notification_author_subscription (
    user_id, author_id, level, in_app_enabled, websocket_enabled, email_enabled, digest_enabled, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
ON CONFLICT (user_id, author_id) DO UPDATE
SET level = EXCLUDED.level,
    in_app_enabled = EXCLUDED.in_app_enabled,
    websocket_enabled = EXCLUDED.websocket_enabled,
    email_enabled = EXCLUDED.email_enabled,
    digest_enabled = EXCLUDED.digest_enabled,
    updated_at = EXCLUDED.updated_at`, userID, authorID, level, inApp, websocket, email, digest, now)
	if err != nil {
		t.Fatalf("insert author subscription: %v", err)
	}
}

func insertNotificationPreference(t *testing.T, ctx context.Context, db *sql.DB, userID int64, notificationType string, channel string, enabled bool, now time.Time) {
	t.Helper()
	_, err := db.ExecContext(ctx, `
INSERT INTO notification_user_preference (user_id, notification_type, channel, enabled, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $5)
ON CONFLICT (user_id, notification_type, channel) DO UPDATE
SET enabled = EXCLUDED.enabled, updated_at = EXCLUDED.updated_at`, userID, notificationType, channel, enabled, now)
	if err != nil {
		t.Fatalf("insert notification preference: %v", err)
	}
}

func insertNotificationDND(t *testing.T, ctx context.Context, db *sql.DB, userID int64, now time.Time) {
	t.Helper()
	_, err := db.ExecContext(ctx, `
INSERT INTO notification_user_dnd (user_id, enabled, start_time, end_time, timezone, categories, channels, created_at, updated_at)
VALUES ($1, TRUE, '22:00', '07:00', 'UTC', ARRAY['CONTENT']::varchar[], ARRAY['WEBSOCKET']::varchar[], $2, $2)
ON CONFLICT (user_id) DO UPDATE
SET enabled = EXCLUDED.enabled,
    start_time = EXCLUDED.start_time,
    end_time = EXCLUDED.end_time,
    timezone = EXCLUDED.timezone,
    categories = EXCLUDED.categories,
    channels = EXCLUDED.channels,
    updated_at = EXCLUDED.updated_at`, userID, now)
	if err != nil {
		t.Fatalf("insert notification dnd: %v", err)
	}
}

func assertIntegrationCount(t *testing.T, ctx context.Context, db *sql.DB, table string, predicate string, ids []int64, want int64) {
	t.Helper()
	var count int64
	query := fmt.Sprintf("SELECT count(*) FROM %s WHERE %s", table, predicate)
	var err error
	if len(ids) == 1 {
		err = db.QueryRowContext(ctx, query, ids[0]).Scan(&count)
	} else {
		err = db.QueryRowContext(ctx, query, pq.Array(ids)).Scan(&count)
	}
	if err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if count != want {
		t.Fatalf("%s count = %d, want %d", table, count, want)
	}
}

func assertDeliveryStatus(t *testing.T, ctx context.Context, db *sql.DB, campaignID int64, recipientID int64, channel string, want string) {
	t.Helper()
	var status string
	err := db.QueryRowContext(ctx, `
SELECT status FROM notification_delivery
WHERE campaign_id = $1 AND recipient_id = $2 AND channel = $3`, campaignID, recipientID, channel).Scan(&status)
	if err != nil {
		t.Fatalf("select delivery status campaign=%d recipient=%d channel=%s: %v", campaignID, recipientID, channel, err)
	}
	if status != want {
		t.Fatalf("delivery status campaign=%d recipient=%d channel=%s = %s, want %s", campaignID, recipientID, channel, status, want)
	}
}

func assertNoDeliveryStatus(t *testing.T, ctx context.Context, db *sql.DB, campaignID int64, recipientID int64, channel string) {
	t.Helper()
	var status string
	err := db.QueryRowContext(ctx, `
SELECT status FROM notification_delivery
WHERE campaign_id = $1 AND recipient_id = $2 AND channel = $3`, campaignID, recipientID, channel).Scan(&status)
	if err == nil {
		t.Fatalf("delivery campaign=%d recipient=%d channel=%s exists with status %s", campaignID, recipientID, channel, status)
	}
	if err != sql.ErrNoRows {
		t.Fatalf("select absent delivery status: %v", err)
	}
}

func cleanupIntegrationFanout(t *testing.T, ctx context.Context, db *sql.DB, campaignID int64, sourceEventID string, recipients []int64, authorID int64) {
	t.Helper()
	_, _ = db.ExecContext(ctx, "DELETE FROM notification_delivery WHERE campaign_id = $1 OR recipient_id = ANY($2)", campaignID, pq.Array(recipients))
	_, _ = db.ExecContext(ctx, "DELETE FROM notification_group_state WHERE recipient_id = ANY($1)", pq.Array(recipients))
	_, _ = db.ExecContext(ctx, "DELETE FROM notification_stats WHERE recipient_id = ANY($1)", pq.Array(recipients))
	_, _ = db.ExecContext(ctx, "DELETE FROM notifications WHERE recipient_id = ANY($1)", pq.Array(recipients))
	_, _ = db.ExecContext(ctx, "DELETE FROM notification_user_preference WHERE user_id = ANY($1)", pq.Array(recipients))
	_, _ = db.ExecContext(ctx, "DELETE FROM notification_user_dnd WHERE user_id = ANY($1)", pq.Array(recipients))
	_, _ = db.ExecContext(ctx, "DELETE FROM notification_author_subscription WHERE user_id = ANY($1) OR author_id = $2", pq.Array(recipients), authorID)
	_, _ = db.ExecContext(ctx, "DELETE FROM notification_campaign_shard WHERE campaign_id = $1", campaignID)
	_, _ = db.ExecContext(ctx, "DELETE FROM notification_campaign WHERE id = $1 OR source_event_id = $2", campaignID, sourceEventID)
}

func cleanupIntegrationCampaignRows(t *testing.T, ctx context.Context, db *sql.DB, sourceEventID string, campaignID int64) {
	t.Helper()
	_, _ = db.ExecContext(ctx, `
DELETE FROM notification_campaign_shard
WHERE campaign_id = $1
   OR campaign_id IN (SELECT id FROM notification_campaign WHERE source_event_id = $2)`, campaignID, sourceEventID)
	_, _ = db.ExecContext(ctx, "DELETE FROM notification_campaign WHERE id = $1 OR source_event_id = $2", campaignID, sourceEventID)
}

func assertIntegrationCampaignShardStatusCount(t *testing.T, ctx context.Context, db *sql.DB, campaignID int64, status string, want int64) {
	t.Helper()
	var count int64
	err := db.QueryRowContext(ctx, `
SELECT count(*) FROM notification_campaign_shard
WHERE campaign_id = $1 AND status = $2`, campaignID, status).Scan(&count)
	if err != nil {
		t.Fatalf("count campaign shards status=%s: %v", status, err)
	}
	if count != want {
		t.Fatalf("campaign shard status=%s count = %d, want %d", status, count, want)
	}
}
