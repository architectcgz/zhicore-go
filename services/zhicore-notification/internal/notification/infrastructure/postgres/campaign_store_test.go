package postgres

import (
	"context"
	"database/sql"
	"strings"
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
	activeSince := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)

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
			"HOT",
			sql.NullTime{Time: activeSince, Valid: true},
			"Hello",
			"Short summary",
			[]byte(`{"internalId":41}`),
			time.Date(2026, 7, 6, 18, 59, 0, 0, time.UTC),
			now,
		).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(7001)))
	mock.ExpectQuery("INSERT INTO notification_campaign_shard").
		WithArgs(int64(7001), "HOT", sql.NullTime{Time: activeSince, Valid: true}, now).
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
		SourceEventID:       "evt_post_published_1",
		CampaignType:        "POST_PUBLISHED",
		AuthorID:            1001,
		PostID:              41,
		ObjectType:          "POST",
		ObjectID:            41,
		AudienceClass:       "HOT",
		AudienceActiveSince: &activeSince,
		Title:               "Hello",
		Excerpt:             "Short summary",
		Payload:             []byte(`{"internalId":41}`),
		PublishedAt:         time.Date(2026, 7, 6, 18, 59, 0, 0, time.UTC),
		CreatedAt:           now,
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

func TestStoreClaimCampaignShardUsesSkipLockedAndConfiguredTimeout(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)
	now := time.Date(2026, 7, 6, 20, 0, 0, 0, time.UTC)
	deadline := now.Add(30 * time.Second)
	activeSince := time.Date(2026, 6, 6, 20, 0, 0, 0, time.UTC)

	mock.ExpectQuery("FOR UPDATE SKIP LOCKED").
		WithArgs("worker-1", now, int64(30)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "campaign_id", "author_id", "post_id", "audience_class", "audience_active_since", "follower_cursor", "attempt_count", "claimed_by", "claim_deadline_at", "title", "excerpt", "payload", "published_at"}).
			AddRow(int64(8001), int64(7001), int64(1001), int64(41), "HOT", activeSince, "", 2, "worker-1", deadline, "Hello", "Excerpt", []byte(`{"postId":41}`), now))

	claim, err := store.ClaimCampaignShard(context.Background(), ports.ClaimCampaignShardInput{
		WorkerID:     "worker-1",
		Now:          now,
		ClaimTimeout: 30 * time.Second,
	})
	if err != nil {
		t.Fatalf("ClaimCampaignShard() error = %v", err)
	}
	if !claim.Found ||
		claim.ShardID != 8001 ||
		claim.CampaignID != 7001 ||
		claim.AuthorID != 1001 ||
		claim.PostID != 41 ||
		claim.AudienceClass != "HOT" ||
		claim.AudienceActiveSince == nil ||
		!claim.AudienceActiveSince.Equal(activeSince) ||
		claim.AttemptCount != 2 ||
		claim.ClaimedBy != "worker-1" ||
		!claim.ClaimDeadlineAt.Equal(deadline) {
		t.Fatalf("claim = %+v", claim)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestStoreFailCampaignShardSchedulesRetry(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)
	now := time.Date(2026, 7, 6, 20, 10, 0, 0, time.UTC)
	deadline := now.Add(30 * time.Second)

	mock.ExpectExec("UPDATE notification_campaign_shard").
		WithArgs(int64(8001), "worker-1", deadline, "USER_FOLLOWER_SHARD_DEGRADED", now, int64(120)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = store.FailCampaignShard(context.Background(), ports.FailCampaignShardInput{
		ShardID:         8001,
		WorkerID:        "worker-1",
		ClaimDeadlineAt: deadline,
		ErrorCode:       "USER_FOLLOWER_SHARD_DEGRADED",
		FailedAt:        now,
		RetryAfter:      2 * time.Minute,
	})
	if err != nil {
		t.Fatalf("FailCampaignShard() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestStoreCompleteCampaignShardUpdatesProgressAndRequeuesWhenHasMore(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)
	now := time.Date(2026, 7, 6, 20, 20, 0, 0, time.UTC)
	deadline := now.Add(30 * time.Second)

	mock.ExpectExec("UPDATE notification_campaign_shard").
		WithArgs(int64(8001), "worker-1", deadline, int64(2), int64(2), int64(0), int64(0), "cursor-2", true, now).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = store.CompleteCampaignShard(context.Background(), ports.CompleteCampaignShardInput{
		ShardID:         8001,
		WorkerID:        "worker-1",
		ClaimDeadlineAt: deadline,
		ProcessedCount:  2,
		SuccessCount:    2,
		NextCursor:      "cursor-2",
		HasMore:         true,
		CompletedAt:     now,
	})
	if err != nil {
		t.Fatalf("CompleteCampaignShard() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestStoreCompleteCampaignShardReturnsLeaseLostWhenClaimTokenDoesNotMatch(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)
	now := time.Date(2026, 7, 6, 20, 25, 0, 0, time.UTC)
	deadline := now.Add(30 * time.Second)

	mock.ExpectExec("UPDATE notification_campaign_shard").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = store.CompleteCampaignShard(context.Background(), ports.CompleteCampaignShardInput{
		ShardID:         8001,
		WorkerID:        "stale-worker",
		ClaimDeadlineAt: deadline,
		CompletedAt:     now,
	})
	if err != ports.ErrShardLeaseLost {
		t.Fatalf("CompleteCampaignShard() error = %v, want lease lost", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestCampaignDeliveryDecisionHonorsMutedAuthorSubscription(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })

	mock.ExpectQuery("WITH subscription").
		WithArgs(int64(2001), int64(1001), "POST_PUBLISHED_BY_FOLLOWING").
		WillReturnRows(campaignDecisionRows("MUTED", true, true, true, false, "", "", "UTC", "{}", "{}"))
	mock.ExpectRollback()

	decision, err := campaignDeliveryDecisionForRecipient(context.Background(), tx, 2001, 1001, "POST_PUBLISHED_BY_FOLLOWING", "CONTENT", time.Now())
	if err != nil {
		t.Fatalf("campaignDeliveryDecisionForRecipient() error = %v", err)
	}
	if !decision.SkipAll || decision.InAppEnabled || decision.WebsocketEnabled || decision.DigestOnly {
		t.Fatalf("decision = %+v, want muted skip all", decision)
	}
}

func TestCampaignDeliveryDecisionHonorsDigestOnlySubscriptionWithExplicitEmailOptIn(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })

	mock.ExpectQuery("WITH subscription").
		WithArgs(int64(2001), int64(1001), "POST_PUBLISHED_BY_FOLLOWING").
		WillReturnRows(campaignDecisionRows("DIGEST_ONLY", true, true, true, false, "", "", "UTC", "{}", "{}"))
	mock.ExpectRollback()

	decision, err := campaignDeliveryDecisionForRecipient(context.Background(), tx, 2001, 1001, "POST_PUBLISHED_BY_FOLLOWING", "CONTENT", time.Now())
	if err != nil {
		t.Fatalf("campaignDeliveryDecisionForRecipient() error = %v", err)
	}
	if !decision.DigestOnly || decision.SkipAll || decision.InAppEnabled || decision.WebsocketEnabled {
		t.Fatalf("decision = %+v, want digest only without inbox/websocket", decision)
	}
}

func TestCampaignDeliveryDecisionLetsUserEmailPreferenceOverrideDigestOnlySubscription(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })

	mock.ExpectQuery("WITH subscription").
		WithArgs(int64(2001), int64(1001), "POST_PUBLISHED_BY_FOLLOWING").
		WillReturnRows(campaignDecisionRowsWithEmailPreference("DIGEST_ONLY", true, true, false, true, false, "", "", "UTC", "{}", "{}"))
	mock.ExpectRollback()

	decision, err := campaignDeliveryDecisionForRecipient(context.Background(), tx, 2001, 1001, "POST_PUBLISHED_BY_FOLLOWING", "CONTENT", time.Now())
	if err != nil {
		t.Fatalf("campaignDeliveryDecisionForRecipient() error = %v", err)
	}
	if decision.DigestOnly || decision.SkipAll || decision.InAppEnabled || decision.WebsocketEnabled {
		t.Fatalf("decision = %+v, want global email preference to suppress digest-only subscription", decision)
	}
}

func TestCampaignDeliveryDecisionDefaultsEmailToExplicitOptIn(t *testing.T) {
	if !strings.Contains(getCampaignDeliveryDecisionSQL, "COALESCE((SELECT enabled FROM email_preference), FALSE)") {
		t.Fatalf("email preference must default to disabled so email delivery is explicit opt-in")
	}
}

func TestCampaignDeliveryDecisionHonorsActiveDNDForWebsocket(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	mock.ExpectBegin()
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback() })

	mock.ExpectQuery("WITH subscription").
		WithArgs(int64(2001), int64(1001), "POST_PUBLISHED_BY_FOLLOWING").
		WillReturnRows(campaignDecisionRows("ALL", true, true, true, true, "22:00", "07:00", "UTC", "{CONTENT}", "{WEBSOCKET}"))
	mock.ExpectRollback()

	decision, err := campaignDeliveryDecisionForRecipient(
		context.Background(),
		tx,
		2001,
		1001,
		"POST_PUBLISHED_BY_FOLLOWING",
		"CONTENT",
		time.Date(2026, 7, 7, 23, 30, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("campaignDeliveryDecisionForRecipient() error = %v", err)
	}
	if !decision.InAppEnabled || decision.WebsocketEnabled || decision.DigestOnly || decision.SkipAll {
		t.Fatalf("decision = %+v, want in-app only while websocket is suppressed by dnd", decision)
	}
}

func campaignDecisionRows(level string, inApp bool, websocket bool, digest bool, dnd bool, start string, end string, timezone string, categories string, channels string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"level",
		"in_app_enabled",
		"websocket_enabled",
		"email_preference_enabled",
		"digest_enabled",
		"dnd_enabled",
		"start_time",
		"end_time",
		"timezone",
		"categories",
		"channels",
	}).AddRow(level, inApp, websocket, true, digest, dnd, start, end, timezone, categories, channels)
}

