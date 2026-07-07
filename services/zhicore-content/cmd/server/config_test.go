package main

import (
	"strings"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
	contentruntime "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/runtime"
)

func TestLoadContentServerConfigRequiresRuntimeDependencies(t *testing.T) {
	_, err := LoadContentServerConfig(mapLookup(nil))
	if err == nil {
		t.Fatal("LoadContentServerConfig() error = nil, want required dependency error")
	}
	for _, want := range []string{
		"ZHICORE_CONTENT_POSTGRES_DSN",
		"ZHICORE_CONTENT_MONGO_URI",
		"ZHICORE_CONTENT_REDIS_ADDR",
		"ZHICORE_CONTENT_RABBITMQ_URL",
		"ZHICORE_CONTENT_USER_SERVICE_BASE_URL",
		"ZHICORE_CONTENT_FILE_SERVICE_BASE_URL",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("LoadContentServerConfig() error = %v, want mention %s", err, want)
		}
	}
}

func TestLoadContentServerConfigAppliesDefaultsAndEnvOverrides(t *testing.T) {
	cfg, err := LoadContentServerConfig(mapLookup(map[string]string{
		"ZHICORE_CONTENT_POSTGRES_DSN":                     "postgres://content:secret@127.0.0.1:5432/content",
		"ZHICORE_CONTENT_MONGO_URI":                        "mongodb://content:secret@127.0.0.1:27017",
		"ZHICORE_CONTENT_REDIS_ADDR":                       "127.0.0.1:6379",
		"ZHICORE_CONTENT_RABBITMQ_URL":                     "amqp://content:secret@127.0.0.1:5672/",
		"ZHICORE_CONTENT_RABBITMQ_EXCHANGE":                "zhicore.content.events",
		"ZHICORE_CONTENT_RABBITMQ_PUBLISH_CONFIRM_TIMEOUT": "4s",
		"ZHICORE_CONTENT_USER_SERVICE_BASE_URL":            "http://127.0.0.1:18081",
		"ZHICORE_CONTENT_FILE_SERVICE_BASE_URL":            "http://127.0.0.1:18082",
		"ZHICORE_CONTENT_HTTP_ADDR":                        ":19080",
		"ZHICORE_CONTENT_HTTP_READ_HEADER_TIMEOUT":         "3s",
		"ZHICORE_CONTENT_HTTP_READ_TIMEOUT":                "7s",
		"ZHICORE_CONTENT_HTTP_WRITE_TIMEOUT":               "9s",
		"ZHICORE_CONTENT_HTTP_IDLE_TIMEOUT":                "11s",
		"ZHICORE_CONTENT_HTTP_SHUTDOWN_TIMEOUT":            "22s",
		"ZHICORE_CONTENT_HTTP_MAX_JSON_BODY":               "1MiB",
		"ZHICORE_CONTENT_WORKERS_CLEANUP_ENABLED":          "true",
	}))
	if err != nil {
		t.Fatalf("LoadContentServerConfig() error = %v", err)
	}

	if cfg.ServiceName != "zhicore-content" || cfg.HTTP.Addr != ":19080" {
		t.Fatalf("service/http config = (%q, %q), want zhicore-content/:19080", cfg.ServiceName, cfg.HTTP.Addr)
	}
	if cfg.HTTP.ReadHeaderTimeout != 3*time.Second ||
		cfg.HTTP.ReadTimeout != 7*time.Second ||
		cfg.HTTP.WriteTimeout != 9*time.Second ||
		cfg.HTTP.IdleTimeout != 11*time.Second ||
		cfg.HTTP.ShutdownTimeout != 22*time.Second {
		t.Fatalf(
			"http timeouts = (%s, %s, %s, %s, %s), want 3s/7s/9s/11s/22s",
			cfg.HTTP.ReadHeaderTimeout,
			cfg.HTTP.ReadTimeout,
			cfg.HTTP.WriteTimeout,
			cfg.HTTP.IdleTimeout,
			cfg.HTTP.ShutdownTimeout,
		)
	}
	if cfg.HTTP.MaxJSONBodyBytes != 1<<20 {
		t.Fatalf("MaxJSONBodyBytes = %d, want 1MiB", cfg.HTTP.MaxJSONBodyBytes)
	}
	if !cfg.Workers.CleanupEnabled || cfg.Workers.RepairEnabled || cfg.Workers.OutboxEnabled {
		t.Fatalf("workers = %#v, want only cleanup enabled", cfg.Workers)
	}
	if cfg.Mongo.Database != "zhicore_content" || cfg.Mongo.BodyCollection != "post_bodies" {
		t.Fatalf("mongo defaults = (%q, %q), want zhicore_content/post_bodies", cfg.Mongo.Database, cfg.Mongo.BodyCollection)
	}
	if cfg.Redis.Addr != "127.0.0.1:6379" || cfg.Redis.DialTimeout != 200*time.Millisecond ||
		cfg.Redis.ReadTimeout != 200*time.Millisecond || cfg.Redis.WriteTimeout != 200*time.Millisecond ||
		cfg.Redis.PoolSize != 10 {
		t.Fatalf("redis config = %#v, want addr override and timeout/pool defaults", cfg.Redis)
	}
	if cfg.RabbitMQ.Exchange != "zhicore.content.events" || cfg.RabbitMQ.PublishConfirmTimeout != 4*time.Second {
		t.Fatalf("rabbitmq config = %#v, want overridden exchange and confirm timeout", cfg.RabbitMQ)
	}
}

