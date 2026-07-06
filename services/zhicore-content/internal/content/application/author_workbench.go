package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

const (
	defaultAuthorPostLimit = 20
	maxAuthorPostLimit     = 100
)

type ListAuthorPostsQuery struct {
	Actor  *Actor
	Status string
	Cursor string
	Limit  int
}

type ListAuthorDraftsQuery struct {
	Actor  *Actor
	Cursor string
	Limit  int
}

type AuthorPostPageResult struct {
	Items      []PostSummary
	NextCursor string
	HasMore    bool
	Limit      int
}

type GetAuthorDraftQuery struct {
	Actor  *Actor
	PostID string
}

type AuthorDraftResult struct {
	PostID        string
	PostVersion   int64
	Title         string
	Summary       string
	CoverFileID   string
	Status        string
	DraftBodyID   string
	DraftBodyHash string
	Body          *PostBodyResult
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type UpdateDraftMetaCommand struct {
	Actor           *Actor
	PostID          string
	BasePostVersion int64
	Title           *string
	Summary         *string
	CoverFileID     *string
	TopicID         *string
	CategoryID      *string
	Tags            *[]string
}

type DeleteAuthorDraftCommand struct {
	Actor  *Actor
	PostID string
}

type DraftMutationResult struct {
	PostID      string
	PostVersion int64
	Title       string
	Summary     string
	CoverFileID string
	UpdatedAt   time.Time
}

type authorPostCursorToken struct {
	UpdatedAt string `json:"updatedAt"`
	PostID    string `json:"postId"`
}

func (s *Service) ListAuthorPosts(ctx context.Context, query ListAuthorPostsQuery) (AuthorPostPageResult, error) {
	if query.Actor == nil || query.Actor.UserID == 0 {
		return AuthorPostPageResult{}, ErrLoginRequired
	}
	status, err := normalizeAuthorPostStatus(query.Status)
	if err != nil {
		return AuthorPostPageResult{}, err
	}
	return s.listAuthorPosts(ctx, query.Actor.UserID, status, query.Cursor, query.Limit)
}

func (s *Service) ListAuthorDrafts(ctx context.Context, query ListAuthorDraftsQuery) (AuthorPostPageResult, error) {
	if query.Actor == nil || query.Actor.UserID == 0 {
		return AuthorPostPageResult{}, ErrLoginRequired
	}
	return s.listAuthorPosts(ctx, query.Actor.UserID, string(domain.PostStatusDraft), query.Cursor, query.Limit)
}

func (s *Service) listAuthorPosts(ctx context.Context, ownerID int64, status, rawCursor string, rawLimit int) (AuthorPostPageResult, error) {
	if s.queries == nil {
		return AuthorPostPageResult{}, ErrDependencyUnavailable
	}
	cursor, err := decodeAuthorPostCursor(rawCursor)
	if err != nil {
		return AuthorPostPageResult{}, err
	}
	limit := normalizeAuthorPostLimit(rawLimit)
	records, err := s.queries.ListAuthorPosts(ctx, ports.AuthorPostListQuery{
		OwnerID: ownerID,
		Status:  status,
		Cursor:  cursor,
		Limit:   limit + 1,
	})
	if err != nil {
		return AuthorPostPageResult{}, fmt.Errorf("%w: list author posts", ErrDependencyUnavailable)
	}
	hasMore := len(records) > limit
	if hasMore {
		records = records[:limit]
	}
	nextCursor := ""
	if hasMore && len(records) > 0 {
		nextCursor = encodeAuthorPostCursor(records[len(records)-1])
	}
	return AuthorPostPageResult{Items: mapPostSummaries(records), NextCursor: nextCursor, HasMore: hasMore, Limit: limit}, nil
}

func (s *Service) GetAuthorDraft(ctx context.Context, query GetAuthorDraftQuery) (AuthorDraftResult, error) {
	if query.Actor == nil || query.Actor.UserID == 0 {
		return AuthorDraftResult{}, ErrLoginRequired
	}
	if strings.TrimSpace(query.PostID) == "" {
		return AuthorDraftResult{}, ErrInvalidArgument
	}
	draft, err := s.queries.GetDraftPost(ctx, query.PostID)
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			return AuthorDraftResult{}, err
		}
		return AuthorDraftResult{}, fmt.Errorf("%w: get draft", ErrDependencyUnavailable)
	}
	if draft.Post.OwnerID != query.Actor.UserID {
		return AuthorDraftResult{}, domain.ErrForbidden
	}
	if draft.Post.Status == domain.PostStatusDeleted {
		return AuthorDraftResult{}, domain.ErrPostDeleted
	}
	result := mapAuthorDraft(draft)
	if draft.Post.DraftBodyID != "" {
		body, err := s.readDraftBody(ctx, draft.Post.ID, draft.Post.DraftBodyID, draft.Post.DraftBodyHash)
		if err != nil {
			return AuthorDraftResult{}, err
		}
		result.Body = &body
	}
	return result, nil
}

