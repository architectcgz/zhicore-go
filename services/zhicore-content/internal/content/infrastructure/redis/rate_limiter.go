package redis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
	goredis "github.com/redis/go-redis/v9"
)

type RateLimitRule struct {
	Limit          int
	Window         time.Duration
	Fallback       ports.RateLimitFallback
	FallbackWindow time.Duration
	FailClosed     bool
}

func NewFixedWindowRateLimiter(client goredis.Cmdable, rules map[ports.RateLimitType]RateLimitRule) ports.RateLimiter {
	return &fixedWindowRateLimiter{
		client:        client,
		rules:         rules,
		fallback:      newLocalMemoryRateLimiter(),
		degradedSince: make(map[ports.RateLimitType]time.Time),
		now:           time.Now,
	}
}

type fixedWindowRateLimiter struct {
	client        goredis.Cmdable
	rules         map[ports.RateLimitType]RateLimitRule
	fallback      *localMemoryRateLimiter
	degradedMu    sync.Mutex
	degradedSince map[ports.RateLimitType]time.Time
	now           func() time.Time
}

func (l *fixedWindowRateLimiter) Check(ctx context.Context, request ports.RateLimitRequest) ports.RateLimitDecision {
	rule, ok := l.rules[request.LimitType]
	if !ok {
		rule = RateLimitRule{Limit: 1, Window: time.Minute, Fallback: ports.RateLimitFallbackNone, FailClosed: true}
	}
	now := l.now()
	key := redisRateLimitKey(request, rule, now)
	count, err := l.client.Incr(ctx, key).Result()
	if err != nil {
		return l.degradedDecision(request, rule, now)
	}
	if count == 1 {
		if err := l.client.Expire(ctx, key, rule.Window).Err(); err != nil {
			return l.degradedDecision(request, rule, now)
		}
	}
	l.markRedisHealthy(request.LimitType)
	if count > int64(rule.Limit) {
		return ports.RateLimitDecision{
			Outcome:    ports.RateLimitOutcomeRejectTooFrequent,
			PublicCode: 1003,
			Reason:     "fixed_window_limit_exceeded",
			LimitType:  request.LimitType,
			RetryAfter: l.retryAfter(ctx, key, rule),
			Fallback:   ports.RateLimitFallbackNone,
		}
	}
	return ports.RateLimitDecision{
		Outcome:   ports.RateLimitOutcomeAllow,
		Reason:    "fixed_window_allow",
		LimitType: request.LimitType,
		Fallback:  ports.RateLimitFallbackNone,
	}
}

func (l *fixedWindowRateLimiter) retryAfter(ctx context.Context, key string, rule RateLimitRule) time.Duration {
	ttl, err := l.client.TTL(ctx, key).Result()
	if err != nil || ttl <= 0 {
		return rule.Window
	}
	return ttl
}

