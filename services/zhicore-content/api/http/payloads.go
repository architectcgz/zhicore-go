package httpapi

import (
	"encoding/json"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/application"
)

type createPostReq struct {
	Title       string       `json:"title"`
	Summary     string       `json:"summary"`
	CoverFileID string       `json:"coverFileId"`
	TopicID     string       `json:"topicId"`
	CategoryID  string       `json:"categoryId"`
	Tags        []string     `json:"tags"`
	Body        *postBodyReq `json:"body"`
}

type postBodyReq struct {
	SchemaVersion int                `json:"schemaVersion"`
	Blocks        application.Blocks `json:"blocks"`
}

type saveDraftBodyReq struct {
	BasePostVersion   int64              `json:"basePostVersion"`
	BaseDraftBodyID   string             `json:"baseDraftBodyId"`
	BaseDraftBodyHash string             `json:"baseDraftBodyHash"`
	SchemaVersion     int                `json:"schemaVersion"`
	Blocks            application.Blocks `json:"blocks"`
	ClientSavedAt     string             `json:"clientSavedAt"`
}

type publishPostReq struct {
	BasePostVersion int64  `json:"basePostVersion"`
	DraftBodyID     string `json:"draftBodyId"`
	DraftBodyHash   string `json:"draftBodyHash"`
}

type postLifecycleReq struct {
	BasePostVersion int64 `json:"basePostVersion"`
}

type schedulePostReq struct {
	BasePostVersion int64  `json:"basePostVersion"`
	DraftBodyID     string `json:"draftBodyId"`
	DraftBodyHash   string `json:"draftBodyHash"`
	ScheduledAt     string `json:"scheduledAt"`
}

type batchGetPostsReq struct {
	PostIDs        []string `json:"postIds"`
	IncludeDeleted bool     `json:"includeDeleted"`
}

type updateDraftMetaReq struct {
	BasePostVersion int64     `json:"basePostVersion"`
	Title           *string   `json:"title"`
	Summary         *string   `json:"summary"`
	CoverFileID     *string   `json:"coverFileId"`
	TopicID         *string   `json:"topicId"`
	CategoryID      *string   `json:"categoryId"`
	Tags            *[]string `json:"tags"`
}

type createPostResp struct {
	PostID      string `json:"postId"`
	PostVersion int64  `json:"postVersion"`
}

type saveDraftBodyResp struct {
	PostID        string `json:"postId"`
	PostVersion   int64  `json:"postVersion"`
	DraftBodyID   string `json:"draftBodyId"`
	DraftBodyHash string `json:"draftBodyHash"`
	SavedAt       string `json:"savedAt"`
	WordCount     int    `json:"wordCount"`
}

type publishPostResp struct {
	PostID      string `json:"postId"`
	PostVersion int64  `json:"postVersion"`
	PublishedAt string `json:"publishedAt"`
}

type postLifecycleResp struct {
	PostID      string `json:"postId"`
	PostVersion int64  `json:"postVersion"`
	Status      string `json:"status"`
	UpdatedAt   string `json:"updatedAt"`
}

type schedulePostResp struct {
	PostID      string `json:"postId"`
	PostVersion int64  `json:"postVersion"`
	Status      string `json:"status"`
	ScheduledAt string `json:"scheduledAt"`
}

type postBodyResp struct {
	BodyID        string          `json:"bodyId"`
	SchemaVersion int             `json:"schemaVersion"`
	Format        string          `json:"format"`
	Blocks        json.RawMessage `json:"blocks"`
	PlainText     string          `json:"plainText"`
	ContentHash   string          `json:"contentHash"`
	SizeBytes     int             `json:"sizeBytes"`
	CreatedAt     string          `json:"createdAt"`
}

type cursorPageResp[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"nextCursor,omitempty"`
	HasMore    bool   `json:"hasMore"`
	Limit      int    `json:"limit"`
}

type postDetailResp struct {
	Post postSummaryResp `json:"post"`
	Body *postBodyResp   `json:"body,omitempty"`
}

type postSummaryResp struct {
	PostID             string        `json:"postId"`
	AuthorID           string        `json:"authorId"`
	AuthorName         string        `json:"authorName,omitempty"`
	AuthorAvatarFileID string        `json:"authorAvatarFileId,omitempty"`
	Title              string        `json:"title"`
	Summary            string        `json:"summary,omitempty"`
	CoverFileID        string        `json:"coverFileId,omitempty"`
	Status             string        `json:"status"`
	PostVersion        int64         `json:"postVersion"`
	PublishedAt        string        `json:"publishedAt,omitempty"`
	CreatedAt          string        `json:"createdAt"`
	UpdatedAt          string        `json:"updatedAt"`
	Stats              postStatsResp `json:"stats"`
}

type postStatsResp struct {
	ViewCount     int64 `json:"viewCount"`
	LikeCount     int64 `json:"likeCount"`
	FavoriteCount int64 `json:"favoriteCount"`
	CommentCount  int64 `json:"commentCount"`
}

type batchGetPostsResp struct {
	Items          []postSummaryResp `json:"items"`
	MissingPostIDs []string          `json:"missingPostIds"`
}

type authorDraftResp struct {
	PostID        string        `json:"postId"`
	PostVersion   int64         `json:"postVersion"`
	Title         string        `json:"title"`
	Summary       string        `json:"summary,omitempty"`
	CoverFileID   string        `json:"coverFileId,omitempty"`
	Status        string        `json:"status"`
	DraftBodyID   string        `json:"draftBodyId,omitempty"`
	DraftBodyHash string        `json:"draftBodyHash,omitempty"`
	Body          *postBodyResp `json:"body,omitempty"`
	CreatedAt     string        `json:"createdAt"`
	UpdatedAt     string        `json:"updatedAt"`
}

type draftMutationResp struct {
	PostID      string `json:"postId"`
	PostVersion int64  `json:"postVersion"`
	Title       string `json:"title,omitempty"`
	Summary     string `json:"summary,omitempty"`
	CoverFileID string `json:"coverFileId,omitempty"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
}

type adminOutboxRetryReq struct {
	Reason string `json:"reason"`
}

type adminOutboxListResp struct {
	Items []adminOutboxEventResp `json:"items"`
	Page  int                    `json:"page"`
	Size  int                    `json:"size"`
	Total int64                  `json:"total"`
}

type adminOutboxEventResp struct {
	EventID          string `json:"eventId"`
	EventType        string `json:"eventType"`
	AggregateType    string `json:"aggregateType"`
	AggregateID      string `json:"aggregateId"`
	AggregateVersion int64  `json:"aggregateVersion"`
	Status           string `json:"status"`
	RetryCount       int    `json:"retryCount"`
	LastError        string `json:"lastError"`
	OccurredAt       string `json:"occurredAt"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
}

type adminOutboxRetryResp struct {
	EventID    string `json:"eventId"`
	Status     string `json:"status"`
	RetryCount int    `json:"retryCount"`
	RetriedAt  string `json:"retriedAt"`
}

type tagResp struct {
	TagID     string `json:"tagId"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	PostCount int64  `json:"postCount"`
}

type updatePostTagsReq struct {
	BasePostVersion int64    `json:"basePostVersion"`
	Tags            *[]string `json:"tags"`
}

type postTagsMutationResp struct {
	PostID      string    `json:"postId"`
	PostVersion int64     `json:"postVersion"`
	Tags        []tagResp `json:"tags"`
	UpdatedAt   string    `json:"updatedAt"`
}
