package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/architectcgz/zhicore-go/libs/kit/taskworker"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

type BodyRepairWorker struct {
	runner *taskworker.Runner[ports.BodyRepairTaskClaim]
}

type BodyRepairWorkerDeps struct {
	Tasks ports.BodyRepairTaskStore
	Clock ports.Clock
}

type BodyRepairWorkerConfig struct {
	WorkerID        string
	BatchSize       int
	StaleClaimAfter time.Duration
	RetryBackoff    time.Duration
	DeadThreshold   int
}

func NewBodyRepairWorker(deps BodyRepairWorkerDeps, config BodyRepairWorkerConfig) *BodyRepairWorker {
	config = normalizeRepairWorkerConfig(config)
	return &BodyRepairWorker{
		runner: taskworker.NewRunner[ports.BodyRepairTaskClaim](
			repairTaskStoreAdapter{tasks: deps.Tasks},
			taskworker.HandlerFunc[ports.BodyRepairTaskClaim](handleRepairTask),
			deps.Clock,
			taskworker.Config{
				WorkerID:        config.WorkerID,
				BatchSize:       config.BatchSize,
				StaleClaimAfter: config.StaleClaimAfter,
				RetryBackoff:    config.RetryBackoff,
				DeadThreshold:   config.DeadThreshold,
			},
		),
	}
}

func (w *BodyRepairWorker) RunUntilIdle(ctx context.Context) error {
	return w.runner.RunUntilIdle(ctx)
}

func handleRepairTask(ctx context.Context, task ports.BodyRepairTaskClaim) error {
	if strings.TrimSpace(task.BodyID) == "" {
		return fmt.Errorf("repair task body id is empty")
	}
	if strings.TrimSpace(task.TaskType) == "" {
		return fmt.Errorf("repair task type is empty for body %s", task.BodyID)
	}
	// First-stage repair workers never invent a replacement body or read draft
	// content as a substitute for the published pointer. They persist a clear
	// manual-repair reason so admin tooling can surface the data incident.
	return fmt.Errorf(
		"manual repair required: taskType=%s postID=%d bodyID=%s expectedHash=%s observedHash=%s",
		task.TaskType,
		task.PostID,
		task.BodyID,
		task.ExpectedHash,
		task.ObservedHash,
	)
}

func normalizeRepairWorkerConfig(config BodyRepairWorkerConfig) BodyRepairWorkerConfig {
	if strings.TrimSpace(config.WorkerID) == "" {
		config.WorkerID = "content-body-repair"
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
		config.DeadThreshold = 1
	}
	return config
}

type repairTaskStoreAdapter struct {
	tasks ports.BodyRepairTaskStore
}

func (a repairTaskStoreAdapter) Claim(ctx context.Context, options taskworker.ClaimOptions) ([]ports.BodyRepairTaskClaim, error) {
	return a.tasks.Claim(ctx, ports.TaskClaimRequest{
		WorkerID:    options.WorkerID,
		Limit:       options.BatchSize,
		Now:         options.Now,
		StaleBefore: options.StaleBefore,
	})
}

func (a repairTaskStoreAdapter) MarkSucceeded(ctx context.Context, task ports.BodyRepairTaskClaim, success taskworker.Success) error {
	return a.tasks.MarkSucceeded(ctx, task.ID, success.WorkerID, success.CompletedAt)
}

func (a repairTaskStoreAdapter) MarkFailed(ctx context.Context, task ports.BodyRepairTaskClaim, failure taskworker.Failure) error {
	return a.tasks.MarkFailed(ctx, ports.TaskFailure{
		TaskID:        task.ID,
		WorkerID:      failure.WorkerID,
		Error:         failure.Error,
		NextRetryAt:   failure.NextRetryAt,
		DeadThreshold: failure.DeadThreshold,
		Now:           failure.FailedAt,
	})
}
