package ports

import (
	"context"
	"time"
)

type OutboxPublisher interface {
	Append(ctx context.Context, tx Tx, event OutboxEvent) error
}

type IntegrationEventPublisher interface {
	PublishIntegrationEvent(ctx context.Context, event OutboxEvent) error
}

type OutboxEvent struct {
	ID               int64
	EventID          string
	EventType        string
	PayloadVersion   int
	AggregateType    string
	AggregateID      string
	AggregateVersion int64
	PayloadJSON      []byte
	OccurredAt       time.Time
	AttemptCount     int
}

type OutboxClaimOptions struct {
	DispatcherID string
	BatchSize    int
	StaleAfter   time.Duration
	Now          time.Time
}

type OutboxPublished struct {
	ID           int64
	DispatcherID string
	PublishedAt  time.Time
}

type OutboxFailure struct {
	ID           int64
	DispatcherID string
	AttemptCount int
	NextRetryAt  *time.Time
	Dead         bool
	LastError    string
	FailedAt     time.Time
}

type OutboxDispatchRepository interface {
	ClaimPendingOutbox(ctx context.Context, options OutboxClaimOptions) ([]OutboxEvent, error)
	MarkOutboxPublished(ctx context.Context, published OutboxPublished) error
	MarkOutboxFailed(ctx context.Context, failure OutboxFailure) error
}
