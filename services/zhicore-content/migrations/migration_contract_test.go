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
		"CREATE TABLE domain_event_tasks",
		"CREATE TABLE content_body_cleanup_tasks",
		"UNIQUE (body_id, task_type)",
		"CREATE TABLE content_body_repair_tasks",
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
		"DROP TABLE IF EXISTS outbox_events",
		"DROP TABLE IF EXISTS post_stats",
		"DROP TABLE IF EXISTS posts",
	} {
		if !strings.Contains(down, fragment) {
			t.Fatalf("down migration missing %q", fragment)
		}
	}
}

func readContentPublishCoreMigration(t *testing.T, suffix string) string {
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
		if strings.Contains(entry.Name(), "create_content_publish_core") {
			matches = append(matches, entry.Name())
		}
	}
	if len(matches) != 1 {
		t.Fatalf("migration files ending %s = %v, want exactly one create_content_publish_core migration", suffix, matches)
	}

	body, err := os.ReadFile(filepath.Join(".", matches[0]))
	if err != nil {
		t.Fatalf("read %s: %v", matches[0], err)
	}
	return string(body)
}