func (l *fixedWindowRateLimiter) degradedDecision(request ports.RateLimitRequest, rule RateLimitRule, now time.Time) ports.RateLimitDecision {
	if rule.FailClosed || rule.Fallback == ports.RateLimitFallbackNone {
		return ports.RateLimitDecision{
			Outcome:    ports.RateLimitOutcomeDegradedDenyUnavailable,
			PublicCode: 1004,
			Reason:     "redis_unavailable_fail_closed",
			LimitType:  request.LimitType,
			Fallback:   rule.Fallback,
		}
	}
	if !l.fallbackWindowAllows(request.LimitType, rule, now) {
		return ports.RateLimitDecision{
			Outcome:    ports.RateLimitOutcomeDegradedDenyUnavailable,
			PublicCode: 1004,
			Reason:     "redis_unavailable_fallback_window_exceeded",
			LimitType:  request.LimitType,
			Fallback:   rule.Fallback,
		}
	}
	switch rule.Fallback {
	case ports.RateLimitFallbackLocalMemory:
		// Local fallback is only a short outage budget; it is per process and cannot enforce a global quota in multi-instance deployments.
		allowed, retryAfter := l.fallback.allow(localMemoryRateLimitKey(request, rule, now), rule, now)
		if !allowed {
			return ports.RateLimitDecision{
				Outcome:    ports.RateLimitOutcomeRejectTooFrequent,
				PublicCode: 1003,
				Reason:     "local_memory_limit_exceeded",
				LimitType:  request.LimitType,
				RetryAfter: retryAfter,
				Fallback:   ports.RateLimitFallbackLocalMemory,
			}
		}
		return ports.RateLimitDecision{
			Outcome:   ports.RateLimitOutcomeDegradedAllowLocal,
			Reason:    "redis_unavailable_local_memory_allow",
			LimitType: request.LimitType,
			Fallback:  ports.RateLimitFallbackLocalMemory,
		}
	case ports.RateLimitFallbackGatewayOnly:
		return ports.RateLimitDecision{
			Outcome:   ports.RateLimitOutcomeDegradedAllowLocal,
			Reason:    "redis_unavailable_gateway_only_allow",
			LimitType: request.LimitType,
			Fallback:  ports.RateLimitFallbackGatewayOnly,
		}
	default:
		return ports.RateLimitDecision{
			Outcome:    ports.RateLimitOutcomeDegradedDenyUnavailable,
			PublicCode: 1004,
			Reason:     "redis_unavailable_unknown_fallback",
			LimitType:  request.LimitType,
			Fallback:   rule.Fallback,
		}
	}
}

func (l *fixedWindowRateLimiter) fallbackWindowAllows(limitType ports.RateLimitType, rule RateLimitRule, now time.Time) bool {
	if rule.FallbackWindow <= 0 {
		return false
	}
	l.degradedMu.Lock()
	defer l.degradedMu.Unlock()

	startedAt, ok := l.degradedSince[limitType]
	if !ok {
		l.degradedSince[limitType] = now
		return true
	}
	return !now.After(startedAt.Add(rule.FallbackWindow))
}

func (l *fixedWindowRateLimiter) markRedisHealthy(limitType ports.RateLimitType) {
	l.degradedMu.Lock()
	defer l.degradedMu.Unlock()
	delete(l.degradedSince, limitType)
}

func redisRateLimitKey(request ports.RateLimitRequest, rule RateLimitRule, now time.Time) string {
	windowStart := now.Truncate(rule.Window).Unix()
	return "content:rate_limit:" + string(request.LimitType) + ":" + strconv.FormatInt(windowStart, 10) + ":" + hashedRateLimitParts(request)
}

func localMemoryRateLimitKey(request ports.RateLimitRequest, rule RateLimitRule, now time.Time) string {
	windowStart := now.Truncate(rule.Window).Unix()
	return string(request.LimitType) + ":" + strconv.FormatInt(windowStart, 10) + ":" + hashedRateLimitParts(request)
}

func hashedRateLimitParts(request ports.RateLimitRequest) string {
	parts := strings.Join([]string{
		string(request.LimitType),
		strings.TrimSpace(request.Subject),
		strings.TrimSpace(request.Resource),
		strings.TrimSpace(request.Operation),
	}, "\x00")
	sum := sha256.Sum256([]byte(parts))
	return hex.EncodeToString(sum[:])
}

type localMemoryRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]localMemoryRateLimitBucket
}

type localMemoryRateLimitBucket struct {
	count   int
	expires time.Time
}

func newLocalMemoryRateLimiter() *localMemoryRateLimiter {
	return &localMemoryRateLimiter{buckets: make(map[string]localMemoryRateLimitBucket)}
}

func (l *localMemoryRateLimiter) allow(key string, rule RateLimitRule, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	bucket := l.buckets[key]
	if bucket.expires.IsZero() || !now.Before(bucket.expires) {
		bucket = localMemoryRateLimitBucket{expires: now.Truncate(rule.Window).Add(rule.Window)}
	}
	bucket.count++
	l.buckets[key] = bucket
	if bucket.count > rule.Limit {
		return false, bucket.expires.Sub(now)
	}
	return true, 0
}
