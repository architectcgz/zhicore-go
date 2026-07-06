package postgres

import (
	"embed"
	"fmt"
)

//go:embed sql/*.sql
var sqlFiles embed.FS

func mustSQL(name string) string {
	content, err := sqlFiles.ReadFile("sql/" + name)
	if err != nil {
		panic(fmt.Sprintf("content postgres SQL not found: %s: %v", name, err))
	}
	return string(content)
}

var (
	insertPostSQL                     = mustSQL("insert_post.sql")
	insertPostStatsSQL                = mustSQL("insert_post_stats.sql")
	selectPostForUpdateSQL            = mustSQL("select_post_for_update.sql")
	updateDraftBodySQL                = mustSQL("update_draft_body.sql")
	publishPostSQL                    = mustSQL("publish_post.sql")
	classifyPostMutationMissSQL       = mustSQL("classify_post_mutation_miss.sql")
	selectPublishedBodyPointerSQL     = mustSQL("select_published_body_pointer.sql")
	listPublishedPostsSQL             = mustSQL("list_published_posts.sql")
	getPublishedPostDetailSQL         = mustSQL("get_published_post_detail.sql")
	batchGetPublishedPostSummariesSQL = mustSQL("batch_get_published_post_summaries.sql")
	selectBodyReferencedSQL           = mustSQL("select_body_referenced.sql")
	insertOutboxEventSQL              = mustSQL("insert_outbox_event.sql")
	upsertCleanupTaskSQL              = mustSQL("upsert_cleanup_task.sql")
	upsertRepairTaskSQL               = mustSQL("upsert_repair_task.sql")
	claimCleanupTasksSQL              = mustSQL("claim_cleanup_tasks.sql")
	claimRepairTasksSQL               = mustSQL("claim_repair_tasks.sql")
	markCleanupTaskSucceededSQL       = mustSQL("mark_cleanup_task_succeeded.sql")
	markRepairTaskSucceededSQL        = mustSQL("mark_repair_task_succeeded.sql")
	markCleanupTaskFailedSQL          = mustSQL("mark_cleanup_task_failed.sql")
	markRepairTaskFailedSQL           = mustSQL("mark_repair_task_failed.sql")
	listAdminOutboxEventsSQL          = mustSQL("list_admin_outbox_events.sql")
	retryAdminOutboxEventSQL          = mustSQL("retry_admin_outbox_event.sql")
)
