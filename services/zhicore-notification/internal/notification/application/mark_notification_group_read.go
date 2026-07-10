package application

import (
	"context"
	"strings"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

// MarkNotificationGroupRead marks only the caller-owned aggregation. The repository owns the
// transaction so notification rows, group counters and user stats change atomically.
func (s *Service) MarkNotificationGroupRead(ctx context.Context, command MarkNotificationGroupReadCommand) (MarkNotificationGroupReadResult, error) {
	if err := requireActor(command.Actor); err != nil {
		return MarkNotificationGroupReadResult{}, err
	}
	groupID := strings.TrimSpace(command.GroupID)
	if groupID == "" {
		return MarkNotificationGroupReadResult{}, ErrInvalidRequest
	}
	readAt := s.clock.Now()
	result, err := s.commands.MarkGroupRead(ctx, ports.MarkGroupReadInput{RecipientID: command.Actor.UserID, GroupID: groupID, ReadAt: readAt})
	if err != nil {
		return MarkNotificationGroupReadResult{}, mapPortsError(err)
	}
	if err := s.unread.Delete(ctx, unreadCacheKeys(command.Actor.UserID)...); err != nil {
		return MarkNotificationGroupReadResult{}, err
	}
	return MarkNotificationGroupReadResult{GroupID: result.GroupID, Read: true, ChangedCount: result.ChangedCount, UnreadCount: result.UnreadCount, ReadAt: result.ReadAt}, nil
}
