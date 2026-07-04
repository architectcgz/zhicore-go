package migrations_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUserProfileMigrationDefinesProfileAndOutboxSchema(t *testing.T) {
	root := "."
	up := readOnlyMigration(t, root, ".up.sql")
	down := readOnlyMigration(t, root, ".down.sql")

	requiredUpFragments := []string{
		"CREATE TABLE users",
		"public_id VARCHAR(64) NOT NULL",
		"account_id BIGINT NOT NULL",
		"nickname VARCHAR(15) NOT NULL",
		"CREATE UNIQUE INDEX ux_users_public_id",
		"CREATE UNIQUE INDEX ux_users_account_id",
		"CREATE UNIQUE INDEX ux_users_nickname",
		"CREATE TABLE outbox_events",
		"event_id VARCHAR(64) NOT NULL",
		"payload_json JSONB NOT NULL",
		"claimed_by VARCHAR(128)",
		"claim_started_at TIMESTAMPTZ",
		"CHECK (status IN ('PENDING', 'CLAIMING', 'PUBLISHED', 'FAILED', 'DEAD'))",
		"CHECK (user_status IN ('ACTIVE', 'DEACTIVATED', 'DELETED'))",
	}
	for _, fragment := range requiredUpFragments {
		if !strings.Contains(up, fragment) {
			t.Fatalf("up migration missing %q", fragment)
		}
	}

	requiredDownFragments := []string{
		"DROP TABLE IF EXISTS outbox_events",
		"DROP TABLE IF EXISTS users",
	}
	for _, fragment := range requiredDownFragments {
		if !strings.Contains(down, fragment) {
			t.Fatalf("down migration missing %q", fragment)
		}
	}
}

func readOnlyMigration(t *testing.T, root, suffix string) string {
	t.Helper()

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read migration dir: %v", err)
	}

	var matches []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), suffix) {
			continue
		}
		if strings.Contains(entry.Name(), "create_user_profile_tables") {
			matches = append(matches, entry.Name())
		}
	}
	if len(matches) != 1 {
		t.Fatalf("migration files ending %s = %v, want exactly one create_user_profile_tables migration", suffix, matches)
	}

	body, err := os.ReadFile(filepath.Join(root, matches[0]))
	if err != nil {
		t.Fatalf("read %s: %v", matches[0], err)
	}
	return string(body)
}
