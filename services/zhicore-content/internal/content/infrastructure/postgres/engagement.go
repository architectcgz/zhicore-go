package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func (s *Store) MutateEngagement(ctx context.Context, tx ports.Tx, input ports.EngagementMutationInput) (ports.EngagementMutationRecord, error) {
	execer, err := s.execer(tx)
	if err != nil {
		return ports.EngagementMutationRecord{}, err
	}
	query, err := engagementMutationSQL(input.Action)
	if err != nil {
		return ports.EngagementMutationRecord{}, err
	}
	record, err := scanEngagementMutationRecord(execer.QueryRowContext(ctx, query, input.PostID, input.ActorID, input.OccurredAt))
	if errors.Is(err, sql.ErrNoRows) {
		return ports.EngagementMutationRecord{}, domain.ErrPostNotFound
	}
	if err != nil {
		return ports.EngagementMutationRecord{}, fmt.Errorf("mutate content engagement: %w", err)
	}
	return record, nil
}

func (s *Store) GetPostEngagement(ctx context.Context, postID string) (ports.PostEngagementRecord, error) {
	var record ports.PostEngagementRecord
	err := s.db.QueryRowContext(ctx, getPostEngagementSQL, postID).Scan(
		&record.PostID,
		&record.Stats.ViewCount,
		&record.Stats.LikeCount,
		&record.Stats.FavoriteCount,
		&record.Stats.CommentCount,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ports.PostEngagementRecord{}, domain.ErrPostNotFound
	}
	if err != nil {
		return ports.PostEngagementRecord{}, fmt.Errorf("get content post engagement: %w", err)
	}
	return record, nil
}

func (s *Store) BatchGetViewerStatus(ctx context.Context, userID int64, postIDs []string) ([]ports.EngagementStatusRecord, error) {
	rows, err := s.db.QueryContext(ctx, batchGetViewerEngagementStatusSQL, userID, pq.Array(postIDs))
	if err != nil {
		return nil, fmt.Errorf("batch get content engagement viewer status: %w", err)
	}
	defer rows.Close()

	records := make([]ports.EngagementStatusRecord, 0, len(postIDs))
	for rows.Next() {
		var record ports.EngagementStatusRecord
		if err := rows.Scan(&record.PostID, &record.Liked, &record.Favorited); err != nil {
			return nil, fmt.Errorf("scan content engagement viewer status: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate content engagement viewer status: %w", err)
	}
	return records, nil
}

func engagementMutationSQL(action ports.EngagementAction) (string, error) {
	switch action {
	case ports.EngagementActionLike:
		return mutateLikeEngagementSQL, nil
	case ports.EngagementActionUnlike:
		return mutateUnlikeEngagementSQL, nil
	case ports.EngagementActionFavorite:
		return mutateFavoriteEngagementSQL, nil
	case ports.EngagementActionUnfavorite:
		return mutateUnfavoriteEngagementSQL, nil
	default:
		return "", fmt.Errorf("unsupported content engagement action %q", action)
	}
}

func scanEngagementMutationRecord(row *sql.Row) (ports.EngagementMutationRecord, error) {
	var record ports.EngagementMutationRecord
	err := row.Scan(
		&record.PostInternalID,
		&record.PostID,
		&record.AuthorID,
		&record.ActorID,
		&record.Changed,
		&record.Liked,
		&record.Favorited,
		&record.AggregateVersion,
		&record.Stats.ViewCount,
		&record.Stats.LikeCount,
		&record.Stats.FavoriteCount,
		&record.Stats.CommentCount,
	)
	if err != nil {
		return ports.EngagementMutationRecord{}, err
	}
	return record, nil
}
