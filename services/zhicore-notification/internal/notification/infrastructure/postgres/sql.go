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
	insertInteractionNotificationSQL      = mustSQL("insert_interaction_notification.sql")
	upsertInteractionNotificationGroupSQL = mustSQL("upsert_interaction_notification_group.sql")
	markConsumedEventSQL                  = mustSQL("mark_consumed_event.sql")
	selectNotificationForMarkReadSQL      = mustSQL("select_notification_for_mark_read.sql")
	updateNotificationReadSQL             = mustSQL("update_notification_read.sql")
	decrementGroupUnreadSQL               = mustSQL("decrement_group_unread.sql")
	markAllNotificationsReadSQL           = mustSQL("mark_all_notifications_read.sql")
	resetGroupUnreadSQL                   = mustSQL("reset_group_unread.sql")
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
)