func TestLoadContentServerConfigRejectsInvalidDuration(t *testing.T) {
	_, err := LoadContentServerConfig(mapLookup(map[string]string{
		"ZHICORE_CONTENT_POSTGRES_DSN":          "postgres://content:secret@127.0.0.1:5432/content",
		"ZHICORE_CONTENT_MONGO_URI":             "mongodb://content:secret@127.0.0.1:27017",
		"ZHICORE_CONTENT_RABBITMQ_URL":          "amqp://content:secret@127.0.0.1:5672/",
		"ZHICORE_CONTENT_USER_SERVICE_BASE_URL": "http://127.0.0.1:18081",
		"ZHICORE_CONTENT_FILE_SERVICE_BASE_URL": "http://127.0.0.1:18082",
		"ZHICORE_CONTENT_HTTP_READ_TIMEOUT":     "soon",
	}))
	if err == nil || !strings.Contains(err.Error(), "ZHICORE_CONTENT_HTTP_READ_TIMEOUT") {
		t.Fatalf("LoadContentServerConfig() error = %v, want invalid read timeout", err)
	}
}

func TestLoadContentServerConfigRejectsNonPositiveDuration(t *testing.T) {
	testCases := []struct {
		name    string
		envName string
		value   string
	}{
		{name: "read header timeout zero", envName: "ZHICORE_CONTENT_HTTP_READ_HEADER_TIMEOUT", value: "0s"},
		{name: "read header timeout negative", envName: "ZHICORE_CONTENT_HTTP_READ_HEADER_TIMEOUT", value: "-1s"},
		{name: "read timeout zero", envName: "ZHICORE_CONTENT_HTTP_READ_TIMEOUT", value: "0s"},
		{name: "read timeout negative", envName: "ZHICORE_CONTENT_HTTP_READ_TIMEOUT", value: "-1s"},
		{name: "write timeout zero", envName: "ZHICORE_CONTENT_HTTP_WRITE_TIMEOUT", value: "0s"},
		{name: "write timeout negative", envName: "ZHICORE_CONTENT_HTTP_WRITE_TIMEOUT", value: "-1s"},
		{name: "idle timeout zero", envName: "ZHICORE_CONTENT_HTTP_IDLE_TIMEOUT", value: "0s"},
		{name: "idle timeout negative", envName: "ZHICORE_CONTENT_HTTP_IDLE_TIMEOUT", value: "-1s"},
		{name: "shutdown timeout zero", envName: "ZHICORE_CONTENT_HTTP_SHUTDOWN_TIMEOUT", value: "0s"},
		{name: "shutdown timeout negative", envName: "ZHICORE_CONTENT_HTTP_SHUTDOWN_TIMEOUT", value: "-1s"},
		{name: "rabbitmq publish confirm timeout zero", envName: "ZHICORE_CONTENT_RABBITMQ_PUBLISH_CONFIRM_TIMEOUT", value: "0s"},
		{name: "rabbitmq publish confirm timeout negative", envName: "ZHICORE_CONTENT_RABBITMQ_PUBLISH_CONFIRM_TIMEOUT", value: "-1s"},
		{name: "redis dial timeout zero", envName: "ZHICORE_CONTENT_REDIS_DIAL_TIMEOUT", value: "0s"},
		{name: "redis dial timeout negative", envName: "ZHICORE_CONTENT_REDIS_DIAL_TIMEOUT", value: "-1s"},
		{name: "redis read timeout zero", envName: "ZHICORE_CONTENT_REDIS_READ_TIMEOUT", value: "0s"},
		{name: "redis read timeout negative", envName: "ZHICORE_CONTENT_REDIS_READ_TIMEOUT", value: "-1s"},
		{name: "redis write timeout zero", envName: "ZHICORE_CONTENT_REDIS_WRITE_TIMEOUT", value: "0s"},
		{name: "redis write timeout negative", envName: "ZHICORE_CONTENT_REDIS_WRITE_TIMEOUT", value: "-1s"},
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

func TestLoadContentServerConfigRejectsShutdownTimeoutAboveThirtySeconds(t *testing.T) {
	values := validRequiredConfigValues()
	values["ZHICORE_CONTENT_HTTP_SHUTDOWN_TIMEOUT"] = "31s"

	_, err := LoadContentServerConfig(mapLookup(values))
	if err == nil || !strings.Contains(err.Error(), "ZHICORE_CONTENT_HTTP_SHUTDOWN_TIMEOUT") {
		t.Fatalf("LoadContentServerConfig() error = %v, want mention ZHICORE_CONTENT_HTTP_SHUTDOWN_TIMEOUT", err)
	}
}

func TestLoadContentServerConfigRejectsInvalidMaxJSONBody(t *testing.T) {
	testCases := []struct {
		name  string
		value string
	}{
		{name: "missing explicit unit", value: "1048576"},
		{name: "zero bytes", value: "0B"},
		{name: "negative bytes", value: "-1MiB"},
		{name: "overflow", value: "9223372036854775807GiB"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			values := validRequiredConfigValues()
			values["ZHICORE_CONTENT_HTTP_MAX_JSON_BODY"] = tc.value

			_, err := LoadContentServerConfig(mapLookup(values))
			if err == nil || !strings.Contains(err.Error(), "ZHICORE_CONTENT_HTTP_MAX_JSON_BODY") {
				t.Fatalf("LoadContentServerConfig() error = %v, want mention ZHICORE_CONTENT_HTTP_MAX_JSON_BODY", err)
			}
		})
	}
}

func TestLoadContentServerConfigRejectsPresentButEmptyEnv(t *testing.T) {
	testCases := []struct {
		name    string
		envName string
	}{
		{name: "http addr", envName: "ZHICORE_CONTENT_HTTP_ADDR"},
		{name: "http read header timeout", envName: "ZHICORE_CONTENT_HTTP_READ_HEADER_TIMEOUT"},
		{name: "http read timeout", envName: "ZHICORE_CONTENT_HTTP_READ_TIMEOUT"},
		{name: "http write timeout", envName: "ZHICORE_CONTENT_HTTP_WRITE_TIMEOUT"},
		{name: "http idle timeout", envName: "ZHICORE_CONTENT_HTTP_IDLE_TIMEOUT"},
		{name: "http shutdown timeout", envName: "ZHICORE_CONTENT_HTTP_SHUTDOWN_TIMEOUT"},
		{name: "http max body", envName: "ZHICORE_CONTENT_HTTP_MAX_JSON_BODY"},
		{name: "cleanup enabled", envName: "ZHICORE_CONTENT_WORKERS_CLEANUP_ENABLED"},
		{name: "repair enabled", envName: "ZHICORE_CONTENT_WORKERS_REPAIR_ENABLED"},
		{name: "outbox enabled", envName: "ZHICORE_CONTENT_WORKERS_OUTBOX_ENABLED"},
		{name: "postgres dsn", envName: "ZHICORE_CONTENT_POSTGRES_DSN"},
		{name: "mongo uri", envName: "ZHICORE_CONTENT_MONGO_URI"},
		{name: "redis addr", envName: "ZHICORE_CONTENT_REDIS_ADDR"},
		{name: "redis username", envName: "ZHICORE_CONTENT_REDIS_USERNAME"},
		{name: "redis password", envName: "ZHICORE_CONTENT_REDIS_PASSWORD"},
		{name: "redis db", envName: "ZHICORE_CONTENT_REDIS_DB"},
		{name: "redis dial timeout", envName: "ZHICORE_CONTENT_REDIS_DIAL_TIMEOUT"},
		{name: "redis read timeout", envName: "ZHICORE_CONTENT_REDIS_READ_TIMEOUT"},
		{name: "redis write timeout", envName: "ZHICORE_CONTENT_REDIS_WRITE_TIMEOUT"},
		{name: "redis pool size", envName: "ZHICORE_CONTENT_REDIS_POOL_SIZE"},
		{name: "rabbitmq url", envName: "ZHICORE_CONTENT_RABBITMQ_URL"},
		{name: "rabbitmq exchange", envName: "ZHICORE_CONTENT_RABBITMQ_EXCHANGE"},
		{name: "rabbitmq publish confirm timeout", envName: "ZHICORE_CONTENT_RABBITMQ_PUBLISH_CONFIRM_TIMEOUT"},
		{name: "user service base url", envName: "ZHICORE_CONTENT_USER_SERVICE_BASE_URL"},
		{name: "file service base url", envName: "ZHICORE_CONTENT_FILE_SERVICE_BASE_URL"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			values := validRequiredConfigValues()
			values[tc.envName] = ""

			_, err := LoadContentServerConfig(mapLookup(values))
			if err == nil {
				t.Fatalf("LoadContentServerConfig() error = nil, want mention %s", tc.envName)
			}
			if err.Error() != tc.envName+": value must not be empty" {
				t.Fatalf("LoadContentServerConfig() error = %v, want exact empty error for %s", err, tc.envName)
			}
		})
	}
}

