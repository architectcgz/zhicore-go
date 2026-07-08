package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func (s *Store) ListAdminPosts(ctx context.Context, query ports.AdminPostListQuery) (ports.AdminPostPage, error) {
	offset := (query.Page - 1) * query.Size
	rows, err := s.db.QueryContext(ctx, listAdminPostsSQL, query.Status, query.AuthorID, query.Size, offset)
	if err != nil {
		return ports.AdminPostPage{}, fmt.Errorf("list admin content posts: %w", err)
	}
	defer rows.Close()

	page := ports.AdminPostPage{Page: query.Page, Size: query.Size}
	for rows.Next() {
		var item ports.AdminPostRecord
		var total int64
		if err := scanAdminPostRecord(rows, &item, &total); err != nil {
			return ports.AdminPostPage{}, fmt.Errorf("scan admin content post: %w", err)
		}
		page.Items = append(page.Items, item)
		page.Total = total
	}
	if err := rows.Err(); err != nil {
		return ports.AdminPostPage{}, fmt.Errorf("iterate admin content posts: %w", err)
	}
	return page, nil
}

func (s *Store) DeleteAdminPost(ctx context.Context, tx ports.Tx, command ports.AdminPostDeleteCommand) (ports.AdminPostDeleteRecord, error) {
	execer, err := s.execer(tx)
	if err != nil {
		return ports.AdminPostDeleteRecord{}, err
	}
	before, err := scanPostRecord(execer.QueryRowContext(ctx, selectPostForUpdateSQL, command.PublicID))
	if errors.Is(err, sql.ErrNoRows) {
		return ports.AdminPostDeleteRecord{}, domain.ErrPostNotFound
	}
	if err != nil {
		return ports.AdminPostDeleteRecord{}, fmt.Errorf("select admin content post for delete: %w", err)
	}
	if before.Status == domain.PostStatusDeleted {
		return ports.AdminPostDeleteRecord{}, domain.ErrPostDeleted
	}

	after, err := scanPostRecord(execer.QueryRowContext(ctx, adminDeletePostSQL, command.PublicID, command.DeletedAt))
	if errors.Is(err, sql.ErrNoRows) {
		return ports.AdminPostDeleteRecord{}, domain.ErrPostDeleted
	}
	if err != nil {
		return ports.AdminPostDeleteRecord{}, fmt.Errorf("admin delete content post: %w", err)
	}
	if _, err := execer.ExecContext(ctx, insertAdminPostAuditSQL,
		before.ID,
		before.PublicID,
		command.AdminUserID,
		"DELETE",
		command.Reason,
		string(before.Status),
		string(after.Status),
		command.DeletedAt,
	); err != nil {
		return ports.AdminPostDeleteRecord{}, fmt.Errorf("insert admin post audit: %w", err)
	}
	// A pending schedule is an operational side effect of this post. Cancel it
	// atomically with the admin delete so the scheduler cannot later republish
	// content that an administrator has removed.
	if _, err := execer.ExecContext(ctx, cancelScheduledPublishEventSQL, before.ID, command.DeletedAt); err != nil {
		return ports.AdminPostDeleteRecord{}, fmt.Errorf("cancel scheduled publish event for admin deleted post: %w", err)
	}
	return ports.AdminPostDeleteRecord{Before: before, After: after}, nil
}

func scanAdminPostRecord(row rowScanner, item *ports.AdminPostRecord, total *int64) error {
	var authorAvatar, title, summary, cover sql.NullString
	var status string
	var publishedAt sql.NullTime
	if err := row.Scan(
		&item.PostID,
		&item.AuthorID,
		&item.AuthorName,
		&authorAvatar,
		&title,
		&summary,
		&cover,
		&status,
		&item.PostVersion,
		&publishedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.ViewCount,
		&item.LikeCount,
		&item.FavoriteCount,
		&item.CommentCount,
		total,
	); err != nil {
		return err
	}
	item.AuthorAvatarFileID = authorAvatar.String
	item.Title = title.String
	item.Summary = summary.String
	item.CoverFileID = cover.String
	item.Status = domain.PostStatus(status)
	item.PublishedAt = publishedAt.Time
	return nil
}
