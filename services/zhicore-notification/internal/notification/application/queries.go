package application

import (
	"context"
	"strings"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

const (
	defaultListSize = 20
	maxListSize     = 50
)

func (s *Service) GetUnreadCount(ctx context.Context, query GetUnreadCountQuery) (UnreadCountResult, error) {
	if err := requireActor(query.Actor); err != nil {
		return UnreadCountResult{}, err
	}
	if count, hit, err := s.unread.GetUnreadCount(ctx, query.Actor.UserID); err != nil {
		return UnreadCountResult{}, err
	} else if hit {
		return UnreadCountResult{UnreadCount: count}, nil
	}
	count, err := s.queries.GetUnreadCount(ctx, query.Actor.UserID)
	if err != nil {
		return UnreadCountResult{}, mapPortsError(err)
	}
	if err := s.unread.SetUnreadCount(ctx, query.Actor.UserID, count); err != nil {
		return UnreadCountResult{}, err
	}
	return UnreadCountResult{UnreadCount: count}, nil
}

func (s *Service) GetUnreadBreakdown(ctx context.Context, query GetUnreadBreakdownQuery) (UnreadBreakdownResult, error) {
	if err := requireActor(query.Actor); err != nil {
		return UnreadBreakdownResult{}, err
	}
	breakdown, err := s.queries.GetUnreadBreakdown(ctx, query.Actor.UserID)
	if err != nil {
		return UnreadBreakdownResult{}, mapPortsError(err)
	}
	return UnreadBreakdownResult{
		Total:       breakdown.Total,
		Interaction: breakdown.Interaction,
		Content:     breakdown.Content,
		Social:      breakdown.Social,
		System:      breakdown.System,
		Security:    breakdown.Security,
	}, nil
}

func (s *Service) ListAggregatedNotifications(ctx context.Context, query ListNotificationsQuery) (NotificationPage, error) {
	if err := requireActor(query.Actor); err != nil {
		return NotificationPage{}, err
	}
	size := query.Size
	if size <= 0 {
		size = defaultListSize
	}
	if size > maxListSize {
		size = maxListSize
	}
	page, err := s.queries.ListAggregated(ctx, ports.ListAggregatedQuery{
		RecipientID: query.Actor.UserID,
		Cursor:      query.Cursor,
		Size:        size,
		Category:    query.Category,
		UnreadOnly:  query.UnreadOnly,
	})
	if err != nil {
		return NotificationPage{}, mapPortsError(err)
	}
	items := make([]AggregatedNotification, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, AggregatedNotification{
			GroupID:           item.GroupID,
			Type:              item.Type,
			Category:          item.Category,
			TargetType:        item.TargetType,
			TargetID:          item.TargetID,
			TotalCount:        item.TotalCount,
			UnreadCount:       item.UnreadCount,
			LatestTime:        item.LatestTime,
			LatestContent:     item.LatestContent,
			ActorIDs:          item.ActorIDs,
			ActorTotalCount:   item.ActorTotalCount,
			RecentActors:      notificationActorSnapshots(item.RecentActors),
			AggregatedContent: item.AggregatedContent,
		})
	}
	return NotificationPage{Items: items, NextCursor: page.NextCursor, HasMore: page.HasMore, RepairSignal: page.RepairSignal}, nil
}

func notificationActorSnapshots(items []ports.NotificationActorSnapshot) []NotificationActorSnapshot {
	result := make([]NotificationActorSnapshot, 0, len(items))
	for _, item := range items {
		result = append(result, NotificationActorSnapshot{PublicID: item.PublicID, DisplayName: item.DisplayName, AvatarURL: item.AvatarURL})
	}
	return result
}

func (s *Service) ListNotificationGroupActors(ctx context.Context, query ListNotificationGroupActorsQuery) (NotificationActorPage, error) {
	if err := requireActor(query.Actor); err != nil {
		return NotificationActorPage{}, err
	}
	if strings.TrimSpace(query.GroupID) == "" {
		return NotificationActorPage{}, ErrInvalidRequest
	}
	size := query.Size
	if size <= 0 {
		size = defaultListSize
	}
	if size > maxListSize {
		size = maxListSize
	}
	page, err := s.queries.ListGroupActors(ctx, ports.ListGroupActorsQuery{RecipientID: query.Actor.UserID, GroupID: query.GroupID, Cursor: query.Cursor, Size: size})
	if err != nil {
		return NotificationActorPage{}, mapPortsError(err)
	}
	items := make([]NotificationActor, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, NotificationActor{PublicID: item.PublicID, DisplayName: item.DisplayName, AvatarURL: item.AvatarURL, EventCount: item.EventCount, LatestOccurredAt: item.LatestOccurredAt})
	}
	return NotificationActorPage{Items: items, NextCursor: page.NextCursor, HasMore: page.HasMore}, nil
}
