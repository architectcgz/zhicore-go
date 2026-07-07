package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-notification/internal/notification/ports"
)

func (s *Store) PlanPostPublishedCampaign(ctx context.Context, input ports.PlanPostPublishedCampaignInput) (ports.PlanCampaignResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ports.PlanCampaignResult{}, fmt.Errorf("begin notification campaign transaction: %w", err)
	}
	defer tx.Rollback()

	if err := insertConsumedEvent(ctx, tx, input.Event); errors.Is(err, sql.ErrNoRows) {
		if commitErr := tx.Commit(); commitErr != nil {
			return ports.PlanCampaignResult{}, fmt.Errorf("commit duplicate campaign event: %w", commitErr)
		}
		return ports.PlanCampaignResult{}, ports.ErrDuplicateConsumedEvent
	} else if err != nil {
		return ports.PlanCampaignResult{}, err
	}

	campaignID, created, err := insertPostPublishedCampaign(ctx, tx, input)
	if err != nil {
		return ports.PlanCampaignResult{}, err
	}
	var shardID int64
	if created {
		shardID, err = insertInitialCampaignShard(ctx, tx, campaignID, input.AudienceClass, input.AudienceActiveSince, input.CreatedAt)
		if err != nil {
			return ports.PlanCampaignResult{}, err
		}
	}
	if err := markConsumedEvent(ctx, tx, input.Event.EventID, input.CreatedAt); err != nil {
		return ports.PlanCampaignResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return ports.PlanCampaignResult{}, fmt.Errorf("commit notification campaign transaction: %w", err)
	}
	return ports.PlanCampaignResult{Created: created, CampaignID: campaignID, ShardID: shardID}, nil
}

func insertPostPublishedCampaign(ctx context.Context, tx *sql.Tx, input ports.PlanPostPublishedCampaignInput) (int64, bool, error) {
	var campaignID int64
	err := tx.QueryRowContext(ctx, insertPostPublishedCampaignSQL,
		input.SourceEventID,
		input.CampaignType,
		input.AuthorID,
		input.PostID,
		input.ObjectType,
		input.ObjectID,
		input.AudienceClass,
		nullableTime(input.AudienceActiveSince),
		input.Title,
		input.Excerpt,
		input.Payload,
		input.PublishedAt,
		input.CreatedAt,
	).Scan(&campaignID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("insert post published campaign: %w", err)
	}
	return campaignID, true, nil
}

func insertInitialCampaignShard(ctx context.Context, tx *sql.Tx, campaignID int64, audienceClass string, audienceActiveSince *time.Time, createdAt time.Time) (int64, error) {
	var shardID int64
	if err := tx.QueryRowContext(ctx, insertInitialCampaignShardSQL, campaignID, audienceClass, nullableTime(audienceActiveSince), createdAt).Scan(&shardID); err != nil {
		return 0, fmt.Errorf("insert initial campaign shard: %w", err)
	}
	return shardID, nil
}

func (s *Store) ClaimCampaignShard(ctx context.Context, input ports.ClaimCampaignShardInput) (ports.ClaimedCampaignShard, error) {
	var claim ports.ClaimedCampaignShard
	var activeSince sql.NullTime
	err := s.db.QueryRowContext(ctx, claimCampaignShardSQL,
		input.WorkerID,
		input.Now,
		int64(input.ClaimTimeout/time.Second),
	).Scan(
		&claim.ShardID,
		&claim.CampaignID,
		&claim.AuthorID,
		&claim.PostID,
		&claim.AudienceClass,
		&activeSince,
		&claim.FollowerCursor,
		&claim.AttemptCount,
		&claim.ClaimDeadlineAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ports.ClaimedCampaignShard{}, nil
	}
	if err != nil {
		return ports.ClaimedCampaignShard{}, fmt.Errorf("claim campaign shard: %w", err)
	}
	if activeSince.Valid {
		claim.AudienceActiveSince = &activeSince.Time
	}
	claim.Found = true
	return claim, nil
}

func (s *Store) FailCampaignShard(ctx context.Context, input ports.FailCampaignShardInput) error {
	_, err := s.db.ExecContext(ctx,
		failCampaignShardSQL,
		input.ShardID,
		input.ErrorCode,
		input.FailedAt,
		int64(input.RetryAfter/time.Second),
	)
	if err != nil {
		return fmt.Errorf("fail campaign shard: %w", err)
	}
	return nil
}

func nullableTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}
