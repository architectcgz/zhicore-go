package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/ports"
)

const (
	defaultOutboxBatchSize    = 50
	defaultOutboxMaxAttempts  = 5
	defaultOutboxRetryBackoff = time.Minute
	defaultOutboxStaleAfter   = 5 * time.Minute
)

type OutboxDispatcherConfig struct {
	DispatcherID string
	BatchSize    int
	MaxAttempts  int
	RetryBackoff time.Duration
	StaleAfter   time.Duration
	Repository   ports.OutboxDispatchRepository
	Publisher    ports.IntegrationEventPublisher
	Clock        ports.Clock
}

type OutboxDispatcher struct {
	dispatcherID string
	batchSize    int
	maxAttempts  int
	retryBackoff time.Duration
	staleAfter   time.Duration
	repository   ports.OutboxDispatchRepository
	publisher    ports.IntegrationEventPublisher
	clock        ports.Clock
}

type OutboxDispatchResult struct {
	Claimed   int
	Published int
	Failed    int
	Dead      int
}

func NewOutboxDispatcher(config OutboxDispatcherConfig) (*OutboxDispatcher, error) {
	if strings.TrimSpace(config.DispatcherID) == "" {
		return nil, errors.New("outbox dispatcher id is required")
	}
	if config.Repository == nil {
		return nil, errors.New("outbox dispatch repository is required")
	}
	if config.Publisher == nil {
		return nil, errors.New("integration event publisher is required")
	}
	if config.Clock == nil {
		return nil, errors.New("clock is required")
	}
	if config.BatchSize <= 0 {
		config.BatchSize = defaultOutboxBatchSize
	}
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = defaultOutboxMaxAttempts
	}
	if config.RetryBackoff <= 0 {
		config.RetryBackoff = defaultOutboxRetryBackoff
	}
	if config.StaleAfter <= 0 {
		config.StaleAfter = defaultOutboxStaleAfter
	}
	return &OutboxDispatcher{
		dispatcherID: strings.TrimSpace(config.DispatcherID),
		batchSize:    config.BatchSize,
		maxAttempts:  config.MaxAttempts,
		retryBackoff: config.RetryBackoff,
		staleAfter:   config.StaleAfter,
		repository:   config.Repository,
		publisher:    config.Publisher,
		clock:        config.Clock,
	}, nil
}

func (d *OutboxDispatcher) DispatchOnce(ctx context.Context) (OutboxDispatchResult, error) {
	now := d.clock.Now()
	events, err := d.repository.ClaimPendingOutbox(ctx, ports.OutboxClaimOptions{
		DispatcherID: d.dispatcherID,
		BatchSize:    d.batchSize,
		StaleAfter:   d.staleAfter,
		Now:          now,
	})
	if err != nil {
		return OutboxDispatchResult{}, fmt.Errorf("claim comment outbox events: %w", err)
	}

	result := OutboxDispatchResult{Claimed: len(events)}
	for _, event := range events {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if err := d.publisher.PublishIntegrationEvent(ctx, event); err != nil {
			failure := d.failure(event, err, d.clock.Now())
			if markErr := d.repository.MarkOutboxFailed(ctx, failure); markErr != nil {
				return result, fmt.Errorf("mark comment outbox failed: %w", markErr)
			}
			if failure.Dead {
				result.Dead++
			} else {
				result.Failed++
			}
			continue
		}
		if err := d.repository.MarkOutboxPublished(ctx, ports.OutboxPublished{
			ID:           event.ID,
			DispatcherID: d.dispatcherID,
			PublishedAt:  d.clock.Now(),
		}); err != nil {
			return result, fmt.Errorf("mark comment outbox published: %w", err)
		}
		result.Published++
	}
	return result, nil
}

func (d *OutboxDispatcher) Run(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		interval = time.Second
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := d.DispatchOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			return err
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (d *OutboxDispatcher) failure(event ports.OutboxEvent, publishErr error, failedAt time.Time) ports.OutboxFailure {
	attempts := event.AttemptCount + 1
	failure := ports.OutboxFailure{
		ID:           event.ID,
		DispatcherID: d.dispatcherID,
		AttemptCount: attempts,
		Dead:         attempts >= d.maxAttempts,
		LastError:    publishErr.Error(),
		FailedAt:     failedAt,
	}
	if !failure.Dead {
		nextRetryAt := failedAt.Add(d.retryBackoff)
		failure.NextRetryAt = &nextRetryAt
	}
	return failure
}
