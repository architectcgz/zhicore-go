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
		panic(fmt.Sprintf("comment postgres SQL not found: %s: %v", name, err))
	}
	return string(content)
}

var (
	createCommentSQL              = mustSQL("create_comment.sql")
	findCommentForMutationSQL     = mustSQL("find_comment_for_mutation.sql")
	findReplyGuardPreviewSQL      = mustSQL("find_reply_guard_preview.sql")
	softDeleteSubtreeSQL          = mustSQL("soft_delete_subtree.sql")
	insertCommentStatsSQL         = mustSQL("insert_comment_stats.sql")
	incrementReplyCountSQL        = mustSQL("increment_reply_count.sql")
	decrementReplyCountSQL        = mustSQL("decrement_reply_count.sql")
	incrementPostStatsTopLevelSQL = mustSQL("increment_post_stats_top_level.sql")
	incrementPostStatsReplySQL    = mustSQL("increment_post_stats_reply.sql")
	decrementPostStatsSQL         = mustSQL("decrement_post_stats.sql")
	getPostStatsSQL               = mustSQL("get_post_stats.sql")
	insertHotRankSQL              = mustSQL("insert_hot_rank.sql")
	insertRecommendedRankSQL      = mustSQL("insert_recommended_rank.sql")
	hideTopLevelRanksSQL          = mustSQL("hide_top_level_ranks.sql")
	upsertLikeSQL                 = mustSQL("upsert_like.sql")
	deleteLikeSQL                 = mustSQL("delete_like.sql")
	insertCounterDeltaSQL         = mustSQL("insert_counter_delta.sql")
	listTopLevelRecommendedSQL    = mustSQL("list_top_level_recommended.sql")
	listTopLevelHotSQL            = mustSQL("list_top_level_hot.sql")
	listTopLevelTimeSQL           = mustSQL("list_top_level_time.sql")
	getCommentDetailSQL           = mustSQL("get_comment_detail.sql")
	checkRootCommentSQL           = mustSQL("check_root_comment.sql")
	countRepliesSQL               = mustSQL("count_replies.sql")
	listRepliesHotSQL             = mustSQL("list_replies_hot.sql")
	listRepliesTimeSQL            = mustSQL("list_replies_time.sql")
	batchViewerLikedSQL           = mustSQL("batch_viewer_liked.sql")
)
