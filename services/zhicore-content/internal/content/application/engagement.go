package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

const maxBatchEngagementPostIDs = 100

type EngagementCommand struct {
	Actor  *Actor
	PostID string
}

type GetPostEngagementQuery struct {
	Actor  *Actor
	PostID string
}

type BatchGetEngagementStatusQuery struct {
	Actor   *Actor
	PostIDs []string
}

type EngagementResult struct {
	PostID    string
	Liked     bool
	Favorited bool
	Stats     PostStats
}

type PostEngagementResult struct {
	PostID string
	Stats  PostStats
	Viewer *EngagementViewer
}

type BatchEngagementStatusResult struct {
	Items []EngagementStatusItem
}

type EngagementStatusItem struct {
	PostID    string
	Liked     triBool
	Favorited triBool
	Degraded  bool
}

type EngagementViewer struct {
	Liked     triBool
	Favorited triBool
	Degraded  bool
}

type triBool struct {
	value bool
	known bool
}

func KnownBool(value bool) triBool {
	return triBool{value: value, known: true}
}

func UnknownBool() triBool {
	return triBool{}
}

func (b triBool) Ptr() *bool {
	if !b.known {
		return nil
	}
	value := b.value
	return &value
}

func (s *Service) LikePost(ctx context.Context, cmd EngagementCommand) (EngagementResult, error) {
	return s.mutateEngagement(ctx, cmd, ports.EngagementActionLike)
}

func (s *Service) UnlikePost(ctx context.Context, cmd EngagementCommand) (EngagementResult, error) {
	return s.mutateEngagement(ctx, cmd, ports.EngagementActionUnlike)
}

func (s *Service) FavoritePost(ctx context.Context, cmd EngagementCommand) (EngagementResult, error) {
	return s.mutateEngagement(ctx, cmd, ports.EngagementActionFavorite)
}

func (s *Service) UnfavoritePost(ctx context.Context, cmd EngagementCommand) (EngagementResult, error) {
	return s.mutateEngagement(ctx, cmd, ports.EngagementActionUnfavorite)
}

func (s *Service) mutateEngagement(ctx context.Context, cmd EngagementCommand, action ports.EngagementAction) (EngagementResult, error) {
	if cmd.Actor == nil || cmd.Actor.UserID == 0 {
		return EngagementResult{}, ErrLoginRequired
	}
	postID := strings.TrimSpace(cmd.PostID)
	if postID == "" {
		return EngagementResult{}, ErrInvalidArgument
	}
	if err := s.enforceRateLimit(ctx, actorRateLimitRequest(ports.RateLimitTypeEngagementWrite, cmd.Actor, postID, engagementRateLimitOperation(action))); err != nil {
		return EngagementResult{}, err
	}
	if s.engagement == nil || s.tx == nil || s.clock == nil {
		return EngagementResult{}, ErrDependencyUnavailable
	}
	now := s.clock.Now()
	var record ports.EngagementMutationRecord
	err := s.tx.WithinTx(ctx, func(ctx context.Context, tx ports.Tx) error {
		var err error
		record, err = s.engagement.MutateEngagement(ctx, tx, ports.EngagementMutationInput{
			PostID:     postID,
			ActorID:    cmd.Actor.UserID,
			Action:     action,
			OccurredAt: now,
		})
		if err != nil || !record.Changed {
			return err
		}
		if s.outbox == nil || s.engagementStats == nil || s.users == nil {
			return ErrDependencyUnavailable
		}
		actorSnapshot, err := s.users.GetOwnerSnapshot(ctx, cmd.Actor.UserID)
		if err != nil || strings.TrimSpace(actorSnapshot.PublicID) == "" || strings.TrimSpace(actorSnapshot.DisplayName) == "" {
			return ErrDependencyUnavailable
		}
		event, err := newEngagementOutboxEvent(record, action, actorSnapshot, now)
		if err != nil {
			return err
		}
		if err := s.outbox.Append(ctx, tx, event); err != nil {
			return err
		}
		return s.engagementStats.Append(ctx, tx, newEngagementStatsDeltaTask(record, action, now))
	})
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			return EngagementResult{}, err
		}
		return EngagementResult{}, fmt.Errorf("%w: mutate engagement", ErrDependencyUnavailable)
	}
	if s.engagementCache != nil {
		_ = s.engagementCache.StoreMutation(ctx, record)
	}
	return mapEngagementResult(record), nil
}

