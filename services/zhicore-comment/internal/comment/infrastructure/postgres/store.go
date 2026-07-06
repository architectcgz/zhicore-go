package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	kitoutbox "github.com/architectcgz/zhicore-go/libs/kit/postgres/outbox"
	"github.com/architectcgz/zhicore-go/libs/kit/postgres/sqlarg"
	"github.com/lib/pq"

	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-comment/internal/comment/ports"
)

type sqlExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type txContextKey struct{}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) execer(ctx context.Context) sqlExecutor {
	if tx, ok := ctx.Value(txContextKey{}).(*sql.Tx); ok && tx != nil {
		return tx
	}
	return s.db
}

type TransactionRunner struct {
	db *sql.DB
}

func NewTransactionRunner(db *sql.DB) *TransactionRunner {
	return &TransactionRunner{db: db}
}

func (r *TransactionRunner) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin comment transaction: %w", err)
	}
	defer tx.Rollback()

	txCtx := context.WithValue(ctx, txContextKey{}, tx)
	if err := fn(txCtx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit comment transaction: %w", err)
	}
	return nil
}

func (s *Store) FindReplyGuardPreview(ctx context.Context, postID domain.PostID, parentID domain.CommentID) (ports.ReplyGuardPreview, bool, error) {
	var authorID int64
	err := s.execer(ctx).QueryRowContext(ctx, findReplyGuardPreviewSQL, string(postID), int64(parentID)).Scan(&authorID)
	if errors.Is(err, sql.ErrNoRows) {
		return ports.ReplyGuardPreview{}, false, nil
	}
	if err != nil {
		return ports.ReplyGuardPreview{}, false, fmt.Errorf("find reply guard preview: %w", err)
	}
	return ports.ReplyGuardPreview{ParentAuthorID: domain.UserID(authorID)}, true, nil
}

func (s *Store) FindReplyTarget(ctx context.Context, postID domain.PostID, parentID domain.CommentID) (ports.ReplyTarget, error) {
	parent, err := s.FindCommentForMutation(ctx, postID, parentID)
	if err != nil {
		return ports.ReplyTarget{}, domain.ErrParentCommentNotFound
	}
	if parent.Status != domain.CommentStatusNormal {
		return ports.ReplyTarget{}, domain.ErrParentCommentNotFound
	}
	if parent.IsTopLevel() {
		return ports.ReplyTarget{Parent: parent, Root: parent}, nil
	}
	root, err := s.FindCommentForMutation(ctx, postID, parent.RootID)
	if err != nil || root.Status != domain.CommentStatusNormal || !root.IsTopLevel() {
		return ports.ReplyTarget{}, domain.ErrRootCommentNotFound
	}
	return ports.ReplyTarget{Parent: parent, Root: root}, nil
}

func (s *Store) FindCommentForMutation(ctx context.Context, postID domain.PostID, commentID domain.CommentID) (domain.Comment, error) {
	comment, err := scanComment(s.execer(ctx).QueryRowContext(ctx, findCommentForMutationSQL, string(postID), int64(commentID)))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Comment{}, domain.ErrCommentNotFound
	}
	if err != nil {
		return domain.Comment{}, fmt.Errorf("find comment for mutation: %w", err)
	}
	return comment, nil
}

func (s *Store) Create(ctx context.Context, draft domain.Comment) (domain.Comment, error) {
	comment, err := scanComment(s.execer(ctx).QueryRowContext(ctx, createCommentSQL,
		string(draft.PostID),
		int64(draft.ContentInternalID),
		int64(draft.AuthorID),
		nullableCommentID(draft.RootID),
		nullableCommentID(draft.ParentID),
		draft.Content,
		nullableStringArray(draft.Media.ImageFileIDs),
		sqlarg.NonBlankString(draft.Media.VoiceFileID),
		sqlarg.Int(draft.Media.VoiceDuration),
		string(draft.Status),
		draft.CreatedAt,
		draft.UpdatedAt,
	))
	if err != nil {
		return domain.Comment{}, fmt.Errorf("insert comment: %w", err)
	}
	return comment, nil
}

