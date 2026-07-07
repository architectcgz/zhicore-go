package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	notificationruntime "github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/runtime"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/application"
	notificationclients "github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/infrastructure/clients"
	notificationpostgres "github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/infrastructure/postgres"
	notificationpublicid "github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/infrastructure/publicid"
	notificationredis "github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/infrastructure/redis"
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

	codec, err := notificationpublicid.NewCodec(notificationpublicid.Config{
		Prefix:        cfg.PublicID.Prefix,
		ActiveVersion: cfg.PublicID.ActiveVersion,
		Secrets:       cfg.PublicID.Secrets,
	})
	if err != nil {
		closeNamedClosers(closers)
		return openedNotificationRuntime{}, fmt.Errorf("build notification public id codec: %w", err)
	}

	store := notificationpostgres.NewStoreWithCodec(postgresDB, codec)
	service, err := application.NewService(application.Dependencies{
		Commands:   store,
		Queries:    store,
		Unread:     notificationredis.NewUnreadCache(redisClientAdapter{client: redisClient}, 5*time.Minute),
		IDs:        codec,
		Settings:   store,
		Deliveries: store,
	})
	if err != nil {
		closeNamedClosers(closers)
		return openedNotificationRuntime{}, fmt.Errorf("build notification application service: %w", err)
	}

	followers := notificationclients.NewUserFollowerClient(notificationclients.UserFollowerClientConfig{
		BaseURL: cfg.UserService.BaseURL,
		Timeout: cfg.UserService.Timeout,
	})
	campaignWorkers, err := buildCampaignShardWorkers(cfg.Campaign.MaxConcurrentShardJobs, time.Second, func(workerID string) (func(context.Context) error, error) {
		executor, err := application.NewCampaignShardExecutor(application.CampaignShardExecutorDeps{
			Campaigns: store,
			Followers: followers,
		}, application.CampaignShardExecutorConfig{
			WorkerID:     workerID,
			ClaimTimeout: cfg.Campaign.ClaimTimeout,
			BatchSize:    cfg.Campaign.ShardBatchSize,
			RetryDelay:   cfg.Campaign.ClaimTimeout,
		})
		if err != nil {
			return nil, err
		}
		return func(ctx context.Context) error {
			_, err := executor.ExecuteOnce(ctx)
			return err
		}, nil
	})
	if err != nil {
		closeNamedClosers(closers)
		return openedNotificationRuntime{}, fmt.Errorf("build notification campaign shard workers: %w", err)
	}

	module, err := notificationruntime.Build(notificationruntime.Deps{
		Service: service,
		Health: notificationruntime.HealthDeps{
			ServiceName: cfg.ServiceName,
			Dependencies: []notificationruntime.DependencyCheck{
				postgresPingChecker{db: postgresDB},
				redisPingChecker{client: redisClient},
				rabbitMQHealthChecker{connection: rabbitConn, channel: rabbitChannel},
			},
			Workers: []notificationruntime.WorkerDescriptor{
				{Name: "cleanup_consumed_events", Enabled: false, Ready: false},
			},
		},
		Workers: campaignWorkers,
	})
	if err != nil {
		closeNamedClosers(closers)
		return openedNotificationRuntime{}, fmt.Errorf("build notification runtime module: %w", err)
	}

	return openedNotificationRuntime{Module: module, Closers: closers}, nil
}

type campaignShardRunOnceFactory func(workerID string) (func(context.Context) error, error)

func buildCampaignShardWorkers(maxJobs int, interval time.Duration, buildRunOnce campaignShardRunOnceFactory) ([]notificationruntime.WorkerRunner, error) {
	if maxJobs <= 0 {
		return nil, fmt.Errorf("campaign max concurrent shard jobs must be greater than zero")
	}
	if buildRunOnce == nil {
		return nil, fmt.Errorf("campaign shard worker factory is required")
	}
	workers := make([]notificationruntime.WorkerRunner, 0, maxJobs)
	for index := 1; index <= maxJobs; index++ {
		// Each worker needs a stable unique identity because shard completion and
		// failure use the worker id as part of the DB lease token.
		workerID := fmt.Sprintf("zhicore-notification:campaign-shard:%d", index)
		runOnce, err := buildRunOnce(workerID)
		if err != nil {
			return nil, err
		}
		workers = append(workers, notificationruntime.NewLoopWorker(fmt.Sprintf("campaign_shard_%d", index), interval, runOnce))
	}
	return workers, nil
}

type redisClientAdapter struct {
	client *goredis.Client
}

func (a redisClientAdapter) Get(ctx context.Context, key string) (string, error) {
	value, err := a.client.Get(ctx, key).Result()
	if errors.Is(err, goredis.Nil) {
		return "", notificationredis.ErrCacheMiss
	}
	return value, err
}

func (a redisClientAdapter) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return a.client.Set(ctx, key, value, ttl).Err()
}

func (a redisClientAdapter) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return a.client.Del(ctx, keys...).Err()
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
