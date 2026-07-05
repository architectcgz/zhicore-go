package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestOutboxDispatcher(t *testing.T) {
	t.Run("claims a batch publishes events and marks them published", func(t *testing.T) {
		deps := newOutboxDispatcherDeps()
		deps.repository.claimResults = [][]ports.OutboxEvent{{
			{ID: 41, EventID: "evt_post_published_1", EventType: "content.post.published"},
			{ID: 42, EventID: "evt_post_published_2", EventType: "content.post.published"},
		}}
		worker := newTestOutboxDispatcher(deps)

		err := worker.RunUntilIdle(context.Background())
		if err != nil {
			t.Fatalf("RunUntilIdle() error = %v", err)
		}
		if deps.repository.claimCalls != 2 {
			t.Fatalf("claim calls = %d, want first batch and final idle claim", deps.repository.claimCalls)
		}
		request := deps.repository.claimRequests[0]
		if request.DispatcherID != "content-outbox:test" || request.BatchSize != 10 || request.StaleAfter != 5*time.Minute || !request.Now.Equal(deps.clock.now) {
			t.Fatalf("claim request = %+v", request)
		}
		if len(deps.publisher.published) != 2 || deps.publisher.published[0].ID != 41 || deps.publisher.published[1].ID != 42 {
			t.Fatalf("published events = %+v", deps.publisher.published)
		}
		if len(deps.repository.published) != 2 || deps.repository.published[0].ID != 41 || deps.repository.published[0].DispatcherID != "content-outbox:test" {
			t.Fatalf("marked published = %+v", deps.repository.published)
		}
		if len(deps.repository.failed) != 0 {
			t.Fatalf("failed = %+v, want none", deps.repository.failed)
		}
	})

	t.Run("marks publish failure with retry backoff", func(t *testing.T) {
		deps := newOutboxDispatcherDeps()
		deps.repository.claimResults = [][]ports.OutboxEvent{{
			{ID: 43, EventID: "evt_retry", EventType: "content.post.published", AttemptCount: 1},
		}}
		deps.publisher.errByID[43] = errors.New("rabbitmq publish failed")
		worker := newTestOutboxDispatcher(deps)

		err := worker.RunUntilIdle(context.Background())
		if err != nil {
			t.Fatalf("RunUntilIdle() error = %v", err)
		}
		if len(deps.repository.failed) != 1 {
			t.Fatalf("failed = %+v, want one failure", deps.repository.failed)
		}
		got := deps.repository.failed[0]
		if got.ID != 43 || got.DispatcherID != "content-outbox:test" || got.AttemptCount != 2 || got.Dead {
			t.Fatalf("failure = %+v, want retry attempt 2", got)
		}
		if got.NextRetryAt == nil || !got.NextRetryAt.Equal(deps.clock.now.Add(time.Minute)) {
			t.Fatalf("next retry = %v, want clock + backoff", got.NextRetryAt)
		}
		if got.LastError != "rabbitmq publish failed" || !got.FailedAt.Equal(deps.clock.now) {
			t.Fatalf("failure error/time = %+v", got)
		}
	})

	t.Run("marks event dead when max attempts is reached", func(t *testing.T) {
		deps := newOutboxDispatcherDeps()
		deps.repository.claimResults = [][]ports.OutboxEvent{{
			{ID: 44, EventID: "evt_dead", EventType: "content.post.published", AttemptCount: 2},
		}}
		deps.publisher.errByID[44] = errors.New("rabbitmq publish failed")
		worker := newTestOutboxDispatcher(deps)

		err := worker.RunUntilIdle(context.Background())
		if err != nil {
			t.Fatalf("RunUntilIdle() error = %v", err)
		}
		if len(deps.repository.failed) != 1 {
			t.Fatalf("failed = %+v, want one dead failure", deps.repository.failed)
		}
		got := deps.repository.failed[0]
		if got.ID != 44 || got.AttemptCount != 3 || !got.Dead || got.NextRetryAt != nil {
			t.Fatalf("failure = %+v, want dead attempt without next retry", got)
		}
	})

	t.Run("does not claim new work after shutdown cancellation", func(t *testing.T) {
		deps := newOutboxDispatcherDeps()
		deps.repository.claimResults = [][]ports.OutboxEvent{
			{{ID: 45, EventID: "evt_first", EventType: "content.post.published"}},
			{{ID: 46, EventID: "evt_second", EventType: "content.post.published"}},
		}
		ctx, cancel := context.WithCancel(context.Background())
		deps.publisher.afterPublish = cancel
		worker := newTestOutboxDispatcher(deps)

		err := worker.RunUntilIdle(ctx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("RunUntilIdle() error = %v, want context.Canceled", err)
		}
		if deps.repository.claimCalls != 1 {
			t.Fatalf("claim calls = %d, want no second claim after cancel", deps.repository.claimCalls)
		}
		if len(deps.publisher.published) != 1 || deps.publisher.published[0].ID != 45 {
			t.Fatalf("published events = %+v, want first only", deps.publisher.published)
		}
	})

	t.Run("does not claim when context is already canceled", func(t *testing.T) {
		deps := newOutboxDispatcherDeps()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		worker := newTestOutboxDispatcher(deps)

		err := worker.RunUntilIdle(ctx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("RunUntilIdle() error = %v, want context.Canceled", err)
		}
		if deps.repository.claimCalls != 0 {
			t.Fatalf("claim calls = %d, want none after pre-canceled context", deps.repository.claimCalls)
		}
	})
}