func TestLoadContentServerConfigAcceptsStrictBoolLiterals(t *testing.T) {
	values := validRequiredConfigValues()
	values["ZHICORE_CONTENT_WORKERS_CLEANUP_ENABLED"] = "true"
	values["ZHICORE_CONTENT_WORKERS_REPAIR_ENABLED"] = "false"
	values["ZHICORE_CONTENT_WORKERS_OUTBOX_ENABLED"] = "false"

	cfg, err := LoadContentServerConfig(mapLookup(values))
	if err != nil {
		t.Fatalf("LoadContentServerConfig() error = %v", err)
	}
	if !cfg.Workers.CleanupEnabled || cfg.Workers.RepairEnabled || cfg.Workers.OutboxEnabled {
		t.Fatalf("workers = %#v, want cleanup=true repair=false outbox=false", cfg.Workers)
	}
}

func TestLoadContentServerConfigRejectsNonCanonicalBoolValues(t *testing.T) {
	for _, value := range []string{"TRUE", "1", "t", "yes"} {
		t.Run(value, func(t *testing.T) {
			values := validRequiredConfigValues()
			values["ZHICORE_CONTENT_WORKERS_CLEANUP_ENABLED"] = value

			_, err := LoadContentServerConfig(mapLookup(values))
			if err == nil || !strings.Contains(err.Error(), "ZHICORE_CONTENT_WORKERS_CLEANUP_ENABLED") {
				t.Fatalf("LoadContentServerConfig() error = %v, want mention ZHICORE_CONTENT_WORKERS_CLEANUP_ENABLED", err)
			}
		})
	}
}

