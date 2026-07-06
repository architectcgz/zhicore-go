package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

func (s *Service) MarkNotificationRead(ctx context.Context, command MarkNotificationReadCommand) (MarkNotificationReadResult, error) {
	if err := requireActor(command.Actor); err != nil {
		return MarkNotificationReadResult{}, err
	}
	publicID := strings.TrimSpace(command.NotificationID)
	if publicID == "" {
		return MarkNotificationReadResult{}, ErrInvalidRequest
	}
	internalID, err := s.ids.Decode(publicID)
	if err != nil {
		return MarkNotificationReadResult{}, ErrInvalidRequest
	}

	readAt := s.clock.Now()
	result, err := s.commands.MarkRead(ctx, ports.MarkReadInput{
		NotificationID: int64(internalID),
		RecipientID:    command.Actor.UserID,
		ReadAt:         readAt,
	})
	if err != nil {
		return MarkNotificationReadResult{}, mapPortsError(err)
	}
	if err := s.unread.Delete(ctx, unreadCacheKeys(command.Actor.UserID)...); err != nil {
		return MarkNotificationReadResult{}, fmt.Errorf("invalidate notification unread cache: %w", err)
	}
	return MarkNotificationReadResult{
		NotificationID: result.PublicID,
		Read:           true,
		Changed:        result.Changed,
		ReadAt:         result.ReadAt,
	}, nil
}
