package taskworker

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRunnerClaimsProcessesAndMarksSucceeded(t *testing.T) {
	now := time.Date(2026, 7, 5, 14, 0, 0, 0, time.UTC)
	store := &fakeStore[int]{claims: [][]int{{10, 11}}}
	var handled []int
	runner := NewRunner[int](store, HandlerFunc[int](func(ctx context.Context, task int) error {
		handled = append(handled, task)
		return nil
	}), fixedClock{now: now}, Config{
		WorkerID:        "worker-a",
		BatchSize:       2,
		StaleClaimAfter: 5 * time.Minute,
		RetryBackoff:    time.Minute,
		DeadThreshold:   3,
	})

	if err := runner.RunUntilIdle(context.Background()); err != nil {
		t.Fatalf("RunUntilIdle() error = %v", err)
	}
	if len(store.claimsSeen) != 2 {
		t.Fatalf("claim calls = %d, want claimed batch then idle", len(store.claimsSeen))
	}
	if got := store.claimsSeen[0]; got.WorkerID != "worker-a" || got.BatchSize != 2 || got.Now != now || got.StaleBefore != now.Add(-5*time.Minute) {
		t.Fatalf("claim options = %+v", got)
	}
	if len(handled) != 2 || handled[0] != 10 || handled[1] != 11 {
		t.Fatalf("handled = %+v, want claimed tasks in order", handled)
	}
	if len(store.succeeded) != 2 || store.succeeded[0].task != 10 || store.succeeded[1].task != 11 {
		t.Fatalf("succeeded = %+v, want both tasks", store.succeeded)
	}
}

func TestRunnerMarksFailedAndContinues(t *testing.T) {
	now := time.Date(2026, 7, 5, 14, 5, 0, 0, time.UTC)
	store := &fakeStore[int]{claims: [][]int{{20, 21}}}
	processErr := errors.New("temporary dependency failure")
	runner := NewRunner[int](store, HandlerFunc[int](func(ctx context.Context, task int) error {
		if task == 20 {
			return processErr
		}
		return nil
	}), fixedClock{now: now}, Config{
		WorkerID:      "worker-b",
		RetryBackoff:  2 * time.Minute,
		DeadThreshold: 4,
	})

	if err := runner.RunUntilIdle(context.Background()); err != nil {
		t.Fatalf("RunUntilIdle() error = %v", err)
	}
	if len(store.failed) != 1 || store.failed[0].task != 20 {
		t.Fatalf("failed = %+v, want task 20", store.failed)
	}
	got := store.failed[0].failure
	if got.WorkerID != "worker-b" || got.Error != processErr.Error() || got.NextRetryAt != now.Add(2*time.Minute) || got.DeadThreshold != 4 || got.FailedAt != now {
		t.Fatalf("failure = %+v, want backoff and threshold", got)
	}
	if len(store.succeeded) != 1 || store.succeeded[0].task != 21 {
		t.Fatalf("succeeded = %+v, want task 21", store.succeeded)
	}
}

func TestRunnerStopsBeforeClaimWhenContextCanceled(t *testing.T) {
	store := &fakeStore[int]{claims: [][]int{{1}}}
	runner := NewRunner[int](store, HandlerFunc[int](func(ctx context.Context, task int) error {
		return nil
	}), fixedClock{now: time.Unix(0, 0).UTC()}, Config{WorkerID: "worker-c"})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := runner.RunUntilIdle(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunUntilIdle() error = %v, want context.Canceled", err)
	}
	if len(store.claimsSeen) != 0 {
		t.Fatalf("claim calls = %d, want none", len(store.claimsSeen))
	}
}

func TestRunnerStopsBeforeNextClaimAfterContextCancellation(t *testing.T) {
	store := &fakeStore[int]{claims: [][]int{{1}, {2}}}
	ctx, cancel := context.WithCancel(context.Background())
	runner := NewRunner[int](store, HandlerFunc[int](func(ctx context.Context, task int) error {
		cancel()
		return nil
	}), fixedClock{now: time.Unix(0, 0).UTC()}, Config{WorkerID: "worker-d"})

	err := runner.RunUntilIdle(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RunUntilIdle() error = %v, want context.Canceled", err)
	}
	if len(store.claimsSeen) != 1 {
		t.Fatalf("claim calls = %d, want no second claim", len(store.claimsSeen))
	}
}

type fakeStore[T comparable] struct {
	claims      [][]T
	claimsSeen  []ClaimOptions
	succeeded   []fakeSuccess[T]
	failed      []fakeFailure[T]
	claimErr    error
	succeedErr  error
	markFailErr error
}

func (f *fakeStore[T]) Claim(ctx context.Context, options ClaimOptions) ([]T, error) {
	f.claimsSeen = append(f.claimsSeen, options)
	if f.claimErr != nil {
		return nil, f.claimErr
	}
	if len(f.claims) == 0 {
		return nil, nil
	}
	tasks := f.claims[0]
	f.claims = f.claims[1:]
	return tasks, nil
}

func (f *fakeStore[T]) MarkSucceeded(ctx context.Context, task T, success Success) error {
	f.succeeded = append(f.succeeded, fakeSuccess[T]{task: task, success: success})
	return f.succeedErr
}

func (f *fakeStore[T]) MarkFailed(ctx context.Context, task T, failure Failure) error {
	f.failed = append(f.failed, fakeFailure[T]{task: task, failure: failure})
	return f.markFailErr
}

type fakeSuccess[T comparable] struct {
	task    T
	success Success
}

type fakeFailure[T comparable] struct {
	task    T
	failure Failure
}

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}
