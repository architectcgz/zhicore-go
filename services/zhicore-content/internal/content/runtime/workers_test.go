package runtime

import (
	"context"
	"errors"
	"testing"
)

func TestContentWorkersReturnsEnabledDescriptors(t *testing.T) {
	deps := validDeps(t)
	deps.Config.Workers.CleanupEnabled = true
	deps.Config.Workers.RepairEnabled = true
	deps.Config.Workers.OutboxEnabled = true

	module, err := Build(deps)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	cleanup := findWorkerDescriptor(module.Workers, "content-body-cleanup")
	if cleanup == nil || !cleanup.Enabled || cleanup.DisabledReason != "" || cleanup.Checker == nil || cleanup.Runner == nil {
		t.Fatalf("cleanup worker = %#v, want enabled descriptor with checker and runner", cleanup)
	}
	repair := findWorkerDescriptor(module.Workers, "content-body-repair")
	if repair == nil || !repair.Enabled || repair.DisabledReason != "" || repair.Checker == nil || repair.Runner == nil {
		t.Fatalf("repair worker = %#v, want enabled descriptor with checker and runner", repair)
	}
	outbox := findWorkerDescriptor(module.Workers, "content-outbox-dispatcher")
	if outbox == nil || !outbox.Enabled || outbox.DisabledReason != "" || outbox.Checker == nil || outbox.Runner == nil {
		t.Fatalf("outbox worker = %#v, want enabled descriptor with checker and runner", outbox)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := cleanup.Runner.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("cleanup Run(canceled) error = %v, want context.Canceled", err)
	}
	if err := repair.Runner.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("repair Run(canceled) error = %v, want context.Canceled", err)
	}
	if err := outbox.Runner.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("outbox Run(canceled) error = %v, want context.Canceled", err)
	}
}

func findWorkerDescriptor(workers []WorkerDescriptor, name string) *WorkerDescriptor {
	for i := range workers {
		if workers[i].Name == name {
			return &workers[i]
		}
	}
	return nil
}
