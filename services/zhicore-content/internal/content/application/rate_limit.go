package application

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func (s *Service) enforceRateLimit(ctx context.Context, request ports.RateLimitRequest) error {
	if s.limiter == nil {
		return nil
	}
	decision := s.limiter.Check(ctx, request)
	if decision.LimitType == "" {
		decision.LimitType = request.LimitType
	}
	if strings.TrimSpace(decision.Operation) == "" {
		decision.Operation = request.Operation
	}
	if s.observe != nil {
		s.observe.ObserveRateLimitDecision(ctx, decision)
	}

	switch decision.Outcome {
	case ports.RateLimitOutcomeAllow, ports.RateLimitOutcomeDegradedAllowLocal:
		return nil
	case ports.RateLimitOutcomeRejectTooFrequent:
		return fmt.Errorf("%w: %s", ErrRateLimited, decisionReason(decision))
	case ports.RateLimitOutcomeDegradedDenyUnavailable:
		// High-side-effect paths fail closed when the distributed budget cannot be confirmed.
		return fmt.Errorf("%w: %s", ErrDependencyUnavailable, decisionReason(decision))
	case ports.RateLimitOutcomeNoopSuccess:
		// No-op success must be handled by a use case that can return an empty success without side effects.
		return fmt.Errorf("%w: noop rate limit outcome requires explicit use case handling", ErrDependencyUnavailable)
	default:
		return fmt.Errorf("%w: unknown rate limit outcome", ErrDependencyUnavailable)
	}
}

func actorRateLimitRequest(limitType ports.RateLimitType, actor *Actor, resource, operation string) ports.RateLimitRequest {
	return ports.RateLimitRequest{
		LimitType: limitType,
		Subject:   actorRateLimitSubject(actor),
		Resource:  strings.TrimSpace(resource),
		Operation: strings.TrimSpace(operation),
	}
}

func publicRateLimitRequest(subject, resource, operation string) ports.RateLimitRequest {
	return ports.RateLimitRequest{
		LimitType: ports.RateLimitTypePublicRead,
		Subject:   publicRateLimitSubject(subject),
		Resource:  strings.TrimSpace(resource),
		Operation: strings.TrimSpace(operation),
	}
}

func actorRateLimitSubject(actor *Actor) string {
	if actor == nil || actor.UserID == 0 {
		return "anonymous"
	}
	return "actor:" + strconv.FormatInt(actor.UserID, 10)
}

func publicRateLimitSubject(subject string) string {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return "anonymous"
	}
	return subject
}

func decisionReason(decision ports.RateLimitDecision) string {
	reason := strings.TrimSpace(decision.Reason)
	if reason == "" {
		return "rate_limit_decision"
	}
	return reason
}
