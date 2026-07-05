package ports

import (
	"context"
	"time"
)

type PostContentStore interface {
	WriteDraftBody(ctx context.Context, input WriteBodyInput) (StoredBody, error)
	WriteSnapshotBody(ctx context.Context, input WriteBodyInput) (StoredBody, error)
	ReadBody(ctx context.Context, bodyID string) (StoredBody, error)
	DeleteBody(ctx context.Context, bodyID string) error
}

type WriteBodyInput struct {
	PostPublicID  string
	OwnerID       int64
	SchemaVersion int
	Blocks        Blocks
	CanonicalJSON []byte
	PlainText     string
	ContentHash   string
	SizeBytes     int
	BlockCount    int
	CreatedAt     time.Time
}

type StoredBody struct {
	ID            string
	SchemaVersion int
	Blocks        Blocks
	CanonicalJSON []byte
	PlainText     string
	ContentHash   string
	SizeBytes     int
	BlockCount    int
	CreatedAt     time.Time
}
