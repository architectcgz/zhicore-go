package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

const campaignFollowerShardDegraded = "USER_FOLLOWER_SHARD_DEGRADED"

type CampaignShardExecutorDeps struct {
	Campaigns ports.CampaignRepository
	Followers ports.UserFollowerClient
	Clock     ports.Clock
}

type CampaignShardExecutorConfig struct {
	WorkerID     string
	ClaimTimeout time.Duration
	BatchSize    int
	RetryDelay   time.Duration
}

type CampaignShardExecutionResult struct {
	Claimed       bool
	ShardID       int64
	CampaignID    int64
	FollowerCount int
	NextCursor    string
	HasMore       bool
}

type CampaignShardExecutor struct {
	campaigns ports.CampaignRepository
	followers ports.UserFollowerClient
	clock     ports.Clock
	config    CampaignShardExecutorConfig
}

func NewCampaignShardExecutor(deps CampaignShardExecutorDeps, config CampaignShardExecutorConfig) (*CampaignShardExecutor, error) {
	if deps.Campaigns == nil {
		return nil, fmt.Errorf("campaign repository is required")
	}
	if deps.Followers == nil {
		return nil, fmt.Errorf("user follower client is required")
	}
	if deps.Clock == nil {
		deps.Clock = systemClock{}
	}
	if strings.TrimSpace(config.WorkerID) == "" {
		return nil, fmt.Errorf("campaign shard worker id is required")
	}
	if config.ClaimTimeout <= 0 {
		return nil, fmt.Errorf("campaign shard claim timeout must be positive")
	}
	if config.BatchSize <= 0 {
		return nil, fmt.Errorf("campaign shard batch size must be positive")
	}
	if config.RetryDelay <= 0 {
		return nil, fmt.Errorf("campaign shard retry delay must be positive")
	}
	return &CampaignShardExecutor{
		campaigns: deps.Campaigns,
		followers: deps.Followers,
		clock:     deps.Clock,
		config:    config,
	}, nil
}

func (e *CampaignShardExecutor) ExecuteOnce(ctx context.Context) (CampaignShardExecutionResult, error) {
	now := e.clock.Now()
	claim, err := e.campaigns.ClaimCampaignShard(ctx, ports.ClaimCampaignShardInput{
		WorkerID:     strings.TrimSpace(e.config.WorkerID),
		Now:          now,
		ClaimTimeout: e.config.ClaimTimeout,
	})
	if err != nil {
		return CampaignShardExecutionResult{}, err
	}
	if !claim.Found {
		return CampaignShardExecutionResult{}, nil
	}
	page, err := e.followers.ListFollowerShard(ctx, ports.ListFollowerShardInput{
		FollowingID:   claim.AuthorID,
		AudienceClass: claim.AudienceClass,
		ActiveSince:   claim.AudienceActiveSince,
		Cursor:        claim.FollowerCursor,
		Limit:         e.config.BatchSize,
	})
	if err != nil {
		// A HOT shard cannot silently fall back to ALL followers. User degraded
		// means the shard stays retryable instead of being treated as empty.
		_ = e.campaigns.FailCampaignShard(ctx, ports.FailCampaignShardInput{
			ShardID:    claim.ShardID,
			ErrorCode:  campaignFollowerShardDegraded,
			FailedAt:   now,
			RetryAfter: e.config.RetryDelay,
		})
		return CampaignShardExecutionResult{}, err
	}
	return CampaignShardExecutionResult{
		Claimed:       true,
		ShardID:       claim.ShardID,
		CampaignID:    claim.CampaignID,
		FollowerCount: len(page.FollowerIDs),
		NextCursor:    page.NextCursor,
		HasMore:       page.HasMore,
	}, nil
}
