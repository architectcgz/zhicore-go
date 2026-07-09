package ports

import (
	"context"
	"time"
)

type RateLimitType string

const (
	RateLimitTypePublicRead       RateLimitType = "public_read"
	RateLimitTypeDraftWrite       RateLimitType = "draft_write"
	RateLimitTypePublishLifecycle RateLimitType = "publish_lifecycle"
	RateLimitTypeEngagementWrite  RateLimitType = "engagement_write"
	RateLimitTypeEngagementRead   RateLimitType = "engagement_read"
	RateLimitTypeAdminCommand     RateLimitType = "admin_command"
	RateLimitTypeInternalClient   RateLimitType = "internal_client"
)

type RateLimitOutcome string

const (
	RateLimitOutcomeAllow                   RateLimitOutcome = "ALLOW"
	RateLimitOutcomeRejectTooFrequent       RateLimitOutcome = "REJECT_TOO_FREQUENT"
	RateLimitOutcomeDegradedAllowLocal      RateLimitOutcome = "DEGRADED_ALLOW_LOCAL"
	RateLimitOutcomeDegradedDenyUnavailable RateLimitOutcome = "DEGRADED_DENY_UNAVAILABLE"
)

type RateLimitFallback string

const (
	RateLimitFallbackNone        RateLimitFallback = "none"
	RateLimitFallbackLocalMemory RateLimitFallback = "local_memory"
	RateLimitFallbackGatewayOnly RateLimitFallback = "gateway_only"
)

type RateLimitRequest struct {
	LimitType RateLimitType
	Subject   string
	Resource  string
	Operation string
}

type RateLimitDecision struct {
	Outcome    RateLimitOutcome
	PublicCode int
	Reason     string
	LimitType  RateLimitType
	Operation  string
	RetryAfter time.Duration
	Fallback   RateLimitFallback
}

type RateLimiter interface {
	Check(context.Context, RateLimitRequest) RateLimitDecision
}

type ContentObserver interface {
	ObserveRateLimitDecision(context.Context, RateLimitDecision)
	ObserveWorkerResult(context.Context, WorkerResult)
}

type WorkerResultStatus string

const (
	WorkerResultStatusSuccess WorkerResultStatus = "success"
	WorkerResultStatusFailed  WorkerResultStatus = "failed"
)

type WorkerResult struct {
	Worker     string
	Operation  string
	Status     WorkerResultStatus
	ErrorClass string
	Duration   time.Duration
}
