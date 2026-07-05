package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/architectcgz/zhicore-go/libs/kit/taskworker"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

type OutboxDispatcher struct {
	publisher ports.IntegrationEventPublisher
	runner    *taskworker.Runner[ports.OutboxEvent]
}

type OutboxDispatcherDeps struct {
	Repository ports.OutboxDispatchRepository
	Publisher  ports.IntegrationEventPublisher
	Clock      ports.Clock
}

type OutboxDispatcherConfig struct {
	DispatcherID    string
	BatchSize       int
	StaleClaimAfter time.Duration
	RetryBackoff    time.Duration
	DeadThreshold   int
}

func NewOutboxDispatcher(deps OutboxDispatcherDeps, config OutboxDispatcherConfig) *OutboxDispatcher {
	config = normalizeOutboxDispatcherConfig(config)
	dispatcher := &OutboxDispatcher{publisher: deps.Publisher}
	dispatcher.runner = taskworker.NewRunner[ports.OutboxEvent](
		outboxDispatchStoreAdapter{repository: deps.Repository, staleClaimAfter: config.StaleClaimAfter},
		taskworker.HandlerFunc[ports.OutboxEvent](dispatcher.publish),
		deps.Clock,
		taskworker.Config{
			WorkerID:        config.DispatcherID,
			BatchSize:       config.BatchSize,
			StaleClaimAfter: config.StaleClaimAfter,
			RetryBackoff:    config.RetryBackoff,
			DeadThreshold:   config.DeadThreshold,
		},
	)
	return dispatcher
}

func (d *OutboxDispatcher) RunUntilIdle(ctx context.Context) error {
	return d.runner.RunUntilIdle(ctx)
}

func (d *OutboxDispatcher) publish(ctx context.Context, event ports.OutboxEvent) error {
	if d.publisher == nil {
		return fmt.Errorf("integration event publisher is required")
	}
	return d.publisher.PublishIntegrationEvent(ctx, event)
}

func normalizeOutboxDispatcherConfig(config OutboxDispatcherConfig) OutboxDispatcherConfig {
	if strings.TrimSpace(config.DispatcherID) == "" {
		config.DispatcherID = "zhicore-content:outbox-dispatcher"
	} else {
		config.DispatcherID = strings.TrimSpace(config.DispatcherID)
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 50
	}
	if config.StaleClaimAfter <= 0 {
		config.StaleClaimAfter = 5 * time.Minute
	}
	if config.RetryBackoff <= 0 {
		config.RetryBackoff = time.Minute
	}
	if config.DeadThreshold <= 0 {
		config.DeadThreshold = 5
	}
	return config
}

type outboxDispatchStoreAdapter struct {
	repository      ports.OutboxDispatchRepository
	staleClaimAfter time.Duration
}

func (a outboxDispatchStoreAdapter) Claim(ctx context.Context, options taskworker.ClaimOptions) ([]ports.OutboxEvent, error) {
	return a.repository.ClaimPendingOutbox(ctx, ports.OutboxClaimOptions{
		DispatcherID: options.WorkerID,
		BatchSize:    options.BatchSize,
		StaleAfter:   a.staleClaimAfter,
		Now:          options.Now,
	})
}

func (a outboxDispatchStoreAdapter) MarkSucceeded(ctx context.Context, event ports.OutboxEvent, success taskworker.Success) error {
	return a.repository.MarkOutboxPublished(ctx, ports.OutboxPublished{
		ID:           event.ID,
		DispatcherID: success.WorkerID,
		PublishedAt:  success.CompletedAt,
	})
}

func (a outboxDispatchStoreAdapter) MarkFailed(ctx context.Context, event ports.OutboxEvent, failure taskworker.Failure) error {
	attempts := event.AttemptCount + 1
	dead := attempts >= failure.DeadThreshold
	outboxFailure := ports.OutboxFailure{
		ID:           event.ID,
		DispatcherID: failure.WorkerID,
		AttemptCount: attempts,
		Dead:         dead,
		LastError:    failure.Error,
		FailedAt:     failure.FailedAt,
	}
	if !dead {
		// Retry scheduling belongs with the dispatcher because RabbitMQ publish
		// succeeded or failed outside the database transaction that claimed the row.
		outboxFailure.NextRetryAt = &failure.NextRetryAt
	}
	return a.repository.MarkOutboxFailed(ctx, outboxFailure)
}