func TestLoadContentServerConfigRejectsInvalidRequiredServiceURLs(t *testing.T) {
	testCases := []struct {
		name    string
		envName string
		value   string
	}{
		{name: "user missing scheme", envName: "ZHICORE_CONTENT_USER_SERVICE_BASE_URL", value: "127.0.0.1:18081"},
		{name: "user non http scheme", envName: "ZHICORE_CONTENT_USER_SERVICE_BASE_URL", value: "ftp://user:secret@127.0.0.1:18081"},
		{name: "user bad url", envName: "ZHICORE_CONTENT_USER_SERVICE_BASE_URL", value: "http://user:secret@"},
		{name: "user empty hostname", envName: "ZHICORE_CONTENT_USER_SERVICE_BASE_URL", value: "http://user:secret@:18081"},
		{name: "file missing scheme", envName: "ZHICORE_CONTENT_FILE_SERVICE_BASE_URL", value: "127.0.0.1:18082"},
		{name: "file non http scheme", envName: "ZHICORE_CONTENT_FILE_SERVICE_BASE_URL", value: "ftp://file:secret@127.0.0.1:18082"},
		{name: "file bad url", envName: "ZHICORE_CONTENT_FILE_SERVICE_BASE_URL", value: "https://file:secret@"},
		{name: "file empty hostname", envName: "ZHICORE_CONTENT_FILE_SERVICE_BASE_URL", value: "https://file:secret@:18082"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			values := validRequiredConfigValues()
			values[tc.envName] = tc.value

			_, err := LoadContentServerConfig(mapLookup(values))
			if err == nil {
				t.Fatalf("LoadContentServerConfig() error = nil, want invalid url for %s", tc.envName)
			}
			if !strings.Contains(err.Error(), tc.envName) {
				t.Fatalf("LoadContentServerConfig() error = %v, want mention %s", err, tc.envName)
			}
			if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), tc.value) {
				t.Fatalf("LoadContentServerConfig() error leaked raw url: %v", err)
			}
		})
	}
}