func (s *Store) SoftDeleteSubtree(ctx context.Context, input ports.DeleteSubtreeInput) (ports.DeleteSubtreeResult, error) {
	entry, err := s.FindCommentForMutation(ctx, input.PostID, input.CommentID)
	if err != nil {
		return ports.DeleteSubtreeResult{}, err
	}
	rootID := entry.ID
	if entry.IsReply() {
		rootID = entry.RootID
	}
	if entry.Status != domain.CommentStatusNormal {
		if input.AllowDeleted {
			return ports.DeleteSubtreeResult{Entry: entry, RootID: rootID, AlreadyDeleted: true}, nil
		}
		return ports.DeleteSubtreeResult{}, domain.ErrCommentNotFound
	}

	var affected int64
	if err := s.execer(ctx).QueryRowContext(ctx, softDeleteSubtreeSQL,
		string(input.PostID),
		int64(input.CommentID),
		int64(input.DeletedBy),
		input.DeletedByRole,
		input.DeleteReason,
		input.DeletedAt,
	).Scan(&affected); err != nil {
		return ports.DeleteSubtreeResult{}, fmt.Errorf("soft delete comment subtree: %w", err)
	}
	return ports.DeleteSubtreeResult{Entry: entry, RootID: rootID, AffectedCount: int(affected)}, nil
}

func (s *Store) InitializeTopLevelRanks(ctx context.Context, comment domain.Comment, now time.Time) error {
	if _, err := s.execer(ctx).ExecContext(ctx, insertHotRankSQL, int64(comment.ID), string(comment.PostID), now); err != nil {
		return fmt.Errorf("initialize comment hot rank: %w", err)
	}
	if _, err := s.execer(ctx).ExecContext(ctx, insertRecommendedRankSQL, int64(comment.ID), string(comment.PostID), now); err != nil {
		return fmt.Errorf("initialize comment recommended rank: %w", err)
	}
	return nil
}

func (s *Store) HideTopLevelRanks(ctx context.Context, commentID domain.CommentID, now time.Time) error {
	if _, err := s.execer(ctx).ExecContext(ctx, hideTopLevelRanksSQL, int64(commentID), now); err != nil {
		return fmt.Errorf("hide comment ranks: %w", err)
	}
	return nil
}

func (s *Store) UpsertLike(ctx context.Context, input ports.LikeMutationInput) (bool, error) {
	result, err := s.execer(ctx).ExecContext(ctx, upsertLikeSQL, int64(input.CommentID), int64(input.UserID), input.Now)
	if err != nil {
		return false, fmt.Errorf("upsert comment like: %w", err)
	}
	return rowsChanged(result)
}

func (s *Store) DeleteLike(ctx context.Context, input ports.LikeMutationInput) (bool, error) {
	result, err := s.execer(ctx).ExecContext(ctx, deleteLikeSQL, int64(input.CommentID), int64(input.UserID))
	if err != nil {
		return false, fmt.Errorf("delete comment like: %w", err)
	}
	return rowsChanged(result)
}

func (s *Store) AppendCounterDelta(ctx context.Context, delta ports.CommentCounterDelta) error {
	if _, err := s.execer(ctx).ExecContext(ctx, insertCounterDeltaSQL, int64(delta.CommentID), string(delta.PostID), delta.CounterType, delta.DeltaValue, delta.CreatedAt); err != nil {
		return fmt.Errorf("append comment counter delta: %w", err)
	}
	return nil
}

func (s *Store) Initialize(ctx context.Context, commentID domain.CommentID, now time.Time) error {
	if _, err := s.execer(ctx).ExecContext(ctx, insertCommentStatsSQL, int64(commentID), now); err != nil {
		return fmt.Errorf("initialize comment stats: %w", err)
	}
	return nil
}

func (s *Store) IncrementReplyCount(ctx context.Context, rootID domain.CommentID, now time.Time) error {
	if _, err := s.execer(ctx).ExecContext(ctx, incrementReplyCountSQL, int64(rootID), now); err != nil {
		return fmt.Errorf("increment comment reply count: %w", err)
	}
	return nil
}

