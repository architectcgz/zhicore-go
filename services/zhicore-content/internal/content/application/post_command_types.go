package application

import "time"

type CreatePostCommand struct {
	Actor       *Actor
	Title       string
	Summary     string
	CoverFileID string
	TopicID     string
	CategoryID  string
	Tags        []string
	Body        *PostBodyInput
}

type SaveDraftBodyCommand struct {
	Actor             *Actor
	PostID            string
	BasePostVersion   int64
	BaseDraftBodyID   string
	BaseDraftBodyHash string
	Body              PostBodyInput
}

type PublishPostCommand struct {
	Actor           *Actor
	PostID          string
	BasePostVersion int64
	DraftBodyID     string
	DraftBodyHash   string
}

type PostLifecycleCommand struct {
	Actor           *Actor
	PostID          string
	BasePostVersion int64
}

type SchedulePostCommand struct {
	Actor           *Actor
	PostID          string
	BasePostVersion int64
	DraftBodyID     string
	DraftBodyHash   string
	ScheduledAt     time.Time
}
