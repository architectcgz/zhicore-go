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
	listAuthorPostsSQL                = mustSQL("list_author_posts.sql")
	getDraftPostSQL                   = mustSQL("get_draft_post.sql")
	updateDraftMetaSQL                = mustSQL("update_draft_meta.sql")
	deleteDraftSQL                    = mustSQL("delete_draft.sql")
	unpublishPostSQL                  = mustSQL("unpublish_post.sql")
	deletePostSQL                     = mustSQL("delete_post.sql")
	restorePostSQL                    = mustSQL("restore_post.sql")
	schedulePostSQL                   = mustSQL("schedule_post.sql")
	upsertScheduledPublishEventSQL    = mustSQL("upsert_scheduled_publish_event.sql")
	cancelScheduleSQL                 = mustSQL("cancel_schedule.sql")
	cancelScheduledPublishEventSQL    = mustSQL("cancel_scheduled_publish_event.sql")
	selectBodyReferencedSQL           = mustSQL("select_body_referenced.sql")
	insertOutboxEventSQL              = mustSQL("insert_outbox_event.sql")
	upsertCleanupTaskSQL              = mustSQL("upsert_cleanup_task.sql")
	upsertRepairTaskSQL               = mustSQL("upsert_repair_task.sql")
	insertEngagementStatsTaskSQL      = mustSQL("insert_engagement_stats_task.sql")
	claimCleanupTasksSQL              = mustSQL("claim_cleanup_tasks.sql")
	claimRepairTasksSQL               = mustSQL("claim_repair_tasks.sql")
	claimEngagementStatsTasksSQL      = mustSQL("claim_engagement_stats_tasks.sql")
	markCleanupTaskSucceededSQL       = mustSQL("mark_cleanup_task_succeeded.sql")
	markRepairTaskSucceededSQL        = mustSQL("mark_repair_task_succeeded.sql")
	applyEngagementStatsTaskSQL       = mustSQL("apply_engagement_stats_task.sql")
	markCleanupTaskFailedSQL          = mustSQL("mark_cleanup_task_failed.sql")
	markRepairTaskFailedSQL           = mustSQL("mark_repair_task_failed.sql")
	markEngagementStatsTaskFailedSQL  = mustSQL("mark_engagement_stats_task_failed.sql")
	listAdminOutboxEventsSQL          = mustSQL("list_admin_outbox_events.sql")
	retryAdminOutboxEventSQL          = mustSQL("retry_admin_outbox_event.sql")
	listTagsSQL                       = mustSQL("list_tags.sql")
	getTagBySlugSQL                   = mustSQL("get_tag_by_slug.sql")
	searchTagsSQL                     = mustSQL("search_tags.sql")
	listHotTagsSQL                    = mustSQL("list_hot_tags.sql")
	listPublishedPostsByTagSQL        = mustSQL("list_published_posts_by_tag.sql")
	listPostTagsSQL                   = mustSQL("list_post_tags.sql")
	selectPublishedPostExistsSQL      = mustSQL("select_published_post_exists.sql")
	listPostTagsByIDSQL               = mustSQL("list_post_tags_by_id.sql")
	selectPostTagIDsSQL               = mustSQL("select_post_tag_ids.sql")
	selectTagsBySlugsSQL              = mustSQL("select_tags_by_slugs.sql")
	deletePostTagsSQL                 = mustSQL("delete_post_tags.sql")
	deletePostTagSQL                  = mustSQL("delete_post_tag.sql")
	insertPostTagSQL                  = mustSQL("insert_post_tag.sql")
	refreshTagStatsSQL                = mustSQL("refresh_tag_stats.sql")
	touchPostTagsSQL                  = mustSQL("touch_post_tags.sql")
	mutateLikeEngagementSQL           = mustSQL("mutate_like_engagement.sql")
	mutateUnlikeEngagementSQL         = mustSQL("mutate_unlike_engagement.sql")
	mutateFavoriteEngagementSQL       = mustSQL("mutate_favorite_engagement.sql")
	mutateUnfavoriteEngagementSQL     = mustSQL("mutate_unfavorite_engagement.sql")
	getPostEngagementSQL              = mustSQL("get_post_engagement.sql")
	batchGetViewerEngagementStatusSQL = mustSQL("batch_get_viewer_engagement_status.sql")
)
