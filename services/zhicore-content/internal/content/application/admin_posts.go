package application

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

const (
	defaultAdminPostPage         = 1
	defaultAdminPostSize         = 20
	maxAdminPostSize             = 100
	defaultAdminPostDeleteReason = "ADMIN_DELETED"
)

type ListAdminPostsQuery struct {
	Actor    *Actor
	Status   string
	AuthorID int64
	Page     int
	Size     int
}

type ListAdminPostsResult struct {
	Items []AdminPostItem
	Page  int
	Size  int
	Total int64
}

type AdminPostItem struct {
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

type DeleteAdminPostCommand struct {
	Actor  *Actor
	PostID string
	Reason string
}

type DeleteAdminPostResult struct {
	PostID string
	Status string
}

func (s *Service) ListAdminPosts(ctx context.Context, query ListAdminPostsQuery) (ListAdminPostsResult, error) {
	if err := requireAdminActor(query.Actor); err != nil {
		return ListAdminPostsResult{}, err
	}
	if s.adminPosts == nil {
		return ListAdminPostsResult{}, ErrDependencyUnavailable
	}
	status, err := normalizeAdminPostStatus(query.Status)
	if err != nil {
		return ListAdminPostsResult{}, err
	}
	if query.AuthorID < 0 {
		return ListAdminPostsResult{}, ErrInvalidArgument
	}
	page, size := normalizeAdminPostPage(query.Page, query.Size)
	result, err := s.adminPosts.ListAdminPosts(ctx, ports.AdminPostListQuery{
		Status:   status,
		AuthorID: query.AuthorID,
		Page:     page,
		Size:     size,
	})
	if err != nil {
		return ListAdminPostsResult{}, fmt.Errorf("%w: list admin posts", ErrDependencyUnavailable)
	}
	return ListAdminPostsResult{
		Items: mapAdminPostItems(result.Items),
		Page:  result.Page,
		Size:  result.Size,
		Total: result.Total,
	}, nil
}

func (s *Service) DeleteAdminPost(ctx context.Context, command DeleteAdminPostCommand) (DeleteAdminPostResult, error) {
	if err := requireAdminActor(command.Actor); err != nil {
		return DeleteAdminPostResult{}, err
	}
	if s.adminPosts == nil || s.tx == nil || s.outbox == nil || s.clock == nil {
		return DeleteAdminPostResult{}, ErrDependencyUnavailable
	}
	postID := strings.TrimSpace(command.PostID)
	if postID == "" {
		return DeleteAdminPostResult{}, ErrInvalidArgument
	}
	reason := strings.TrimSpace(command.Reason)
	if reason == "" {
		reason = defaultAdminPostDeleteReason
	}
	deletedAt := s.clock.Now()

	var changed ports.AdminPostDeleteRecord
	err := s.tx.WithinTx(ctx, func(ctx context.Context, tx ports.Tx) error {
		var err error
		changed, err = s.adminPosts.DeleteAdminPost(ctx, tx, ports.AdminPostDeleteCommand{
			PublicID:    postID,
			AdminUserID: command.Actor.UserID,
			Reason:      reason,
			DeletedAt:   deletedAt,
		})
		if err != nil {
			return err
		}
		event, err := newPostVisibilityChangedOutboxEvent(
			changed.Before,
			changed.After,
			postVisibilityForStatus(changed.Before.Status),
			postVisibilityForStatus(changed.After.Status),
			false,
			defaultAdminPostDeleteReason,
			deletedAt,
		)
		if err != nil {
			return err
		}
		return s.outbox.Append(ctx, tx, event)
	})
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) || errors.Is(err, domain.ErrPostDeleted) {
			return DeleteAdminPostResult{}, err
		}
		return DeleteAdminPostResult{}, fmt.Errorf("%w: delete admin post", ErrDependencyUnavailable)
	}
	return DeleteAdminPostResult{PostID: changed.After.PublicID, Status: string(changed.After.Status)}, nil
}

func normalizeAdminPostStatus(raw string) (string, error) {
	status := strings.ToUpper(strings.TrimSpace(raw))
	if status == "" {
		return "", nil
	}
	switch domain.PostStatus(status) {
	case domain.PostStatusDraft, domain.PostStatusPublished, domain.PostStatusScheduled, domain.PostStatusDeleted:
		return status, nil
	default:
		return "", ErrInvalidArgument
	}
}

func normalizeAdminPostPage(page, size int) (int, int) {
	if page <= 0 {
		page = defaultAdminPostPage
	}
	if size <= 0 {
		size = defaultAdminPostSize
	}
	if size > maxAdminPostSize {
		size = maxAdminPostSize
	}
	return page, size
}

func mapAdminPostItems(records []ports.AdminPostRecord) []AdminPostItem {
	items := make([]AdminPostItem, 0, len(records))
	for _, record := range records {
		items = append(items, AdminPostItem{
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
		})
	}
	return items
}
