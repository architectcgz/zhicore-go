package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestContentServerConfigExposesRedactedSummary(t *testing.T) {
	cfg, err := LoadContentServerConfig(mapLookup(sensitiveConfigValues()))
	if err != nil {
		t.Fatalf("LoadContentServerConfig() error = %v", err)
	}

	summary := cfg.RedactedSummary()
	assertNoSensitiveConfigLeak(t, summary)
	for _, want := range []string{
		"zhicore-content",
		":19080",
		"7s",
		"9s",
		"11s",
		"22s",
		"cleanup",
		"user.internal:18081",
		"file.internal:18082",
		"mongo.local:27017",
		"mq.local:5672",
		"zhicore_content",
		"post_bodies",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("summary = %s, want mention %q", summary, want)
		}
	}
}

func TestContentServerConfigFormattingIsRedacted(t *testing.T) {
	cfg, err := LoadContentServerConfig(mapLookup(sensitiveConfigValues()))
	if err != nil {
		t.Fatalf("LoadContentServerConfig() error = %v", err)
	}

	renderedValues := []string{
		fmt.Sprint(cfg),
		fmt.Sprintf("%v", cfg),
		fmt.Sprintf("%+v", cfg),
		fmt.Sprintf("%#v", cfg),
		fmt.Sprint(cfg.Postgres),
		fmt.Sprintf("%+v", cfg.Postgres),
		fmt.Sprintf("%#v", cfg.Postgres),
		fmt.Sprint(cfg.Mongo),
		fmt.Sprintf("%+v", cfg.Mongo),
		fmt.Sprintf("%#v", cfg.Mongo),
		fmt.Sprint(cfg.RabbitMQ),
		fmt.Sprintf("%+v", cfg.RabbitMQ),
		fmt.Sprintf("%#v", cfg.RabbitMQ),
		fmt.Sprint(cfg.UserService),
		fmt.Sprintf("%+v", cfg.UserService),
		fmt.Sprintf("%#v", cfg.UserService),
		fmt.Sprint(cfg.FileService),
		fmt.Sprintf("%+v", cfg.FileService),
		fmt.Sprintf("%#v", cfg.FileService),
	}

	for _, rendered := range renderedValues {
		assertNoSensitiveConfigLeak(t, rendered)
	}
}

func sensitiveConfigValues() map[string]string {
	return map[string]string{
		"ZHICORE_CONTENT_POSTGRES_DSN":             "postgres://content:secret@127.0.0.1:5432/content",
		"ZHICORE_CONTENT_MONGO_URI":                "mongodb://content:secret@mongo.local:27017",
		"ZHICORE_CONTENT_RABBITMQ_URL":             "amqp://content:secret@mq.local:5672/",
		"ZHICORE_CONTENT_USER_SERVICE_BASE_URL":    "https://user:secret@user.internal:18081/base",
		"ZHICORE_CONTENT_FILE_SERVICE_BASE_URL":    "http://file:secret@file.internal:18082/api",
		"ZHICORE_CONTENT_HTTP_ADDR":                ":19080",
		"ZHICORE_CONTENT_HTTP_READ_HEADER_TIMEOUT": "3s",
		"ZHICORE_CONTENT_HTTP_READ_TIMEOUT":        "7s",
		"ZHICORE_CONTENT_HTTP_WRITE_TIMEOUT":       "9s",
		"ZHICORE_CONTENT_HTTP_IDLE_TIMEOUT":        "11s",
		"ZHICORE_CONTENT_HTTP_SHUTDOWN_TIMEOUT":    "22s",
		"ZHICORE_CONTENT_HTTP_MAX_JSON_BODY":       "1MiB",
		"ZHICORE_CONTENT_WORKERS_CLEANUP_ENABLED":  "true",
	}
}

func assertNoSensitiveConfigLeak(t *testing.T, rendered string) {
	t.Helper()

	for _, forbidden := range []string{
		"secret",
		"postgres://content:secret@127.0.0.1:5432/content",
		"mongodb://content:secret@mongo.local:27017",
		"amqp://content:secret@mq.local:5672/",
		"https://user:secret@user.internal:18081/base",
		"http://file:secret@file.internal:18082/api",
		"content:secret@",
		"user:secret@",
		"file:secret@",
	} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("rendered config leaked %q: %s", forbidden, rendered)
		}
	}
}
