package comment

const (
	EventCommentCreated = "comment.created"
	EventCommentDeleted = "comment.deleted"
	EventCommentLiked   = "comment.liked"
	EventCommentUnliked = "comment.unliked"
)

// CommentCreatedPayload is the version 1 payload for comment.created.
// Reply-only fields are pointers so top-level comment events omit them.
type CommentCreatedPayload struct {
	CommentID      int64  `json:"commentId"`
	PublicID       string `json:"publicId"`
	InternalID     int64  `json:"internalId"`
	PostAuthorID   int64  `json:"postAuthorId"`
	AuthorID       int64  `json:"authorId"`
	RootID         *int64 `json:"rootId,omitempty"`
	RootAuthorID   *int64 `json:"rootAuthorId,omitempty"`
	ParentID       *int64 `json:"parentId,omitempty"`
	ParentAuthorID *int64 `json:"parentAuthorId,omitempty"`
	HasImages      bool   `json:"hasImages"`
	HasVoice       bool   `json:"hasVoice"`
	CreatedAt      string `json:"createdAt"`
}

// CommentDeletedPayload is the version 1 payload for comment.deleted.
type CommentDeletedPayload struct {
	CommentID     int64   `json:"commentId"`
	PublicID      string  `json:"publicId"`
	InternalID    int64   `json:"internalId"`
	RootID        *int64  `json:"rootId,omitempty"`
	AuthorID      int64   `json:"authorId"`
	DeletedBy     int64   `json:"deletedBy"`
	DeletedByRole string  `json:"deletedByRole"`
	DeleteReason  *string `json:"deleteReason,omitempty"`
	DeletedAt     string  `json:"deletedAt"`
	IsRoot        bool    `json:"isRoot"`
	AffectedCount int     `json:"affectedCount"`
}

// CommentLikedPayload is the version 1 payload for comment.liked.
type CommentLikedPayload struct {
	CommentID       int64  `json:"commentId"`
	PublicID        string `json:"publicId"`
	InternalID      int64  `json:"internalId"`
	CommentAuthorID int64  `json:"commentAuthorId"`
	LikedBy         int64  `json:"likedBy"`
	OccurredAt      string `json:"occurredAt"`
}

// CommentUnlikedPayload is the version 1 payload for comment.unliked.
type CommentUnlikedPayload struct {
	CommentID       int64  `json:"commentId"`
	PublicID        string `json:"publicId"`
	InternalID      int64  `json:"internalId"`
	CommentAuthorID int64  `json:"commentAuthorId"`
	UnlikedBy       int64  `json:"unlikedBy"`
	OccurredAt      string `json:"occurredAt"`
}
