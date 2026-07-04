package migrations_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommentCoreMigrationDefinesOwnedTablesAndRollback(t *testing.T) {
	up := readCommentMigration(t, ".up.sql")
	down := readCommentMigration(t, ".down.sql")

	for _, fragment := range []string{
		"CREATE TABLE comments",
		"content_internal_id BIGINT NOT NULL",
		"image_file_ids TEXT[]",
		"CHECK (COALESCE(cardinality(image_file_ids), 0) <= 9)",
		"CREATE TABLE comment_stats",
		"CREATE TABLE comment_post_stats",
		"CREATE TABLE comment_likes",
		"CREATE TABLE comment_counter_deltas",
		"CREATE TABLE comment_hot_rank",
		"CREATE TABLE comment_recommended_rank",
		"CREATE TABLE outbox_events",
		"CHECK (status IN ('PENDING', 'CLAIMING', 'PUBLISHED', 'FAILED', 'DEAD'))",
		"CREATE INDEX ix_comments_post_top_time",
		"CREATE INDEX ix_comment_recommended_rank_post",
	} {
		if !strings.Contains(up, fragment) {
			t.Fatalf("up migration missing %q", fragment)
		}
	}

	for _, fragment := range []string{
		"DROP TABLE IF EXISTS outbox_events",
		"DROP TABLE IF EXISTS comment_recommended_rank",
		"DROP TABLE IF EXISTS comment_hot_rank",
		"DROP TABLE IF EXISTS comment_counter_deltas",
		"DROP TABLE IF EXISTS comment_likes",
		"DROP TABLE IF EXISTS comment_post_stats",
		"DROP TABLE IF EXISTS comment_stats",
		"DROP TABLE IF EXISTS comments",
	} {
		if !strings.Contains(down, fragment) {
			t.Fatalf("down migration missing %q", fragment)
		}
	}
}

func readCommentMigration(t *testing.T, suffix string) string {
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
		if strings.Contains(entry.Name(), "create_comment_core_tables") {
			matches = append(matches, entry.Name())
		}
	}
	if len(matches) != 1 {
		t.Fatalf("migration files ending %s = %v, want exactly one create_comment_core_tables migration", suffix, matches)
	}
	body, err := os.ReadFile(filepath.Join(".", matches[0]))
	if err != nil {
		t.Fatalf("read %s: %v", matches[0], err)
	}
	return string(body)
}
