package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/architectcgz/zhicore-go/libs/kit/postgres/sqlarg"
	"github.com/lib/pq"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func (s *Store) ListPublishedPosts(ctx context.Context, query ports.PostListQuery) ([]ports.PostSummaryRecord, error) {
	rows, err := s.db.QueryContext(ctx, listPublishedPostsSQL,
		query.AuthorID,
		sqlarg.Time(query.Cursor.PublishedAt),
		query.Cursor.PublicID,
		query.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list published content posts: %w", err)
	}
	defer rows.Close()

	records := make([]ports.PostSummaryRecord, 0)
	for rows.Next() {
		record, err := scanPostSummaryRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan published content post: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate published content posts: %w", err)
	}
	return records, nil
}

func (s *Store) GetPublishedPostDetail(ctx context.Context, publicID string) (ports.PostDetailRecord, error) {
	detail, err := scanPostDetailRecord(s.db.QueryRowContext(ctx, getPublishedPostDetailSQL, publicID))
	if errors.Is(err, sql.ErrNoRows) {
		return ports.PostDetailRecord{}, domain.ErrPostNotFound
	}
	if err != nil {
		return ports.PostDetailRecord{}, fmt.Errorf("get published content post detail: %w", err)
	}
	return detail, nil
}

func (s *Store) BatchGetPublishedPostSummaries(ctx context.Context, publicIDs []string) ([]ports.PostSummaryRecord, error) {
	rows, err := s.db.QueryContext(ctx, batchGetPublishedPostSummariesSQL, pq.Array(publicIDs))
	if err != nil {
		return nil, fmt.Errorf("batch get published content posts: %w", err)
	}
	defer rows.Close()

	records := make([]ports.PostSummaryRecord, 0)
	for rows.Next() {
		record, err := scanPostSummaryRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan batch published content post: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate batch published content posts: %w", err)
	}
	return records, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanPostSummaryRecord(row rowScanner) (ports.PostSummaryRecord, error) {
	var record ports.PostSummaryRecord
	var authorAvatar, summary, cover sql.NullString
	var publishedAt sql.NullTime
	if err := row.Scan(
		&record.PostID,
		&record.AuthorID,
		&record.AuthorName,
		&authorAvatar,
		&record.Title,
		&summary,
		&cover,
		&record.Status,
		&record.PostVersion,
		&publishedAt,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.ViewCount,
		&record.LikeCount,
		&record.FavoriteCount,
		&record.CommentCount,
	); err != nil {
		return ports.PostSummaryRecord{}, err
	}
	record.AuthorAvatarFileID = authorAvatar.String
	record.Summary = summary.String
	record.CoverFileID = cover.String
	record.PublishedAt = publishedAt.Time
	return record, nil
}

func scanPostDetailRecord(row rowScanner) (ports.PostDetailRecord, error) {
	var detail ports.PostDetailRecord
	var authorAvatar, summary, cover sql.NullString
	var bodyID, bodyHash sql.NullString
	if err := row.Scan(
		&detail.InternalPostID,
		&detail.Summary.PostID,
		&detail.Summary.AuthorID,
		&detail.Summary.AuthorName,
		&authorAvatar,
		&detail.Summary.Title,
		&summary,
		&cover,
		&detail.Summary.Status,
		&detail.Summary.PostVersion,
		&detail.Summary.PublishedAt,
		&detail.Summary.CreatedAt,
		&detail.Summary.UpdatedAt,
		&detail.Summary.ViewCount,
		&detail.Summary.LikeCount,
		&detail.Summary.FavoriteCount,
		&detail.Summary.CommentCount,
		&bodyID,
		&bodyHash,
	); err != nil {
		return ports.PostDetailRecord{}, err
	}
	detail.Summary.AuthorAvatarFileID = authorAvatar.String
	detail.Summary.Summary = summary.String
	detail.Summary.CoverFileID = cover.String
	detail.PublishedBodyID = bodyID.String
	detail.PublishedHash = bodyHash.String
	return detail, nil
}
