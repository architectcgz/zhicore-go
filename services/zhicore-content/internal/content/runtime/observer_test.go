package runtime

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/architectcgz/zhicore-go/libs/kit/observability"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestContentRateLimitObserverRecordsDecisionMetric(t *testing.T) {
	recorder := &recordingMetricsRecorder{}
	observer := NewRateLimitObserver(recorder)

	observer.ObserveRateLimitDecision(context.Background(), ports.RateLimitDecision{
		Outcome:   ports.RateLimitOutcomeRejectTooFrequent,
		Reason:    "fixed_window_limit_exceeded",
		LimitType: ports.RateLimitTypePublicRead,
		Operation: "list_published_posts",
		Fallback:  ports.RateLimitFallbackNone,
	})

	if recorder.calls != 1 {
		t.Fatalf("recorder calls = %d, want 1", recorder.calls)
	}
	if recorder.name != "zhicore_content_rate_limit_decisions_total" {
		t.Fatalf("metric name = %q, want content rate limit counter", recorder.name)
	}
	wantLabels := observability.Labels{
		"service":   "zhicore-content",
		"operation": "list_published_posts",
		"limitType": "public_read",
		"reason":    "fixed_window_limit_exceeded",
		"outcome":   "REJECT_TOO_FREQUENT",
		"fallback":  "none",
		"status":    "rate_limited",
	}
	if !reflect.DeepEqual(recorder.labels, wantLabels) {
		t.Fatalf("metric labels = %#v, want %#v", recorder.labels, wantLabels)
	}
	if err := observability.ValidateLowCardinalityLabels(recorder.labels); err != nil {
		t.Fatalf("metric labels are not low-cardinality: %v", err)
	}
}

func TestContentRateLimitObserverIgnoresRecorderFailure(t *testing.T) {
	observer := NewRateLimitObserver(failingMetricsRecorder{})

	observer.ObserveRateLimitDecision(context.Background(), ports.RateLimitDecision{
		Outcome:   ports.RateLimitOutcomeDegradedDenyUnavailable,
		Reason:    "redis_unavailable_fail_closed",
		LimitType: ports.RateLimitTypePublishLifecycle,
		Operation: "publish_post",
		Fallback:  ports.RateLimitFallbackNone,
	})
}

func TestContentObserverRecordsWorkerResultMetric(t *testing.T) {
	recorder := &recordingMetricsRecorder{}
	observer := NewRateLimitObserver(recorder)

	observer.ObserveWorkerResult(context.Background(), ports.WorkerResult{
		Worker:     "content-outbox-dispatcher",
		Operation:  "worker.run_until_idle",
		Status:     ports.WorkerResultStatusFailed,
		ErrorClass: "failed",
	})

	if recorder.calls != 1 {
		t.Fatalf("recorder calls = %d, want 1", recorder.calls)
	}
	if recorder.name != "zhicore_content_worker_jobs_total" {
		t.Fatalf("metric name = %q, want worker jobs counter", recorder.name)
	}
	wantLabels := observability.Labels{
		"service":    "zhicore-content",
		"worker":     "content-outbox-dispatcher",
		"operation":  "worker.run_until_idle",
		"status":     "failed",
		"errorClass": "failed",
	}
	if !reflect.DeepEqual(recorder.labels, wantLabels) {
		t.Fatalf("metric labels = %#v, want %#v", recorder.labels, wantLabels)
	}
	if err := observability.ValidateLowCardinalityLabels(recorder.labels); err != nil {
		t.Fatalf("metric labels are not low-cardinality: %v", err)
	}
}

type recordingMetricsRecorder struct {
	calls  int
	name   string
	labels observability.Labels
}

func (r *recordingMetricsRecorder) IncrementCounter(_ context.Context, name string, labels observability.Labels) error {
	r.calls++
	r.name = name
	r.labels = labels
	return nil
}

type failingMetricsRecorder struct{}

func (failingMetricsRecorder) IncrementCounter(context.Context, string, observability.Labels) error {
	return errors.New("metrics backend unavailable")
}
