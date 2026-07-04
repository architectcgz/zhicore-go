package ports

import (
	"context"
	"time"
)

type OutboxMessage struct {
	EventType  string
	OccurredAt time.Time
	Payload    []byte
}

type OutboxPublisher interface {
	Publish(ctx context.Context, message OutboxMessage) error
}
