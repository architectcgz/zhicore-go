package migrations_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContentPublishCoreMigrationContract(t *testing.T) {
	up := readContentPublishCoreMigration(t, ".up.sql")
	down := readContentPublishCoreMigration(t, ".down.sql")

	for _, fragment := range []string{
		"BEGIN;",
		"CREATE TABLE posts",
		"public_id VARCHAR(64) NOT NULL",
		"owner_id BIGINT NOT NULL",
		"post_version BIGINT NOT NULL",
		"published_body_id VARCHAR(64)",
		"published_body_hash VARCHAR(80)",
		"draft_body_id VARCHAR(64)",
		"draft_body_hash VARCHAR(80)",
		"CREATE UNIQUE INDEX ux_posts_public_id",
		"CHECK (status IN ('DRAFT', 'PUBLISHED', 'SCHEDULED', 'DELETED'))",
		"status <> 'PUBLISHED'",
		"published_body_hash IS NOT NULL",
		"published_plain_text_length IS NOT NULL",
		"published_at IS NOT NULL",
		"CREATE TABLE post_stats",
		"CHECK (view_count >= 0)",
		"CHECK (like_count >= 0)",
		"CHECK (favorite_count >= 0)",
		"CHECK (comment_count >= 0)",
		"CREATE TABLE outbox_events",
		"event_id VARCHAR(64) NOT NULL",
		"payload_json JSONB NOT NULL",
		"aggregate_version BIGINT NULL",
		"CHECK (status IN ('PENDING', 'CLAIMING', 'PUBLISHED', 'FAILED', 'DEAD'))",
		"CREATE TABLE outbox_retry_audit",
		"admin_user_id BIGINT NOT NULL",
		"retry_reason TEXT NOT NULL",
		"CREATE TABLE domain_event_tasks",
		"CREATE TABLE content_body_cleanup_tasks",
		"UNIQUE (body_id, task_type)",
		"CREATE TABLE content_body_repair_tasks",
		"body_id VARCHAR(64) NOT NULL",
		"COMMIT;",
	} {
		if !strings.Contains(up, fragment) {
			t.Fatalf("up migration missing %q", fragment)
		}
	}

	for _, fragment := range []string{
		"DROP TABLE IF EXISTS content_body_repair_tasks",
		"DROP TABLE IF EXISTS content_body_cleanup_tasks",
		"DROP TABLE IF EXISTS domain_event_tasks",
		"DROP TABLE IF EXISTS outbox_retry_audit",
		"DROP TABLE IF EXISTS outbox_events",
		"DROP TABLE IF EXISTS post_stats",
		"DROP TABLE IF EXISTS posts",
	} {
		if !strings.Contains(down, fragment) {
			t.Fatalf("down migration missing %q", fragment)
		}
	}
}

func TestScheduledPublishMigrationContract(t *testing.T) {
	up := readNamedMigration(t, "add_scheduled_publish_events", ".up.sql")
	down := readNamedMigration(t, "add_scheduled_publish_events", ".down.sql")

	for _, fragment := range []string{
		"CREATE TABLE scheduled_publish_events",
		"post_id BIGINT NOT NULL REFERENCES posts (id)",
		"draft_body_id VARCHAR(64) NOT NULL",
		"draft_body_hash VARCHAR(80) NOT NULL",
		"scheduled_at TIMESTAMPTZ NOT NULL",
		"CHECK (status IN ('PENDING', 'CANCELED', 'EXECUTED', 'FAILED', 'DEAD'))",
		"CREATE UNIQUE INDEX ux_scheduled_publish_events_pending_post",
	} {
		if !strings.Contains(up, fragment) {
			t.Fatalf("scheduled publish up migration missing %q", fragment)
		}
	}

	if !strings.Contains(down, "DROP TABLE IF EXISTS scheduled_publish_events") {
		t.Fatalf("scheduled publish down migration missing drop table")
	}
}

func readContentPublishCoreMigration(t *testing.T, suffix string) string {
	return readNamedMigration(t, "create_content_publish_core", suffix)
}

func readNamedMigration(t *testing.T, namePart, suffix string) string {
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
