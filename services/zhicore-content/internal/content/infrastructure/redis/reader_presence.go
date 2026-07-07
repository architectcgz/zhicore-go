package redis

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

type ReaderPresenceClient interface {
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
	CountPrefix(ctx context.Context, prefix string) (int, error)
}

type ReaderPresenceStore struct {
	client ReaderPresenceClient
}

func NewReaderPresenceStore(client ReaderPresenceClient) *ReaderPresenceStore {
	return &ReaderPresenceStore{client: client}
}

func (s *ReaderPresenceStore) UpsertReaderSession(ctx context.Context, input ports.ReaderSessionInput) (ports.ReaderPresenceRecord, error) {
	if err := s.ensureClient(); err != nil {
		return ports.ReaderPresenceRecord{}, err
	}
	ttl := input.TTL
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	if err := s.client.Set(ctx, readerPresenceSessionKey(input.PostID, input.SessionID), strconv.FormatInt(input.UserID, 10), ttl); err != nil {
		return ports.ReaderPresenceRecord{}, fmt.Errorf("upsert content reader presence: %w", err)
	}
	count, err := s.client.CountPrefix(ctx, readerPresencePostPrefix(input.PostID))
	if err != nil {
		return ports.ReaderPresenceRecord{}, fmt.Errorf("count content reader presence: %w", err)
	}
	return ports.ReaderPresenceRecord{PostID: input.PostID, OnlineCount: count, TTL: ttl}, nil
}

func (s *ReaderPresenceStore) DeleteReaderSession(ctx context.Context, input ports.ReaderSessionInput) error {
	if err := s.ensureClient(); err != nil {
		return err
	}
	if err := s.client.Del(ctx, readerPresenceSessionKey(input.PostID, input.SessionID)); err != nil {
		return fmt.Errorf("delete content reader presence: %w", err)
	}
	return nil
}

func (s *ReaderPresenceStore) GetReaderPresence(ctx context.Context, postID string) (ports.ReaderPresenceRecord, error) {
	if err := s.ensureClient(); err != nil {
		return ports.ReaderPresenceRecord{}, err
	}
	count, err := s.client.CountPrefix(ctx, readerPresencePostPrefix(postID))
	if err != nil {
		return ports.ReaderPresenceRecord{}, fmt.Errorf("count content reader presence: %w", err)
	}
	return ports.ReaderPresenceRecord{PostID: postID, OnlineCount: count, TTL: 30 * time.Second}, nil
}

func (s *ReaderPresenceStore) ensureClient() error {
	if s == nil || s.client == nil {
		return fmt.Errorf("content reader presence client is required")
	}
	return nil
}

func readerPresenceSessionKey(postID, sessionID string) string {
	return fmt.Sprintf("%s%s", readerPresencePostPrefix(postID), sessionID)
}

func readerPresencePostPrefix(postID string) string {
	return fmt.Sprintf("content:presence:post:%s:session:", postID)
}

var _ ports.ReaderPresenceStore = (*ReaderPresenceStore)(nil)