func (s *Store) DecrementReplyCount(ctx context.Context, rootID domain.CommentID, by int, now time.Time) error {
	if _, err := s.execer(ctx).ExecContext(ctx, decrementReplyCountSQL, int64(rootID), by, now); err != nil {
		return fmt.Errorf("decrement comment reply count: %w", err)
	}
	return nil
}

func (s *Store) IncrementForTopLevel(ctx context.Context, postID domain.PostID, now time.Time) error {
	if _, err := s.execer(ctx).ExecContext(ctx, incrementPostStatsTopLevelSQL, string(postID), now); err != nil {
		return fmt.Errorf("increment top level comment post stats: %w", err)
	}
	return nil
}

func (s *Store) IncrementForReply(ctx context.Context, postID domain.PostID, now time.Time) error {
	if _, err := s.execer(ctx).ExecContext(ctx, incrementPostStatsReplySQL, string(postID), now); err != nil {
		return fmt.Errorf("increment reply comment post stats: %w", err)
	}
	return nil
}

func (s *Store) DecrementForDelete(ctx context.Context, postID domain.PostID, affectedCount int, topLevelDeleted bool, now time.Time) error {
	if _, err := s.execer(ctx).ExecContext(ctx, decrementPostStatsSQL, string(postID), affectedCount, topLevelDeleted, now); err != nil {
		return fmt.Errorf("decrement comment post stats: %w", err)
	}
	return nil
}

func (s *Store) Get(ctx context.Context, postID domain.PostID) (domain.CommentPostStats, error) {
	var stats domain.CommentPostStats
	err := s.execer(ctx).QueryRowContext(ctx, getPostStatsSQL, string(postID)).Scan(&stats.PostID, &stats.TotalComments, &stats.TotalTopLevelComments)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.CommentPostStats{PostID: postID}, nil
	}
	if err != nil {
		return domain.CommentPostStats{}, fmt.Errorf("get comment post stats: %w", err)
	}
	return stats, nil
}

func (s *Store) ListTopLevelComments(ctx context.Context, query ports.TopLevelCommentPageQuery) (ports.TopLevelCommentPage, error) {
	sqlText := listTopLevelRecommendedSQL
	switch query.Sort {
	case domain.CommentSortHot:
		sqlText = listTopLevelHotSQL
	case domain.CommentSortTime:
		sqlText = listTopLevelTimeSQL
	}
	page := query.Page
	if page < 1 {
		page = 1
	}
	size := query.Size
	if size < 1 {
		size = 20
	}
	offset := (page - 1) * size
	rows, err := s.execer(ctx).QueryContext(ctx, sqlText, string(query.PostID), size, offset)
	if err != nil {
		return ports.TopLevelCommentPage{}, fmt.Errorf("list top level comments: %w", err)
	}
	defer rows.Close()

	var items []ports.TopLevelCommentRecord
	for rows.Next() {
		record, err := scanCommentRecord(rows)
		if err != nil {
			return ports.TopLevelCommentPage{}, err
		}
		items = append(items, record)
	}
	if err := rows.Err(); err != nil {
		return ports.TopLevelCommentPage{}, fmt.Errorf("iterate top level comments: %w", err)
	}
	return ports.TopLevelCommentPage{Items: items}, nil
}

func (s *Store) GetCommentDetail(ctx context.Context, postID domain.PostID, commentID domain.CommentID) (ports.TopLevelCommentRecord, error) {
	record, err := scanCommentRecord(s.execer(ctx).QueryRowContext(ctx, getCommentDetailSQL, string(postID), int64(commentID)))
	if errors.Is(err, sql.ErrNoRows) {
		return ports.TopLevelCommentRecord{}, domain.ErrCommentNotFound
	}
	if err != nil {
		return ports.TopLevelCommentRecord{}, err
	}
	return record, nil
}

