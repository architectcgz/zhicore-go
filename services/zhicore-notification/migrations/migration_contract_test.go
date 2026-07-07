package migrations_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNotificationInboxCoreMigrationDefinesInboxGroupStateAndConsumedEvents(t *testing.T) {
	up := readNotificationMigration(t, "create_notification_inbox_core", ".up.sql")
	down := readNotificationMigration(t, "create_notification_inbox_core", ".down.sql")

	for _, fragment := range []string{
		"BEGIN;",
		"CREATE TABLE notifications",
		"public_id VARCHAR(32) NOT NULL",
		"recipient_id BIGINT NOT NULL",
		"actor_id BIGINT NULL",
		"source_event_id VARCHAR(128) NOT NULL",
		"dedupe_key VARCHAR(256) NOT NULL",
		"group_key VARCHAR(256) NOT NULL",
		"payload JSONB NOT NULL DEFAULT '{}'::jsonb",
		"expires_at TIMESTAMPTZ NULL",
		"CHECK (public_id <> '')",
		"CHECK (category IN ('INTERACTION', 'CONTENT', 'SOCIAL', 'SYSTEM', 'SECURITY'))",
		"CHECK (notification_type IN ('POST_LIKED', 'POST_COMMENTED', 'COMMENT_REPLIED', 'USER_FOLLOWED', 'POST_PUBLISHED_BY_FOLLOWING', 'POST_PUBLISHED_DIGEST', 'SYSTEM_ANNOUNCEMENT', 'SECURITY_ALERT'))",
		"CHECK (importance IN ('NORMAL', 'HIGH', 'CRITICAL'))",
		"CREATE UNIQUE INDEX ux_notifications_public_id",
		"CREATE UNIQUE INDEX ux_notifications_source_event_id",
		"CREATE UNIQUE INDEX ux_notifications_recipient_dedupe_key",
		"CREATE INDEX ix_notifications_expires_at",
		"CREATE TABLE notification_group_state",
		"recipient_id BIGINT NOT NULL",
		"latest_notification_id BIGINT NOT NULL REFERENCES notifications (id)",
		"latest_actor_ids BIGINT[] NOT NULL DEFAULT '{}'",
		"aggregated_content JSONB NOT NULL DEFAULT '{}'::jsonb",
		"PRIMARY KEY (recipient_id, group_key)",
		"CHECK (total_count >= 0)",
		"CHECK (unread_count >= 0)",
		"CHECK (unread_count <= total_count)",
		"CREATE TABLE consumed_events",
		"event_id VARCHAR(128) NOT NULL PRIMARY KEY",
		"payload_hash VARCHAR(128) NOT NULL",
		"expires_at TIMESTAMPTZ NOT NULL",
		"CHECK (status IN ('PROCESSING', 'CONSUMED', 'FAILED', 'DEAD'))",
		"CREATE INDEX ix_consumed_events_expires_at",
		"COMMIT;",
	} {
		if !strings.Contains(up, fragment) {
			t.Fatalf("up migration missing %q", fragment)
		}
	}

	for _, fragment := range []string{
		"DROP TABLE IF EXISTS consumed_events",
		"DROP TABLE IF EXISTS notification_group_state",
		"DROP TABLE IF EXISTS notifications",
	} {
		if !strings.Contains(down, fragment) {
			t.Fatalf("down migration missing %q", fragment)
		}
	}
}

