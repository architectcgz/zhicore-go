package main

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

type NotificationServerConfig struct {
	ServiceName    string
	HTTP           NotificationHTTPConfig
	Postgres       NotificationPostgresConfig
	Redis          NotificationRedisConfig
	RabbitMQ       NotificationRabbitMQConfig
	PublicID       NotificationPublicIDConfig
	Consumer       NotificationConsumerConfig
	RealtimeFanout NotificationRealtimeFanoutConfig
	Campaign       NotificationCampaignConfig
}

type NotificationHTTPConfig struct {
	Addr              string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

type NotificationPostgresConfig struct {
	DSN string
}

type NotificationRedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type NotificationRabbitMQConfig struct {
	URL string
}

type NotificationPublicIDConfig struct {
	Prefix        string
	ActiveVersion uint8
	Secrets       map[uint8]string
}

type NotificationConsumerConfig struct {
	ConsumedEventsRetention time.Duration
}

type NotificationRealtimeFanoutConfig struct {
	Timeout time.Duration
}

type NotificationCampaignConfig struct {
	ClaimTimeout           time.Duration
	ShardBatchSize         int
	MaxConcurrentShardJobs int
}

func (c NotificationPostgresConfig) String() string {
	return fmt.Sprintf("{DSN:%s}", redactedPresence(c.DSN))
}

func (c NotificationPostgresConfig) GoString() string {
	return c.String()
}

func (c NotificationRedisConfig) String() string {
	return fmt.Sprintf("{Addr:%s Password:%s DB:%d}", c.Addr, redactedPresence(c.Password), c.DB)
}

func (c NotificationRedisConfig) GoString() string {
	return c.String()
}

func (c NotificationRabbitMQConfig) String() string {
	return fmt.Sprintf("{URL:%s}", redactedURLSummary(c.URL))
}

func (c NotificationRabbitMQConfig) GoString() string {
	return c.String()
}

func (c NotificationPublicIDConfig) String() string {
	return fmt.Sprintf("{Prefix:%s ActiveVersion:%d Secrets:%s}", c.Prefix, c.ActiveVersion, redactedSecretVersions(c.Secrets))
}

func (c NotificationPublicIDConfig) GoString() string {
	return c.String()
}

func (c NotificationServerConfig) RedactedSummary() string {
	return fmt.Sprintf(
		"service=%s http.addr=%s http.readHeaderTimeout=%s http.readTimeout=%s http.writeTimeout=%s http.idleTimeout=%s http.shutdownTimeout=%s postgres=%s redis.addr=%s redis.password=%s redis.db=%d rabbitmq.url=%s publicID.prefix=%s publicID.activeVersion=%d publicID.keyVersions=%s consumer.consumedEventsRetention=%s realtimeFanout.timeout=%s campaign.claimTimeout=%s campaign.shardBatchSize=%d campaign.maxConcurrentShardJobs=%d",
		c.ServiceName,
		c.HTTP.Addr,
		c.HTTP.ReadHeaderTimeout,
		c.HTTP.ReadTimeout,
		c.HTTP.WriteTimeout,
		c.HTTP.IdleTimeout,
		c.HTTP.ShutdownTimeout,
		redactedPresence(c.Postgres.DSN),
		c.Redis.Addr,
		redactedPresence(c.Redis.Password),
		c.Redis.DB,
		redactedURLSummary(c.RabbitMQ.URL),
		c.PublicID.Prefix,
		c.PublicID.ActiveVersion,
		redactedSecretVersions(c.PublicID.Secrets),
		c.Consumer.ConsumedEventsRetention,
		c.RealtimeFanout.Timeout,
		c.Campaign.ClaimTimeout,
		c.Campaign.ShardBatchSize,
		c.Campaign.MaxConcurrentShardJobs,
	)
}

func (c NotificationServerConfig) String() string {
	return c.RedactedSummary()
}

func (c NotificationServerConfig) GoString() string {
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

func redactedSecretVersions(secrets map[uint8]string) string {
	if len(secrets) == 0 {
		return "missing"
	}
	versions := make([]string, 0, len(secrets))
	for version := range secrets {
		versions = append(versions, fmt.Sprintf("%d:<redacted>", version))
	}
	return strings.Join(versions, ",")
}
