package runtime

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/application"
	notificationclients "github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/infrastructure/clients"
	notificationpostgres "github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/infrastructure/postgres"
	notificationpublicid "github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/infrastructure/publicid"
	notificationredis "github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/infrastructure/redis"
	goredis "github.com/redis/go-redis/v9"
)

type PublicIDConfig struct {
	Prefix        string
	ActiveVersion uint8
	Secrets       map[uint8]string
}

type UserServiceConfig struct {
	BaseURL string
	Timeout time.Duration
}

type CampaignConfig struct {
	ClaimTimeout             time.Duration
	ShardBatchSize           int
	MaxConcurrentShardJobs   int
	WorkerInterval           time.Duration
	UnreadCacheRetentionTime time.Duration
}

type DefaultDeps struct {
	ServiceName string
	PostgresDB  *sql.DB
	RedisClient *goredis.Client
	PublicID    PublicIDConfig
	UserService UserServiceConfig
	Campaign    CampaignConfig
	Health      HealthDeps
}

func BuildDefault(deps DefaultDeps) (*Module, error) {
	codec, err := notificationpublicid.NewCodec(notificationpublicid.Config{
		Prefix:        deps.PublicID.Prefix,
		ActiveVersion: deps.PublicID.ActiveVersion,
		Secrets:       deps.PublicID.Secrets,
	})
	if err != nil {
		return nil, fmt.Errorf("build notification public id codec: %w", err)
	}

	store := notificationpostgres.NewStoreWithCodec(deps.PostgresDB, codec)
	service, err := application.NewService(application.Dependencies{
		Commands:   store,
		Queries:    store,
		Unread:     notificationredis.NewUnreadCache(redisClientAdapter{client: deps.RedisClient}, unreadCacheTTL(deps.Campaign.UnreadCacheRetentionTime)),
		IDs:        codec,
		Settings:   store,
		Deliveries: store,
	})
	if err != nil {
		return nil, fmt.Errorf("build notification application service: %w", err)
	}

	followers := notificationclients.NewUserFollowerClient(notificationclients.UserFollowerClientConfig{
		BaseURL: deps.UserService.BaseURL,
		Timeout: deps.UserService.Timeout,
	})
	campaignWorkers, err := BuildCampaignShardWorkers(deps.Campaign.MaxConcurrentShardJobs, workerInterval(deps.Campaign.WorkerInterval), func(workerID string) (func(context.Context) error, error) {
		executor, err := application.NewCampaignShardExecutor(application.CampaignShardExecutorDeps{
			Campaigns: store,
			Followers: followers,
		}, application.CampaignShardExecutorConfig{
			WorkerID:     workerID,
			ClaimTimeout: deps.Campaign.ClaimTimeout,
			BatchSize:    deps.Campaign.ShardBatchSize,
			RetryDelay:   deps.Campaign.ClaimTimeout,
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
		return nil, fmt.Errorf("build notification campaign shard workers: %w", err)
	}

	health := deps.Health
	if health.ServiceName == "" {
		health.ServiceName = deps.ServiceName
	}
	health.Workers = append([]WorkerDescriptor{
		{Name: "cleanup_consumed_events", Enabled: false, Ready: false},
	}, health.Workers...)
	return Build(Deps{
		Service: service,
		Health:  health,
		Workers: campaignWorkers,
	})
}

type campaignShardRunOnceFactory func(workerID string) (func(context.Context) error, error)

func BuildCampaignShardWorkers(maxJobs int, interval time.Duration, buildRunOnce campaignShardRunOnceFactory) ([]WorkerRunner, error) {
	if maxJobs <= 0 {
		return nil, fmt.Errorf("campaign max concurrent shard jobs must be greater than zero")
	}
	if buildRunOnce == nil {
		return nil, fmt.Errorf("campaign shard worker factory is required")
	}
	workers := make([]WorkerRunner, 0, maxJobs)
	for index := 1; index <= maxJobs; index++ {
		// The worker id is part of the shard lease token; reusing it would let
		// one worker complete or fail a shard claimed by another worker.
		workerID := fmt.Sprintf("zhicore-notification:campaign-shard:%d", index)
		runOnce, err := buildRunOnce(workerID)
		if err != nil {
			return nil, err
		}
		workers = append(workers, NewLoopWorker(fmt.Sprintf("campaign_shard_%d", index), interval, runOnce))
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

func workerInterval(interval time.Duration) time.Duration {
	if interval <= 0 {
		return time.Second
	}
	return interval
}

func unreadCacheTTL(ttl time.Duration) time.Duration {
	if ttl <= 0 {
		return 5 * time.Minute
	}
	return ttl
}
