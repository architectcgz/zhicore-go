package postgres

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

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

	var publicID, groupKey, category string
	var isRead bool
	var readAt sql.NullTime
	err = tx.QueryRowContext(ctx, selectNotificationForMarkReadSQL, input.NotificationID, input.RecipientID).Scan(&publicID, &groupKey, &category, &isRead, &readAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ports.MarkReadResult{}, ports.ErrNotificationNotFound
	}
	if err != nil {
		return ports.MarkReadResult{}, fmt.Errorf("select notification for mark read: %w", err)
	}
	var lockedGroupKey string
	if err := tx.QueryRowContext(ctx, lockNotificationGroupByKeySQL, input.RecipientID, groupKey).Scan(&lockedGroupKey); err != nil {
		return ports.MarkReadResult{}, fmt.Errorf("lock notification group before single read: %w", err)
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
	if _, err := tx.ExecContext(ctx, decrementNotificationStatsSQL, input.RecipientID, category, input.ReadAt); err != nil {
		return ports.MarkReadResult{}, fmt.Errorf("decrement notification stats unread count: %w", err)
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
	rows, err := tx.QueryContext(ctx, lockAllNotificationGroupsSQL, input.RecipientID)
	if err != nil {
		return ports.MarkAllReadResult{}, fmt.Errorf("lock notification groups before read all: %w", err)
	}
	if err := rows.Close(); err != nil {
		return ports.MarkAllReadResult{}, fmt.Errorf("close notification group locks: %w", err)
	}

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
	if _, err := tx.ExecContext(ctx, resetNotificationStatsSQL, input.RecipientID, input.ReadAt); err != nil {
		return ports.MarkAllReadResult{}, fmt.Errorf("reset notification stats unread count: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return ports.MarkAllReadResult{}, fmt.Errorf("commit notification mark all read: %w", err)
	}
	return ports.MarkAllReadResult{AffectedCount: affected, ReadAt: input.ReadAt}, nil
}

func (s *Store) MarkGroupRead(ctx context.Context, input ports.MarkGroupReadInput) (ports.MarkGroupReadResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ports.MarkGroupReadResult{}, fmt.Errorf("begin notification group read transaction: %w", err)
	}
	defer tx.Rollback()
	var groupKey, category string
	var unread int64
	err = tx.QueryRowContext(ctx, selectNotificationGroupForMarkReadSQL, input.RecipientID, input.GroupID).Scan(&groupKey, &category, &unread)
	if errors.Is(err, sql.ErrNoRows) {
		return ports.MarkGroupReadResult{}, ports.ErrNotificationNotFound
	}
	if err != nil {
		return ports.MarkGroupReadResult{}, fmt.Errorf("lock notification group: %w", err)
	}
	result, err := tx.ExecContext(ctx, markNotificationGroupReadSQL, input.RecipientID, input.GroupID, input.ReadAt)
	if err != nil {
		return ports.MarkGroupReadResult{}, fmt.Errorf("mark notification group rows read: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return ports.MarkGroupReadResult{}, err
	}
	if _, err = tx.ExecContext(ctx, resetSingleGroupUnreadSQL, input.RecipientID, groupKey, input.ReadAt); err != nil {
		return ports.MarkGroupReadResult{}, fmt.Errorf("reset notification group unread count: %w", err)
	}
	if changed > 0 {
		if _, err = tx.ExecContext(ctx, decrementNotificationGroupStatsSQL, input.RecipientID, category, changed, input.ReadAt); err != nil {
			return ports.MarkGroupReadResult{}, fmt.Errorf("decrement notification group stats: %w", err)
		}
	}
	if err = tx.Commit(); err != nil {
		return ports.MarkGroupReadResult{}, fmt.Errorf("commit notification group read: %w", err)
	}
	return ports.MarkGroupReadResult{GroupID: input.GroupID, ChangedCount: changed, UnreadCount: 0, ReadAt: input.ReadAt}, nil
}

func (s *Store) ListGroupActors(ctx context.Context, query ports.ListGroupActorsQuery) (ports.NotificationActorPage, error) {
	cursorTime, cursorPublicID, ok := decodeNotificationActorCursor(query.Cursor)
	if query.Cursor != "" && !ok {
		return ports.NotificationActorPage{}, ports.ErrInvalidQuery
	}
	var ownerMarker int
	if err := s.db.QueryRowContext(ctx, ensureNotificationGroupOwnerSQL, query.RecipientID, query.GroupID).Scan(&ownerMarker); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ports.NotificationActorPage{}, ports.ErrNotificationNotFound
		}
		return ports.NotificationActorPage{}, fmt.Errorf("check notification group owner: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, listNotificationGroupActorsSQL, query.RecipientID, query.GroupID, cursorTime, cursorPublicID, query.Size+1)
	if err != nil {
		return ports.NotificationActorPage{}, fmt.Errorf("list notification group actors: %w", err)
	}
	defer rows.Close()
	items := make([]ports.NotificationActor, 0, query.Size)
	for rows.Next() {
		var item ports.NotificationActor
		if err := rows.Scan(&item.PublicID, &item.DisplayName, &item.AvatarURL, &item.EventCount, &item.LatestOccurredAt); err != nil {
			return ports.NotificationActorPage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ports.NotificationActorPage{}, err
	}
	page := ports.NotificationActorPage{Items: items}
	if len(items) > query.Size {
		page.HasMore = true
		page.Items = items[:query.Size]
		last := page.Items[len(page.Items)-1]
		page.NextCursor = encodeNotificationActorCursor(last.LatestOccurredAt, last.PublicID)
	}
	return page, nil
}

func (s *Store) GetUnreadCount(ctx context.Context, recipientID int64) (int64, error) {
	var count int64
	if err := s.db.QueryRowContext(ctx, getUnreadCountSQL, recipientID).Scan(&count); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("get notification unread count: %w", err)
	}
	return count, nil
}

func (s *Store) GetUnreadBreakdown(ctx context.Context, recipientID int64) (ports.UnreadBreakdown, error) {
	var result ports.UnreadBreakdown
	err := s.db.QueryRowContext(ctx, getUnreadBreakdownSQL, recipientID).Scan(
		&result.Total,
		&result.Interaction,
		&result.Content,
		&result.Social,
		&result.System,
		&result.Security,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ports.UnreadBreakdown{}, nil
	}
	if err != nil {
		return ports.UnreadBreakdown{}, fmt.Errorf("get notification unread breakdown: %w", err)
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
	cursorTime, cursorGroupID, ok := decodeNotificationCursor(query.Cursor)
	if query.Cursor != "" && !ok {
		return ports.AggregatedNotificationPage{}, ports.ErrInvalidQuery
	}
	rows, err := s.db.QueryContext(ctx, listAggregatedFromGroupStateSQL, query.RecipientID, query.Category, query.UnreadOnly, cursorTime, cursorGroupID, limit+1)
	if err != nil {
		return ports.AggregatedNotificationPage{}, fmt.Errorf("list notification group state: %w", err)
	}
	defer rows.Close()

	items := make([]ports.AggregatedNotification, 0, limit)
	for rows.Next() {
		var item ports.AggregatedNotification
		var recentActorsJSON []byte
		if err := rows.Scan(
			&item.GroupID,
			&item.Type,
			&item.Category,
			&item.TargetType,
			&item.TargetID,
			&item.TotalCount,
			&item.UnreadCount,
			&item.LatestTime,
			&item.LatestContent,
			&item.AggregatedContent,
			&item.ActorTotalCount,
			&recentActorsJSON,
		); err != nil {
			return ports.AggregatedNotificationPage{}, fmt.Errorf("scan notification group state: %w", err)
		}
		if err := json.Unmarshal(recentActorsJSON, &item.RecentActors); err != nil {
			return ports.AggregatedNotificationPage{}, fmt.Errorf("decode notification group recent actors: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ports.AggregatedNotificationPage{}, fmt.Errorf("iterate notification group state: %w", err)
	}
	return pageFromAggregatedItems(items, limit), nil
}

func (s *Store) listAggregatedFromInbox(ctx context.Context, query ports.ListAggregatedQuery, limit int) (ports.AggregatedNotificationPage, error) {
	cursorTime, cursorGroupID, ok := decodeNotificationCursor(query.Cursor)
	if query.Cursor != "" && !ok {
		return ports.AggregatedNotificationPage{}, ports.ErrInvalidQuery
	}
	rows, err := s.db.QueryContext(ctx, listAggregatedFromInboxSQL, query.RecipientID, query.Category, query.UnreadOnly, cursorTime, cursorGroupID, limit+1)
	if err != nil {
		return ports.AggregatedNotificationPage{}, fmt.Errorf("list notification inbox fallback aggregation: %w", err)
	}
	defer rows.Close()

	items := make([]ports.AggregatedNotification, 0, limit)
	for rows.Next() {
		var item ports.AggregatedNotification
		var recentActorsJSON []byte
		if err := rows.Scan(
			&item.GroupID,
			&item.Type,
			&item.Category,
			&item.TargetType,
			&item.TargetID,
			&item.TotalCount,
			&item.UnreadCount,
			&item.LatestTime,
			&item.LatestContent,
			&item.AggregatedContent,
			&item.ActorTotalCount,
			&recentActorsJSON,
		); err != nil {
			return ports.AggregatedNotificationPage{}, fmt.Errorf("scan notification inbox fallback aggregation: %w", err)
		}
		if err := json.Unmarshal(recentActorsJSON, &item.RecentActors); err != nil {
			return ports.AggregatedNotificationPage{}, fmt.Errorf("decode notification inbox fallback recent actors: %w", err)
		}
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
		page.NextCursor = encodeNotificationCursor(page.Items[len(page.Items)-1].LatestTime, page.Items[len(page.Items)-1].GroupID)
	}
	return page
}

type notificationCursor struct {
	OccurredAt string `json:"occurredAt"`
	GroupID    string `json:"groupId"`
}

type notificationActorCursor struct {
	OccurredAt string `json:"occurredAt"`
	PublicID   string `json:"publicId"`
}

func encodeNotificationCursor(occurredAt time.Time, groupID string) string {
	payload, _ := json.Marshal(notificationCursor{OccurredAt: occurredAt.UTC().Format(time.RFC3339Nano), GroupID: groupID})
	return base64.RawURLEncoding.EncodeToString(payload)
}
func decodeNotificationCursor(raw string) (string, string, bool) {
	payload, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return "", "", false
	}
	var cursor notificationCursor
	if json.Unmarshal(payload, &cursor) != nil || cursor.GroupID == "" {
		return "", "", false
	}
	if _, err := time.Parse(time.RFC3339Nano, cursor.OccurredAt); err != nil {
		return "", "", false
	}
	return cursor.OccurredAt, cursor.GroupID, true
}

func encodeNotificationActorCursor(occurredAt time.Time, publicID string) string {
	payload, _ := json.Marshal(notificationActorCursor{OccurredAt: occurredAt.UTC().Format(time.RFC3339Nano), PublicID: publicID})
	return base64.RawURLEncoding.EncodeToString(payload)
}

func decodeNotificationActorCursor(raw string) (string, string, bool) {
	payload, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return "", "", false
	}
	var cursor notificationActorCursor
	if json.Unmarshal(payload, &cursor) != nil || cursor.PublicID == "" {
		return "", "", false
	}
	if _, err := time.Parse(time.RFC3339Nano, cursor.OccurredAt); err != nil {
		return "", "", false
	}
	return cursor.OccurredAt, cursor.PublicID, true
}
