package application

import (
	"context"
	"fmt"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

func (s *Service) MarkAllNotificationsRead(ctx context.Context, command MarkAllNotificationsReadCommand) (MarkAllNotificationsReadResult, error) {
	if err := requireActor(command.Actor); err != nil {
		return MarkAllNotificationsReadResult{}, err
	}
	readAt := s.clock.Now()
	result, err := s.commands.MarkAllRead(ctx, ports.MarkAllReadInput{RecipientID: command.Actor.UserID, ReadAt: readAt})
	if err != nil {
		return MarkAllNotificationsReadResult{}, mapPortsError(err)
	}
	if err := s.unread.Delete(ctx, unreadCacheKeys(command.Actor.UserID)...); err != nil {
		return MarkAllNotificationsReadResult{}, fmt.Errorf("invalidate notification unread cache: %w", err)
	}
	return MarkAllNotificationsReadResult{
		ReadAll:       true,
		AffectedCount: result.AffectedCount,
		ReadAt:        result.ReadAt,
	}, nil
}
