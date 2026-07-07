package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	notificationruntime "github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/runtime"
	_ "github.com/lib/pq"
	amqp "github.com/rabbitmq/amqp091-go"
	goredis "github.com/redis/go-redis/v9"
)

type openedNotificationRuntime struct {
	Module  *notificationruntime.Module
	Closers []namedCloser
}

func openNotificationRuntimeDependencies(ctx context.Context, cfg NotificationServerConfig) (openedNotificationRuntime, error) {
	postgresDB, err := sql.Open("postgres", cfg.Postgres.DSN)
	if err != nil {
		return openedNotificationRuntime{}, fmt.Errorf("open postgres dependency: %w", err)
	}
	closers := []namedCloser{{name: "postgres", closer: postgresDB}}
	if err := postgresDB.PingContext(ctx); err != nil {
		closeNamedClosers(closers)
		return openedNotificationRuntime{}, fmt.Errorf("ping postgres dependency: %w", err)
	}

	redisClient := goredis.NewClient(&goredis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	closers = append(closers, namedCloser{name: "redis", closer: redisClient})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		closeNamedClosers(closers)
		return openedNotificationRuntime{}, fmt.Errorf("ping redis dependency: %w", err)
	}

	rabbitConn, err := amqp.Dial(cfg.RabbitMQ.URL)
	if err != nil {
		closeNamedClosers(closers)
		return openedNotificationRuntime{}, fmt.Errorf("open rabbitmq dependency: %w", err)
	}
	rabbitChannel, err := rabbitConn.Channel()
	if err != nil {
		_ = rabbitConn.Close()
		closeNamedClosers(closers)
		return openedNotificationRuntime{}, fmt.Errorf("open rabbitmq channel: %w", err)
	}
	closers = append(closers,
		namedCloser{name: "rabbitmq channel", closer: rabbitChannel},
		namedCloser{name: "rabbitmq connection", closer: rabbitConn},
	)

	module, err := notificationruntime.BuildDefault(notificationruntime.DefaultDeps{
		ServiceName: cfg.ServiceName,
		PostgresDB:  postgresDB,
		RedisClient: redisClient,
		PublicID: notificationruntime.PublicIDConfig{
			Prefix:        cfg.PublicID.Prefix,
			ActiveVersion: cfg.PublicID.ActiveVersion,
			Secrets:       cfg.PublicID.Secrets,
		},
		UserService: notificationruntime.UserServiceConfig{
			BaseURL: cfg.UserService.BaseURL,
			Timeout: cfg.UserService.Timeout,
		},
		Campaign: notificationruntime.CampaignConfig{
			ClaimTimeout:           cfg.Campaign.ClaimTimeout,
			ShardBatchSize:         cfg.Campaign.ShardBatchSize,
			MaxConcurrentShardJobs: cfg.Campaign.MaxConcurrentShardJobs,
		},
		Health: notificationruntime.HealthDeps{
			ServiceName: cfg.ServiceName,
			Dependencies: []notificationruntime.DependencyCheck{
				postgresPingChecker{db: postgresDB},
				redisPingChecker{client: redisClient},
				rabbitMQHealthChecker{connection: rabbitConn, channel: rabbitChannel},
			},
		},
	})
	if err != nil {
		closeNamedClosers(closers)
		return openedNotificationRuntime{}, fmt.Errorf("build notification runtime module: %w", err)
	}

	return openedNotificationRuntime{Module: module, Closers: closers}, nil
}

type postgresPingChecker struct {
	db *sql.DB
}

func (c postgresPingChecker) Name() string {
	return "postgres"
}

func (c postgresPingChecker) Check(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

type redisPingChecker struct {
	client *goredis.Client
}

func (c redisPingChecker) Name() string {
	return "redis"
}

func (c redisPingChecker) Check(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

type rabbitMQHealthChecker struct {
	connection *amqp.Connection
	channel    *amqp.Channel
}

func (c rabbitMQHealthChecker) Name() string {
	return "rabbitmq"
}

func (c rabbitMQHealthChecker) Check(context.Context) error {
	if c.connection == nil || c.connection.IsClosed() {
		return errors.New("rabbitmq connection is closed")
	}
	if c.channel == nil || c.channel.IsClosed() {
		return errors.New("rabbitmq channel is closed")
	}
	return nil
}
