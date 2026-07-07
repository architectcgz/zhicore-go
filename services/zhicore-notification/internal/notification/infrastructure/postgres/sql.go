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
		panic(fmt.Sprintf("notification postgres SQL not found: %s: %v", name, err))
	}
	return string(content)
}

var (
	insertConsumedEventSQL                = mustSQL("insert_consumed_event.sql")
	nextNotificationIDSQL                 = mustSQL("next_notification_id.sql")
	nextDeliveryIDSQL                     = mustSQL("next_delivery_id.sql")
	insertInteractionNotificationSQL      = mustSQL("insert_interaction_notification.sql")
	upsertInteractionNotificationGroupSQL = mustSQL("upsert_interaction_notification_group.sql")
	markConsumedEventSQL                  = mustSQL("mark_consumed_event.sql")
	selectNotificationForMarkReadSQL      = mustSQL("select_notification_for_mark_read.sql")
	updateNotificationReadSQL             = mustSQL("update_notification_read.sql")
	decrementGroupUnreadSQL               = mustSQL("decrement_group_unread.sql")
	incrementNotificationStatsSQL         = mustSQL("increment_notification_stats.sql")
	decrementNotificationStatsSQL         = mustSQL("decrement_notification_stats.sql")
	markAllNotificationsReadSQL           = mustSQL("mark_all_notifications_read.sql")
	resetGroupUnreadSQL                   = mustSQL("reset_group_unread.sql")
	resetNotificationStatsSQL             = mustSQL("reset_notification_stats.sql")
	getUnreadCountSQL                     = mustSQL("get_unread_count.sql")
	getUnreadBreakdownSQL                 = mustSQL("get_unread_breakdown.sql")
	listAggregatedFromGroupStateSQL       = mustSQL("list_aggregated_from_group_state.sql")
	listAggregatedFromInboxSQL            = mustSQL("list_aggregated_from_inbox.sql")
	getNotificationPreferencesSQL         = mustSQL("get_notification_preferences.sql")
	deleteNotificationPreferencesSQL      = mustSQL("delete_notification_preferences.sql")
	insertNotificationPreferenceSQL       = mustSQL("insert_notification_preference.sql")
	getNotificationDNDSQL                 = mustSQL("get_notification_dnd.sql")
	upsertNotificationDNDSQL              = mustSQL("upsert_notification_dnd.sql")
	getAuthorSubscriptionSQL              = mustSQL("get_author_subscription.sql")
	upsertAuthorSubscriptionSQL           = mustSQL("upsert_author_subscription.sql")
	listDeliveriesSQL                     = mustSQL("list_deliveries.sql")
	retryDeliverySQL                      = mustSQL("retry_delivery.sql")
	insertPostPublishedCampaignSQL        = mustSQL("insert_post_published_campaign.sql")
	insertInitialCampaignShardSQL         = mustSQL("insert_initial_campaign_shard.sql")
	claimCampaignShardSQL                 = mustSQL("claim_campaign_shard.sql")
	failCampaignShardSQL                  = mustSQL("fail_campaign_shard.sql")
	completeCampaignShardSQL              = mustSQL("complete_campaign_shard.sql")
	insertCampaignDeliverySQL             = mustSQL("insert_campaign_delivery.sql")
	getCampaignDeliveryDecisionSQL        = mustSQL("get_campaign_delivery_decision.sql")
	lockRebuildGroupStateSQL              = mustSQL("lock_rebuild_group_state.sql")
	deleteGroupStateForRebuildSQL         = mustSQL("delete_group_state_for_rebuild.sql")
	rebuildGroupStateSQL                  = mustSQL("rebuild_group_state.sql")
)
