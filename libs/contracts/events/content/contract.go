package content

import "time"

// PostPublishedPayload is the version 1 payload for content.post.published.
// The Content application fills this provider-owned contract before writing
// the outbox event so consumers and producer tests share one payload shape.
type PostPublishedPayload struct {
	PublicID          string         `json:"publicId"`
	InternalID        int64          `json:"internalId"`
	AuthorID          int64          `json:"authorId"`
	Author            AuthorSnapshot `json:"author"`
	Title             string         `json:"title"`
	Summary           string         `json:"summary,omitempty"`
	CoverFileID       string         `json:"coverFileId,omitempty"`
	PublishedAt       time.Time      `json:"publishedAt"`
	PublishedBodyID   string         `json:"publishedBodyId,omitempty"`
	PublishedBodyHash string         `json:"publishedBodyHash,omitempty"`
}

// AuthorSnapshot makes follower notifications independently renderable without
// resolving User for every inbox row at notification-list read time.
type AuthorSnapshot struct {
	PublicID    string `json:"publicId"`
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl,omitempty"`
}

// PostVisibilityChangedPayload is the version 1 payload for
// content.post.visibility_changed. It captures lifecycle visibility facts
// without forcing consumers to infer public visibility from Content status.
type PostVisibilityChangedPayload struct {
	PublicID      string    `json:"publicId"`
	InternalID    int64     `json:"internalId"`
	AuthorID      int64     `json:"authorId,omitempty"`
	OldVisibility string    `json:"oldVisibility"`
	NewVisibility string    `json:"newVisibility"`
	PublicVisible bool      `json:"publicVisible"`
	Reason        string    `json:"reason"`
	ChangedAt     time.Time `json:"changedAt"`
}
