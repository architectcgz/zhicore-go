package application

import (
	"time"
)

type PostBodyInput struct {
	SchemaVersion int
	Blocks        Blocks
}

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

type CreatePostResult struct {
	PostID      string
	PostVersion int64
}

type SaveDraftBodyCommand struct {
	Actor             *Actor
	PostID            string
	BasePostVersion   int64
	BaseDraftBodyID   string
	BaseDraftBodyHash string
	Body              PostBodyInput
}

type SaveDraftBodyResult struct {
	PostID        string
	PostVersion   int64
	DraftBodyID   string
	DraftBodyHash string
	SavedAt       time.Time
	WordCount     int
}

type PublishPostCommand struct {
	Actor           *Actor
	PostID          string
	BasePostVersion int64
	DraftBodyID     string
	DraftBodyHash   string
}

type PublishPostResult struct {
	PostID      string
	PostVersion int64
	PublishedAt time.Time
}

type PostLifecycleCommand struct {
	Actor           *Actor
	PostID          string
	BasePostVersion int64
}

type PostLifecycleResult struct {
	PostID      string
	PostVersion int64
	Status      string
	UpdatedAt   time.Time
}

type SchedulePostCommand struct {
	Actor           *Actor
	PostID          string
	BasePostVersion int64
	DraftBodyID     string
	DraftBodyHash   string
	ScheduledAt     time.Time
}

type SchedulePostResult struct {
	PostID      string
	PostVersion int64
	Status      string
	ScheduledAt time.Time
}

type GetPublishedPostBodyQuery struct {
	PostID string
}

type GetPublishedPostBodyResult struct {
	BodyID        string
	SchemaVersion int
	CanonicalJSON []byte
	PlainText     string
	ContentHash   string
	SizeBytes     int
	CreatedAt     time.Time
}
