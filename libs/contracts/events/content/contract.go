package content

import "time"

// PostPublishedPayload is the version 1 payload for content.post.published.
// The Content application fills this provider-owned contract before writing
// the outbox event so consumers and producer tests share one payload shape.
type PostPublishedPayload struct {
	PublicID          string    `json:"publicId"`
	InternalID        int64     `json:"internalId"`
	AuthorID          int64     `json:"authorId"`
	Title             string    `json:"title"`
	Summary           string    `json:"summary,omitempty"`
	CoverFileID       string    `json:"coverFileId,omitempty"`
	PublishedAt       time.Time `json:"publishedAt"`
	PublishedBodyID   string    `json:"publishedBodyId,omitempty"`
	PublishedBodyHash string    `json:"publishedBodyHash,omitempty"`
}
