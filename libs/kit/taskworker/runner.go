// Package taskworker contains reusable claim-process-ack worker control flow.
package taskworker

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Clock interface {
	Now() time.Time
}

type ClaimOptions struct {
	WorkerID    string
	BatchSize   int
	StaleBefore time.Time
	Now         time.Time
}

type Failure struct {
	WorkerID      string
	Error         string
	NextRetryAt   time.Time
	DeadThreshold int
	FailedAt      time.Time
}

type Success struct {
	WorkerID    string
	CompletedAt time.Time
}

type Store[T any] interface {
	Claim(context.Context, ClaimOptions) ([]T, error)
	MarkSucceeded(context.Context, T, Success) error
	MarkFailed(context.Context, T, Failure) error
}

type Handler[T any] interface {
	Handle(context.Context, T) error
}

type HandlerFunc[T any] func(context.Context, T) error

func (f HandlerFunc[T]) Handle(ctx context.Context, task T) error {
	return f(ctx, task)
}

type Config struct {
	WorkerID        string
	BatchSize       int
	StaleClaimAfter time.Duration
	RetryBackoff    time.Duration
	DeadThreshold   int
	MarkTimeout     time.Duration
}

type Runner[T any] struct {
	store   Store[T]
	handler Handler[T]
	clock   Clock
	config  Config
}

func NewRunner[T any](store Store[T], handler Handler[T], clock Clock, config Config) *Runner[T] {
	return &Runner[T]{
		store:   store,
		handler: handler,
		clock:   clock,
		config:  normalizeConfig(config),
	}
}

func (r *Runner[T]) RunUntilIdle(ctx context.Context) error {
	if err := r.validate(); err != nil {
		return err
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		now := r.clock.Now()
		tasks, err := r.store.Claim(ctx, ClaimOptions{
			WorkerID:    r.config.WorkerID,
			BatchSize:   r.config.BatchSize,
			Now:         now,
			StaleBefore: now.Add(-r.config.StaleClaimAfter),
		})
		if err != nil {
			return fmt.Errorf("claim tasks: %w", err)
		}
		if len(tasks) == 0 {
			return nil
		}
		for _, task := range tasks {
			if err := r.handler.Handle(ctx, task); err != nil {
				if markErr := r.markFailed(ctx, task, err); markErr != nil {
					return markErr
				}
				if err := ctx.Err(); err != nil {
					return err
				}
				continue
			}
			markCtx, cancel := r.markContext(ctx)
			err := r.store.MarkSucceeded(markCtx, task, Success{
				WorkerID:    r.config.WorkerID,
				CompletedAt: r.clock.Now(),
			})
			cancel()
			if err != nil {
				return fmt.Errorf("mark task succeeded: %w", err)
			}
			if err := ctx.Err(); err != nil {
				return err
			}
		}
	}
}

func (r *Runner[T]) markFailed(ctx context.Context, task T, taskErr error) error {
	now := r.clock.Now()
	markCtx, cancel := r.markContext(ctx)
	defer cancel()
	if err := r.store.MarkFailed(markCtx, task, Failure{
		WorkerID:      r.config.WorkerID,
		Error:         taskErr.Error(),
		NextRetryAt:   now.Add(r.config.RetryBackoff),
		DeadThreshold: r.config.DeadThreshold,
		FailedAt:      now,
	}); err != nil {
		return fmt.Errorf("mark task failed: %w", err)
	}
	return nil
}

func (r *Runner[T]) markContext(ctx context.Context) (context.Context, context.CancelFunc) {
	// Marking a claimed task is recovery state, not more business work. It keeps
	// request/lifecycle values for observability while ignoring shutdown cancel
	// long enough to persist success or retry metadata within a bounded budget.
	return context.WithTimeout(context.WithoutCancel(ctx), r.config.MarkTimeout)
}

func (r *Runner[T]) validate() error {
	if r.store == nil {
		return fmt.Errorf("taskworker store is required")
	}
	if r.handler == nil {
		return fmt.Errorf("taskworker handler is required")
	}
	if r.clock == nil {
		return fmt.Errorf("taskworker clock is required")
	}
	return nil
}

func normalizeConfig(config Config) Config {
	if strings.TrimSpace(config.WorkerID) == "" {
		config.WorkerID = "taskworker"
	} else {
		config.WorkerID = strings.TrimSpace(config.WorkerID)
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}
	if config.StaleClaimAfter <= 0 {
		config.StaleClaimAfter = 5 * time.Minute
	}
	if config.RetryBackoff <= 0 {
		config.RetryBackoff = time.Minute
	}
	if config.DeadThreshold <= 0 {
		config.DeadThreshold = 3
	}
	if config.MarkTimeout <= 0 {
		config.MarkTimeout = 5 * time.Second
	}
	return config
}
