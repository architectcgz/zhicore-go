package ports

import (
	"context"
	"errors"
	"time"
)

var ErrOutboxEventNotFound = errors.New("outbox event not found")

type OutboxPublisher interface {
	Append(ctx context.Context, tx Tx, event OutboxEvent) error
}

type IntegrationEventPublisher interface {
	PublishIntegrationEvent(ctx context.Context, event OutboxEvent) error
}

type OutboxAdminRepository interface {
	ListOutboxEvents(ctx context.Context, query OutboxEventQuery) (OutboxEventPage, error)
	RetryOutboxEvent(ctx context.Context, command OutboxRetryCommand) (OutboxRetryResult, error)
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

type OutboxEventQuery struct {
	Status    string
	EventType string
	Page      int
	Size      int
}

type OutboxEventRecord struct {
	EventID          string
	EventType        string
	AggregateType    string
	AggregateID      string
	AggregateVersion int64
	Status           string
	AttemptCount     int
	LastError        string
	OccurredAt       time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type OutboxEventPage struct {
	Items []OutboxEventRecord
	Page  int
	Size  int
	Total int64
}

type OutboxRetryCommand struct {
	EventID     string
	AdminUserID int64
	Reason      string
	RetriedAt   time.Time
}

type OutboxRetryResult struct {
	EventID    string
	Status     string
	RetryCount int
	RetriedAt  time.Time
}
