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
)
