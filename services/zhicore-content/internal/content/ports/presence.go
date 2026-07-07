package ports

import (
	"context"
	"time"
)

type ReaderPresenceStore interface {
	UpsertReaderSession(ctx context.Context, input ReaderSessionInput) (ReaderPresenceRecord, error)
	DeleteReaderSession(ctx context.Context, input ReaderSessionInput) error
	GetReaderPresence(ctx context.Context, postID string) (ReaderPresenceRecord, error)
}

type ReaderSessionInput struct {
	PostID    string
	SessionID string
	UserID    int64
	TTL       time.Duration
	Now       time.Time
}

type ReaderPresenceRecord struct {
	PostID      string
	OnlineCount int
	TTL         time.Duration
	Degraded    bool
}