func TestNotificationPreferenceAndDeliveryMigrationDefinesSettingsAndLedger(t *testing.T) {
	up := readNotificationMigration(t, "add_notification_preference_and_delivery", ".up.sql")
	down := readNotificationMigration(t, "add_notification_preference_and_delivery", ".down.sql")

	for _, fragment := range []string{
		"BEGIN;",
		"CREATE TABLE notification_user_preference",
		"user_id BIGINT NOT NULL",
		"notification_type VARCHAR(64) NOT NULL",
		"channel VARCHAR(32) NOT NULL",
		"enabled BOOLEAN NOT NULL",
		"PRIMARY KEY (user_id, notification_type, channel)",
		"CHECK (channel IN ('IN_APP', 'WEBSOCKET', 'EMAIL', 'SMS'))",
		"CREATE TABLE notification_user_dnd",
		"start_time TIME NOT NULL",
		"end_time TIME NOT NULL",
		"timezone VARCHAR(64) NOT NULL",
		"categories VARCHAR(64)[] NOT NULL DEFAULT '{}'",
		"channels VARCHAR(32)[] NOT NULL DEFAULT '{}'",
		"CHECK (start_time <> end_time)",
		"CREATE TABLE notification_author_subscription",
		"author_id BIGINT NOT NULL",
		"level VARCHAR(32) NOT NULL",
		"CHECK (level IN ('ALL', 'DIGEST_ONLY', 'MUTED'))",
		"PRIMARY KEY (user_id, author_id)",
		"CREATE TABLE notification_delivery",
		"public_id VARCHAR(32) NOT NULL",
		"recipient_id BIGINT NOT NULL",
		"notification_id BIGINT NULL REFERENCES notifications (id)",
		"channel VARCHAR(32) NOT NULL",
		"status VARCHAR(64) NOT NULL",
		"dedupe_key VARCHAR(256) NOT NULL",
		"CHECK (status IN ('IN_APP', 'WEBSOCKET_PENDING', 'WEBSOCKET_DISPATCHED', 'DIGEST_PENDING', 'DIGEST_DELIVERED', 'SKIPPED', 'FAILED'))",
		"CREATE UNIQUE INDEX ux_notification_delivery_public_id",
		"CREATE UNIQUE INDEX ux_notification_delivery_dedupe_key",
		"CREATE INDEX ix_notification_delivery_recipient_created_at",
		"COMMIT;",
	} {
		if !strings.Contains(up, fragment) {
			t.Fatalf("up migration missing %q", fragment)
		}
	}

	for _, fragment := range []string{
		"DROP TABLE IF EXISTS notification_delivery",
		"DROP TABLE IF EXISTS notification_author_subscription",
		"DROP TABLE IF EXISTS notification_user_dnd",
		"DROP TABLE IF EXISTS notification_user_preference",
	} {
		if !strings.Contains(down, fragment) {
			t.Fatalf("down migration missing %q", fragment)
		}
	}
}

func TestNotificationStatsMigrationDefinesUserUnreadReadModel(t *testing.T) {
	up := readNotificationMigration(t, "add_notification_stats", ".up.sql")
	down := readNotificationMigration(t, "add_notification_stats", ".down.sql")

	for _, fragment := range []string{
		"BEGIN;",
		"CREATE TABLE notification_stats",
		"recipient_id BIGINT PRIMARY KEY",
		"unread_total BIGINT NOT NULL DEFAULT 0",
		"unread_interaction BIGINT NOT NULL DEFAULT 0",
		"unread_content BIGINT NOT NULL DEFAULT 0",
		"unread_social BIGINT NOT NULL DEFAULT 0",
		"unread_system BIGINT NOT NULL DEFAULT 0",
		"unread_security BIGINT NOT NULL DEFAULT 0",
		"CHECK (unread_total >= 0)",
		"CHECK (unread_interaction >= 0)",
		"CHECK (unread_total = unread_interaction + unread_content + unread_social + unread_system + unread_security)",
		"COMMIT;",
	} {
		if !strings.Contains(up, fragment) {
			t.Fatalf("up migration missing %q", fragment)
		}
	}

	if !strings.Contains(down, "DROP TABLE IF EXISTS notification_stats") {
		t.Fatalf("down migration missing notification_stats drop")
	}
	if strings.Contains(down, "DROP TABLE IF EXISTS notifications") || strings.Contains(down, "DROP TABLE IF EXISTS notification_group_state") {
		t.Fatalf("notification_stats down migration must not drop inbox core tables")
	}
}

