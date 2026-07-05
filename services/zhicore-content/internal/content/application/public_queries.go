package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

const (
	defaultPublicPostLimit = 20
	maxPublicPostLimit     = 100
	maxPublicBatchPostIDs  = 100
)

type ListPublishedPostsQuery struct {
	AuthorID   string
	Tag        string
	CategoryID string
	Cursor     string
	Limit      int
	Sort       string
}

type ListPublishedPostsResult struct {
	Items      []PostSummary
	NextCursor string
	HasMore    bool
	Limit      int
}

type GetPostDetailQuery struct {
	PostID string
}

type GetPostDetailResult struct {
	Post PostSummary
	Body *PostBodyResult
}

type BatchGetPublishedPostsQuery struct {
	PostIDs        []string
	IncludeDeleted bool
}

type BatchGetPublishedPostsResult struct {
	Items          []PostSummary
	MissingPostIDs []string
}

type PostSummary struct {
	PostID             string
	AuthorID           string
	AuthorName         string
	AuthorAvatarFileID string
	Title              string
	Summary            string
	CoverFileID        string
	Status             string
	PostVersion        int64
	PublishedAt        time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
	Stats              PostStats
}

type PostStats struct {
	ViewCount     int64
	LikeCount     int64
	FavoriteCount int64
	CommentCount  int64
}

type PostBodyResult struct {
	BodyID        string
	SchemaVersion int
	CanonicalJSON []byte
	PlainText     string
	ContentHash   string
	SizeBytes     int
	CreatedAt     time.Time
}

type publicPostCursorToken struct {
	PublishedAt string `json:"publishedAt"`
	PostID      string `json:"postId"`
}

func (s *Service) ListPublishedPosts(ctx context.Context, query ListPublishedPostsQuery) (ListPublishedPostsResult, error) {
	if s.queries == nil {
		return ListPublishedPostsResult{}, ErrDependencyUnavailable
	}
	if sortKey := strings.TrimSpace(query.Sort); sortKey != "" && sortKey != "latest" {
		return ListPublishedPostsResult{}, ErrInvalidArgument
	}
	if strings.TrimSpace(query.Tag) != "" || strings.TrimSpace(query.CategoryID) != "" {
		return ListPublishedPostsResult{}, ErrInvalidArgument
	}
	authorID, err := parseOptionalAuthorID(query.AuthorID)
	if err != nil {
		return ListPublishedPostsResult{}, err
	}
	cursor, err := decodePublicPostCursor(query.Cursor)
	if err != nil {
		return ListPublishedPostsResult{}, err
	}
	limit := normalizePublicPostLimit(query.Limit)
	records, err := s.queries.ListPublishedPosts(ctx, ports.PostListQuery{
		AuthorID: authorID,
		Cursor:   cursor,
		Limit:    limit + 1,
	})
	if err != nil {
		return ListPublishedPostsResult{}, fmt.Errorf("%w: list published posts", ErrDependencyUnavailable)
	}

	hasMore := len(records) > limit
	if hasMore {
		records = records[:limit]
	}
	items := mapPostSummaries(records)
	nextCursor := ""
	if hasMore && len(records) > 0 {
		nextCursor = encodePublicPostCursor(records[len(records)-1])
	}
	return ListPublishedPostsResult{Items: items, NextCursor: nextCursor, HasMore: hasMore, Limit: limit}, nil
}

func (s *Service) GetPostDetail(ctx context.Context, query GetPostDetailQuery) (GetPostDetailResult, error) {
	if s.queries == nil {
		return GetPostDetailResult{}, ErrDependencyUnavailable
	}
	postID := strings.TrimSpace(query.PostID)
	if postID == "" {
		return GetPostDetailResult{}, ErrInvalidArgument
	}
	detail, err := s.queries.GetPublishedPostDetail(ctx, postID)
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			return GetPostDetailResult{}, err
		}
		return GetPostDetailResult{}, fmt.Errorf("%w: get post detail", ErrDependencyUnavailable)
	}
	body, err := s.readPublishedBody(ctx, detail.InternalPostID, detail.PublishedBodyID, detail.PublishedHash)
	if err != nil {
		return GetPostDetailResult{}, err
	}
	return GetPostDetailResult{Post: mapPostSummary(detail.Summary), Body: &body}, nil
}

func (s *Service) BatchGetPublishedPosts(ctx context.Context, query BatchGetPublishedPostsQuery) (BatchGetPublishedPostsResult, error) {
	if s.queries == nil {
		return BatchGetPublishedPostsResult{}, ErrDependencyUnavailable
	}
	ids, err := normalizeBatchPostIDs(query.PostIDs)
	if err != nil {
		return BatchGetPublishedPostsResult{}, err
	}
	records, err := s.queries.BatchGetPublishedPostSummaries(ctx, ids)
	if err != nil {
		return BatchGetPublishedPostsResult{}, fmt.Errorf("%w: batch get published posts", ErrDependencyUnavailable)
	}
	byID := make(map[string]ports.PostSummaryRecord, len(records))
	for _, record := range records {
		byID[record.PostID] = record
	}

	items := make([]PostSummary, 0, len(records))
	missing := make([]string, 0)
	for _, id := range ids {
		record, ok := byID[id]
		if !ok {
			missing = append(missing, id)
			continue
		}
		items = append(items, mapPostSummary(record))
	}
	return BatchGetPublishedPostsResult{Items: items, MissingPostIDs: missing}, nil
}

