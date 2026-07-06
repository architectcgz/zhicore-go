package outbox

import (
	"embed"
	"fmt"
)

//go:embed sql/*.sql
var sqlFiles embed.FS

func mustSQL(name string) string {
	content, err := sqlFiles.ReadFile("sql/" + name)
	if err != nil {
		panic(fmt.Sprintf("outbox SQL template not found: %s: %v", name, err))
	}
	return string(content)
}

var (
	claimPendingTemplate                     = mustSQL("claim_pending.sql")
	claimPendingWithAggregateVersionTemplate = mustSQL("claim_pending_with_aggregate_version.sql")
	markPublishedTemplate                    = mustSQL("mark_published.sql")
	markFailedTemplate                       = mustSQL("mark_failed.sql")
	insertTemplate                           = mustSQL("insert.sql")
)
