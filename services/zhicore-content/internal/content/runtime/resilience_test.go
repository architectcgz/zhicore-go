package runtime

import (
	"testing"
	"time"
)

func TestDefaultResilienceConfigDefinesProviderOperationMatrix(t *testing.T) {
	cfg := DefaultResilienceConfig()

	assertResiliencePolicy(t, cfg, "postgres", "post.command_tx", 3*time.Second, 1, "postgres.post.command_tx", 20)
	assertResiliencePolicy(t, cfg, "postgres", "post.query", 2*time.Second, 2, "postgres.post.query", 20)
	assertResiliencePolicy(t, cfg, "postgres", "engagement.query", 500*time.Millisecond, 1, "postgres.engagement.query", 5)
	assertResiliencePolicy(t, cfg, "mongo", "body.write_draft", 5*time.Second, 2, "mongo.body.write_draft", 10)
	assertResiliencePolicy(t, cfg, "mongo", "body.write_snapshot", 5*time.Second, 2, "mongo.body.write_snapshot", 5)
	assertResiliencePolicy(t, cfg, "mongo", "body.read_published", 3*time.Second, 2, "mongo.body.read_published", 20)
	assertResiliencePolicy(t, cfg, "redis", "rate_limit.check", 200*time.Millisecond, 1, "redis.rate_limit.check", 20)
	assertResiliencePolicy(t, cfg, "redis", "post.cache", 200*time.Millisecond, 1, "redis.post.cache", 20)
	assertResiliencePolicy(t, cfg, "redis", "engagement.cache", 200*time.Millisecond, 1, "redis.engagement.cache", 20)
	assertResiliencePolicy(t, cfg, "user-service", "profile.get_summary", 3*time.Second, 2, "user.profile.get_summary", 20)
	assertResiliencePolicy(t, cfg, "upload-service", "file.validate_ref", 3*time.Second, 2, "upload.file.validate_ref", 20)
	assertResiliencePolicy(t, cfg, "upload-service", "file.resolve_url", 2*time.Second, 2, "upload.file.resolve_url", 20)
	assertResiliencePolicy(t, cfg, "rabbitmq", "outbox.publish", 3*time.Second, 5, "rabbitmq.outbox.publish", 5)
}

func assertResiliencePolicy(t *testing.T, cfg ResilienceConfig, provider, operation string, timeout time.Duration, maxAttempts int, breakerKey string, maxInFlight int) {
	t.Helper()
	policy, ok := cfg.Policy(provider, operation)
	if !ok {
		t.Fatalf("resilience policy %s/%s missing", provider, operation)
	}
	if policy.Timeout != timeout || policy.MaxAttempts != maxAttempts ||
		policy.CircuitBreakerKey != breakerKey || policy.MaxInFlight != maxInFlight {
		t.Fatalf("resilience policy %s/%s = %#v, want timeout=%s maxAttempts=%d breaker=%s maxInFlight=%d",
			provider,
			operation,
			policy,
			timeout,
			maxAttempts,
			breakerKey,
			maxInFlight,
		)
	}
}
