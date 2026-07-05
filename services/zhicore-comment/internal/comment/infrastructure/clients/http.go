package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/ports"
)

const defaultCallerService = "zhicore-comment"

type Config struct {
	BaseURL       string
	CallerService string
	HTTPClient    *http.Client
}

type baseClient struct {
	baseURL       string
	callerService string
	httpClient    *http.Client
}

func newBaseClient(config Config) baseClient {
	callerService := strings.TrimSpace(config.CallerService)
	if callerService == "" {
		callerService = defaultCallerService
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return baseClient{
		baseURL:       strings.TrimRight(strings.TrimSpace(config.BaseURL), "/"),
		callerService: callerService,
		httpClient:    httpClient,
	}
}

type ContentClient struct {
	base baseClient
}

func NewContentClient(config Config) *ContentClient {
	return &ContentClient{base: newBaseClient(config)}
}

func (c *ContentClient) CheckPostCommentable(ctx context.Context, postID domain.PostID) (ports.CommentablePost, error) {
	path := "/api/v1/internal/posts/" + url.PathEscape(string(postID)) + "/comment-context"
	var data contentCommentContext
	if err := c.base.doJSON(ctx, http.MethodGet, path, "comment.check_post_commentable", nil, &data); err != nil {
		return ports.CommentablePost{}, mapProviderError(err, providerContent)
	}
	if !data.Commentable {
		return ports.CommentablePost{}, ports.ErrPostNotFound
	}
	return ports.CommentablePost{
		PostID:            domain.PostID(data.PostID),
		ContentInternalID: domain.ContentInternalID(data.InternalID),
		AuthorID:          domain.UserID(data.AuthorID),
	}, nil
}

type UserClient struct {
	base baseClient
}

func NewUserClient(config Config) *UserClient {
	return &UserClient{base: newBaseClient(config)}
}

func (c *UserClient) EnsureUserCanComment(ctx context.Context, userID domain.UserID) error {
	if userID <= 0 {
		return ports.ErrUserUnavailable
	}
	var data userAvailabilityBatch
	err := c.base.doJSON(ctx, http.MethodPost, "/api/v1/internal/users/batch-availability", "comment.check_user_availability", userIDsRequest{UserIDs: []int64{int64(userID)}}, &data)
	if err != nil {
		return mapProviderError(err, providerUser)
	}
	for _, item := range data.Items {
		if domain.UserID(item.UserID) == userID && item.Available {
			return nil
		}
	}
	return ports.ErrUserUnavailable
}

func (c *UserClient) BatchGetAuthorSummaries(ctx context.Context, userIDs []domain.UserID) (map[domain.UserID]ports.AuthorSummary, error) {
	result := make(map[domain.UserID]ports.AuthorSummary, len(userIDs))
	if len(userIDs) == 0 {
		return result, nil
	}
	ids := make([]int64, 0, len(userIDs))
	for _, id := range userIDs {
		ids = append(ids, int64(id))
	}
	var data userSimpleBatch
	if err := c.base.doJSON(ctx, http.MethodPost, "/api/v1/internal/users/batch-simple", "comment.batch_get_author_summaries", userIDsRequest{UserIDs: ids}, &data); err != nil {
		return nil, mapProviderError(err, providerUser)
	}
	for _, item := range data.Items {
		result[domain.UserID(item.UserID)] = ports.AuthorSummary{
			UserID:       domain.UserID(item.UserID),
			PublicID:     item.PublicID,
			DisplayName:  item.Nickname,
			AvatarFileID: item.AvatarFileID,
			AvatarURL:    item.AvatarURL,
			Unavailable:  item.Status != "" && item.Status != "ACTIVE",
		}
	}
	for _, missing := range data.MissingUserIDs {
		result[domain.UserID(missing)] = ports.AuthorSummary{UserID: domain.UserID(missing), Unavailable: true}
	}
	return result, nil
}

func (c *UserClient) BatchCheckBlocked(ctx context.Context, pairs []ports.BlockPair) (map[ports.BlockPair]bool, error) {
	result := make(map[ports.BlockPair]bool, len(pairs))
	if len(pairs) == 0 {
		return result, nil
	}
	req := blockPairsRequest{Pairs: make([]blockPairDTO, 0, len(pairs))}
	for _, pair := range pairs {
		result[pair] = false
		req.Pairs = append(req.Pairs, blockPairDTO{BlockerID: int64(pair.BlockerID), BlockedID: int64(pair.BlockedID)})
	}
	var data blockPairsResponse
	if err := c.base.doJSON(ctx, http.MethodPost, "/api/v1/internal/users/blocks/batch-check", "comment.batch_check_blocked", req, &data); err != nil {
		return nil, mapProviderError(err, providerUser)
	}
	for _, item := range data.Items {
		pair := ports.BlockPair{BlockerID: domain.UserID(item.BlockerID), BlockedID: domain.UserID(item.BlockedID)}
		result[pair] = item.Blocked
	}
	return result, nil
}

type FileClient struct {
	base baseClient
}

func NewFileClient(config Config) *FileClient {
	return &FileClient{base: newBaseClient(config)}
}

func (c *FileClient) EnsureCommentMediaReferenced(ctx context.Context, input ports.CommentMediaReferences) error {
	req := commentMediaRequest{
		ImageFileIDs:  append([]string(nil), input.ImageFileIDs...),
		VoiceFileID:   input.VoiceFileID,
		VoiceDuration: input.VoiceDuration,
	}
	if err := c.base.doJSON(ctx, http.MethodPost, "/api/v1/internal/files/comment-media/validate", "comment.ensure_comment_media_referenced", req, nil); err != nil {
		return mapProviderError(err, providerFile)
	}
	return nil
}

func (c baseClient) doJSON(ctx context.Context, method, path, operation string, body any, out any) error {
	if c.baseURL == "" {
		return ports.ErrDependencyUnavailable
	}
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal downstream request: %w", err)
		}
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("build downstream request: %w", err)
	}
	req.Header.Set("X-Caller-Service", c.callerService)
	req.Header.Set("X-Caller-Operation", operation)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return ports.ErrDependencyUnavailable
	}
	defer resp.Body.Close()

	var envelope responseEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return ports.ErrDependencyUnavailable
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices || envelope.Code != 200 {
		return providerError{StatusCode: resp.StatusCode, Code: envelope.Code}
	}
	if out == nil || len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return ports.ErrDependencyUnavailable
	}
	return nil
}

