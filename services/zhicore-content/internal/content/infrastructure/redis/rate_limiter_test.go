package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
	goredis "github.com/redis/go-redis/v9"
)

func TestFixedWindowRateLimiterAllowsAndRejectsByRedisWindow(t *testing.T) {
	client := &stubCmdable{incrValues: []int64{1, 3}, ttl: 20 * time.Second}
	limiter := NewFixedWindowRateLimiter(client, map[ports.RateLimitType]RateLimitRule{
		ports.RateLimitTypePublicRead: {Limit: 2, Window: time.Minute, Fallback: ports.RateLimitFallbackLocalMemory},
	})
	request := ports.RateLimitRequest{LimitType: ports.RateLimitTypePublicRead, Subject: "actor:1", Resource: "post:1", Operation: "read"}

	allowed := limiter.Check(context.Background(), request)
	if allowed.Outcome != ports.RateLimitOutcomeAllow || allowed.Reason != "fixed_window_allow" {
		t.Fatalf("first decision = %#v, want allow", allowed)
	}
	if client.expireCalls != 1 {
		t.Fatalf("expire calls = %d, want 1 for new Redis window", client.expireCalls)
	}

	rejected := limiter.Check(context.Background(), request)
	if rejected.Outcome != ports.RateLimitOutcomeRejectTooFrequent ||
		rejected.PublicCode != 1003 ||
		rejected.RetryAfter != 20*time.Second {
		t.Fatalf("second decision = %#v, want fixed-window rejection with retry-after", rejected)
	}
}

func TestFixedWindowRateLimiterUsesLocalFallbackDuringRedisOutage(t *testing.T) {
	client := &stubCmdable{incrErr: errors.New("redis unavailable")}
	limiter := NewFixedWindowRateLimiter(client, map[ports.RateLimitType]RateLimitRule{
		ports.RateLimitTypeDraftWrite: {Limit: 1, Window: time.Minute, Fallback: ports.RateLimitFallbackLocalMemory},
	})
	request := ports.RateLimitRequest{LimitType: ports.RateLimitTypeDraftWrite, Subject: "actor:1", Resource: "post:1", Operation: "save"}

	first := limiter.Check(context.Background(), request)
	if first.Outcome != ports.RateLimitOutcomeDegradedAllowLocal ||
		first.Fallback != ports.RateLimitFallbackLocalMemory {
		t.Fatalf("first decision = %#v, want local fallback allow", first)
	}

	second := limiter.Check(context.Background(), request)
	if second.Outcome != ports.RateLimitOutcomeRejectTooFrequent ||
		second.PublicCode != 1003 ||
		second.Fallback != ports.RateLimitFallbackLocalMemory ||
		second.RetryAfter <= 0 {
		t.Fatalf("second decision = %#v, want local fallback rejection", second)
	}
}

func TestFixedWindowRateLimiterFailsClosedWhenRedisBudgetCannotBeConfirmed(t *testing.T) {
	client := &stubCmdable{incrErr: errors.New("redis unavailable")}
	limiter := NewFixedWindowRateLimiter(client, map[ports.RateLimitType]RateLimitRule{
		ports.RateLimitTypePublishLifecycle: {Limit: 1, Window: time.Minute, Fallback: ports.RateLimitFallbackNone, FailClosed: true},
	})

	decision := limiter.Check(context.Background(), ports.RateLimitRequest{
		LimitType: ports.RateLimitTypePublishLifecycle,
		Subject:   "actor:1",
		Resource:  "post:1",
		Operation: "publish",
	})

	if decision.Outcome != ports.RateLimitOutcomeDegradedDenyUnavailable ||
		decision.PublicCode != 1004 ||
		decision.Reason != "redis_unavailable_fail_closed" {
		t.Fatalf("decision = %#v, want fail-closed dependency unavailable", decision)
	}
}

type stubCmdable struct {
	goredis.Cmdable
	incrValues  []int64
	incrErr     error
	expireErr   error
	ttl         time.Duration
	incrCalls   int
	expireCalls int
}

func (s *stubCmdable) Incr(context.Context, string) *goredis.IntCmd {
	s.incrCalls++
	if s.incrErr != nil {
		return goredis.NewIntResult(0, s.incrErr)
	}
	if len(s.incrValues) == 0 {
		return goredis.NewIntResult(1, nil)
	}
	index := s.incrCalls - 1
	if index >= len(s.incrValues) {
		index = len(s.incrValues) - 1
	}
	return goredis.NewIntResult(s.incrValues[index], nil)
}

func (s *stubCmdable) Expire(context.Context, string, time.Duration) *goredis.BoolCmd {
	s.expireCalls++
	return goredis.NewBoolResult(s.expireErr == nil, s.expireErr)
}

func (s *stubCmdable) TTL(context.Context, string) *goredis.DurationCmd {
	return goredis.NewDurationResult(s.ttl, nil)
}
