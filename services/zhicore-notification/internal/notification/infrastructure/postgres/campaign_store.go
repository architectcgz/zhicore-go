package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

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
		&claim.ClaimedBy,
		&claim.ClaimDeadlineAt,
		&claim.Title,
		&claim.Excerpt,
		&claim.Payload,
		&claim.PublishedAt,
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

func (s *Store) MaterializeCampaignFollowers(ctx context.Context, input ports.MaterializeCampaignFollowersInput) (ports.MaterializeCampaignFollowersResult, error) {
	if len(input.FollowerIDs) == 0 {
		return ports.MaterializeCampaignFollowersResult{}, nil
	}
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = "新作品发布"
	}
	content := strings.TrimSpace(input.Content)
	if content == "" {
		content = title
	}
	payload := input.Payload
	if len(payload) == 0 {
		payload = []byte(`{}`)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ports.MaterializeCampaignFollowersResult{}, fmt.Errorf("begin campaign follower materialization transaction: %w", err)
	}
	defer tx.Rollback()

	result := ports.MaterializeCampaignFollowersResult{ProcessedCount: int64(len(input.FollowerIDs))}
	for _, followerID := range input.FollowerIDs {
		if followerID <= 0 || followerID == input.AuthorID {
			result.SkippedCount++
			continue
		}
		decision, err := campaignDeliveryDecisionForRecipient(ctx, tx, followerID, input.AuthorID, input.NotificationType, input.Category, input.CreatedAt)
		if err != nil {
			return ports.MaterializeCampaignFollowersResult{}, err
		}
		if decision.SkipAll {
			if err := s.insertCampaignDelivery(ctx, tx, input, followerID, nil, "IN_APP", "SKIPPED", input.CreatedAt); err != nil {
				return ports.MaterializeCampaignFollowersResult{}, err
			}
			result.SkippedCount++
			continue
		}
		if decision.DigestOnly {
			if err := s.insertCampaignDelivery(ctx, tx, input, followerID, nil, "EMAIL", "DIGEST_PENDING", input.CreatedAt); err != nil {
				return ports.MaterializeCampaignFollowersResult{}, err
			}
			result.SkippedCount++
			continue
		}
		if !decision.InAppEnabled {
			if err := s.insertCampaignDelivery(ctx, tx, input, followerID, nil, "IN_APP", "SKIPPED", input.CreatedAt); err != nil {
				return ports.MaterializeCampaignFollowersResult{}, err
			}
			result.SkippedCount++
			continue
		}
		notificationID, err := s.nextNotificationID(ctx, tx)
		if err != nil {
			return ports.MaterializeCampaignFollowersResult{}, err
		}
		notificationPublicID, err := s.encodePublicID(notificationID)
		if err != nil {
			return ports.MaterializeCampaignFollowersResult{}, err
		}
		actorID := input.AuthorID
		notificationInput := ports.CreateInteractionNotificationInput{
			RecipientID:      followerID,
			ActorID:          &actorID,
			Category:         input.Category,
			NotificationType: input.NotificationType,
			EventCode:        input.EventCode,
			Importance:       "NORMAL",
			TargetType:       input.TargetType,
			TargetID:         input.TargetID,
			SourceEventID:    fmt.Sprintf("campaign:%d:shard:%d:recipient:%d", input.CampaignID, input.ShardID, followerID),
			DedupeKey:        fmt.Sprintf("campaign:%d:post:%d:recipient:%d", input.CampaignID, input.PostID, followerID),
			GroupKey:         fmt.Sprintf("campaign:%d:post:%d", input.CampaignID, input.PostID),
			Title:            title,
			Content:          content,
			Payload:          payload,
			OccurredAt:       input.OccurredAt,
			CreatedAt:        input.CreatedAt,
		}
		insertedID, created, err := insertNotification(ctx, tx, notificationID, notificationPublicID, notificationInput)
		if err != nil {
			return ports.MaterializeCampaignFollowersResult{}, err
		}
		if !created {
			result.SkippedCount++
			continue
		}
		if err := upsertGroupState(ctx, tx, insertedID, notificationInput); err != nil {
			return ports.MaterializeCampaignFollowersResult{}, err
		}
		if err := incrementNotificationStats(ctx, tx, notificationInput); err != nil {
			return ports.MaterializeCampaignFollowersResult{}, err
		}
		if err := s.insertCampaignDelivery(ctx, tx, input, followerID, &insertedID, "IN_APP", "IN_APP", input.CreatedAt); err != nil {
			return ports.MaterializeCampaignFollowersResult{}, err
		}
		if decision.WebsocketEnabled {
			if err := s.insertCampaignDelivery(ctx, tx, input, followerID, &insertedID, "WEBSOCKET", "WEBSOCKET_PENDING", input.CreatedAt); err != nil {
				return ports.MaterializeCampaignFollowersResult{}, err
			}
		}
		result.SuccessCount++
	}
	if err := tx.Commit(); err != nil {
		return ports.MaterializeCampaignFollowersResult{}, fmt.Errorf("commit campaign follower materialization: %w", err)
	}
	return result, nil
}

