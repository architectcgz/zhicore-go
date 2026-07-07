package application

import "time"

type CreatePostResult struct {
	PostID      string
	PostVersion int64
}

type SaveDraftBodyResult struct {
	PostID        string
	PostVersion   int64
	DraftBodyID   string
	DraftBodyHash string
	SavedAt       time.Time
	WordCount     int
}

type PublishPostResult struct {
	PostID      string
	PostVersion int64
	PublishedAt time.Time
}

type PostLifecycleResult struct {
	PostID      string
	PostVersion int64
	Status      string
	UpdatedAt   time.Time
}

type SchedulePostResult struct {
	PostID      string
	PostVersion int64
	Status      string
	ScheduledAt time.Time
}
