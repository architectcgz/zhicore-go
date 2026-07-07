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

func TestContentTaxonomyMigrationContract(t *testing.T) {
	up := readNamedMigration(t, "add_content_taxonomy", ".up.sql")
	down := readNamedMigration(t, "add_content_taxonomy", ".down.sql")

	for _, fragment := range []string{
		"ALTER TABLE posts ADD COLUMN category_id BIGINT NULL",
		"ALTER TABLE posts ADD COLUMN topic_id BIGINT NULL",
		"CREATE TABLE categories",
		"kind VARCHAR(16) NOT NULL",
		"public_id VARCHAR(64) NOT NULL",
		"slug VARCHAR(96) NOT NULL",
		"CHECK (kind IN ('CATEGORY', 'TOPIC'))",
		"CREATE UNIQUE INDEX ux_categories_kind_slug",
		"CREATE TABLE tags",
		"CREATE UNIQUE INDEX ux_tags_slug",
		"CREATE TABLE tag_stats",
		"post_count BIGINT NOT NULL DEFAULT 0",
		"CREATE TABLE post_tags",
		"position INT NOT NULL",
		"CREATE UNIQUE INDEX ux_post_tags_post_tag",
		"CREATE INDEX ix_post_tags_tag_post",
		"COMMIT;",
	} {
		if !strings.Contains(up, fragment) {
			t.Fatalf("taxonomy up migration missing %q", fragment)
		}
	}

	for _, fragment := range []string{
		"DROP TABLE IF EXISTS post_tags",
		"DROP TABLE IF EXISTS tag_stats",
		"DROP TABLE IF EXISTS tags",
		"DROP TABLE IF EXISTS categories",
		"ALTER TABLE posts DROP COLUMN IF EXISTS topic_id",
		"ALTER TABLE posts DROP COLUMN IF EXISTS category_id",
	} {
		if !strings.Contains(down, fragment) {
			t.Fatalf("taxonomy down migration missing %q", fragment)
		}
	}
}

func TestContentEngagementMigrationContract(t *testing.T) {
	up := readNamedMigration(t, "add_content_engagement", ".up.sql")
	down := readNamedMigration(t, "add_content_engagement", ".down.sql")

	for _, fragment := range []string{
		"BEGIN;",
		"CREATE TABLE post_likes",
		"post_id BIGINT NOT NULL REFERENCES posts (id)",
		"user_id BIGINT NOT NULL",
		"UNIQUE (post_id, user_id)",
		"CREATE INDEX ix_post_likes_user_post",
		"CREATE TABLE post_favorites",
		"CREATE UNIQUE INDEX ux_post_favorites_post_user",
		"CREATE INDEX ix_post_favorites_user_post",
		"COMMIT;",
	} {
		if !strings.Contains(up, fragment) {
			t.Fatalf("engagement up migration missing %q", fragment)
		}
	}

	for _, fragment := range []string{
		"DROP TABLE IF EXISTS post_favorites",
		"DROP TABLE IF EXISTS post_likes",
	} {
		if !strings.Contains(down, fragment) {
			t.Fatalf("engagement down migration missing %q", fragment)
		}
	}
}

func TestAdminPostAuditMigrationContract(t *testing.T) {
	up := readNamedMigration(t, "add_admin_post_audit", ".up.sql")
	down := readNamedMigration(t, "add_admin_post_audit", ".down.sql")

	for _, fragment := range []string{
		"BEGIN;",
		"CREATE TABLE admin_post_audit",
		"post_id BIGINT NOT NULL REFERENCES posts (id)",
		"public_id VARCHAR(64) NOT NULL",
		"admin_user_id BIGINT NOT NULL",
		"action VARCHAR(32) NOT NULL",
		"reason TEXT NOT NULL",
		"previous_status VARCHAR(32) NOT NULL",
		"new_status VARCHAR(32) NOT NULL",
		"occurred_at TIMESTAMPTZ NOT NULL",
		"CHECK (action IN ('DELETE'))",
		"CHECK (previous_status IN ('DRAFT', 'PUBLISHED', 'SCHEDULED', 'DELETED'))",
		"CHECK (new_status IN ('DRAFT', 'PUBLISHED', 'SCHEDULED', 'DELETED'))",
		"CREATE INDEX ix_admin_post_audit_post_created_at",
		"CREATE INDEX ix_admin_post_audit_admin_created_at",
		"COMMIT;",
	} {
		if !strings.Contains(up, fragment) {
			t.Fatalf("admin post audit up migration missing %q", fragment)
		}
	}

	if !strings.Contains(down, "DROP TABLE IF EXISTS admin_post_audit") {
		t.Fatalf("admin post audit down migration missing drop table")
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
