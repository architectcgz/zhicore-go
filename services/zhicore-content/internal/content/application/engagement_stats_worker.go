package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/architectcgz/zhicore-go/libs/kit/taskworker"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

type EngagementStatsWorker struct {
	tasks    ports.EngagementStatsTaskStore
	clock    ports.Clock
	workerID string
	runner   *taskworker.Runner[ports.EngagementStatsDeltaClaim]
}

type EngagementStatsWorkerDeps struct {
	Tasks ports.EngagementStatsTaskStore
	Clock ports.Clock
}

type EngagementStatsWorkerConfig struct {
	WorkerID        string
	BatchSize       int
	StaleClaimAfter time.Duration
	RetryBackoff    time.Duration
	DeadThreshold   int
}

func NewEngagementStatsWorker(deps EngagementStatsWorkerDeps, config EngagementStatsWorkerConfig) *EngagementStatsWorker {
	config = normalizeEngagementStatsWorkerConfig(config)
	worker := &EngagementStatsWorker{
		tasks:    deps.Tasks,
		clock:    deps.Clock,
		workerID: config.WorkerID,
	}
	worker.runner = taskworker.NewRunner[ports.EngagementStatsDeltaClaim](
		engagementStatsTaskStoreAdapter{tasks: deps.Tasks},
		taskworker.HandlerFunc[ports.EngagementStatsDeltaClaim](worker.handle),
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

func (w *EngagementStatsWorker) RunUntilIdle(ctx context.Context) error {
	return w.runner.RunUntilIdle(ctx)
}

func (w *EngagementStatsWorker) handle(ctx context.Context, task ports.EngagementStatsDeltaClaim) error {
	// Applying the stats delta and marking the internal task DONE are one
	// database transaction. Keeping this in the handler lets taskworker route
	// transient apply failures through the normal retry/dead-letter path.
	if err := w.tasks.ApplyClaimed(ctx, task, w.workerID, w.clock.Now()); err != nil {
		if errors.Is(err, ports.ErrTaskClaimLost) {
			return nil
		}
		return err
	}
	return nil
}

func normalizeEngagementStatsWorkerConfig(config EngagementStatsWorkerConfig) EngagementStatsWorkerConfig {
	if strings.TrimSpace(config.WorkerID) == "" {
		config.WorkerID = "content-engagement-stats"
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
		config.DeadThreshold = 5
	}
	return config
}

type engagementStatsTaskStoreAdapter struct {
	tasks ports.EngagementStatsTaskStore
}

func (a engagementStatsTaskStoreAdapter) Claim(ctx context.Context, options taskworker.ClaimOptions) ([]ports.EngagementStatsDeltaClaim, error) {
	return a.tasks.Claim(ctx, ports.TaskClaimRequest{
		WorkerID:    options.WorkerID,
		Limit:       options.BatchSize,
		Now:         options.Now,
		StaleBefore: options.StaleBefore,
	})
}

func (a engagementStatsTaskStoreAdapter) MarkSucceeded(ctx context.Context, task ports.EngagementStatsDeltaClaim, success taskworker.Success) error {
	return nil
}

func (a engagementStatsTaskStoreAdapter) MarkFailed(ctx context.Context, task ports.EngagementStatsDeltaClaim, failure taskworker.Failure) error {
	return a.tasks.MarkFailed(ctx, ports.TaskFailure{
		TaskID:        task.ID,
		WorkerID:      failure.WorkerID,
		Error:         failure.Error,
		NextRetryAt:   failure.NextRetryAt,
		DeadThreshold: failure.DeadThreshold,
		Now:           failure.FailedAt,
	})
}