func TestNotificationCampaignMigrationDefinesCampaignAndShardTables(t *testing.T) {
	up := readNotificationMigration(t, "add_notification_campaign_tables", ".up.sql")
	down := readNotificationMigration(t, "add_notification_campaign_tables", ".down.sql")

	for _, fragment := range []string{
		"BEGIN;",
		"CREATE TABLE notification_campaign",
		"source_event_id VARCHAR(128) NOT NULL",
		"campaign_type VARCHAR(64) NOT NULL",
		"author_id BIGINT NOT NULL",
		"post_id BIGINT NOT NULL",
		"audience_class VARCHAR(32) NOT NULL",
		"audience_active_since TIMESTAMPTZ NULL",
		"audience_estimate BIGINT NULL",
		"payload JSONB NOT NULL DEFAULT '{}'::jsonb",
		"status VARCHAR(64) NOT NULL",
		"CHECK (status IN ('PLANNED', 'PROCESSING', 'COMPLETED', 'FAILED', 'CANCELED'))",
		"CHECK (audience_class IN ('HOT', 'WARM', 'COLD', 'ALL'))",
		"CREATE UNIQUE INDEX ux_notification_campaign_source_event_id",
		"CREATE TABLE notification_campaign_shard",
		"campaign_id BIGINT NOT NULL REFERENCES notification_campaign (id)",
		"audience_class VARCHAR(32) NOT NULL",
		"audience_active_since TIMESTAMPTZ NULL",
		"follower_cursor VARCHAR(256) NOT NULL DEFAULT ''",
		"next_follower_cursor VARCHAR(256) NOT NULL DEFAULT ''",
		"processed_count BIGINT NOT NULL DEFAULT 0",
		"success_count BIGINT NOT NULL DEFAULT 0",
		"skipped_count BIGINT NOT NULL DEFAULT 0",
		"failed_count BIGINT NOT NULL DEFAULT 0",
		"claimed_by VARCHAR(128) NOT NULL DEFAULT ''",
		"claim_deadline_at TIMESTAMPTZ NULL",
		"next_retry_at TIMESTAMPTZ NULL",
		"CHECK (status IN ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED'))",
		"CREATE INDEX ix_notification_campaign_shard_claim",
		"ALTER TABLE notification_delivery",
		"ADD CONSTRAINT fk_notification_delivery_campaign",
		"FOREIGN KEY (campaign_id) REFERENCES notification_campaign (id)",
		"COMMIT;",
	} {
		if !strings.Contains(up, fragment) {
			t.Fatalf("up migration missing %q", fragment)
		}
	}

	for _, fragment := range []string{
		"ALTER TABLE notification_delivery",
		"DROP CONSTRAINT IF EXISTS fk_notification_delivery_campaign",
		"DROP TABLE IF EXISTS notification_campaign_shard",
		"DROP TABLE IF EXISTS notification_campaign",
	} {
		if !strings.Contains(down, fragment) {
			t.Fatalf("down migration missing %q", fragment)
		}
	}
	if strings.Contains(down, "DROP TABLE IF EXISTS notification_delivery") || strings.Contains(down, "DROP TABLE IF EXISTS notifications") {
		t.Fatalf("campaign down migration must not drop delivery or inbox tables")
	}
}

func readNotificationMigration(t *testing.T, namePart, suffix string) string {
	t.Helper()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read migration dir: %v", err)
	}

	var matches []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), suffix) {
			continue
		}
		if strings.Contains(entry.Name(), namePart) {
			matches = append(matches, entry.Name())
		}
	}
	if len(matches) != 1 {
		t.Fatalf("migration files ending %s containing %s = %v, want exactly one", suffix, namePart, matches)
	}

	body, err := os.ReadFile(filepath.Join(".", matches[0]))
	if err != nil {
		t.Fatalf("read %s: %v", matches[0], err)
	}
	return string(body)
}
