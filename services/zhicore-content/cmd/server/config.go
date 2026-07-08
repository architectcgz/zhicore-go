package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	contentruntime "github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/runtime"
)

type ContentServerConfig struct {
	ServiceName string
	HTTP        ContentHTTPConfig
	Postgres    ContentPostgresConfig
	Mongo       ContentMongoConfig
	Redis       contentruntime.RedisConfig
	RabbitMQ    ContentRabbitMQConfig
	UserService ContentDependencyConfig
	FileService ContentDependencyConfig
	Workers     ContentWorkersConfig
	RateLimit   contentruntime.RateLimitConfig
}

type ContentHTTPConfig struct {
	Addr              string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	MaxJSONBodyBytes  int64
}

type ContentPostgresConfig struct {
	DSN string
}

type ContentMongoConfig struct {
	URI            string
	Database       string
	BodyCollection string
}

type ContentRabbitMQConfig struct {
	URL                   string
	Exchange              string
	PublishConfirmTimeout time.Duration
}

type ContentDependencyConfig struct {
	BaseURL string
}

type ContentWorkersConfig struct {
	CleanupEnabled         bool
	RepairEnabled          bool
	OutboxEnabled          bool
	EngagementStatsEnabled bool
}

func (c ContentPostgresConfig) String() string {
	return fmt.Sprintf("{DSN:%s}", redactedPresence(c.DSN))
}

func (c ContentPostgresConfig) GoString() string {
	return c.String()
}

func (c ContentMongoConfig) String() string {
	return fmt.Sprintf(
		"{URI:%s Database:%s BodyCollection:%s}",
		redactedURLSummary(c.URI),
		c.Database,
		c.BodyCollection,
	)
}

func (c ContentMongoConfig) GoString() string {
	return c.String()
}

func (c ContentRabbitMQConfig) String() string {
	return fmt.Sprintf("{URL:%s Exchange:%s PublishConfirmTimeout:%s}", redactedURLSummary(c.URL), c.Exchange, c.PublishConfirmTimeout)
}

func (c ContentRabbitMQConfig) GoString() string {
	return c.String()
}

func (c ContentDependencyConfig) String() string {
	return fmt.Sprintf("{BaseURL:%s}", redactedURLSummary(c.BaseURL))
}

func (c ContentDependencyConfig) GoString() string {
	return c.String()
}

func (c ContentServerConfig) RedactedSummary() string {
	return fmt.Sprintf(
		"service=%s http.addr=%s http.readHeaderTimeout=%s http.readTimeout=%s http.writeTimeout=%s http.idleTimeout=%s http.shutdownTimeout=%s http.maxJSONBodyBytes=%d postgres=%s mongo.uri=%s mongo.database=%s mongo.bodyCollection=%s redis.addr=%s redis.db=%d redis.dialTimeout=%s redis.readTimeout=%s redis.writeTimeout=%s redis.poolSize=%d rabbitmq.url=%s rabbitmq.exchange=%s rabbitmq.publishConfirmTimeout=%s userService=%s fileService=%s workers.cleanup=%t workers.repair=%t workers.outbox=%t workers.engagementStats=%t",
		c.ServiceName,
		c.HTTP.Addr,
		c.HTTP.ReadHeaderTimeout,
		c.HTTP.ReadTimeout,
		c.HTTP.WriteTimeout,
		c.HTTP.IdleTimeout,
		c.HTTP.ShutdownTimeout,
		c.HTTP.MaxJSONBodyBytes,
		redactedPresence(c.Postgres.DSN),
		redactedURLSummary(c.Mongo.URI),
		c.Mongo.Database,
		c.Mongo.BodyCollection,
		c.Redis.Addr,
		c.Redis.DB,
		c.Redis.DialTimeout,
		c.Redis.ReadTimeout,
		c.Redis.WriteTimeout,
		c.Redis.PoolSize,
		redactedURLSummary(c.RabbitMQ.URL),
		c.RabbitMQ.Exchange,
		c.RabbitMQ.PublishConfirmTimeout,
		redactedURLSummary(c.UserService.BaseURL),
		redactedURLSummary(c.FileService.BaseURL),
		c.Workers.CleanupEnabled,
		c.Workers.RepairEnabled,
		c.Workers.OutboxEnabled,
		c.Workers.EngagementStatsEnabled,
	)
}

func (c ContentServerConfig) String() string {
	return c.RedactedSummary()
}

func (c ContentServerConfig) GoString() string {
	return c.RedactedSummary()
}

func redactedPresence(value string) string {
	if strings.TrimSpace(value) == "" {
		return "missing"
	}
	return "<redacted>"
}

func redactedURLSummary(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "<redacted>"
	}
	return parsed.Scheme + "://" + parsed.Host
}
