package runtime

import (
	"testing"
	"time"

	contentredis "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/infrastructure/redis"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func TestRateLimitRulesFromConfigPreservesDefaultSemantics(t *testing.T) {
	rules := rateLimitRulesFromConfig(DefaultRateLimitConfig())

	assertRedisRateLimitRule(t, rules, ports.RateLimitTypePublicRead, 120, time.Minute, ports.RateLimitFallbackLocalMemory, 2*time.Minute, false)
	assertRedisRateLimitRule(t, rules, ports.RateLimitTypeDraftWrite, 30, time.Minute, ports.RateLimitFallbackLocalMemory, 30*time.Second, false)
	assertRedisRateLimitRule(t, rules, ports.RateLimitTypePublishLifecycle, 5, time.Minute, ports.RateLimitFallbackNone, 0, true)
	assertRedisRateLimitRule(t, rules, ports.RateLimitTypeEngagementWrite, 60, time.Minute, ports.RateLimitFallbackLocalMemory, 30*time.Second, false)
	assertRedisRateLimitRule(t, rules, ports.RateLimitTypeEngagementRead, 120, time.Minute, ports.RateLimitFallbackLocalMemory, 2*time.Minute, false)
	assertRedisRateLimitRule(t, rules, ports.RateLimitTypeAdminCommand, 10, time.Minute, ports.RateLimitFallbackNone, 0, true)
	assertRedisRateLimitRule(t, rules, ports.RateLimitTypeInternalClient, 120, time.Minute, ports.RateLimitFallbackNone, 0, true)
}

func TestRateLimitRulesFromConfigUsesOverridesAndBackfillsMissingDefaults(t *testing.T) {
	cfg := DefaultRateLimitConfig()
	cfg.Rules[ports.RateLimitTypePublicRead] = RateLimitRuleConfig{
		Limit:          42,
		Window:         30 * time.Second,
		Fallback:       ports.RateLimitFallbackGatewayOnly,
		FallbackWindow: 45 * time.Second,
		FailClosed:     true,
	}
	delete(cfg.Rules, ports.RateLimitTypeDraftWrite)

	rules := rateLimitRulesFromConfig(cfg)

	assertRedisRateLimitRule(t, rules, ports.RateLimitTypePublicRead, 42, 30*time.Second, ports.RateLimitFallbackGatewayOnly, 45*time.Second, true)
	assertRedisRateLimitRule(t, rules, ports.RateLimitTypeDraftWrite, 30, time.Minute, ports.RateLimitFallbackLocalMemory, 30*time.Second, false)
}

func assertRedisRateLimitRule(t *testing.T, rules map[ports.RateLimitType]contentredis.RateLimitRule, limitType ports.RateLimitType, limit int, window time.Duration, fallback ports.RateLimitFallback, fallbackWindow time.Duration, failClosed bool) {
	t.Helper()
	rule, ok := rules[limitType]
	if !ok {
		t.Fatalf("redis rate limit rule %s missing", limitType)
	}
	if rule.Limit != limit || rule.Window != window || rule.Fallback != fallback || rule.FallbackWindow != fallbackWindow || rule.FailClosed != failClosed {
		t.Fatalf("redis rate limit rule %s = %#v, want limit=%d window=%s fallback=%s fallbackWindow=%s failClosed=%t", limitType, rule, limit, window, fallback, fallbackWindow, failClosed)
	}
}
