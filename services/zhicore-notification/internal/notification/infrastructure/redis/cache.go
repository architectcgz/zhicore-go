package redis

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"
)

var ErrCacheMiss = errors.New("notification cache miss")

type Client interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

type UnreadCache struct {
	client Client
	ttl    time.Duration
}

func NewUnreadCache(client Client, ttl time.Duration) *UnreadCache {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &UnreadCache{client: client, ttl: ttl}
}

func (c *UnreadCache) GetUnreadCount(ctx context.Context, userID int64) (int64, bool, error) {
	raw, err := c.client.Get(ctx, unreadKey(userID))
	if errors.Is(err, ErrCacheMiss) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("get notification unread cache: %w", err)
	}
	count, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("parse notification unread cache: %w", err)
	}
	return count, true, nil
}

func (c *UnreadCache) SetUnreadCount(ctx context.Context, userID int64, count int64) error {
	if count < 0 {
		count = 0
	}
	if err := c.client.Set(ctx, unreadKey(userID), strconv.FormatInt(count, 10), c.ttl); err != nil {
		return fmt.Errorf("set notification unread cache: %w", err)
	}
	return nil
}

func (c *UnreadCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	if err := c.client.Del(ctx, keys...); err != nil {
		return fmt.Errorf("delete notification cache: %w", err)
	}
	return nil
}

func unreadKey(userID int64) string {
	return fmt.Sprintf("notification:%d:unread", userID)
}