func (s *Store) ListRepliesByPage(ctx context.Context, query ports.ReplyCommentPageQuery) (ports.ReplyCommentPage, error) {
	var rootExists bool
	err := s.execer(ctx).QueryRowContext(ctx, checkRootCommentSQL, string(query.PostID), int64(query.RootID)).Scan(&rootExists)
	if errors.Is(err, sql.ErrNoRows) {
		return ports.ReplyCommentPage{}, domain.ErrRootCommentNotFound
	}
	if err != nil {
		return ports.ReplyCommentPage{}, fmt.Errorf("check root comment: %w", err)
	}

	var total int64
	if err := s.execer(ctx).QueryRowContext(ctx, countRepliesSQL, string(query.PostID), int64(query.RootID)).Scan(&total); err != nil {
		return ports.ReplyCommentPage{}, fmt.Errorf("count comment replies: %w", err)
	}

	sqlText := listRepliesHotSQL
	if query.Sort == domain.CommentSortTime {
		sqlText = listRepliesTimeSQL
	}
	page := query.Page
	if page < 1 {
		page = 1
	}
	size := query.Size
	if size < 1 {
		size = 20
	}
	offset := (page - 1) * size
	rows, err := s.execer(ctx).QueryContext(ctx, sqlText, string(query.PostID), int64(query.RootID), size, offset)
	if err != nil {
		return ports.ReplyCommentPage{}, fmt.Errorf("list comment replies: %w", err)
	}
	defer rows.Close()

	items := make([]ports.TopLevelCommentRecord, 0)
	for rows.Next() {
		record, err := scanCommentRecord(rows)
		if err != nil {
			return ports.ReplyCommentPage{}, err
		}
		items = append(items, record)
	}
	if err := rows.Err(); err != nil {
		return ports.ReplyCommentPage{}, fmt.Errorf("iterate comment replies: %w", err)
	}
	return ports.ReplyCommentPage{Items: items, Total: total}, nil
}

func (s *Store) BatchGetViewerLiked(ctx context.Context, viewerID domain.UserID, commentIDs []domain.CommentID) (map[domain.CommentID]bool, error) {
	result := make(map[domain.CommentID]bool, len(commentIDs))
	if viewerID <= 0 || len(commentIDs) == 0 {
		return result, nil
	}
	ids := make([]int64, 0, len(commentIDs))
	for _, id := range commentIDs {
		result[id] = false
		ids = append(ids, int64(id))
	}
	rows, err := s.execer(ctx).QueryContext(ctx, batchViewerLikedSQL, int64(viewerID), pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("batch get viewer liked comments: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan viewer liked comment: %w", err)
		}
		result[domain.CommentID(id)] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate viewer liked comments: %w", err)
	}
	return result, nil
}

type EventIDGenerator interface {
	NewEventID() (string, error)
}

type RandomEventIDGenerator struct{}

func (RandomEventIDGenerator) NewEventID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate outbox event id: %w", err)
	}
	return "evt_" + hex.EncodeToString(raw[:]), nil
}

type OutboxPublisher struct {
	db        *sql.DB
	publisher *kitoutbox.InsertPublisher
}

func NewOutboxPublisher(db *sql.DB, ids EventIDGenerator) *OutboxPublisher {
	if ids == nil {
		ids = RandomEventIDGenerator{}
	}
	return &OutboxPublisher{
		db:        db,
		publisher: kitoutbox.NewInsertPublisher(kitoutbox.Config{Table: "outbox_events"}, ids),
	}
}

func (p *OutboxPublisher) Publish(ctx context.Context, message ports.OutboxMessage) error {
	execer := sqlExecutor(p.db)
	if tx, ok := ctx.Value(txContextKey{}).(*sql.Tx); ok && tx != nil {
		execer = tx
	}
	if err := p.publisher.Publish(ctx, execer, kitoutbox.Message{
		EventType:     message.EventType,
		AggregateType: message.AggregateType,
		AggregateID:   message.AggregateID,
		Payload:       message.Payload,
		OccurredAt:    message.OccurredAt,
	}); err != nil {
		return fmt.Errorf("insert comment outbox event: %w", err)
	}
	return nil
}

