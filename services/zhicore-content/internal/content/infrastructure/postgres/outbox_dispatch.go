package postgres

import (
	"context"
	"database/sql"
	"errors"

	kitoutbox "github.com/architectcgz/zhicore-go/libs/kit/postgres/outbox"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

var ErrOutboxClaimLost = kitoutbox.ErrClaimLost

type OutboxDispatchRepository struct {
	repo *kitoutbox.DispatchRepository
}

func NewOutboxDispatchRepository(db *sql.DB) *OutboxDispatchRepository {
	return &OutboxDispatchRepository{
		repo: kitoutbox.NewDispatchRepository(db, kitoutbox.Config{Table: "outbox_events"}),
	}
}

func (r *OutboxDispatchRepository) ClaimPendingOutbox(ctx context.Context, options ports.OutboxClaimOptions) ([]ports.OutboxEvent, error) {
	events, err := r.repo.ClaimPending(ctx, kitoutbox.ClaimOptions{
		DispatcherID: options.DispatcherID,
		BatchSize:    options.BatchSize,
		StaleAfter:   options.StaleAfter,
		Now:          options.Now,
	})
	if err != nil {
		return nil, err
	}
	result := make([]ports.OutboxEvent, 0, len(events))
	for _, event := range events {
		result = append(result, ports.OutboxEvent{
			ID:             event.ID,
			EventID:        event.EventID,
			EventType:      event.EventType,
			PayloadVersion: event.PayloadVersion,
			AggregateType:  event.AggregateType,
			AggregateID:    event.AggregateID,
			PayloadJSON:    event.Payload,
			OccurredAt:     event.OccurredAt,
			AttemptCount:   event.AttemptCount,
		})
	}
	return result, nil
}

func (r *OutboxDispatchRepository) MarkOutboxPublished(ctx context.Context, published ports.OutboxPublished) error {
	return mapOutboxError(r.repo.MarkPublished(ctx, kitoutbox.Published{
		ID:           published.ID,
		DispatcherID: published.DispatcherID,
		PublishedAt:  published.PublishedAt,
	}))
}

func (r *OutboxDispatchRepository) MarkOutboxFailed(ctx context.Context, failure ports.OutboxFailure) error {
	return mapOutboxError(r.repo.MarkFailed(ctx, kitoutbox.Failure{
		ID:           failure.ID,
		DispatcherID: failure.DispatcherID,
		AttemptCount: failure.AttemptCount,
		NextRetryAt:  failure.NextRetryAt,
		Dead:         failure.Dead,
		LastError:    failure.LastError,
		FailedAt:     failure.FailedAt,
	}))
}

func mapOutboxError(err error) error {
	if errors.Is(err, kitoutbox.ErrClaimLost) {
		return ErrOutboxClaimLost
	}
	return err
}

var _ ports.OutboxDispatchRepository = (*OutboxDispatchRepository)(nil)
