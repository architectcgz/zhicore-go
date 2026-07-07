package application

import "time"

type PostBodyInput struct {
	SchemaVersion int
	Blocks        Blocks
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