func (s *Store) insertCampaignDelivery(ctx context.Context, tx *sql.Tx, input ports.MaterializeCampaignFollowersInput, recipientID int64, notificationID *int64, channel string, status string, createdAt time.Time) error {
	deliveryID, err := s.nextDeliveryID(ctx, tx)
	if err != nil {
		return err
	}
	publicID, err := s.encodePublicID(deliveryID)
	if err != nil {
		return err
	}
	dedupeKey := strings.ToLower(fmt.Sprintf("campaign:%d:recipient:%d:%s", input.CampaignID, recipientID, channel))
	if _, err := tx.ExecContext(ctx, insertCampaignDeliverySQL,
		deliveryID,
		publicID,
		recipientID,
		nullableInt64(notificationID),
		input.CampaignID,
		channel,
		input.NotificationType,
		status,
		dedupeKey,
		createdAt,
	); err != nil {
		return fmt.Errorf("insert campaign delivery: %w", err)
	}
	return nil
}

type campaignDeliveryDecision struct {
	InAppEnabled     bool
	WebsocketEnabled bool
	DigestOnly       bool
	SkipAll          bool
}

func campaignDeliveryDecisionForRecipient(ctx context.Context, tx *sql.Tx, recipientID int64, authorID int64, notificationType string, category string, now time.Time) (campaignDeliveryDecision, error) {
	var level string
	var inAppEnabled bool
	var websocketEnabled bool
	var emailPreferenceEnabled bool
	var digestEnabled bool
	var dndEnabled bool
	var startTime string
	var endTime string
	var timezone string
	var categories pq.StringArray
	var channels pq.StringArray
	err := tx.QueryRowContext(ctx, getCampaignDeliveryDecisionSQL, recipientID, authorID, notificationType).Scan(
		&level,
		&inAppEnabled,
		&websocketEnabled,
		&emailPreferenceEnabled,
		&digestEnabled,
		&dndEnabled,
		&startTime,
		&endTime,
		&timezone,
		&categories,
		&channels,
	)
	if err != nil {
		return campaignDeliveryDecision{}, fmt.Errorf("get campaign delivery decision: %w", err)
	}
	decision := campaignDeliveryDecision{InAppEnabled: inAppEnabled, WebsocketEnabled: websocketEnabled}
	switch level {
	case "MUTED":
		decision.SkipAll = true
		decision.InAppEnabled = false
		decision.WebsocketEnabled = false
	case "DIGEST_ONLY":
		// User preferences are the global gate; a disabled email channel must
		// suppress digest delivery even when the author subscription requests it.
		decision.DigestOnly = digestEnabled && emailPreferenceEnabled
		decision.InAppEnabled = false
		decision.WebsocketEnabled = false
	}
	if decision.WebsocketEnabled && dndApplies(dndEnabled, startTime, endTime, timezone, []string(categories), []string(channels), category, "WEBSOCKET", now) {
		decision.WebsocketEnabled = false
	}
	return decision, nil
}