func campaignDecisionRowsWithEmailPreference(level string, inApp bool, websocket bool, email bool, digest bool, dnd bool, start string, end string, timezone string, categories string, channels string) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"level",
		"in_app_enabled",
		"websocket_enabled",
		"email_enabled",
		"digest_enabled",
		"dnd_enabled",
		"start_time",
		"end_time",
		"timezone",
		"categories",
		"channels",
	}).AddRow(level, inApp, websocket, email, digest, dnd, start, end, timezone, categories, channels)
}

func TestStoreRebuildGroupStateUsesSingleUserAdvisoryLock(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)
	now := time.Date(2026, 7, 6, 20, 30, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery("pg_try_advisory_xact_lock").
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"locked"}).AddRow(true))
	mock.ExpectExec("DELETE FROM notification_group_state").
		WithArgs(int64(42)).
		WillReturnResult(sqlmock.NewResult(0, 3))
	mock.ExpectQuery("INSERT INTO notification_group_state").
		WithArgs(int64(42), now).
		WillReturnRows(sqlmock.NewRows([]string{"rebuilt_count"}).AddRow(int64(2)))
	mock.ExpectCommit()

	result, err := store.RebuildGroupState(context.Background(), ports.RebuildGroupStateInput{RecipientID: 42, RebuiltAt: now})
	if err != nil {
		t.Fatalf("RebuildGroupState() error = %v", err)
	}
	if result.RebuiltGroups != 2 {
		t.Fatalf("result = %+v, want two rebuilt groups", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestStoreRebuildGroupStateReturnsLockedWhenSingleUserRebuildAlreadyRunning(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("new sqlmock: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	store := NewStore(db)

	mock.ExpectBegin()
	mock.ExpectQuery("pg_try_advisory_xact_lock").
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"locked"}).AddRow(false))
	mock.ExpectCommit()

	_, err = store.RebuildGroupState(context.Background(), ports.RebuildGroupStateInput{RecipientID: 42, RebuiltAt: time.Now()})
	if err != ports.ErrRebuildLocked {
		t.Fatalf("RebuildGroupState() error = %v, want ErrRebuildLocked", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
