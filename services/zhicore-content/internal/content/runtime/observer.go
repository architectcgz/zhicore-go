package runtime

import (
	"context"
	"strings"

	"github.com/architectcgz/zhicore-go/libs/kit/observability"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

const contentRateLimitDecisionMetric = "zhicore_content_rate_limit_decisions_total"

type rateLimitObserver struct {
	recorder observability.MetricsRecorder
}

func NewRateLimitObserver(recorder observability.MetricsRecorder) ports.ContentObserver {
	if recorder == nil {
		recorder = observability.NoopMetricsRecorder{}
	}
	return rateLimitObserver{recorder: recorder}
}

func (o rateLimitObserver) ObserveRateLimitDecision(ctx context.Context, decision ports.RateLimitDecision) {
	// Metrics are diagnostic only; recorder failures must not change rate-limit business flow.
	_ = o.recorder.IncrementCounter(ctx, contentRateLimitDecisionMetric, observability.Labels{
		"service":   "zhicore-content",
		"operation": labelOrUnknown(decision.Operation),
		"limitType": labelOrUnknown(string(decision.LimitType)),
		"reason":    labelOrUnknown(decision.Reason),
		"outcome":   labelOrUnknown(string(decision.Outcome)),
		"fallback":  labelOrUnknown(string(decision.Fallback)),
		"status":    rateLimitStatus(decision.Outcome),
	})
}

func labelOrUnknown(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "unknown"
	}
	return value
}

func rateLimitStatus(outcome ports.RateLimitOutcome) string {
	switch outcome {
	case ports.RateLimitOutcomeAllow, ports.RateLimitOutcomeDegradedAllowLocal:
		return "allowed"
	case ports.RateLimitOutcomeRejectTooFrequent:
		return "rate_limited"
	case ports.RateLimitOutcomeDegradedDenyUnavailable:
		return "dependency_unavailable"
	case ports.RateLimitOutcomeNoopSuccess:
		return "noop_success"
	default:
		return "unknown"
	}
}
