package ports

import (
	"context"
	"time"
)

type OutboxPublisher interface {
	Append(ctx context.Context, tx Tx, event OutboxEvent) error
}

type OutboxEvent struct {
	EventType        string
	PayloadVersion   int
	AggregateType    string
	AggregateID      string
	AggregateVersion int64
	PayloadJSON      []byte
	OccurredAt       time.Time
}