type provider string

const (
	providerContent provider = "content"
	providerUser    provider = "user"
	providerFile    provider = "file"
)

type providerError struct {
	StatusCode int
	Code       int
}

func (e providerError) Error() string {
	return fmt.Sprintf("downstream status=%d code=%d", e.StatusCode, e.Code)
}

func mapProviderError(err error, provider provider) error {
	if errors.Is(err, ports.ErrDependencyUnavailable) {
		return ports.ErrDependencyUnavailable
	}
	var perr providerError
	if !errors.As(err, &perr) {
		return ports.ErrDependencyUnavailable
	}
	if perr.Code == 1004 || perr.StatusCode >= http.StatusInternalServerError {
		return ports.ErrDependencyUnavailable
	}
	switch provider {
	case providerContent:
		if perr.Code == 4001 || perr.StatusCode == http.StatusNotFound {
			return ports.ErrPostNotFound
		}
	case providerUser:
		if perr.Code == 3010 {
			return ports.ErrInteractionBlocked
		}
		if perr.Code == 3001 || perr.Code == 3006 || perr.StatusCode == http.StatusForbidden || perr.StatusCode == http.StatusNotFound {
			return ports.ErrUserUnavailable
		}
	case providerFile:
		if perr.StatusCode == http.StatusBadRequest {
			return domain.ErrCommentMediaInvalid
		}
	}
	return ports.ErrDependencyUnavailable
}

type responseEnvelope struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
}

type contentCommentContext struct {
	PostID      string `json:"postId"`
	InternalID  int64  `json:"internalId"`
	AuthorID    int64  `json:"authorId"`
	Commentable bool   `json:"commentable"`
	Status      string `json:"status"`
}

type userIDsRequest struct {
	UserIDs []int64 `json:"userIds"`
}

type userAvailabilityBatch struct {
	Items []struct {
		UserID    int64  `json:"userId"`
		Available bool   `json:"available"`
		Status    string `json:"status"`
	} `json:"items"`
}

type userSimpleBatch struct {
	Items []struct {
		UserID       int64  `json:"userId"`
		PublicID     string `json:"publicId"`
		Nickname     string `json:"nickname"`
		AvatarFileID string `json:"avatarFileId"`
		AvatarURL    string `json:"avatarUrl"`
		Status       string `json:"status"`
	} `json:"items"`
	MissingUserIDs []int64 `json:"missingUserIds"`
}

type blockPairsRequest struct {
	Pairs []blockPairDTO `json:"pairs"`
}

type blockPairDTO struct {
	BlockerID int64 `json:"blockerId"`
	BlockedID int64 `json:"blockedId"`
}

type blockPairsResponse struct {
	Items []struct {
		BlockerID int64 `json:"blockerId"`
		BlockedID int64 `json:"blockedId"`
		Blocked   bool  `json:"blocked"`
	} `json:"items"`
}

type commentMediaRequest struct {
	ImageFileIDs  []string `json:"imageFileIds,omitempty"`
	VoiceFileID   string   `json:"voiceFileId,omitempty"`
	VoiceDuration int      `json:"voiceDuration,omitempty"`
}

var (
	_ ports.ContentPostClient   = (*ContentClient)(nil)
	_ ports.UserProfileClient   = (*UserClient)(nil)
	_ ports.UserRelationClient  = (*UserClient)(nil)
	_ ports.FileReferenceClient = (*FileClient)(nil)
)