func (s *Service) GetPostEngagement(ctx context.Context, query GetPostEngagementQuery) (PostEngagementResult, error) {
	postID := strings.TrimSpace(query.PostID)
	if postID == "" {
		return PostEngagementResult{}, ErrInvalidArgument
	}
	if err := s.enforceRateLimit(ctx, actorRateLimitRequest(ports.RateLimitTypeEngagementRead, query.Actor, postID, "get_post_engagement")); err != nil {
		return PostEngagementResult{}, err
	}
	if s.engagement == nil {
		return PostEngagementResult{}, ErrDependencyUnavailable
	}
	record, err := s.engagement.GetPostEngagement(ctx, postID)
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			return PostEngagementResult{}, err
		}
		return PostEngagementResult{}, fmt.Errorf("%w: get post engagement", ErrDependencyUnavailable)
	}
	result := PostEngagementResult{PostID: record.PostID, Stats: mapPostStats(record.Stats)}
	if query.Actor == nil || query.Actor.UserID == 0 {
		return result, nil
	}
	status, ok := s.readViewerStatus(ctx, query.Actor.UserID, []string{postID})
	if ok && len(status) == 1 {
		item := status[0]
		result.Viewer = &EngagementViewer{Liked: KnownBool(item.Liked), Favorited: KnownBool(item.Favorited)}
		return result, nil
	}
	result.Viewer = &EngagementViewer{Liked: UnknownBool(), Favorited: UnknownBool(), Degraded: true}
	return result, nil
}

func (s *Service) BatchGetEngagementStatus(ctx context.Context, query BatchGetEngagementStatusQuery) (BatchEngagementStatusResult, error) {
	if query.Actor == nil || query.Actor.UserID == 0 {
		return BatchEngagementStatusResult{}, ErrLoginRequired
	}
	ids, err := normalizeEngagementPostIDs(query.PostIDs)
	if err != nil {
		return BatchEngagementStatusResult{}, err
	}
	if err := s.enforceRateLimit(ctx, actorRateLimitRequest(ports.RateLimitTypeEngagementRead, query.Actor, strings.Join(ids, ","), "batch_get_engagement_status")); err != nil {
		return BatchEngagementStatusResult{}, err
	}
	status, ok := s.readViewerStatus(ctx, query.Actor.UserID, ids)
	if !ok {
		status = nil
	}
	byPostID := make(map[string]ports.EngagementStatusRecord, len(status))
	for _, item := range status {
		byPostID[item.PostID] = item
	}
	items := make([]EngagementStatusItem, 0, len(ids))
	for _, id := range ids {
		if item, ok := byPostID[id]; ok {
			items = append(items, EngagementStatusItem{
				PostID:    id,
				Liked:     KnownBool(item.Liked),
				Favorited: KnownBool(item.Favorited),
			})
			continue
		}
		items = append(items, EngagementStatusItem{PostID: id, Liked: UnknownBool(), Favorited: UnknownBool(), Degraded: true})
	}
	return BatchEngagementStatusResult{Items: items}, nil
}

func (s *Service) readViewerStatus(ctx context.Context, userID int64, postIDs []string) ([]ports.EngagementStatusRecord, bool) {
	if len(postIDs) == 0 {
		return nil, true
	}
	if s.engagementCache != nil {
		records, err := s.engagementCache.BatchGetViewerStatus(ctx, userID, postIDs)
		if err == nil && len(records) == len(postIDs) {
			return records, true
		}
	}
	if s.engagement == nil {
		return nil, false
	}
	records, err := s.engagement.BatchGetViewerStatus(ctx, userID, postIDs)
	if err != nil {
		return nil, false
	}
	if s.engagementCache != nil {
		_ = s.engagementCache.StoreViewerStatus(ctx, userID, records)
	}
	return records, true
}