func (s *Service) UpdateDraftMeta(ctx context.Context, cmd UpdateDraftMetaCommand) (DraftMutationResult, error) {
	if cmd.Actor == nil || cmd.Actor.UserID == 0 {
		return DraftMutationResult{}, ErrLoginRequired
	}
	if cmd.TopicID != nil || cmd.CategoryID != nil || cmd.Tags != nil {
		return DraftMutationResult{}, ErrInvalidArgument
	}
	current, err := s.loadPostForDraftWrite(ctx, cmd.PostID)
	if err != nil {
		return DraftMutationResult{}, err
	}
	if current.OwnerID != cmd.Actor.UserID {
		return DraftMutationResult{}, domain.ErrForbidden
	}
	if current.Status == domain.PostStatusDeleted {
		return DraftMutationResult{}, domain.ErrPostDeleted
	}
	if current.Status == domain.PostStatusScheduled {
		// Scheduled publish records pin the current draft metadata and body;
		// editing requires canceling the schedule so the queued job cannot drift.
		return DraftMutationResult{}, domain.ErrDraftConflict
	}
	if current.PostVersion != cmd.BasePostVersion {
		return DraftMutationResult{}, domain.ErrDraftConflict
	}
	var title *string
	if cmd.Title != nil {
		normalized, err := domain.NewPostTitle(*cmd.Title)
		if err != nil {
			return DraftMutationResult{}, err
		}
		value := string(normalized)
		title = &value
	}
	summary := optionalTrimmedString(cmd.Summary)
	cover := optionalStringUpdate(cmd.CoverFileID)
	if cover.Set && cover.Value != "" && s.files != nil {
		if err := s.files.ValidateCoverFile(ctx, cover.Value); err != nil {
			return DraftMutationResult{}, mapFileValidationError(err, ErrCoverUnavailable)
		}
	}
	topic := optionalStringUpdate(cmd.TopicID)
	category := optionalStringUpdate(cmd.CategoryID)
	tags := normalizeOptionalTags(cmd.Tags)

	now := s.clock.Now()
	var updated ports.PostRecord
	err = s.tx.WithinTx(ctx, func(ctx context.Context, tx ports.Tx) error {
		var err error
		updated, err = s.posts.UpdateDraftMeta(ctx, tx, ports.UpdateDraftMetaUpdate{
			PublicID:        cmd.PostID,
			OwnerID:         cmd.Actor.UserID,
			BasePostVersion: cmd.BasePostVersion,
			Title:           title,
			Summary:         summary,
			CoverFileID:     cover,
			TopicID:         topic,
			CategoryID:      category,
			Tags:            tags,
			UpdatedAt:       now,
		})
		return err
	})
	if err != nil {
		if errors.Is(err, domain.ErrDraftConflict) || errors.Is(err, domain.ErrForbidden) ||
			errors.Is(err, domain.ErrPostDeleted) || errors.Is(err, ErrTaxonomyReferenceNotFound) {
			return DraftMutationResult{}, err
		}
		return DraftMutationResult{}, fmt.Errorf("%w: update draft meta", ErrDependencyUnavailable)
	}
	return mapDraftMutation(updated, now), nil
}

