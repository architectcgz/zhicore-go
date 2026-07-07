package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

const (
	defaultTagListLimit   = 20
	maxTagListLimit       = 100
	defaultTagSearchLimit = 10
	maxTagSearchLimit     = 20
	defaultHotTagLimit    = 20
	maxHotTagLimit        = 50
	maxPostTagCount       = 10
)

var tagSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,95}$`)

type ListTagsQuery struct {
	Cursor string
	Limit  int
}

type TagPageResult struct {
	Items      []Tag
	NextCursor string
	HasMore    bool
	Limit      int
}

type GetTagQuery struct {
	Slug string
}

type SearchTagsQuery struct {
	Query string
	Limit int
}

type ListHotTagsQuery struct {
	Limit int
}

type ListPostsByTagQuery struct {
	Slug   string
	Cursor string
	Limit  int
}

type GetPostTagsQuery struct {
	PostID string
}

type UpdatePostTagsCommand struct {
	Actor           *Actor
	PostID          string
	BasePostVersion int64
	Tags            []string
}

type DeletePostTagCommand struct {
	Actor           *Actor
	PostID          string
	BasePostVersion int64
	Slug            string
}

type PostTagsMutationResult struct {
	PostID      string
	PostVersion int64
	Tags        []Tag
	UpdatedAt   time.Time
}

type Tag struct {
	TagID     string
	Name      string
	Slug      string
	PostCount int64
}

type tagCursorToken struct {
	Slug string `json:"slug"`
	ID   int64  `json:"id"`
}

func (s *Service) ListTags(ctx context.Context, query ListTagsQuery) (TagPageResult, error) {
	if s.taxonomy == nil {
		return TagPageResult{}, ErrDependencyUnavailable
	}
	cursor, err := decodeTagCursor(query.Cursor)
	if err != nil {
		return TagPageResult{}, err
	}
	limit := normalizeLimit(query.Limit, defaultTagListLimit, maxTagListLimit)
	records, err := s.taxonomy.ListTags(ctx, ports.TagListQuery{Cursor: cursor, Limit: limit + 1})
	if err != nil {
		return TagPageResult{}, fmt.Errorf("%w: list tags", ErrDependencyUnavailable)
	}
	return mapTagPage(records, limit), nil
}

func (s *Service) GetTag(ctx context.Context, query GetTagQuery) (Tag, error) {
	if s.taxonomy == nil {
		return Tag{}, ErrDependencyUnavailable
	}
	slug, err := normalizeTagSlug(query.Slug)
	if err != nil {
		return Tag{}, err
	}
	record, err := s.taxonomy.GetTagBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, ErrTaxonomyReferenceNotFound) {
			return Tag{}, err
		}
		return Tag{}, fmt.Errorf("%w: get tag", ErrDependencyUnavailable)
	}
	return mapTag(record), nil
}

func (s *Service) SearchTags(ctx context.Context, query SearchTagsQuery) ([]Tag, error) {
	if s.taxonomy == nil {
		return nil, ErrDependencyUnavailable
	}
	raw := strings.ToLower(strings.TrimSpace(query.Query))
	if raw == "" {
		return nil, ErrInvalidArgument
	}
	limit := normalizeLimit(query.Limit, defaultTagSearchLimit, maxTagSearchLimit)
	records, err := s.taxonomy.SearchTags(ctx, ports.TagSearchQuery{Query: raw, Limit: limit})
	if err != nil {
		return nil, fmt.Errorf("%w: search tags", ErrDependencyUnavailable)
	}
	return mapTags(records), nil
}

func (s *Service) ListHotTags(ctx context.Context, query ListHotTagsQuery) ([]Tag, error) {
	if s.taxonomy == nil {
		return nil, ErrDependencyUnavailable
	}
	limit := normalizeLimit(query.Limit, defaultHotTagLimit, maxHotTagLimit)
	records, err := s.taxonomy.ListHotTags(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("%w: list hot tags", ErrDependencyUnavailable)
	}
	return mapTags(records), nil
}

func (s *Service) ListPostsByTag(ctx context.Context, query ListPostsByTagQuery) (ListPublishedPostsResult, error) {
	if s.taxonomy == nil {
		return ListPublishedPostsResult{}, ErrDependencyUnavailable
	}
	slug, err := normalizeTagSlug(query.Slug)
	if err != nil {
		return ListPublishedPostsResult{}, err
	}
	if _, err := s.taxonomy.GetTagBySlug(ctx, slug); err != nil {
		if errors.Is(err, ErrTaxonomyReferenceNotFound) {
			return ListPublishedPostsResult{}, err
		}
		return ListPublishedPostsResult{}, fmt.Errorf("%w: get tag", ErrDependencyUnavailable)
	}
	cursor, err := decodePublicPostCursor(query.Cursor)
	if err != nil {
		return ListPublishedPostsResult{}, err
	}
	limit := normalizePublicPostLimit(query.Limit)
	records, err := s.taxonomy.ListPublishedPostsByTag(ctx, ports.TaggedPostListQuery{Slug: slug, Cursor: cursor, Limit: limit + 1})
	if err != nil {
		return ListPublishedPostsResult{}, fmt.Errorf("%w: list posts by tag", ErrDependencyUnavailable)
	}
	hasMore := len(records) > limit
	if hasMore {
		records = records[:limit]
	}
	nextCursor := ""
	if hasMore && len(records) > 0 {
		nextCursor = encodePublicPostCursor(records[len(records)-1])
	}
	return ListPublishedPostsResult{Items: mapPostSummaries(records), NextCursor: nextCursor, HasMore: hasMore, Limit: limit}, nil
}

func (s *Service) GetPostTags(ctx context.Context, query GetPostTagsQuery) ([]Tag, error) {
	if s.taxonomy == nil {
		return nil, ErrDependencyUnavailable
	}
	postID := strings.TrimSpace(query.PostID)
	if postID == "" {
		return nil, ErrInvalidArgument
	}
	records, err := s.taxonomy.ListPostTags(ctx, postID)
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: list post tags", ErrDependencyUnavailable)
	}
	return mapTags(records), nil
}

func (s *Service) UpdatePostTags(ctx context.Context, cmd UpdatePostTagsCommand) (PostTagsMutationResult, error) {
	if cmd.Actor == nil || cmd.Actor.UserID == 0 {
		return PostTagsMutationResult{}, ErrLoginRequired
	}
	slugs, err := normalizeTagSlugs(cmd.Tags)
	if err != nil {
		return PostTagsMutationResult{}, err
	}
	return s.mutatePostTags(ctx, postTagMutation{
		Actor:           cmd.Actor,
		PostID:          cmd.PostID,
		BasePostVersion: cmd.BasePostVersion,
		Slugs:           slugs,
		Replace:         true,
	})
}

func (s *Service) DeletePostTag(ctx context.Context, cmd DeletePostTagCommand) (PostTagsMutationResult, error) {
	if cmd.Actor == nil || cmd.Actor.UserID == 0 {
		return PostTagsMutationResult{}, ErrLoginRequired
	}
	slug, err := normalizeTagSlug(cmd.Slug)
	if err != nil {
		return PostTagsMutationResult{}, err
	}
	return s.mutatePostTags(ctx, postTagMutation{
		Actor:           cmd.Actor,
		PostID:          cmd.PostID,
		BasePostVersion: cmd.BasePostVersion,
		Slug:            slug,
	})
}

type postTagMutation struct {
	Actor           *Actor
	PostID          string
	BasePostVersion int64
	Slugs           []string
	Slug            string
	Replace         bool
}

func (s *Service) mutatePostTags(ctx context.Context, mutation postTagMutation) (PostTagsMutationResult, error) {
	if s.posts == nil || s.taxonomy == nil || s.tx == nil || s.clock == nil {
		return PostTagsMutationResult{}, ErrDependencyUnavailable
	}
	if strings.TrimSpace(mutation.PostID) == "" || mutation.BasePostVersion <= 0 {
		return PostTagsMutationResult{}, ErrInvalidArgument
	}
	now := s.clock.Now()
	var result ports.PostTagsMutationRecord
	err := s.tx.WithinTx(ctx, func(ctx context.Context, tx ports.Tx) error {
		current, err := s.posts.GetForUpdate(ctx, tx, mutation.PostID)
		if err != nil {
			return err
		}
		if current.OwnerID != mutation.Actor.UserID {
			return domain.ErrForbidden
		}
		if current.Status == domain.PostStatusDeleted {
			return domain.ErrPostDeleted
		}
		if current.PostVersion != mutation.BasePostVersion {
			return domain.ErrDraftConflict
		}
		if mutation.Replace {
			result, err = s.taxonomy.ReplacePostTags(ctx, tx, ports.ReplacePostTagsInput{
				PostInternalID:  current.ID,
				PostPublicID:    current.PublicID,
				ActorID:         mutation.Actor.UserID,
				BasePostVersion: mutation.BasePostVersion,
				Slugs:           append([]string(nil), mutation.Slugs...),
				UpdatedAt:       now,
			})
			return err
		}
		result, err = s.taxonomy.RemovePostTag(ctx, tx, ports.RemovePostTagInput{
			PostInternalID:  current.ID,
			PostPublicID:    current.PublicID,
			ActorID:         mutation.Actor.UserID,
			BasePostVersion: mutation.BasePostVersion,
			Slug:            mutation.Slug,
			UpdatedAt:       now,
		})
		return err
	})
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) || errors.Is(err, domain.ErrForbidden) ||
			errors.Is(err, domain.ErrPostDeleted) || errors.Is(err, domain.ErrDraftConflict) ||
			errors.Is(err, ErrTaxonomyReferenceNotFound) {
			return PostTagsMutationResult{}, err
		}
		return PostTagsMutationResult{}, fmt.Errorf("%w: mutate post tags", ErrDependencyUnavailable)
	}
	return PostTagsMutationResult{
		PostID:      result.PostID,
		PostVersion: result.PostVersion,
		Tags:        mapTags(result.Tags),
		UpdatedAt:   result.UpdatedAt,
	}, nil
}

func normalizeLimit(value, def, max int) int {
	if value <= 0 {
		return def
	}
	if value > max {
		return max
	}
	return value
}

func normalizeTagSlug(raw string) (string, error) {
	slug := strings.ToLower(strings.TrimSpace(raw))
	if !tagSlugPattern.MatchString(slug) {
		return "", ErrInvalidArgument
	}
	return slug, nil
}

func normalizeTagSlugs(raw []string) ([]string, error) {
	if len(raw) > maxPostTagCount {
		return nil, ErrInvalidArgument
	}
	seen := make(map[string]struct{}, len(raw))
	slugs := make([]string, 0, len(raw))
	for _, item := range raw {
		slug, err := normalizeTagSlug(item)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[slug]; ok {
			continue
		}
		seen[slug] = struct{}{}
		slugs = append(slugs, slug)
	}
	return slugs, nil
}

func mapTagPage(records []ports.TagRecord, limit int) TagPageResult {
	hasMore := len(records) > limit
	if hasMore {
		records = records[:limit]
	}
	nextCursor := ""
	if hasMore && len(records) > 0 {
		nextCursor = encodeTagCursor(records[len(records)-1])
	}
	return TagPageResult{Items: mapTags(records), NextCursor: nextCursor, HasMore: hasMore, Limit: limit}
}

func mapTags(records []ports.TagRecord) []Tag {
	items := make([]Tag, 0, len(records))
	for _, record := range records {
		items = append(items, mapTag(record))
	}
	return items
}

func mapTag(record ports.TagRecord) Tag {
	return Tag{TagID: record.PublicID, Name: record.Name, Slug: record.Slug, PostCount: record.PostCount}
}

func encodeTagCursor(record ports.TagRecord) string {
	payload, _ := json.Marshal(tagCursorToken{Slug: record.Slug, ID: record.ID})
	return base64.RawURLEncoding.EncodeToString(payload)
}

func decodeTagCursor(raw string) (ports.TagCursor, error) {
	if strings.TrimSpace(raw) == "" {
		return ports.TagCursor{}, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return ports.TagCursor{}, ErrInvalidArgument
	}
	var token tagCursorToken
	if err := json.Unmarshal(payload, &token); err != nil {
		return ports.TagCursor{}, ErrInvalidArgument
	}
	slug, err := normalizeTagSlug(token.Slug)
	if err != nil || token.ID <= 0 {
		return ports.TagCursor{}, ErrInvalidArgument
	}
	return ports.TagCursor{Slug: slug, ID: token.ID}, nil
}
