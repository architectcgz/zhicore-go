package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/architectcgz/zhicore-go/libs/kit/postgres/sqlarg"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func (s *Store) ListAuthorPosts(ctx context.Context, query ports.AuthorPostListQuery) ([]ports.PostSummaryRecord, error) {
	rows, err := s.db.QueryContext(ctx, listAuthorPostsSQL,
		query.OwnerID,
		query.Status,
		sqlarg.Time(query.Cursor.UpdatedAt),
		query.Cursor.PublicID,
		query.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list author content posts: %w", err)
	}
	defer rows.Close()

	records := make([]ports.PostSummaryRecord, 0)
	for rows.Next() {
		record, err := scanPostSummaryRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan author content post: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate author content posts: %w", err)
	}
	return records, nil
}

func (s *Store) GetDraftPost(ctx context.Context, publicID string) (ports.DraftPostRecord, error) {
	record, err := scanDraftPostRecord(s.db.QueryRowContext(ctx, getDraftPostSQL, publicID))
	if errors.Is(err, sql.ErrNoRows) {
		return ports.DraftPostRecord{}, domain.ErrPostNotFound
	}
	if err != nil {
		return ports.DraftPostRecord{}, fmt.Errorf("get content draft post: %w", err)
	}
	return record, nil
}

func (s *Store) UpdateDraftMeta(ctx context.Context, tx ports.Tx, input ports.UpdateDraftMetaUpdate) (ports.PostRecord, error) {
	execer, err := s.execer(tx)
	if err != nil {
		return ports.PostRecord{}, err
	}
	record, err := scanPostRecord(execer.QueryRowContext(ctx, updateDraftMetaSQL,
		optionalStringPtr(input.Title),
		optionalStringPtr(input.Summary),
		sqlarg.OptionalString(input.CoverFileID.Set, input.CoverFileID.Value),
		input.UpdatedAt,
		input.PublicID,
		input.OwnerID,
		input.BasePostVersion,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return ports.PostRecord{}, classifyMutationMiss(ctx, execer, input.PublicID, input.OwnerID, false)
	}
	if err != nil {
		return ports.PostRecord{}, fmt.Errorf("update content draft meta: %w", err)
	}
	return record, nil
}

func (s *Store) DeleteDraft(ctx context.Context, tx ports.Tx, input ports.DeleteDraftUpdate) (ports.PostRecord, error) {
	execer, err := s.execer(tx)
	if err != nil {
		return ports.PostRecord{}, err
	}
	record, err := scanPostRecord(execer.QueryRowContext(ctx, deleteDraftSQL, input.PublicID, input.OwnerID, input.UpdatedAt))
	if errors.Is(err, sql.ErrNoRows) {
		return ports.PostRecord{}, classifyMutationMiss(ctx, execer, input.PublicID, input.OwnerID, false)
	}
	if err != nil {
		return ports.PostRecord{}, fmt.Errorf("delete content draft: %w", err)
	}
	return record, nil
}

func scanDraftPostRecord(row rowScanner) (ports.DraftPostRecord, error) {
	var draft ports.DraftPostRecord
	var status string
	var draftTitle, draftSummary, draftCover, draftBodyID, draftBodyHash sql.NullString
	var draftSize, draftPlainTextLen sql.NullInt64
	var publishedTitle, publishedSummary, publishedCover, publishedBodyID, publishedBodyHash sql.NullString
	var publishedPlainTextLen sql.NullInt64
	var publishedAt sql.NullTime

	if err := row.Scan(
		&draft.Post.ID,
		&draft.Post.PublicID,
		&draft.Post.OwnerID,
		&status,
		&draft.Post.PostVersion,
		&draftTitle,
		&draftSummary,
		&draftCover,
		&draftBodyID,
		&draftBodyHash,
		&draftSize,
		&draftPlainTextLen,
		&publishedTitle,
		&publishedSummary,
		&publishedCover,
		&publishedBodyID,
		&publishedBodyHash,
		&publishedPlainTextLen,
		&publishedAt,
		&draft.CreatedAt,
		&draft.UpdatedAt,
	); err != nil {
		return ports.DraftPostRecord{}, err
	}
	draft.Post.Status = domain.PostStatus(status)
	draft.Post.DraftTitle = draftTitle.String
	draft.Post.DraftSummary = draftSummary.String
	draft.Post.DraftCoverFileID = draftCover.String
	draft.Post.DraftBodyID = draftBodyID.String
	draft.Post.DraftBodyHash = draftBodyHash.String
	draft.Post.DraftSizeBytes = int(draftSize.Int64)
	draft.Post.DraftPlainTextLength = int(draftPlainTextLen.Int64)
	draft.Post.PublishedTitle = publishedTitle.String
	draft.Post.PublishedSummary = publishedSummary.String
	draft.Post.PublishedCoverFileID = publishedCover.String
	draft.Post.PublishedBodyID = publishedBodyID.String
	draft.Post.PublishedBodyHash = publishedBodyHash.String
	draft.Post.PublishedPlainTextLen = int(publishedPlainTextLen.Int64)
	draft.Post.PublishedAt = publishedAt.Time
	return draft, nil
}

func optionalStringPtr(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}
