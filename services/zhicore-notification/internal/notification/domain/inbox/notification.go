package inbox

import "time"

type Notification struct {
	ID          NotificationID
	PublicID    NotificationPublicID
	RecipientID UserID
	IsRead      bool
	ReadAt      time.Time
}

func (n Notification) AssertRecipient(userID UserID) bool {
	return n.RecipientID == userID
}
