package runtime

import (
	"fmt"
	"strings"
	"time"
)

type ResilienceConfig struct {
	Policies map[string]ResiliencePolicyConfig
}

type ResiliencePolicyConfig struct {
	Provider          string
	Operation         string
	Timeout           time.Duration
	MaxAttempts       int
	CircuitBreakerKey string
	MaxInFlight       int
}

func DefaultResilienceConfig() ResilienceConfig {
	cfg := ResilienceConfig{Policies: make(map[string]ResiliencePolicyConfig)}
	for _, policy := range []ResiliencePolicyConfig{
		{Provider: "postgres", Operation: "post.command_tx", Timeout: 3 * time.Second, MaxAttempts: 1, CircuitBreakerKey: "postgres.post.command_tx", MaxInFlight: 20},
		{Provider: "postgres", Operation: "post.query", Timeout: 2 * time.Second, MaxAttempts: 2, CircuitBreakerKey: "postgres.post.query", MaxInFlight: 20},
		{Provider: "postgres", Operation: "engagement.query", Timeout: 500 * time.Millisecond, MaxAttempts: 1, CircuitBreakerKey: "postgres.engagement.query", MaxInFlight: 5},
		{Provider: "mongo", Operation: "body.write_draft", Timeout: 5 * time.Second, MaxAttempts: 2, CircuitBreakerKey: "mongo.body.write_draft", MaxInFlight: 10},
		{Provider: "mongo", Operation: "body.write_snapshot", Timeout: 5 * time.Second, MaxAttempts: 2, CircuitBreakerKey: "mongo.body.write_snapshot", MaxInFlight: 5},
		{Provider: "mongo", Operation: "body.read_published", Timeout: 3 * time.Second, MaxAttempts: 2, CircuitBreakerKey: "mongo.body.read_published", MaxInFlight: 20},
		{Provider: "redis", Operation: "rate_limit.check", Timeout: 200 * time.Millisecond, MaxAttempts: 1, CircuitBreakerKey: "redis.rate_limit.check", MaxInFlight: 20},
		{Provider: "redis", Operation: "post.cache", Timeout: 200 * time.Millisecond, MaxAttempts: 1, CircuitBreakerKey: "redis.post.cache", MaxInFlight: 20},
		{Provider: "redis", Operation: "engagement.cache", Timeout: 200 * time.Millisecond, MaxAttempts: 1, CircuitBreakerKey: "redis.engagement.cache", MaxInFlight: 20},
		{Provider: "user-service", Operation: "profile.get_summary", Timeout: 3 * time.Second, MaxAttempts: 2, CircuitBreakerKey: "user.profile.get_summary", MaxInFlight: 20},
		{Provider: "upload-service", Operation: "file.validate_ref", Timeout: 3 * time.Second, MaxAttempts: 2, CircuitBreakerKey: "upload.file.validate_ref", MaxInFlight: 20},
		{Provider: "upload-service", Operation: "file.resolve_url", Timeout: 2 * time.Second, MaxAttempts: 2, CircuitBreakerKey: "upload.file.resolve_url", MaxInFlight: 20},
		{Provider: "rabbitmq", Operation: "outbox.publish", Timeout: 3 * time.Second, MaxAttempts: 5, CircuitBreakerKey: "rabbitmq.outbox.publish", MaxInFlight: 5},
	} {
		cfg.Policies[ResiliencePolicyKey(policy.Provider, policy.Operation)] = policy
	}
	return cfg
}

func (c ResilienceConfig) Policy(provider, operation string) (ResiliencePolicyConfig, bool) {
	policy, ok := c.Policies[ResiliencePolicyKey(provider, operation)]
	return policy, ok
}

func ResiliencePolicyKey(provider, operation string) string {
	return strings.TrimSpace(provider) + "\x00" + strings.TrimSpace(operation)
}

func (c ResilienceConfig) Validate() error {
	for key, policy := range c.Policies {
		if strings.TrimSpace(policy.Provider) == "" || strings.TrimSpace(policy.Operation) == "" {
			return fmt.Errorf("resilience policy %s: provider and operation are required", key)
		}
		if policy.Timeout <= 0 {
			return fmt.Errorf("resilience policy %s: timeout must be greater than zero", key)
		}
		if policy.MaxAttempts <= 0 {
			return fmt.Errorf("resilience policy %s: max attempts must be greater than zero", key)
		}
		if strings.TrimSpace(policy.CircuitBreakerKey) == "" {
			return fmt.Errorf("resilience policy %s: circuit breaker key is required", key)
		}
		if policy.MaxInFlight <= 0 {
			return fmt.Errorf("resilience policy %s: max in-flight must be greater than zero", key)
		}
	}
	return nil
}