func TestLoadContentServerConfigRejectsInvalidRequiredDependencyURLs(t *testing.T) {
	testCases := []struct {
		name    string
		envName string
		value   string
	}{
		{name: "mongo missing scheme", envName: "ZHICORE_CONTENT_MONGO_URI", value: "content:secret@127.0.0.1:27017"},
		{name: "mongo wrong scheme", envName: "ZHICORE_CONTENT_MONGO_URI", value: "http://content:secret@127.0.0.1:27017"},
		{name: "mongo empty hostname", envName: "ZHICORE_CONTENT_MONGO_URI", value: "mongodb://content:secret@:27017"},
		{name: "rabbitmq missing scheme", envName: "ZHICORE_CONTENT_RABBITMQ_URL", value: "content:secret@127.0.0.1:5672/"},
		{name: "rabbitmq wrong scheme", envName: "ZHICORE_CONTENT_RABBITMQ_URL", value: "https://content:secret@127.0.0.1:5672/"},
		{name: "rabbitmq empty hostname", envName: "ZHICORE_CONTENT_RABBITMQ_URL", value: "amqp://content:secret@:5672/"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			values := validRequiredConfigValues()
			values[tc.envName] = tc.value

			_, err := LoadContentServerConfig(mapLookup(values))
			if err == nil {
				t.Fatalf("LoadContentServerConfig() error = nil, want invalid url for %s", tc.envName)
			}
			if !strings.Contains(err.Error(), tc.envName) {
				t.Fatalf("LoadContentServerConfig() error = %v, want mention %s", err, tc.envName)
			}
			if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), tc.value) {
				t.Fatalf("LoadContentServerConfig() error leaked raw url: %v", err)
			}
		})
	}
}

func TestLoadContentServerConfigAppliesHTTPTimeoutDefaults(t *testing.T) {
	cfg, err := LoadContentServerConfig(mapLookup(validRequiredConfigValues()))
	if err != nil {
		t.Fatalf("LoadContentServerConfig() error = %v", err)
	}

	if cfg.HTTP.Addr != ":8080" {
		t.Fatalf("HTTP.Addr = %q, want :8080", cfg.HTTP.Addr)
	}
	if cfg.HTTP.ReadHeaderTimeout != 2*time.Second ||
		cfg.HTTP.ReadTimeout != 5*time.Second ||
		cfg.HTTP.WriteTimeout != 10*time.Second ||
		cfg.HTTP.IdleTimeout != 60*time.Second ||
		cfg.HTTP.ShutdownTimeout != 20*time.Second {
		t.Fatalf("http defaults = %#v, want 2s/5s/10s/60s/20s", cfg.HTTP)
	}
	if cfg.HTTP.MaxJSONBodyBytes != 1<<20 {
		t.Fatalf("MaxJSONBodyBytes = %d, want 1MiB", cfg.HTTP.MaxJSONBodyBytes)
	}
	if cfg.RabbitMQ.Exchange != "zhicore.events" || cfg.RabbitMQ.PublishConfirmTimeout != 3*time.Second {
		t.Fatalf("rabbitmq defaults = %#v, want zhicore.events/3s", cfg.RabbitMQ)
	}
}

func TestLoadContentServerConfigAppliesRateLimitDefaults(t *testing.T) {
	cfg, err := LoadContentServerConfig(mapLookup(validRequiredConfigValues()))
	if err != nil {
		t.Fatalf("LoadContentServerConfig() error = %v", err)
	}

	assertRateLimitRule(t, cfg.RateLimit, ports.RateLimitTypePublicRead, 120, time.Minute, ports.RateLimitFallbackLocalMemory, false)
	assertRateLimitRule(t, cfg.RateLimit, ports.RateLimitTypeDraftWrite, 30, time.Minute, ports.RateLimitFallbackLocalMemory, false)
	assertRateLimitRule(t, cfg.RateLimit, ports.RateLimitTypePublishLifecycle, 5, time.Minute, ports.RateLimitFallbackNone, true)
	assertRateLimitRule(t, cfg.RateLimit, ports.RateLimitTypeEngagementWrite, 60, time.Minute, ports.RateLimitFallbackLocalMemory, false)
	assertRateLimitRule(t, cfg.RateLimit, ports.RateLimitTypeEngagementRead, 120, time.Minute, ports.RateLimitFallbackLocalMemory, false)
	assertRateLimitRule(t, cfg.RateLimit, ports.RateLimitTypeAdminCommand, 10, time.Minute, ports.RateLimitFallbackNone, true)
	assertRateLimitRule(t, cfg.RateLimit, ports.RateLimitTypeInternalClient, 120, time.Minute, ports.RateLimitFallbackNone, true)
}

