package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"github.com/architectcgz/zhicore-go/libs/kit/postgres/sqlarg"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

func (s *Store) ListTags(ctx context.Context, query ports.TagListQuery) ([]ports.TagRecord, error) {
	rows, err := s.db.QueryContext(ctx, listTagsSQL, query.Cursor.Slug, query.Cursor.ID, query.Limit)
	if err != nil {
		return nil, fmt.Errorf("list content tags: %w", err)
	}
	return scanTagRows(rows, "list content tags")
}

func (s *Store) GetTagBySlug(ctx context.Context, slug string) (ports.TagRecord, error) {
	record, err := scanTagRecord(s.db.QueryRowContext(ctx, getTagBySlugSQL, slug))
	if errors.Is(err, sql.ErrNoRows) {
		return ports.TagRecord{}, ports.ErrTaxonomyReferenceNotFound
	}
	if err != nil {
		return ports.TagRecord{}, fmt.Errorf("get content tag by slug: %w", err)
	}
	return record, nil
}

func (s *Store) SearchTags(ctx context.Context, query ports.TagSearchQuery) ([]ports.TagRecord, error) {
	rows, err := s.db.QueryContext(ctx, searchTagsSQL, query.Query, query.Limit)
	if err != nil {
		return nil, fmt.Errorf("search content tags: %w", err)
	}
	return scanTagRows(rows, "search content tags")
}

func (s *Store) ListHotTags(ctx context.Context, limit int) ([]ports.TagRecord, error) {
	rows, err := s.db.QueryContext(ctx, listHotTagsSQL, limit)
	if err != nil {
		return nil, fmt.Errorf("list hot content tags: %w", err)
	}
	return scanTagRows(rows, "list hot content tags")
}

func (s *Store) ListPublishedPostsByTag(ctx context.Context, query ports.TaggedPostListQuery) ([]ports.PostSummaryRecord, error) {
	rows, err := s.db.QueryContext(ctx, listPublishedPostsByTagSQL,
		query.Slug,
		sqlarg.Time(query.Cursor.PublishedAt),
		query.Cursor.PublicID,
		query.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list published content posts by tag: %w", err)
	}
	defer rows.Close()

	records := make([]ports.PostSummaryRecord, 0)
	for rows.Next() {
		record, err := scanPostSummaryRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan published content post by tag: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate published content posts by tag: %w", err)
	}
	return records, nil
}

func (s *Store) ListPostTags(ctx context.Context, publicID string) ([]ports.TagRecord, error) {
	rows, err := s.db.QueryContext(ctx, listPostTagsSQL, publicID)
	if err != nil {
		return nil, fmt.Errorf("list content post tags: %w", err)
	}
	records, err := scanTagRows(rows, "list content post tags")
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		exists, err := s.publishedPostExists(ctx, publicID)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, domain.ErrPostNotFound
		}
	}
	return records, nil
}

func (s *Store) ReplacePostTags(ctx context.Context, tx ports.Tx, input ports.ReplacePostTagsInput) (ports.PostTagsMutationRecord, error) {
	execer, err := s.execer(tx)
	if err != nil {
		return ports.PostTagsMutationRecord{}, err
	}
	oldIDs, err := s.selectPostTagIDs(ctx, execer, input.PostInternalID)
	if err != nil {
		return ports.PostTagsMutationRecord{}, err
	}
	tags, err := s.selectTagsBySlugs(ctx, execer, input.Slugs)
	if err != nil {
		return ports.PostTagsMutationRecord{}, err
	}
	ordered, err := orderTagsBySlugs(input.Slugs, tags)
	if err != nil {
		return ports.PostTagsMutationRecord{}, err
	}
	if _, err := execer.ExecContext(ctx, deletePostTagsSQL, input.PostInternalID); err != nil {
		return ports.PostTagsMutationRecord{}, fmt.Errorf("delete content post tags: %w", err)
	}
	newIDs := make([]int64, 0, len(ordered))
	for position, tag := range ordered {
		if _, err := execer.ExecContext(ctx, insertPostTagSQL, input.PostInternalID, tag.ID, position); err != nil {
			return ports.PostTagsMutationRecord{}, fmt.Errorf("insert content post tag: %w", err)
		}
		newIDs = append(newIDs, tag.ID)
	}
	if err := refreshTagStats(ctx, execer, mergeInt64IDs(oldIDs, newIDs), input.UpdatedAt); err != nil {
		return ports.PostTagsMutationRecord{}, err
	}
	version, err := touchPostTags(ctx, execer, input.PostPublicID, input.ActorID, input.BasePostVersion, input.UpdatedAt)
	if err != nil {
		return ports.PostTagsMutationRecord{}, err
	}
	return ports.PostTagsMutationRecord{PostID: input.PostPublicID, PostVersion: version, Tags: ordered, UpdatedAt: input.UpdatedAt}, nil
}

