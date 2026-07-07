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
	Rules map[ports.RateLimitType]RateLimitRuleConfig
}

type RateLimitRuleConfig struct {
	Limit      int
	Window     time.Duration
	Fallback   ports.RateLimitFallback
	FailClosed bool
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
	return RateLimitConfig{Rules: map[ports.RateLimitType]RateLimitRuleConfig{
		ports.RateLimitTypePublicRead:       {Limit: 120, Window: time.Minute, Fallback: ports.RateLimitFallbackLocalMemory},
		ports.RateLimitTypeDraftWrite:       {Limit: 30, Window: time.Minute, Fallback: ports.RateLimitFallbackLocalMemory},
		ports.RateLimitTypePublishLifecycle: {Limit: 5, Window: time.Minute, Fallback: ports.RateLimitFallbackNone, FailClosed: true},
		ports.RateLimitTypeEngagementWrite:  {Limit: 60, Window: time.Minute, Fallback: ports.RateLimitFallbackLocalMemory},
		ports.RateLimitTypeEngagementRead:   {Limit: 120, Window: time.Minute, Fallback: ports.RateLimitFallbackLocalMemory},
		ports.RateLimitTypeAdminCommand:     {Limit: 10, Window: time.Minute, Fallback: ports.RateLimitFallbackNone, FailClosed: true},
		ports.RateLimitTypeInternalClient:   {Limit: 120, Window: time.Minute, Fallback: ports.RateLimitFallbackNone, FailClosed: true},
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
			Limit:      rule.Limit,
			Window:     rule.Window,
			Fallback:   rule.Fallback,
			FailClosed: rule.FailClosed,
		}
	}
	for limitType, rule := range config.Rules {
		if rule.Limit <= 0 || rule.Window <= 0 {
			continue
		}
		rules[limitType] = contentredis.RateLimitRule{
			Limit:      rule.Limit,
			Window:     rule.Window,
			Fallback:   rule.Fallback,
			FailClosed: rule.FailClosed,
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

type noopContentObserver struct{}

func NewNoopObserver() ports.ContentObserver {
	return noopContentObserver{}
}

func (noopContentObserver) ObserveRateLimitDecision(context.Context, ports.RateLimitDecision) {}
