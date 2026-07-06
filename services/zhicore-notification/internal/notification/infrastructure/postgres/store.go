package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

type Store struct {
	db    *sql.DB
	codec ports.NotificationPublicIDCodec
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func NewStoreWithCodec(db *sql.DB, codec ports.NotificationPublicIDCodec) *Store {
	return &Store{db: db, codec: codec}
}

func (s *Store) MarkRead(ctx context.Context, input ports.MarkReadInput) (ports.MarkReadResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ports.MarkReadResult{}, fmt.Errorf("begin notification mark read transaction: %w", err)
	}
	defer tx.Rollback()

	var publicID, groupKey string
	var isRead bool
	var readAt sql.NullTime
	err = tx.QueryRowContext(ctx, selectNotificationForMarkReadSQL, input.NotificationID, input.RecipientID).Scan(&publicID, &groupKey, &isRead, &readAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ports.MarkReadResult{}, ports.ErrNotificationNotFound
	}
	if err != nil {
		return ports.MarkReadResult{}, fmt.Errorf("select notification for mark read: %w", err)
	}

	if isRead {
		if err := tx.Commit(); err != nil {
			return ports.MarkReadResult{}, fmt.Errorf("commit notification repeated mark read: %w", err)
		}
		return ports.MarkReadResult{NotificationID: input.NotificationID, PublicID: publicID, Changed: false, ReadAt: readAt.Time}, nil
	}

	if _, err := tx.ExecContext(ctx, updateNotificationReadSQL, input.NotificationID, input.RecipientID, input.ReadAt); err != nil {
		return ports.MarkReadResult{}, fmt.Errorf("update notification read state: %w", err)
	}
	// Group state is a derived read model; clamp at zero so repeated or repaired reads never create negative unread counts.
	if _, err := tx.ExecContext(ctx, decrementGroupUnreadSQL, input.RecipientID, groupKey, input.ReadAt); err != nil {
		return ports.MarkReadResult{}, fmt.Errorf("decrement notification group unread count: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return ports.MarkReadResult{}, fmt.Errorf("commit notification mark read: %w", err)
	}
	return ports.MarkReadResult{NotificationID: input.NotificationID, PublicID: publicID, Changed: true, ReadAt: input.ReadAt}, nil
}

func (s *Store) MarkAllRead(ctx context.Context, input ports.MarkAllReadInput) (ports.MarkAllReadResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ports.MarkAllReadResult{}, fmt.Errorf("begin notification mark all read transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, markAllNotificationsReadSQL, input.RecipientID, input.ReadAt)
	if err != nil {
		return ports.MarkAllReadResult{}, fmt.Errorf("mark all notifications read: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return ports.MarkAllReadResult{}, fmt.Errorf("count marked notifications: %w", err)
	}
	if _, err := tx.ExecContext(ctx, resetGroupUnreadSQL, input.RecipientID, input.ReadAt); err != nil {
		return ports.MarkAllReadResult{}, fmt.Errorf("reset notification group unread count: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return ports.MarkAllReadResult{}, fmt.Errorf("commit notification mark all read: %w", err)
	}
	return ports.MarkAllReadResult{AffectedCount: affected, ReadAt: input.ReadAt}, nil
}

func (s *Store) GetUnreadCount(ctx context.Context, recipientID int64) (int64, error) {
	var count int64
	if err := s.db.QueryRowContext(ctx, getUnreadCountSQL, recipientID).Scan(&count); err != nil {
		return 0, fmt.Errorf("get notification unread count: %w", err)
	}
	return count, nil
}

func (s *Store) GetUnreadBreakdown(ctx context.Context, recipientID int64) (ports.UnreadBreakdown, error) {
	rows, err := s.db.QueryContext(ctx, getUnreadBreakdownSQL, recipientID)
	if err != nil {
		return ports.UnreadBreakdown{}, fmt.Errorf("get notification unread breakdown: %w", err)
	}
	defer rows.Close()

	var result ports.UnreadBreakdown
	for rows.Next() {
		var category string
		var count int64
		if err := rows.Scan(&category, &count); err != nil {
			return ports.UnreadBreakdown{}, fmt.Errorf("scan notification unread breakdown: %w", err)
		}
		result.Total += count
		switch category {
		case "INTERACTION":
			result.Interaction = count
		case "CONTENT":
			result.Content = count
		case "SOCIAL":
			result.Social = count
		case "SYSTEM":
			result.System = count
		case "SECURITY":
			result.Security = count
		}
	}
	if err := rows.Err(); err != nil {
		return ports.UnreadBreakdown{}, fmt.Errorf("iterate notification unread breakdown: %w", err)
	}
	return result, nil
}

func (s *Store) ListAggregated(ctx context.Context, query ports.ListAggregatedQuery) (ports.AggregatedNotificationPage, error) {
	limit := query.Size
	if limit <= 0 {
		limit = 20
	}
	page, err := s.listAggregatedFromGroupState(ctx, query, limit)
	if err != nil {
		return ports.AggregatedNotificationPage{}, err
	}
	if len(page.Items) > 0 || query.Cursor != "" {
		return page, nil
	}
	// notification_group_state is a synchronous read model. If it is missing, fall back to the inbox truth source
	// so users still see notifications while a later repair path rebuilds the group state.
	page, err = s.listAggregatedFromInbox(ctx, query, limit)
	if err != nil {
		return ports.AggregatedNotificationPage{}, err
	}
	page.RepairSignal = true
	return page, nil
}

func (s *Store) listAggregatedFromGroupState(ctx context.Context, query ports.ListAggregatedQuery, limit int) (ports.AggregatedNotificationPage, error) {
	rows, err := s.db.QueryContext(ctx, listAggregatedFromGroupStateSQL, query.RecipientID, query.Category, query.UnreadOnly, limit+1)
	if err != nil {
		return ports.AggregatedNotificationPage{}, fmt.Errorf("list notification group state: %w", err)
	}
	defer rows.Close()

	items := make([]ports.AggregatedNotification, 0, limit)
	for rows.Next() {
		var item ports.AggregatedNotification
		var actorIDs pq.Int64Array
		if err := rows.Scan(
			&item.Type,
			&item.Category,
			&item.TargetType,
			&item.TargetID,
			&item.TotalCount,
			&item.UnreadCount,
			&item.LatestTime,
			&item.LatestContent,
			&actorIDs,
			&item.AggregatedContent,
		); err != nil {
			return ports.AggregatedNotificationPage{}, fmt.Errorf("scan notification group state: %w", err)
		}
		item.ActorIDs = []int64(actorIDs)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ports.AggregatedNotificationPage{}, fmt.Errorf("iterate notification group state: %w", err)
	}
	return pageFromAggregatedItems(items, limit), nil
}

func (s *Store) listAggregatedFromInbox(ctx context.Context, query ports.ListAggregatedQuery, limit int) (ports.AggregatedNotificationPage, error) {
	rows, err := s.db.QueryContext(ctx, listAggregatedFromInboxSQL, query.RecipientID, query.Category, query.UnreadOnly, limit+1)
	if err != nil {
		return ports.AggregatedNotificationPage{}, fmt.Errorf("list notification inbox fallback aggregation: %w", err)
	}
	defer rows.Close()

	items := make([]ports.AggregatedNotification, 0, limit)
	for rows.Next() {
		var item ports.AggregatedNotification
		var actorIDs pq.Int64Array
		if err := rows.Scan(
			&item.Type,
			&item.Category,
			&item.TargetType,
			&item.TargetID,
			&item.TotalCount,
			&item.UnreadCount,
			&item.LatestTime,
			&item.LatestContent,
			&actorIDs,
			&item.AggregatedContent,
		); err != nil {
			return ports.AggregatedNotificationPage{}, fmt.Errorf("scan notification inbox fallback aggregation: %w", err)
		}
		item.ActorIDs = []int64(actorIDs)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ports.AggregatedNotificationPage{}, fmt.Errorf("iterate notification inbox fallback aggregation: %w", err)
	}
	return pageFromAggregatedItems(items, limit), nil
}

func pageFromAggregatedItems(items []ports.AggregatedNotification, limit int) ports.AggregatedNotificationPage {
	page := ports.AggregatedNotificationPage{Items: items}
	if len(items) > limit {
		page.HasMore = true
		page.Items = items[:limit]
		page.NextCursor = page.Items[len(page.Items)-1].LatestTime.UTC().Format(time.RFC3339Nano)
	}
	return page
}