func (s *Store) RemovePostTag(ctx context.Context, tx ports.Tx, input ports.RemovePostTagInput) (ports.PostTagsMutationRecord, error) {
	execer, err := s.execer(tx)
	if err != nil {
		return ports.PostTagsMutationRecord{}, err
	}
	tag, err := scanTagRecord(execer.QueryRowContext(ctx, getTagBySlugSQL, input.Slug))
	if errors.Is(err, sql.ErrNoRows) {
		return ports.PostTagsMutationRecord{}, ports.ErrTaxonomyReferenceNotFound
	}
	if err != nil {
		return ports.PostTagsMutationRecord{}, fmt.Errorf("get content tag for removal: %w", err)
	}
	if _, err := execer.ExecContext(ctx, deletePostTagSQL, input.PostInternalID, tag.ID); err != nil {
		return ports.PostTagsMutationRecord{}, fmt.Errorf("delete content post tag: %w", err)
	}
	if err := refreshTagStats(ctx, execer, []int64{tag.ID}, input.UpdatedAt); err != nil {
		return ports.PostTagsMutationRecord{}, err
	}
	version, err := touchPostTags(ctx, execer, input.PostPublicID, input.ActorID, input.BasePostVersion, input.UpdatedAt)
	if err != nil {
		return ports.PostTagsMutationRecord{}, err
	}
	tags, err := s.listPostTagsByInternalID(ctx, execer, input.PostInternalID)
	if err != nil {
		return ports.PostTagsMutationRecord{}, err
	}
	return ports.PostTagsMutationRecord{PostID: input.PostPublicID, PostVersion: version, Tags: tags, UpdatedAt: input.UpdatedAt}, nil
}

func (s *Store) selectPostTagIDs(ctx context.Context, execer sqlExecutor, postID int64) ([]int64, error) {
	rows, err := execer.QueryContext(ctx, selectPostTagIDsSQL, postID)
	if err != nil {
		return nil, fmt.Errorf("select content post tag ids: %w", err)
	}
	defer rows.Close()
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan content post tag id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate content post tag ids: %w", err)
	}
	return ids, nil
}

func (s *Store) listPostTagsByInternalID(ctx context.Context, execer sqlExecutor, postID int64) ([]ports.TagRecord, error) {
	rows, err := execer.QueryContext(ctx, listPostTagsByIDSQL, postID)
	if err != nil {
		return nil, fmt.Errorf("list content post tags by id: %w", err)
	}
	return scanTagRows(rows, "list content post tags by id")
}

func (s *Store) publishedPostExists(ctx context.Context, publicID string) (bool, error) {
	var exists bool
	if err := s.db.QueryRowContext(ctx, selectPublishedPostExistsSQL, publicID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check content published post exists: %w", err)
	}
	return exists, nil
}

func refreshTagStats(ctx context.Context, execer sqlExecutor, tagIDs []int64, updatedAt any) error {
	if len(tagIDs) == 0 {
		return nil
	}
	if _, err := execer.ExecContext(ctx, refreshTagStatsSQL, pq.Array(tagIDs), updatedAt); err != nil {
		return fmt.Errorf("refresh content tag stats: %w", err)
	}
	return nil
}

func touchPostTags(ctx context.Context, execer sqlExecutor, publicID string, actorID, baseVersion int64, updatedAt any) (int64, error) {
	var version int64
	err := execer.QueryRowContext(ctx, touchPostTagsSQL, updatedAt, publicID, actorID, baseVersion).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, domain.ErrDraftConflict
	}
	if err != nil {
		return 0, fmt.Errorf("touch content post tags: %w", err)
	}
	return version, nil
}

func scanTagRows(rows *sql.Rows, operation string) ([]ports.TagRecord, error) {
	defer rows.Close()
	records := make([]ports.TagRecord, 0)
	for rows.Next() {
		record, err := scanTagRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("%s scan row: %w", operation, err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s iterate rows: %w", operation, err)
	}
	return records, nil
}

func scanTagRecord(row rowScanner) (ports.TagRecord, error) {
	var record ports.TagRecord
	if err := row.Scan(&record.ID, &record.PublicID, &record.Name, &record.Slug, &record.PostCount); err != nil {
		return ports.TagRecord{}, err
	}
	return record, nil
}

func (s *Store) selectTagsBySlugs(ctx context.Context, execer sqlExecutor, slugs []string) ([]ports.TagRecord, error) {
	if len(slugs) == 0 {
		return nil, nil
	}
	rows, err := execer.QueryContext(ctx, selectTagsBySlugsSQL, pq.Array(slugs))
	if err != nil {
		return nil, fmt.Errorf("select content tags by slugs: %w", err)
	}
	return scanTagRows(rows, "select content tags by slugs")
}

func orderTagsBySlugs(slugs []string, records []ports.TagRecord) ([]ports.TagRecord, error) {
	bySlug := make(map[string]ports.TagRecord, len(records))
	for _, record := range records {
		bySlug[record.Slug] = record
	}
	ordered := make([]ports.TagRecord, 0, len(slugs))
	for _, slug := range slugs {
		record, ok := bySlug[slug]
		if !ok {
			return nil, ports.ErrTaxonomyReferenceNotFound
		}
		ordered = append(ordered, record)
	}
	return ordered, nil
}

func mergeInt64IDs(left, right []int64) []int64 {
	seen := make(map[int64]struct{}, len(left)+len(right))
	merged := make([]int64, 0, len(left)+len(right))
	for _, id := range append(left, right...) {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		merged = append(merged, id)
	}
	return merged
}