func TestLoadContentServerConfigAppliesRateLimitEnvOverrides(t *testing.T) {
	values := validRequiredConfigValues()
	values["ZHICORE_CONTENT_RATE_LIMIT_PUBLIC_READ_LIMIT"] = "42"
	values["ZHICORE_CONTENT_RATE_LIMIT_PUBLIC_READ_WINDOW"] = "30s"
	values["ZHICORE_CONTENT_RATE_LIMIT_PUBLIC_READ_FALLBACK"] = "gateway_only"
	values["ZHICORE_CONTENT_RATE_LIMIT_PUBLIC_READ_FAIL_CLOSED"] = "true"

	cfg, err := LoadContentServerConfig(mapLookup(values))
	if err != nil {
		t.Fatalf("LoadContentServerConfig() error = %v", err)
	}

	assertRateLimitRule(t, cfg.RateLimit, ports.RateLimitTypePublicRead, 42, 30*time.Second, ports.RateLimitFallbackGatewayOnly, true)
	assertRateLimitRule(t, cfg.RateLimit, ports.RateLimitTypeDraftWrite, 30, time.Minute, ports.RateLimitFallbackLocalMemory, false)
}

func TestLoadContentServerConfigRejectsInvalidRateLimitEnv(t *testing.T) {
	testCases := []struct {
		name    string
		envName string
		value   string
	}{
		{name: "limit not int", envName: "ZHICORE_CONTENT_RATE_LIMIT_PUBLIC_READ_LIMIT", value: "many"},
		{name: "limit zero", envName: "ZHICORE_CONTENT_RATE_LIMIT_PUBLIC_READ_LIMIT", value: "0"},
		{name: "window invalid", envName: "ZHICORE_CONTENT_RATE_LIMIT_PUBLIC_READ_WINDOW", value: "soon"},
		{name: "window zero", envName: "ZHICORE_CONTENT_RATE_LIMIT_PUBLIC_READ_WINDOW", value: "0s"},
		{name: "fallback invalid", envName: "ZHICORE_CONTENT_RATE_LIMIT_PUBLIC_READ_FALLBACK", value: "redis"},
		{name: "fail closed non canonical", envName: "ZHICORE_CONTENT_RATE_LIMIT_PUBLIC_READ_FAIL_CLOSED", value: "TRUE"},
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

func assertRateLimitRule(t *testing.T, cfg contentruntime.RateLimitConfig, limitType ports.RateLimitType, limit int, window time.Duration, fallback ports.RateLimitFallback, failClosed bool) {
	t.Helper()
	rule, ok := cfg.Rules[limitType]
	if !ok {
		t.Fatalf("rate limit rule %s missing", limitType)
	}
	if rule.Limit != limit || rule.Window != window || rule.Fallback != fallback || rule.FailClosed != failClosed {
		t.Fatalf("rate limit rule %s = %#v, want limit=%d window=%s fallback=%s failClosed=%t", limitType, rule, limit, window, fallback, failClosed)
	}
}

func validRequiredConfigValues() map[string]string {
	return map[string]string{
		"ZHICORE_CONTENT_POSTGRES_DSN":          "postgres://content:secret@127.0.0.1:5432/content",
		"ZHICORE_CONTENT_MONGO_URI":             "mongodb://content:secret@127.0.0.1:27017",
		"ZHICORE_CONTENT_REDIS_ADDR":            "127.0.0.1:6379",
		"ZHICORE_CONTENT_RABBITMQ_URL":          "amqp://content:secret@127.0.0.1:5672/",
		"ZHICORE_CONTENT_USER_SERVICE_BASE_URL": "http://127.0.0.1:18081",
		"ZHICORE_CONTENT_FILE_SERVICE_BASE_URL": "http://127.0.0.1:18082",
	}
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