func (s *Service) DeleteAuthorDraft(ctx context.Context, cmd DeleteAuthorDraftCommand) (DraftMutationResult, error) {
	if cmd.Actor == nil || cmd.Actor.UserID == 0 {
		return DraftMutationResult{}, ErrLoginRequired
	}

	now := s.clock.Now()
	var updated ports.PostRecord
	err := s.tx.WithinTx(ctx, func(ctx context.Context, tx ports.Tx) error {
		current, err := s.posts.GetForUpdate(ctx, tx, cmd.PostID)
		if err != nil {
			return err
		}
		if current.OwnerID != cmd.Actor.UserID {
			return domain.ErrForbidden
		}
		if current.Status == domain.PostStatusDeleted {
			return domain.ErrPostDeleted
		}
		if current.Status == domain.PostStatusScheduled {
			// Deleting the draft would orphan the pending scheduled publish
			// intent, so authors must cancel the schedule first.
			return domain.ErrDraftConflict
		}
		updated, err = s.posts.DeleteDraft(ctx, tx, ports.DeleteDraftUpdate{PublicID: cmd.PostID, OwnerID: cmd.Actor.UserID, UpdatedAt: now})
		if err != nil {
			return err
		}
		if current.DraftBodyID != "" && s.cleanup != nil {
			return s.cleanup.Append(ctx, tx, ports.BodyCleanupTask{
				PostID:    current.ID,
				BodyID:    current.DraftBodyID,
				TaskType:  "OLD_DRAFT",
				Reason:    "draft_replaced",
				CreatedAt: now,
			})
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, domain.ErrForbidden) || errors.Is(err, domain.ErrPostDeleted) ||
			errors.Is(err, domain.ErrPostNotFound) || errors.Is(err, domain.ErrDraftConflict) {
			return DraftMutationResult{}, err
		}
		return DraftMutationResult{}, fmt.Errorf("%w: delete draft", ErrDependencyUnavailable)
	}
	return mapDraftMutation(updated, now), nil
}

func (s *Service) readDraftBody(ctx context.Context, postID int64, bodyID, expectedHash string) (PostBodyResult, error) {
	body, err := s.bodies.ReadBody(ctx, bodyID)
	if err != nil {
		s.appendRepairTask(ctx, ports.BodyRepairTask{
			PostID:       postID,
			BodyID:       bodyID,
			TaskType:     "draft_body_missing",
			ExpectedHash: expectedHash,
			CreatedAt:    s.clock.Now(),
		})
		if errors.Is(err, domain.ErrBodyUnavailable) {
			return PostBodyResult{}, err
		}
		return PostBodyResult{}, fmt.Errorf("%w: read draft body", ErrDependencyUnavailable)
	}
	if expectedHash != "" && body.ContentHash != expectedHash {
		s.appendRepairTask(ctx, ports.BodyRepairTask{
			PostID:       postID,
			BodyID:       bodyID,
			TaskType:     "body_hash_mismatch",
			ExpectedHash: expectedHash,
			ObservedHash: body.ContentHash,
			CreatedAt:    s.clock.Now(),
		})
		return PostBodyResult{}, domain.ErrBodyInconsistent
	}
	normalized, err := s.validateStoredBody(ctx, body)
	if err != nil {
		return PostBodyResult{}, err
	}
	return PostBodyResult{
		BodyID:        body.ID,
		SchemaVersion: body.SchemaVersion,
		CanonicalJSON: append([]byte(nil), normalized.CanonicalJSON...),
		PlainText:     normalized.PlainText,
		ContentHash:   normalized.ContentHash,
		SizeBytes:     normalized.SizeBytes,
		CreatedAt:     body.CreatedAt,
	}, nil
}

func normalizeAuthorPostStatus(raw string) (string, error) {
	status := strings.ToUpper(strings.TrimSpace(raw))
	if status == "" || status == "ALL" {
		return "", nil
	}
	switch domain.PostStatus(status) {
	case domain.PostStatusDraft, domain.PostStatusPublished, domain.PostStatusScheduled, domain.PostStatusDeleted:
		return status, nil
	default:
		return "", ErrInvalidArgument
	}
}

func normalizeAuthorPostLimit(limit int) int {
	if limit <= 0 {
		return defaultAuthorPostLimit
	}
	if limit > maxAuthorPostLimit {
		return maxAuthorPostLimit
	}
	return limit
}

func decodeAuthorPostCursor(raw string) (ports.AuthorPostCursor, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ports.AuthorPostCursor{}, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(trimmed)
	if err != nil {
		return ports.AuthorPostCursor{}, ErrInvalidArgument
	}
	var token authorPostCursorToken
	if err := json.Unmarshal(payload, &token); err != nil {
		return ports.AuthorPostCursor{}, ErrInvalidArgument
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, token.UpdatedAt)
	if err != nil || strings.TrimSpace(token.PostID) == "" {
		return ports.AuthorPostCursor{}, ErrInvalidArgument
	}
	return ports.AuthorPostCursor{UpdatedAt: updatedAt, PublicID: token.PostID}, nil
}

func encodeAuthorPostCursor(record ports.PostSummaryRecord) string {
	payload, _ := json.Marshal(authorPostCursorToken{
		UpdatedAt: record.UpdatedAt.UTC().Format(time.RFC3339Nano),
		PostID:    record.PostID,
	})
	return base64.RawURLEncoding.EncodeToString(payload)
}

func optionalTrimmedString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

func optionalStringUpdate(value *string) ports.OptionalStringUpdate {
	if value == nil {
		return ports.OptionalStringUpdate{}
	}
	return ports.OptionalStringUpdate{Set: true, Value: strings.TrimSpace(*value)}
}

func normalizeOptionalTags(tags *[]string) *[]string {
	if tags == nil {
		return nil
	}
	normalized := make([]string, 0, len(*tags))
	for _, tag := range *tags {
		if trimmed := strings.TrimSpace(tag); trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return &normalized
}

func mapAuthorDraft(record ports.DraftPostRecord) AuthorDraftResult {
	return AuthorDraftResult{
		PostID:        record.Post.PublicID,
		PostVersion:   record.Post.PostVersion,
		Title:         record.Post.DraftTitle,
		Summary:       record.Post.DraftSummary,
		CoverFileID:   record.Post.DraftCoverFileID,
		Status:        string(record.Post.Status),
		DraftBodyID:   record.Post.DraftBodyID,
		DraftBodyHash: record.Post.DraftBodyHash,
		CreatedAt:     record.CreatedAt,
		UpdatedAt:     record.UpdatedAt,
	}
}

func mapDraftMutation(record ports.PostRecord, updatedAt time.Time) DraftMutationResult {
	return DraftMutationResult{
		PostID:      record.PublicID,
		PostVersion: record.PostVersion,
		Title:       record.DraftTitle,
		Summary:     record.DraftSummary,
		CoverFileID: record.DraftCoverFileID,
		UpdatedAt:   updatedAt,
	}
}
