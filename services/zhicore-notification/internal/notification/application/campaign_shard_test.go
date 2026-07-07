package application

import (
	"context"
	"testing"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

func TestCampaignShardExecutorRequestsHotActiveFollowerShard(t *testing.T) {
	activeSince := time.Date(2026, 6, 6, 10, 0, 0, 0, time.UTC)
	clock := fakeInteractionClock{now: time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)}
	campaigns := &fakeCampaignStore{
		claim: ports.ClaimedCampaignShard{
			Found:               true,
			ShardID:             8001,
			CampaignID:          7001,
			AuthorID:            1001,
			PostID:              41,
			AudienceClass:       "HOT",
			AudienceActiveSince: &activeSince,
			FollowerCursor:      "cursor-1",
		},
	}
	followers := &fakeFollowerClient{
		page: ports.FollowerShardPage{
			FollowerIDs: []int64{2001, 2002},
			NextCursor:  "cursor-2",
			HasMore:     true,
		},
	}
	executor, err := NewCampaignShardExecutor(CampaignShardExecutorDeps{
		Campaigns: campaigns,
		Followers: followers,
		Clock:     clock,
	}, CampaignShardExecutorConfig{
		WorkerID:     "worker-1",
		ClaimTimeout: 30 * time.Second,
		BatchSize:    250,
		RetryDelay:   time.Minute,
	})
	if err != nil {
		t.Fatalf("NewCampaignShardExecutor() error = %v", err)
	}

	result, err := executor.ExecuteOnce(context.Background())
	if err != nil {
		t.Fatalf("ExecuteOnce() error = %v", err)
	}
	if !result.Claimed || result.FollowerCount != 2 || !result.HasMore {
		t.Fatalf("result = %+v", result)
	}
	if campaigns.claimInput.WorkerID != "worker-1" || campaigns.claimInput.ClaimTimeout != 30*time.Second || !campaigns.claimInput.Now.Equal(clock.now) {
		t.Fatalf("claim input = %+v", campaigns.claimInput)
	}
	if followers.input.FollowingID != 1001 ||
		followers.input.AudienceClass != "HOT" ||
		followers.input.ActiveSince == nil ||
		!followers.input.ActiveSince.Equal(activeSince) ||
		followers.input.Cursor != "cursor-1" ||
		followers.input.Limit != 250 {
		t.Fatalf("follower input = %+v", followers.input)
	}
}

func TestCampaignShardExecutorRetriesWhenFollowerShardDegraded(t *testing.T) {
	clock := fakeInteractionClock{now: time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC)}
	campaigns := &fakeCampaignStore{
		claim: ports.ClaimedCampaignShard{
			Found:         true,
			ShardID:       8001,
			CampaignID:    7001,
			AuthorID:      1001,
			PostID:        41,
			AudienceClass: "HOT",
		},
	}
	followers := &fakeFollowerClient{err: ports.ErrDependencyUnavailable}
	executor, err := NewCampaignShardExecutor(CampaignShardExecutorDeps{
		Campaigns: campaigns,
		Followers: followers,
		Clock:     clock,
	}, CampaignShardExecutorConfig{
		WorkerID:     "worker-1",
		ClaimTimeout: 30 * time.Second,
		BatchSize:    250,
		RetryDelay:   2 * time.Minute,
	})
	if err != nil {
		t.Fatalf("NewCampaignShardExecutor() error = %v", err)
	}

	_, err = executor.ExecuteOnce(context.Background())
	if err != ports.ErrDependencyUnavailable {
		t.Fatalf("ExecuteOnce() error = %v, want dependency unavailable", err)
	}
	if len(campaigns.failed) != 1 {
		t.Fatalf("failed shards = %+v, want one retry mark", campaigns.failed)
	}
	failed := campaigns.failed[0]
	if failed.ShardID != 8001 || failed.ErrorCode != "USER_FOLLOWER_SHARD_DEGRADED" || failed.RetryAfter != 2*time.Minute || !failed.FailedAt.Equal(clock.now) {
		t.Fatalf("failed shard = %+v", failed)
	}
}

type fakeFollowerClient struct {
	input ports.ListFollowerShardInput
	page  ports.FollowerShardPage
	err   error
}

func (f *fakeFollowerClient) ListFollowerShard(ctx context.Context, input ports.ListFollowerShardInput) (ports.FollowerShardPage, error) {
	f.input = input
	if f.err != nil {
		return ports.FollowerShardPage{}, f.err
	}
	return f.page, nil
}
