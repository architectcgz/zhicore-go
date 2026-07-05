package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/ports"
)

func TestOutboxDispatcherPublishesClaimedEventsAndMarksPublished(t *testing.T) {
	now := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)
	repo := &fakeOutboxDispatchRepository{
		events: []ports.OutboxEvent{{
			ID:            10,
			EventID:       "evt_10",
			EventType:     "comment.created",
			AggregateType: "comment",
			AggregateID:   "100",
			Payload:       []byte(`{"commentId":100}`),
			OccurredAt:    now.Add(-time.Minute),
		}},
	}
	publisher := &fakeIntegrationEventPublisher{}
	worker, err := NewOutboxDispatcher(OutboxDispatcherConfig{
		DispatcherID: "zhicore-comment:outbox-dispatcher:test",
		BatchSize:    10,
		MaxAttempts:  3,
		RetryBackoff: time.Minute,
		Repository:   repo,
		Publisher:    publisher,
		Clock:        fixedClock{now: now},
	})
	if err != nil {
		t.Fatalf("NewOutboxDispatcher() error = %v", err)
	}

	result, err := worker.DispatchOnce(context.Background())
	if err != nil {
		t.Fatalf("DispatchOnce() error = %v", err)
	}

	if result.Claimed != 1 || result.Published != 1 || result.Failed != 0 || result.Dead != 0 {
		t.Fatalf("dispatch result = %#v", result)
	}
	if len(publisher.published) != 1 || publisher.published[0].EventID != "evt_10" {
		t.Fatalf("published events = %#v", publisher.published)
	}
	if len(repo.publishedIDs) != 1 || repo.publishedIDs[0] != 10 {
		t.Fatalf("published ids = %#v", repo.publishedIDs)
	}
	if repo.lastClaim.DispatcherID != "zhicore-comment:outbox-dispatcher:test" || repo.lastClaim.BatchSize != 10 {
		t.Fatalf("claim options = %#v", repo.lastClaim)
	}
}

func TestOutboxDispatcherMarksRetryAndDeadOnPublishFailure(t *testing.T) {
	now := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)
	publishErr := errors.New("rabbitmq unavailable")
	repo := &fakeOutboxDispatchRepository{
		events: []ports.OutboxEvent{
			{ID: 20, EventID: "evt_retry", EventType: "comment.liked", AggregateType: "comment", AggregateID: "200", AttemptCount: 1, OccurredAt: now},
			{ID: 21, EventID: "evt_dead", EventType: "comment.unliked", AggregateType: "comment", AggregateID: "201", AttemptCount: 2, OccurredAt: now},
		},
	}
	publisher := &fakeIntegrationEventPublisher{err: publishErr}
	worker, err := NewOutboxDispatcher(OutboxDispatcherConfig{
		DispatcherID: "zhicore-comment:outbox-dispatcher:test",
		BatchSize:    10,
		MaxAttempts:  3,
		RetryBackoff: 2 * time.Minute,
		Repository:   repo,
		Publisher:    publisher,
		Clock:        fixedClock{now: now},
	})
	if err != nil {
		t.Fatalf("NewOutboxDispatcher() error = %v", err)
	}

	result, err := worker.DispatchOnce(context.Background())
	if err != nil {
		t.Fatalf("DispatchOnce() error = %v", err)
	}

	if result.Claimed != 2 || result.Published != 0 || result.Failed != 1 || result.Dead != 1 {
		t.Fatalf("dispatch result = %#v", result)
	}
	if len(repo.failures) != 2 {
		t.Fatalf("failure records = %#v", repo.failures)
	}
	if repo.failures[0].Dead || repo.failures[0].NextRetryAt == nil || !repo.failures[0].NextRetryAt.Equal(now.Add(2*time.Minute)) {
		t.Fatalf("retry failure = %#v", repo.failures[0])
	}
	if !repo.failures[1].Dead || repo.failures[1].NextRetryAt != nil {
		t.Fatalf("dead failure = %#v", repo.failures[1])
	}
	if !strings.Contains(repo.failures[0].LastError, "rabbitmq unavailable") {
		t.Fatalf("last error = %q", repo.failures[0].LastError)
	}
}

func TestOutboxDispatcherRunStopsBeforeClaimWhenContextCanceled(t *testing.T) {
	repo := &fakeOutboxDispatchRepository{}
	worker, err := NewOutboxDispatcher(OutboxDispatcherConfig{
		DispatcherID: "zhicore-comment:outbox-dispatcher:test",
		BatchSize:    10,
		MaxAttempts:  3,
		RetryBackoff: time.Minute,
		Repository:   repo,
		Publisher:    &fakeIntegrationEventPublisher{},
		Clock:        fixedClock{now: time.Unix(0, 0).UTC()},
	})
	if err != nil {
		t.Fatalf("NewOutboxDispatcher() error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = worker.Run(ctx, time.Millisecond)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context canceled", err)
	}
	if repo.claimCalls != 0 {
		t.Fatalf("claim calls = %d, want 0", repo.claimCalls)
	}
}

type fakeOutboxDispatchRepository struct {
	events       []ports.OutboxEvent
	lastClaim    ports.OutboxClaimOptions
	publishedIDs []int64
	failures     []ports.OutboxFailure
	claimCalls   int
}

func (f *fakeOutboxDispatchRepository) ClaimPendingOutbox(ctx context.Context, options ports.OutboxClaimOptions) ([]ports.OutboxEvent, error) {
	f.claimCalls++
	f.lastClaim = options
	return append([]ports.OutboxEvent(nil), f.events...), nil
}

func (f *fakeOutboxDispatchRepository) MarkOutboxPublished(ctx context.Context, eventID int64, publishedAt time.Time) error {
	f.publishedIDs = append(f.publishedIDs, eventID)
	return nil
}

func (f *fakeOutboxDispatchRepository) MarkOutboxFailed(ctx context.Context, failure ports.OutboxFailure) error {
	f.failures = append(f.failures, failure)
	return nil
}

type fakeIntegrationEventPublisher struct {
	err       error
	published []ports.OutboxEvent
}

func (f *fakeIntegrationEventPublisher) PublishIntegrationEvent(ctx context.Context, event ports.OutboxEvent) error {
	if f.err != nil {
		return f.err
	}
	f.published = append(f.published, event)
	return nil
}
