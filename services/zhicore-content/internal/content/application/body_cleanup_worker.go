package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/architectcgz/zhicore-go/libs/kit/taskworker"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

type BodyCleanupWorker struct {
	bodies     ports.PostContentStore
	references ports.BodyReferenceChecker
	runner     *taskworker.Runner[ports.BodyCleanupTaskClaim]
}

type BodyCleanupWorkerDeps struct {
	Tasks      ports.BodyCleanupTaskStore
	Bodies     ports.PostContentStore
	References ports.BodyReferenceChecker
	Clock      ports.Clock
}

type BodyCleanupWorkerConfig struct {
	WorkerID        string
	BatchSize       int
	StaleClaimAfter time.Duration
	RetryBackoff    time.Duration
	DeadThreshold   int
}

func NewBodyCleanupWorker(deps BodyCleanupWorkerDeps, config BodyCleanupWorkerConfig) *BodyCleanupWorker {
	config = normalizeCleanupWorkerConfig(config)
	worker := &BodyCleanupWorker{
		bodies:     deps.Bodies,
		references: deps.References,
	}
	worker.runner = taskworker.NewRunner[ports.BodyCleanupTaskClaim](
		cleanupTaskStoreAdapter{tasks: deps.Tasks},
		taskworker.HandlerFunc[ports.BodyCleanupTaskClaim](worker.handle),
		deps.Clock,
		taskworker.Config{
			WorkerID:        config.WorkerID,
			BatchSize:       config.BatchSize,
			StaleClaimAfter: config.StaleClaimAfter,
			RetryBackoff:    config.RetryBackoff,
			DeadThreshold:   config.DeadThreshold,
		},
	)
	return worker
}

func (w *BodyCleanupWorker) RunUntilIdle(ctx context.Context) error {
	return w.runner.RunUntilIdle(ctx)
}

func (w *BodyCleanupWorker) handle(ctx context.Context, task ports.BodyCleanupTaskClaim) error {
	if strings.TrimSpace(task.BodyID) == "" {
		return fmt.Errorf("cleanup task body id is empty")
	}

	referenced, err := w.references.IsBodyReferenced(ctx, task.BodyID)
	if err != nil {
		return fmt.Errorf("check body reference failed: %w", err)
	}
	if referenced {
		// PostgreSQL is the visibility source of truth. A claimed cleanup task is
		// retried instead of deleted whenever draft/published pointers still
		// reference the body, preventing accidental removal of live content.
		return fmt.Errorf("body still referenced by post pointer")
	}

	if err := w.bodies.DeleteBody(ctx, task.BodyID); err != nil {
		return fmt.Errorf("delete body failed: %w", err)
	}
	return nil
}

func normalizeCleanupWorkerConfig(config BodyCleanupWorkerConfig) BodyCleanupWorkerConfig {
	if strings.TrimSpace(config.WorkerID) == "" {
		config.WorkerID = "content-body-cleanup"
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
	return config
}

type cleanupTaskStoreAdapter struct {
	tasks ports.BodyCleanupTaskStore
}

func (a cleanupTaskStoreAdapter) Claim(ctx context.Context, options taskworker.ClaimOptions) ([]ports.BodyCleanupTaskClaim, error) {
	return a.tasks.Claim(ctx, ports.TaskClaimRequest{
		WorkerID:    options.WorkerID,
		Limit:       options.BatchSize,
		Now:         options.Now,
		StaleBefore: options.StaleBefore,
	})
}

func (a cleanupTaskStoreAdapter) MarkSucceeded(ctx context.Context, task ports.BodyCleanupTaskClaim, success taskworker.Success) error {
	return a.tasks.MarkSucceeded(ctx, task.ID, success.WorkerID, success.CompletedAt)
}

func (a cleanupTaskStoreAdapter) MarkFailed(ctx context.Context, task ports.BodyCleanupTaskClaim, failure taskworker.Failure) error {
	return a.tasks.MarkFailed(ctx, ports.TaskFailure{
		TaskID:        task.ID,
		WorkerID:      failure.WorkerID,
		Error:         failure.Error,
		NextRetryAt:   failure.NextRetryAt,
		DeadThreshold: failure.DeadThreshold,
		Now:           failure.FailedAt,
	})
}