type outboxDispatcherDeps struct {
	repository *fakeOutboxDispatchRepository
	publisher  *fakeIntegrationEventPublisher
	clock      fakeClock
}

func newOutboxDispatcherDeps() outboxDispatcherDeps {
	return outboxDispatcherDeps{
		repository: &fakeOutboxDispatchRepository{},
		publisher:  &fakeIntegrationEventPublisher{errByID: map[int64]error{}},
		clock:      fakeClock{now: time.Date(2026, 7, 5, 16, 0, 0, 0, time.UTC)},
	}
}

func newTestOutboxDispatcher(deps outboxDispatcherDeps) *OutboxDispatcher {
	return NewOutboxDispatcher(OutboxDispatcherDeps{
		Repository: deps.repository,
		Publisher:  deps.publisher,
		Clock:      deps.clock,
	}, OutboxDispatcherConfig{
		DispatcherID:    "content-outbox:test",
		BatchSize:       10,
		StaleClaimAfter: 5 * time.Minute,
		RetryBackoff:    time.Minute,
		DeadThreshold:   3,
	})
}

type fakeOutboxDispatchRepository struct {
	claimCalls    int
	claimRequests []ports.OutboxClaimOptions
	claimResults  [][]ports.OutboxEvent
	published     []ports.OutboxPublished
	failed        []ports.OutboxFailure
}

func (f *fakeOutboxDispatchRepository) ClaimPendingOutbox(ctx context.Context, options ports.OutboxClaimOptions) ([]ports.OutboxEvent, error) {
	f.claimCalls++
	f.claimRequests = append(f.claimRequests, options)
	if len(f.claimResults) == 0 {
		return nil, nil
	}
	result := f.claimResults[0]
	f.claimResults = f.claimResults[1:]
	return result, nil
}

func (f *fakeOutboxDispatchRepository) MarkOutboxPublished(ctx context.Context, published ports.OutboxPublished) error {
	f.published = append(f.published, published)
	return nil
}

func (f *fakeOutboxDispatchRepository) MarkOutboxFailed(ctx context.Context, failure ports.OutboxFailure) error {
	f.failed = append(f.failed, failure)
	return nil
}

type fakeIntegrationEventPublisher struct {
	published    []ports.OutboxEvent
	errByID      map[int64]error
	afterPublish func()
}

func (f *fakeIntegrationEventPublisher) PublishIntegrationEvent(ctx context.Context, event ports.OutboxEvent) error {
	f.published = append(f.published, event)
	if f.afterPublish != nil {
		f.afterPublish()
	}
	return f.errByID[event.ID]
}
