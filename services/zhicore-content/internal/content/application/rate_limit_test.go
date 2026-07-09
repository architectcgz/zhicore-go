package application

import (
	"context"
	"errors"
	"testing"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestEnforceRateLimitMapsDecisions(t *testing.T) {
	testCases := []struct {
		name    string
		outcome ports.RateLimitOutcome
		wantErr error
	}{
		{name: "allow", outcome: ports.RateLimitOutcomeAllow},
		{name: "degraded local allow", outcome: ports.RateLimitOutcomeDegradedAllowLocal},
		{name: "too frequent", outcome: ports.RateLimitOutcomeRejectTooFrequent, wantErr: ErrRateLimited},
		{name: "fail closed", outcome: ports.RateLimitOutcomeDegradedDenyUnavailable, wantErr: ErrDependencyUnavailable},
		{name: "unknown outcome fails closed", outcome: ports.RateLimitOutcome("UNKNOWN"), wantErr: ErrDependencyUnavailable},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			observer := &recordingApplicationObserver{}
			service := NewService(Deps{
				Limiter: &fixedDecisionLimiter{decision: ports.RateLimitDecision{
					Outcome:   tc.outcome,
					Reason:    "test_decision",
					LimitType: ports.RateLimitTypePublicRead,
				}},
				Observe: observer,
			})

			err := service.enforceRateLimit(context.Background(), ports.RateLimitRequest{LimitType: ports.RateLimitTypePublicRead})

			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("enforceRateLimit() error = %v, want nil", err)
				}
			} else if !errors.Is(err, tc.wantErr) {
				t.Fatalf("enforceRateLimit() error = %v, want %v", err, tc.wantErr)
			}
			if len(observer.decisions) != 1 || observer.decisions[0].Outcome != tc.outcome {
				t.Fatalf("observer decisions = %#v, want one %s decision", observer.decisions, tc.outcome)
			}
		})
	}
}

func TestEnforceRateLimitBackfillsDecisionOperation(t *testing.T) {
	observer := &recordingApplicationObserver{}
	service := NewService(Deps{
		Limiter: &fixedDecisionLimiter{decision: ports.RateLimitDecision{
			Outcome: ports.RateLimitOutcomeAllow,
			Reason:  "allow",
		}},
		Observe: observer,
	})

	err := service.enforceRateLimit(context.Background(), ports.RateLimitRequest{
		LimitType: ports.RateLimitTypeDraftWrite,
		Operation: "save_draft_body",
	})

	if err != nil {
		t.Fatalf("enforceRateLimit() error = %v, want nil", err)
	}
	if len(observer.decisions) != 1 {
		t.Fatalf("observer decisions = %#v, want one decision", observer.decisions)
	}
	if observer.decisions[0].LimitType != ports.RateLimitTypeDraftWrite {
		t.Fatalf("decision limit type = %q, want request limit type", observer.decisions[0].LimitType)
	}
	if observer.decisions[0].Operation != "save_draft_body" {
		t.Fatalf("decision operation = %q, want request operation", observer.decisions[0].Operation)
	}
}

type fixedDecisionLimiter struct {
	decision ports.RateLimitDecision
}

func (l *fixedDecisionLimiter) Check(context.Context, ports.RateLimitRequest) ports.RateLimitDecision {
	return l.decision
}

type recordingApplicationObserver struct {
	decisions []ports.RateLimitDecision
}

func (o *recordingApplicationObserver) ObserveRateLimitDecision(_ context.Context, decision ports.RateLimitDecision) {
	o.decisions = append(o.decisions, decision)
}

func (o *recordingApplicationObserver) ObserveWorkerResult(context.Context, ports.WorkerResult) {}
