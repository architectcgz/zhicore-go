package application

type MarkNotificationReadCommand struct {
	Actor          Actor
	NotificationID string
}

type MarkAllNotificationsReadCommand struct {
	Actor Actor
}