func dndApplies(enabled bool, startHHMM string, endHHMM string, timezone string, categories []string, channels []string, category string, channel string, now time.Time) bool {
	if !enabled {
		return false
	}
	if !stringListMatches(categories, category) || !stringListMatches(channels, channel) {
		return false
	}
	location, err := time.LoadLocation(strings.TrimSpace(timezone))
	if err != nil {
		location = time.UTC
	}
	local := now.In(location)
	startMinute, ok := parseHHMM(startHHMM)
	if !ok {
		return false
	}
	endMinute, ok := parseHHMM(endHHMM)
	if !ok || startMinute == endMinute {
		return false
	}
	current := local.Hour()*60 + local.Minute()
	if startMinute < endMinute {
		return current >= startMinute && current < endMinute
	}
	return current >= startMinute || current < endMinute
}

func parseHHMM(value string) (int, bool) {
	parsed, err := time.Parse("15:04", strings.TrimSpace(value))
	if err != nil {
		return 0, false
	}
	return parsed.Hour()*60 + parsed.Minute(), true
}

func stringListMatches(values []string, target string) bool {
	if len(values) == 0 {
		return true
	}
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), target) {
			return true
		}
	}
	return false
}

func (s *Store) nextDeliveryID(ctx context.Context, tx *sql.Tx) (int64, error) {
	var id int64
	if err := tx.QueryRowContext(ctx, nextDeliveryIDSQL).Scan(&id); err != nil {
		return 0, fmt.Errorf("allocate notification delivery id: %w", err)
	}
	return id, nil
}

func (s *Store) FailCampaignShard(ctx context.Context, input ports.FailCampaignShardInput) error {
	result, err := s.db.ExecContext(ctx,
		failCampaignShardSQL,
		input.ShardID,
		input.WorkerID,
		input.ClaimDeadlineAt,
		input.ErrorCode,
		input.FailedAt,
		int64(input.RetryAfter/time.Second),
	)
	if err != nil {
		return fmt.Errorf("fail campaign shard: %w", err)
	}
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return ports.ErrShardLeaseLost
	}
	return nil
}

func (s *Store) CompleteCampaignShard(ctx context.Context, input ports.CompleteCampaignShardInput) error {
	result, err := s.db.ExecContext(ctx,
		completeCampaignShardSQL,
		input.ShardID,
		input.WorkerID,
		input.ClaimDeadlineAt,
		input.ProcessedCount,
		input.SuccessCount,
		input.SkippedCount,
		input.FailedCount,
		input.NextCursor,
		input.HasMore,
		input.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("complete campaign shard: %w", err)
	}
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return ports.ErrShardLeaseLost
	}
	return nil
}

func (s *Store) RebuildGroupState(ctx context.Context, input ports.RebuildGroupStateInput) (ports.RebuildGroupStateResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ports.RebuildGroupStateResult{}, fmt.Errorf("begin group state rebuild transaction: %w", err)
	}
	defer tx.Rollback()

	var locked bool
	if err := tx.QueryRowContext(ctx, lockRebuildGroupStateSQL, input.RecipientID).Scan(&locked); err != nil {
		return ports.RebuildGroupStateResult{}, fmt.Errorf("lock group state rebuild: %w", err)
	}
	if !locked {
		if err := tx.Commit(); err != nil {
			return ports.RebuildGroupStateResult{}, fmt.Errorf("commit skipped group state rebuild: %w", err)
		}
		return ports.RebuildGroupStateResult{}, ports.ErrRebuildLocked
	}
	if _, err := tx.ExecContext(ctx, deleteGroupStateForRebuildSQL, input.RecipientID); err != nil {
		return ports.RebuildGroupStateResult{}, fmt.Errorf("delete group state before rebuild: %w", err)
	}
	var rebuilt int64
	if err := tx.QueryRowContext(ctx, rebuildGroupStateSQL, input.RecipientID, input.RebuiltAt).Scan(&rebuilt); err != nil {
		return ports.RebuildGroupStateResult{}, fmt.Errorf("rebuild group state from inbox: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return ports.RebuildGroupStateResult{}, fmt.Errorf("commit group state rebuild: %w", err)
	}
	return ports.RebuildGroupStateResult{RebuiltGroups: rebuilt}, nil
}

func nullableTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}
