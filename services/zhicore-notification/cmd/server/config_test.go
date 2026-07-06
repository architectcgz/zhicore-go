package main

import (
	"strings"
	"testing"
	"time"
)

func TestLoadNotificationServerConfigRequiresCoreDependenciesAndPublicIDSecret(t *testing.T) {
	_, err := LoadNotificationServerConfig(func(string) (string, bool) { return "", false })
	if err == nil {
		t.Fatal("expected missing required config to fail")
	}
}

func TestLoadNotificationServerConfigParsesRuntimeFields(t *testing.T) {
	env := map[string]string{
		envPostgresDSN:                    "postgres://user:pass@localhost:5432/zhicore_notification?sslmode=disable",
		envRedisAddr:                      "localhost:6379",
		envRabbitMQURL:                    "amqp://user:pass@localhost:5672/",
		envPublicIDActiveVersion:          "2",
		envPublicIDSecrets:                "1:old-secret,2:new-secret",
		envConsumedEventsRetention:        "168h",
		envRealtimeFanoutTimeout:          "500ms",
		envCampaignClaimTimeout:           "30s",
		envCampaignShardBatchSize:         "200",
		envCampaignMaxConcurrentShardJobs: "4",
	}

	cfg, err := LoadNotificationServerConfig(func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	})
	if err != nil {
		t.Fatalf("LoadNotificationServerConfig() error = %v", err)
	}
	if cfg.PublicID.ActiveVersion != 2 || cfg.PublicID.Secrets[1] != "old-secret" || cfg.PublicID.Secrets[2] != "new-secret" {
		t.Fatalf("public id config = %#v", cfg.PublicID)
	}
	if cfg.Consumer.ConsumedEventsRetention != 168*time.Hour || cfg.RealtimeFanout.Timeout != 500*time.Millisecond {
		t.Fatalf("runtime durations = %#v %#v", cfg.Consumer, cfg.RealtimeFanout)
	}
	if cfg.Campaign.ShardBatchSize != 200 || cfg.Campaign.MaxConcurrentShardJobs != 4 {
		t.Fatalf("campaign config = %#v", cfg.Campaign)
	}
}

func TestNotificationServerConfigRedactedSummaryDoesNotLeakSecrets(t *testing.T) {
	cfg := DefaultNotificationServerConfig()
	cfg.Postgres.DSN = "postgres://user:secret@localhost:5432/zhicore_notification"
	cfg.Redis.Addr = "localhost:6379"
	cfg.Redis.Password = "redis-secret"
	cfg.RabbitMQ.URL = "amqp://user:rabbit-secret@localhost:5672/"
	cfg.PublicID.Secrets = map[uint8]string{1: "public-id-secret"}

	summary := cfg.RedactedSummary()
	for _, leaked := range []string{"secret", "redis-secret", "rabbit-secret", "public-id-secret", "postgres://user:secret"} {
		if strings.Contains(summary, leaked) {
			t.Fatalf("redacted summary leaked %q: %s", leaked, summary)
		}
	}
	if !strings.Contains(summary, "publicID.activeVersion=1") {
		t.Fatalf("summary missing active version: %s", summary)
	}
}
