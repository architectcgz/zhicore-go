package application

import (
	"context"
	"strings"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

const defaultReaderPresenceTTL = 30 * time.Second

type ReaderSessionCommand struct {
	Actor     *Actor
	PostID    string
	SessionID string
}

type ReaderPresenceQuery struct {
	PostID string
}

type ReaderPresenceResult struct {
	PostID      string
	OnlineCount int
	Degraded    bool
	TTLSeconds  int
}

func (s *Service) UpsertReaderSession(ctx context.Context, cmd ReaderSessionCommand) (ReaderPresenceResult, error) {
	postID, sessionID, err := normalizePresenceIDs(cmd.PostID, cmd.SessionID)
	if err != nil {
		return ReaderPresenceResult{}, err
	}
	if cmd.Actor == nil || cmd.Actor.UserID == 0 || s.presence == nil {
		return degradedReaderPresence(postID), nil
	}
	record, err := s.presence.UpsertReaderSession(ctx, ports.ReaderSessionInput{
		PostID:    postID,
		SessionID: sessionID,
		UserID:    cmd.Actor.UserID,
		TTL:       defaultReaderPresenceTTL,
		Now:       s.now(),
	})
	if err != nil {
		return degradedReaderPresence(postID), nil
	}
	return mapReaderPresence(record, postID), nil
}

func (s *Service) DeleteReaderSession(ctx context.Context, cmd ReaderSessionCommand) error {
	postID, sessionID, err := normalizePresenceIDs(cmd.PostID, cmd.SessionID)
	if err != nil {
		return err
	}
	if cmd.Actor == nil || cmd.Actor.UserID == 0 || s.presence == nil {
		return nil
	}
	_ = s.presence.DeleteReaderSession(ctx, ports.ReaderSessionInput{
		PostID:    postID,
		SessionID: sessionID,
		UserID:    cmd.Actor.UserID,
		TTL:       defaultReaderPresenceTTL,
		Now:       s.now(),
	})
	return nil
}

func (s *Service) GetReaderPresence(ctx context.Context, query ReaderPresenceQuery) (ReaderPresenceResult, error) {
	postID := strings.TrimSpace(query.PostID)
	if postID == "" {
		return ReaderPresenceResult{}, ErrInvalidArgument
	}
	if s.presence == nil {
		return degradedReaderPresence(postID), nil
	}
	record, err := s.presence.GetReaderPresence(ctx, postID)
	if err != nil {
		return degradedReaderPresence(postID), nil
	}
	return mapReaderPresence(record, postID), nil
}

func normalizePresenceIDs(postID, sessionID string) (string, string, error) {
	postID = strings.TrimSpace(postID)
	sessionID = strings.TrimSpace(sessionID)
	if postID == "" || sessionID == "" {
		return "", "", ErrInvalidArgument
	}
	return postID, sessionID, nil
}

func (s *Service) now() time.Time {
	if s.clock == nil {
		return time.Now().UTC()
	}
	return s.clock.Now()
}

func degradedReaderPresence(postID string) ReaderPresenceResult {
	return ReaderPresenceResult{PostID: postID, OnlineCount: 0, Degraded: true, TTLSeconds: int(defaultReaderPresenceTTL.Seconds())}
}

func mapReaderPresence(record ports.ReaderPresenceRecord, fallbackPostID string) ReaderPresenceResult {
	postID := record.PostID
	if postID == "" {
		postID = fallbackPostID
	}
	ttl := record.TTL
	if ttl <= 0 {
		ttl = defaultReaderPresenceTTL
	}
	return ReaderPresenceResult{
		PostID:      postID,
		OnlineCount: record.OnlineCount,
		Degraded:    record.Degraded,
		TTLSeconds:  int(ttl.Seconds()),
	}
}
