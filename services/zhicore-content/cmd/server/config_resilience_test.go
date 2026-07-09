package main

import (
	"strings"
	"testing"
	"time"

	contentruntime "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/runtime"
)

func TestLoadContentServerConfigAppliesResilienceDefaultsAndEnvOverrides(t *testing.T) {
	values := validRequiredConfigValues()
	values["ZHICORE_CONTENT_RESILIENCE_USER_SERVICE_PROFILE_GET_SUMMARY_TIMEOUT"] = "4s"
	values["ZHICORE_CONTENT_RESILIENCE_USER_SERVICE_PROFILE_GET_SUMMARY_MAX_ATTEMPTS"] = "3"
	values["ZHICORE_CONTENT_RESILIENCE_USER_SERVICE_PROFILE_GET_SUMMARY_MAX_IN_FLIGHT"] = "7"

	cfg, err := LoadContentServerConfig(mapLookup(values))
	if err != nil {
		t.Fatalf("LoadContentServerConfig() error = %v", err)
	}

	assertLoadedResiliencePolicy(t, cfg.Resilience, "user-service", "profile.get_summary", 4*time.Second, 3, "user.profile.get_summary", 7)
	assertLoadedResiliencePolicy(t, cfg.Resilience, "postgres", "post.query", 2*time.Second, 2, "postgres.post.query", 20)
}

func TestLoadContentServerConfigRejectsInvalidResilienceEnv(t *testing.T) {
	testCases := []struct {
		name    string
		envName string
		value   string
	}{
		{name: "timeout invalid", envName: "ZHICORE_CONTENT_RESILIENCE_USER_SERVICE_PROFILE_GET_SUMMARY_TIMEOUT", value: "soon"},
		{name: "timeout zero", envName: "ZHICORE_CONTENT_RESILIENCE_USER_SERVICE_PROFILE_GET_SUMMARY_TIMEOUT", value: "0s"},
		{name: "max attempts zero", envName: "ZHICORE_CONTENT_RESILIENCE_USER_SERVICE_PROFILE_GET_SUMMARY_MAX_ATTEMPTS", value: "0"},
		{name: "max in flight zero", envName: "ZHICORE_CONTENT_RESILIENCE_USER_SERVICE_PROFILE_GET_SUMMARY_MAX_IN_FLIGHT", value: "0"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			values := validRequiredConfigValues()
			values[tc.envName] = tc.value

			_, err := LoadContentServerConfig(mapLookup(values))
			if err == nil || !strings.Contains(err.Error(), tc.envName) {
				t.Fatalf("LoadContentServerConfig() error = %v, want mention %s", err, tc.envName)
			}
		})
	}
}

func assertLoadedResiliencePolicy(t *testing.T, cfg contentruntime.ResilienceConfig, provider, operation string, timeout time.Duration, maxAttempts int, breakerKey string, maxInFlight int) {
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
