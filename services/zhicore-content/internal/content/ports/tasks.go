package ports

import (
	"context"
	"time"
)

type BodyCleanupTaskStore interface {
	Append(ctx context.Context, tx Tx, task BodyCleanupTask) error
	AppendOutsideTx(ctx context.Context, task BodyCleanupTask) error
}

type BodyRepairTaskStore interface {
	Append(ctx context.Context, tx Tx, task BodyRepairTask) error
	AppendOutsideTx(ctx context.Context, task BodyRepairTask) error
}

type BodyCleanupTask struct {
	PostID    int64
	BodyID    string
	TaskType  string
	Reason    string
	CreatedAt time.Time
}

type BodyRepairTask struct {
	PostID       int64
	BodyID       string
	TaskType     string
	ExpectedHash string
	ObservedHash string
	CreatedAt    time.Time
}
