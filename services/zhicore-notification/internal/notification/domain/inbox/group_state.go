package inbox

type GroupState struct {
	RecipientID  UserID
	GroupKey     string
	TotalCount   int64
	UnreadCount  int64
	RepairSignal bool
}

func (g GroupState) ValidUnreadCount() bool {
	return g.TotalCount >= 0 && g.UnreadCount >= 0 && g.UnreadCount <= g.TotalCount
}