func rowsChanged(result sql.Result) (bool, error) {
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read affected rows: %w", err)
	}
	return affected > 0, nil
}

func scanComment(row interface {
	Scan(dest ...any) error
}) (domain.Comment, error) {
	var (
		id                int64
		postID            string
		contentInternalID int64
		authorID          int64
		rootID            sql.NullInt64
		parentID          sql.NullInt64
		content           sql.NullString
		imageFileIDs      pq.StringArray
		voiceFileID       sql.NullString
		voiceDuration     sql.NullInt64
		status            string
		createdAt         time.Time
		updatedAt         time.Time
	)
	if err := row.Scan(
		&id,
		&postID,
		&contentInternalID,
		&authorID,
		&rootID,
		&parentID,
		&content,
		&imageFileIDs,
		&voiceFileID,
		&voiceDuration,
		&status,
		&createdAt,
		&updatedAt,
	); err != nil {
		return domain.Comment{}, err
	}
	return domain.NewComment(domain.CommentSeed{
		ID:                domain.CommentID(id),
		PostID:            domain.PostID(postID),
		ContentInternalID: domain.ContentInternalID(contentInternalID),
		AuthorID:          domain.UserID(authorID),
		RootID:            domain.CommentID(nullInt64(rootID)),
		ParentID:          domain.CommentID(nullInt64(parentID)),
		Content:           content.String,
		ImageFileIDs:      []string(imageFileIDs),
		VoiceFileID:       voiceFileID.String,
		VoiceDuration:     int(nullInt64(voiceDuration)),
		Status:            domain.CommentStatus(status),
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	})
}

func scanCommentRecord(row interface {
	Scan(dest ...any) error
}) (ports.TopLevelCommentRecord, error) {
	var (
		id                int64
		postID            string
		contentInternalID int64
		authorID          int64
		rootID            sql.NullInt64
		parentID          sql.NullInt64
		content           sql.NullString
		imageFileIDs      pq.StringArray
		voiceFileID       sql.NullString
		voiceDuration     sql.NullInt64
		status            string
		createdAt         time.Time
		updatedAt         time.Time
		likeCount         int64
		replyCount        int64
	)
	if err := row.Scan(
		&id,
		&postID,
		&contentInternalID,
		&authorID,
		&rootID,
		&parentID,
		&content,
		&imageFileIDs,
		&voiceFileID,
		&voiceDuration,
		&status,
		&createdAt,
		&updatedAt,
		&likeCount,
		&replyCount,
	); err != nil {
		return ports.TopLevelCommentRecord{}, fmt.Errorf("scan top level comment: %w", err)
	}
	comment, err := domain.NewComment(domain.CommentSeed{
		ID:                domain.CommentID(id),
		PostID:            domain.PostID(postID),
		ContentInternalID: domain.ContentInternalID(contentInternalID),
		AuthorID:          domain.UserID(authorID),
		RootID:            domain.CommentID(nullInt64(rootID)),
		ParentID:          domain.CommentID(nullInt64(parentID)),
		Content:           content.String,
		ImageFileIDs:      []string(imageFileIDs),
		VoiceFileID:       voiceFileID.String,
		VoiceDuration:     int(nullInt64(voiceDuration)),
		Status:            domain.CommentStatus(status),
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	})
	if err != nil {
		return ports.TopLevelCommentRecord{}, err
	}
	return ports.TopLevelCommentRecord{
		Comment: comment,
		Stats:   domain.CommentStats{CommentID: comment.ID, LikeCount: likeCount, ReplyCount: replyCount},
	}, nil
}

func nullableCommentID(id domain.CommentID) any {
	if id == 0 {
		return nil
	}
	return int64(id)
}

func nullableStringArray(values []string) any {
	if len(values) == 0 {
		return nil
	}
	return pq.Array(values)
}

func nullInt64(value sql.NullInt64) int64 {
	if !value.Valid {
		return 0
	}
	return value.Int64
}
