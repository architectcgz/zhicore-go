package application

type MarkNotificationReadCommand struct {
	Actor          Actor
	NotificationID string
}

type MarkAllNotificationsReadCommand struct {
	Actor Actor
}

type MarkNotificationGroupReadCommand struct {
	Actor   Actor
	GroupID string
}
