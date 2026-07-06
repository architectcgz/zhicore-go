package application

import (
	"context"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

func (s *Service) ListDeliveries(ctx context.Context, query ListDeliveriesQuery) (DeliveryPage, error) {
	if err := requireActor(query.Actor); err != nil {
		return DeliveryPage{}, err
	}
	if s.deliveries == nil {
		return DeliveryPage{}, ErrDependencyUnavailable
	}
	recipientID := query.Actor.UserID
	isAdmin := hasAdminRole(query.Actor)
	if isAdmin && query.RecipientID > 0 {
		recipientID = query.RecipientID
	}
	page, err := s.deliveries.ListDeliveries(ctx, ports.ListDeliveriesQuery{
		RequesterID: query.Actor.UserID,
		IsAdmin:     isAdmin,
		RecipientID: recipientID,
		Channel:     query.Channel,
		Status:      query.Status,
		Cursor:      query.Cursor,
		Size:        query.Size,
	})
	if err != nil {
		return DeliveryPage{}, mapPortsError(err)
	}
	items := make([]DeliveryResult, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, DeliveryResult{
			DeliveryID:       item.DeliveryID,
			RecipientID:      item.RecipientID,
			NotificationID:   item.NotificationID,
			Channel:          item.Channel,
			NotificationType: item.NotificationType,
			Status:           item.Status,
			Provider:         item.Provider,
			AttemptCount:     item.AttemptCount,
			LastErrorCode:    item.LastErrorCode,
			NextRetryAt:      item.NextRetryAt,
			CreatedAt:        item.CreatedAt,
			UpdatedAt:        item.UpdatedAt,
		})
	}
	return DeliveryPage{Items: items, NextCursor: page.NextCursor, HasMore: page.HasMore}, nil
}

func (s *Service) RetryDelivery(ctx context.Context, command RetryDeliveryCommand) (DeliveryRetryResult, error) {
	if err := requireActor(command.Actor); err != nil {
		return DeliveryRetryResult{}, err
	}
	internalID, err := s.ids.Decode(command.DeliveryID)
	if command.DeliveryID == "" || err != nil {
		return DeliveryRetryResult{}, ErrInvalidRequest
	}
	if s.deliveries == nil {
		return DeliveryRetryResult{}, ErrDependencyUnavailable
	}
	result, err := s.deliveries.RetryDelivery(ctx, ports.RetryDeliveryInput{
		DeliveryID:  int64(internalID),
		RequesterID: command.Actor.UserID,
		IsAdmin:     hasAdminRole(command.Actor),
		RetriedAt:   s.clock.Now(),
	})
	if err != nil {
		return DeliveryRetryResult{}, mapPortsError(err)
	}
	if result.RecipientID != command.Actor.UserID && !hasAdminRole(command.Actor) {
		return DeliveryRetryResult{}, ErrNotificationNotFound
	}
	return DeliveryRetryResult{DeliveryID: result.PublicID, Status: result.Status, Retried: result.Retried}, nil
}