func normalizeEngagementPostIDs(raw []string) ([]string, error) {
	if len(raw) == 0 || len(raw) > maxBatchEngagementPostIDs {
		return nil, ErrInvalidArgument
	}
	seen := make(map[string]struct{}, len(raw))
	ids := make([]string, 0, len(raw))
	for _, item := range raw {
		id := strings.TrimSpace(item)
		if id == "" {
			return nil, ErrInvalidArgument
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids, nil
}

func mapEngagementResult(record ports.EngagementMutationRecord) EngagementResult {
	return EngagementResult{
		PostID:    record.PostID,
		Liked:     record.Liked,
		Favorited: record.Favorited,
		Stats:     mapPostStats(record.Stats),
	}
}

func mapPostStats(record ports.PostStatsRecord) PostStats {
	return PostStats{
		ViewCount:     record.ViewCount,
		LikeCount:     record.LikeCount,
		FavoriteCount: record.FavoriteCount,
		CommentCount:  record.CommentCount,
	}
}

func newEngagementStatsDeltaTask(record ports.EngagementMutationRecord, action ports.EngagementAction, occurredAt time.Time) ports.EngagementStatsDeltaTask {
	metric, delta := engagementStatsDelta(action)
	return ports.EngagementStatsDeltaTask{
		PostInternalID: record.PostInternalID,
		PostID:         record.PostID,
		Metric:         metric,
		Delta:          delta,
		OccurredAt:     occurredAt,
	}
}

func engagementStatsDelta(action ports.EngagementAction) (string, int) {
	switch action {
	case ports.EngagementActionUnlike:
		return "LIKE", -1
	case ports.EngagementActionFavorite:
		return "FAVORITE", 1
	case ports.EngagementActionUnfavorite:
		return "FAVORITE", -1
	default:
		return "LIKE", 1
	}
}

func engagementRateLimitOperation(action ports.EngagementAction) string {
	switch action {
	case ports.EngagementActionUnlike:
		return "unlike_post"
	case ports.EngagementActionFavorite:
		return "favorite_post"
	case ports.EngagementActionUnfavorite:
		return "unfavorite_post"
	default:
		return "like_post"
	}
}

func newEngagementOutboxEvent(record ports.EngagementMutationRecord, action ports.EngagementAction, actor ports.OwnerSnapshot, occurredAt time.Time) (ports.OutboxEvent, error) {
	eventType, actorField := engagementEventShape(action)
	payload := map[string]any{
		"publicId":   record.PostID,
		"internalId": record.PostInternalID,
		"authorId":   record.AuthorID,
		actorField:   record.ActorID,
		"actor":      map[string]string{"publicId": actor.PublicID, "displayName": actor.DisplayName, "avatarUrl": actor.AvatarURL},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return ports.OutboxEvent{}, err
	}
	return ports.OutboxEvent{
		EventType:        eventType,
		PayloadVersion:   1,
		AggregateType:    "post",
		AggregateID:      record.PostID,
		AggregateVersion: record.AggregateVersion,
		PayloadJSON:      payloadJSON,
		OccurredAt:       occurredAt,
	}, nil
}

func engagementEventShape(action ports.EngagementAction) (string, string) {
	switch action {
	case ports.EngagementActionUnlike:
		return "content.post.unliked", "unlikedBy"
	case ports.EngagementActionFavorite:
		return "content.post.favorited", "favoritedBy"
	case ports.EngagementActionUnfavorite:
		return "content.post.unfavorited", "unfavoritedBy"
	default:
		return "content.post.liked", "likedBy"
	}
}
