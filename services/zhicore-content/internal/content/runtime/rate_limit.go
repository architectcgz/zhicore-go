package runtime

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	contentredis "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/infrastructure/redis"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
	goredis "github.com/redis/go-redis/v9"
)

// RateLimitType is re-exported from ports so process-root config loading can
// name runtime-owned rules without importing application ports directly.
type RateLimitType = ports.RateLimitType

const (
	RateLimitTypePublicRead       RateLimitType = ports.RateLimitTypePublicRead
	RateLimitTypeDraftWrite       RateLimitType = ports.RateLimitTypeDraftWrite
	RateLimitTypePublishLifecycle RateLimitType = ports.RateLimitTypePublishLifecycle
	RateLimitTypeEngagementWrite  RateLimitType = ports.RateLimitTypeEngagementWrite
	RateLimitTypeEngagementRead   RateLimitType = ports.RateLimitTypeEngagementRead
	RateLimitTypeAdminCommand     RateLimitType = ports.RateLimitTypeAdminCommand
	RateLimitTypeInternalClient   RateLimitType = ports.RateLimitTypeInternalClient
)

// RateLimitFallback is re-exported for runtime configuration parsing while the
// limiter decision contract remains owned by ports.
type RateLimitFallback = ports.RateLimitFallback

const (
	RateLimitFallbackNone        RateLimitFallback = ports.RateLimitFallbackNone
	RateLimitFallbackLocalMemory RateLimitFallback = ports.RateLimitFallbackLocalMemory
	RateLimitFallbackGatewayOnly RateLimitFallback = ports.RateLimitFallbackGatewayOnly
)

type RedisConfig struct {
	Addr         string
	Username     string
	Password     string
	DB           int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
}

func (c RedisConfig) String() string {
	password := "missing"
	if strings.TrimSpace(c.Password) != "" {
		password = "<redacted>"
	}
	return fmt.Sprintf(
		"{Addr:%s Username:%s Password:%s DB:%d DialTimeout:%s ReadTimeout:%s WriteTimeout:%s PoolSize:%d}",
		c.Addr,
		c.Username,
		password,
		c.DB,
		c.DialTimeout,
		c.ReadTimeout,
		c.WriteTimeout,
		c.PoolSize,
	)
}

func (c RedisConfig) GoString() string {
	return c.String()
}

type RateLimitConfig struct {
	Rules map[RateLimitType]RateLimitRuleConfig
}

type RateLimitRuleConfig struct {
	Limit          int
	Window         time.Duration
	Fallback       RateLimitFallback
	FallbackWindow time.Duration
	FailClosed     bool
}

type RedisRateLimitDependency struct {
	Limiter ports.RateLimiter
	Health  HealthChecker
	Closer  io.Closer
}

func DefaultRedisConfig() RedisConfig {
	return RedisConfig{
		DialTimeout:  200 * time.Millisecond,
		ReadTimeout:  200 * time.Millisecond,
		WriteTimeout: 200 * time.Millisecond,
		PoolSize:     10,
	}
}

func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{Rules: map[RateLimitType]RateLimitRuleConfig{
		RateLimitTypePublicRead:       {Limit: 120, Window: time.Minute, Fallback: RateLimitFallbackLocalMemory, FallbackWindow: 2 * time.Minute},
		RateLimitTypeDraftWrite:       {Limit: 30, Window: time.Minute, Fallback: RateLimitFallbackLocalMemory, FallbackWindow: 30 * time.Second},
		RateLimitTypePublishLifecycle: {Limit: 5, Window: time.Minute, Fallback: RateLimitFallbackNone, FailClosed: true},
		RateLimitTypeEngagementWrite:  {Limit: 60, Window: time.Minute, Fallback: RateLimitFallbackLocalMemory, FallbackWindow: 30 * time.Second},
		RateLimitTypeEngagementRead:   {Limit: 120, Window: time.Minute, Fallback: RateLimitFallbackLocalMemory, FallbackWindow: 2 * time.Minute},
		RateLimitTypeAdminCommand:     {Limit: 10, Window: time.Minute, Fallback: RateLimitFallbackNone, FailClosed: true},
		RateLimitTypeInternalClient:   {Limit: 120, Window: time.Minute, Fallback: RateLimitFallbackNone, FailClosed: true},
	}}
}

func OpenRedisRateLimitDependency(ctx context.Context, cfg RedisConfig, rateLimit RateLimitConfig) (RedisRateLimitDependency, error) {
	client := goredis.NewClient(&goredis.Options{
		Addr:         cfg.Addr,
		Username:     cfg.Username,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolSize:     cfg.PoolSize,
	})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return RedisRateLimitDependency{}, fmt.Errorf("ping redis rate limit dependency: %w", err)
	}
	return RedisRateLimitDependency{
		Limiter: contentredis.NewFixedWindowRateLimiter(client, rateLimitRulesFromConfig(rateLimit)),
		Health:  redisHealthChecker{client: client},
		Closer:  client,
	}, nil
}

func rateLimitRulesFromConfig(config RateLimitConfig) map[ports.RateLimitType]contentredis.RateLimitRule {
	defaults := DefaultRateLimitConfig()
	rules := make(map[ports.RateLimitType]contentredis.RateLimitRule, len(defaults.Rules))
	for limitType, rule := range defaults.Rules {
		rules[limitType] = contentredis.RateLimitRule{
			Limit:          rule.Limit,
			Window:         rule.Window,
			Fallback:       rule.Fallback,
			FallbackWindow: rule.FallbackWindow,
			FailClosed:     rule.FailClosed,
		}
	}
	for limitType, rule := range config.Rules {
		if rule.Limit <= 0 || rule.Window <= 0 {
			continue
		}
		fallbackWindow := rule.FallbackWindow
		if fallbackWindow <= 0 {
			fallbackWindow = rules[limitType].FallbackWindow
		}
		rules[limitType] = contentredis.RateLimitRule{
			Limit:          rule.Limit,
			Window:         rule.Window,
			Fallback:       rule.Fallback,
			FallbackWindow: fallbackWindow,
			FailClosed:     rule.FailClosed,
		}
	}
	return rules
}

type redisHealthChecker struct {
	client goredis.Cmdable
}

func (c redisHealthChecker) Check(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}
