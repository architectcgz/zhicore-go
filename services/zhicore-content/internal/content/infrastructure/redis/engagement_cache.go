package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

const defaultEngagementCacheTTL = 5 * time.Minute

type Client interface {
	MGet(ctx context.Context, keys ...string) ([]string, error)
	MSet(ctx context.Context, values map[string]string, ttl time.Duration) error
}

type EngagementCacheConfig struct {
	TTL time.Duration
}

type EngagementCache struct {
	client Client
	ttl    time.Duration
}

func NewEngagementCache(client Client, config EngagementCacheConfig) *EngagementCache {
	ttl := config.TTL
	if ttl <= 0 {
		ttl = defaultEngagementCacheTTL
	}
	return &EngagementCache{client: client, ttl: ttl}
}

func (c *EngagementCache) BatchGetViewerStatus(ctx context.Context, userID int64, postIDs []string) ([]ports.EngagementStatusRecord, error) {
	if err := c.ensureClient(); err != nil {
		return nil, err
	}
	if len(postIDs) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(postIDs))
	for _, postID := range postIDs {
		keys = append(keys, engagementStatusKey(userID, postID))
	}
	values, err := c.client.MGet(ctx, keys...)
	if err != nil {
		return nil, fmt.Errorf("batch get content engagement cache: %w", err)
	}
	records := make([]ports.EngagementStatusRecord, 0, len(values))
	for index, raw := range values {
		if raw == "" {
			continue
		}
		var status engagementStatusValue
		if err := json.Unmarshal([]byte(raw), &status); err != nil {
			return nil, fmt.Errorf("decode content engagement cache: %w", err)
		}
		records = append(records, ports.EngagementStatusRecord{
			PostID:    postIDs[index],
			Liked:     status.Liked,
			Favorited: status.Favorited,
		})
	}
	return records, nil
}

func (c *EngagementCache) StoreMutation(ctx context.Context, record ports.EngagementMutationRecord) error {
	return c.store(ctx, record.ActorID, []ports.EngagementStatusRecord{{
		PostID:    record.PostID,
		Liked:     record.Liked,
		Favorited: record.Favorited,
	}})
}

func (c *EngagementCache) StoreViewerStatus(ctx context.Context, userID int64, records []ports.EngagementStatusRecord) error {
	return c.store(ctx, userID, records)
}

func (c *EngagementCache) store(ctx context.Context, userID int64, records []ports.EngagementStatusRecord) error {
	if err := c.ensureClient(); err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}
	values := make(map[string]string, len(records))
	for _, record := range records {
		payload, err := json.Marshal(engagementStatusValue{Liked: record.Liked, Favorited: record.Favorited})
		if err != nil {
			return fmt.Errorf("encode content engagement cache: %w", err)
		}
		// Only confirmed viewer states are cached; unknown/degraded is a query
		// outcome owned by application fallback logic and must not become cache fact.
		values[engagementStatusKey(userID, record.PostID)] = string(payload)
	}
	if err := c.client.MSet(ctx, values, c.ttl); err != nil {
		return fmt.Errorf("store content engagement cache: %w", err)
	}
	return nil
}

func (c *EngagementCache) ensureClient() error {
	if c == nil || c.client == nil {
		return fmt.Errorf("content engagement cache client is required")
	}
	return nil
}

func engagementStatusKey(userID int64, postID string) string {
	return fmt.Sprintf("content:engagement:user:%d:post:%s", userID, postID)
}

type engagementStatusValue struct {
	Liked     bool `json:"liked"`
	Favorited bool `json:"favorited"`
}

var _ ports.EngagementCacheStore = (*EngagementCache)(nil)