func (s *Service) readPublishedBody(ctx context.Context, postID int64, bodyID, expectedHash string) (PostBodyResult, error) {
	if bodyID == "" || expectedHash == "" {
		return PostBodyResult{}, domain.ErrPostNotFound
	}
	body, err := s.bodies.ReadBody(ctx, bodyID)
	if err != nil {
		taskType := "mongo_read_error_after_pg_published"
		if errors.Is(err, domain.ErrBodyUnavailable) {
			taskType = "published_body_missing"
		}
		s.appendRepairTask(ctx, ports.BodyRepairTask{
			PostID:       postID,
			BodyID:       bodyID,
			TaskType:     taskType,
			ExpectedHash: expectedHash,
			CreatedAt:    s.clock.Now(),
		})
		if errors.Is(err, domain.ErrBodyUnavailable) {
			return PostBodyResult{}, err
		}
		return PostBodyResult{}, fmt.Errorf("%w: read published body", ErrDependencyUnavailable)
	}
	if body.ContentHash != expectedHash {
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
		taskType := "mongo_read_error_after_pg_published"
		if errors.Is(err, domain.ErrBodyInconsistent) {
			taskType = "body_hash_mismatch"
		}
		s.appendRepairTask(ctx, ports.BodyRepairTask{
			PostID:       postID,
			BodyID:       bodyID,
			TaskType:     taskType,
			ExpectedHash: expectedHash,
			ObservedHash: body.ContentHash,
			CreatedAt:    s.clock.Now(),
		})
		return PostBodyResult{}, err
	}
	if normalized.ContentHash != expectedHash {
		s.appendRepairTask(ctx, ports.BodyRepairTask{
			PostID:       postID,
			BodyID:       bodyID,
			TaskType:     "body_hash_mismatch",
			ExpectedHash: expectedHash,
			ObservedHash: normalized.ContentHash,
			CreatedAt:    s.clock.Now(),
		})
		return PostBodyResult{}, domain.ErrBodyInconsistent
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

func normalizePublicPostLimit(limit int) int {
	if limit <= 0 {
		return defaultPublicPostLimit
	}
	if limit > maxPublicPostLimit {
		return maxPublicPostLimit
	}
	return limit
}

func parseOptionalAuthorID(raw string) (int64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, nil
	}
	value, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil || value <= 0 {
		return 0, ErrInvalidArgument
	}
	return value, nil
}

func decodePublicPostCursor(raw string) (ports.PublishedPostCursor, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ports.PublishedPostCursor{}, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(trimmed)
	if err != nil {
		return ports.PublishedPostCursor{}, ErrInvalidArgument
	}
	var token publicPostCursorToken
	if err := json.Unmarshal(payload, &token); err != nil {
		return ports.PublishedPostCursor{}, ErrInvalidArgument
	}
	publishedAt, err := time.Parse(time.RFC3339Nano, token.PublishedAt)
	if err != nil || strings.TrimSpace(token.PostID) == "" {
		return ports.PublishedPostCursor{}, ErrInvalidArgument
	}
	return ports.PublishedPostCursor{PublishedAt: publishedAt, PublicID: token.PostID}, nil
}

func encodePublicPostCursor(record ports.PostSummaryRecord) string {
	payload, _ := json.Marshal(publicPostCursorToken{
		PublishedAt: record.PublishedAt.UTC().Format(time.RFC3339Nano),
		PostID:      record.PostID,
	})
	return base64.RawURLEncoding.EncodeToString(payload)
}

func normalizeBatchPostIDs(raw []string) ([]string, error) {
	if len(raw) == 0 || len(raw) > maxPublicBatchPostIDs {
		return nil, ErrInvalidArgument
	}
	ids := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, item := range raw {
		id := strings.TrimSpace(item)
		if id == "" {
			return nil, ErrInvalidArgument
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, ErrInvalidArgument
	}
	return ids, nil
}

func mapPostSummaries(records []ports.PostSummaryRecord) []PostSummary {
	items := make([]PostSummary, 0, len(records))
	for _, record := range records {
		items = append(items, mapPostSummary(record))
	}
	return items
}

func mapPostSummary(record ports.PostSummaryRecord) PostSummary {
	return PostSummary{
		PostID:             record.PostID,
		AuthorID:           strconv.FormatInt(record.AuthorID, 10),
		AuthorName:         record.AuthorName,
		AuthorAvatarFileID: record.AuthorAvatarFileID,
		Title:              record.Title,
		Summary:            record.Summary,
		CoverFileID:        record.CoverFileID,
		Status:             string(record.Status),
		PostVersion:        record.PostVersion,
		PublishedAt:        record.PublishedAt,
		CreatedAt:          record.CreatedAt,
		UpdatedAt:          record.UpdatedAt,
		Stats: PostStats{
			ViewCount:     record.ViewCount,
			LikeCount:     record.LikeCount,
			FavoriteCount: record.FavoriteCount,
			CommentCount:  record.CommentCount,
		},
	}
}
