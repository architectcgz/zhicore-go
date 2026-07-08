package ports

import (
	"context"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
)

// AdminPostRepository is a consumer-side port for high-risk admin operations.
// The implementation must keep post mutation, audit, and schedule cancellation
// inside the transaction supplied by the application layer.
type AdminPostRepository interface {
	ListAdminPosts(ctx context.Context, query AdminPostListQuery) (AdminPostPage, error)
	DeleteAdminPost(ctx context.Context, tx Tx, command AdminPostDeleteCommand) (AdminPostDeleteRecord, error)
}

type AdminPostListQuery struct {
	Status   string
	AuthorID int64
	Page     int
	Size     int
}

type AdminPostPage struct {
	Items []AdminPostRecord
	Page  int
	Size  int
	Total int64
}

type AdminPostRecord struct {
	PostID             string
	AuthorID           int64
	AuthorName         string
	AuthorAvatarFileID string
	Title              string
	Summary            string
	CoverFileID        string
	Status             domain.PostStatus
	PostVersion        int64
	PublishedAt        time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
	ViewCount          int64
	LikeCount          int64
	FavoriteCount      int64
	CommentCount       int64
}

type AdminPostDeleteCommand struct {
	PublicID    string
	AdminUserID int64
	Reason      string
	DeletedAt   time.Time
}

type AdminPostDeleteRecord struct {
	Before PostRecord
	After  PostRecord
}
