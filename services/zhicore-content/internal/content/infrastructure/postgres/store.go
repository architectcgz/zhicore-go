package postgres

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/architectcgz/zhicore-go/libs/kit/postgres/sqlarg"
	"github.com/lib/pq"

	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/domain"
	"github.com/architectcgz/zhicore-go/services/zhicore-content/internal/content/ports"
)

type sqlExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type IDGenerator interface {
	NewID() (string, error)
}

type StoreConfig struct {
	PublicIDs IDGenerator
	EventIDs  IDGenerator
}

type Store struct {
	db        *sql.DB
	publicIDs IDGenerator
	eventIDs  IDGenerator
}

func NewStore(db *sql.DB, config StoreConfig) *Store {
	if config.PublicIDs == nil {
		config.PublicIDs = randomIDGenerator{prefix: "post_"}
	}
	if config.EventIDs == nil {
		config.EventIDs = randomIDGenerator{prefix: "evt_"}
	}
	return &Store{
		db:        db,
		publicIDs: config.PublicIDs,
		eventIDs:  config.EventIDs,
	}
}

type TransactionRunner struct {
	db *sql.DB
}

func NewTransactionRunner(db *sql.DB) *TransactionRunner {
	return &TransactionRunner{db: db}
}

func (r *TransactionRunner) WithinTx(ctx context.Context, fn func(ctx context.Context, tx ports.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin content transaction: %w", err)
	}
	defer tx.Rollback()

	if err := fn(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit content transaction: %w", err)
	}
	return nil
}

func (s *Store) CreateDraft(ctx context.Context, tx ports.Tx, input ports.CreateDraftPost) (ports.PostRecord, error) {
	execer, err := s.execer(tx)
	if err != nil {
		return ports.PostRecord{}, err
	}

	const maxPublicIDAttempts = 3
	var lastErr error
	for attempt := 0; attempt < maxPublicIDAttempts; attempt++ {
		publicID, err := s.publicIDs.NewID()
		if err != nil {
			return ports.PostRecord{}, fmt.Errorf("generate content post public id: %w", err)
		}

		record, err := scanPostRecord(execer.QueryRowContext(ctx, insertPostSQL,
			publicID,
			input.OwnerID,
			input.OwnerDisplayName,
			sqlarg.String(input.OwnerAvatarFileID),
			input.OwnerProfileVersion,
			input.Title,
			sqlarg.String(input.Summary),
			sqlarg.String(input.CoverFileID),
			sqlarg.String(input.DraftBodyID),
			sqlarg.String(input.DraftBodyHash),
			sqlarg.Int(input.DraftSizeBytes),
			sqlarg.Int(input.DraftPlainTextLength),
		))
		if err != nil {
			if isUniqueViolation(err, "ux_posts_public_id") {
				lastErr = err
				continue
			}
			return ports.PostRecord{}, fmt.Errorf("insert content post draft: %w", err)
		}

		if _, err := execer.ExecContext(ctx, insertPostStatsSQL, record.ID, time.Now().UTC()); err != nil {
			return ports.PostRecord{}, fmt.Errorf("initialize post stats: %w", err)
		}
		return record, nil
	}

	return ports.PostRecord{}, fmt.Errorf("generate unique content post public id: %w", lastErr)
}

func (s *Store) GetForUpdate(ctx context.Context, tx ports.Tx, publicID string) (ports.PostRecord, error) {
	execer, err := s.execer(tx)
	if err != nil {
		return ports.PostRecord{}, err
	}
	record, err := scanPostRecord(execer.QueryRowContext(ctx, selectPostForUpdateSQL, publicID))
	if errors.Is(err, sql.ErrNoRows) {
		return ports.PostRecord{}, domain.ErrPostNotFound
	}
	if err != nil {
		return ports.PostRecord{}, fmt.Errorf("select content post for update: %w", err)
	}
	return record, nil
}

func (s *Store) SaveDraftBody(ctx context.Context, tx ports.Tx, input ports.SaveDraftBodyUpdate) (ports.PostRecord, error) {
	execer, err := s.execer(tx)
	if err != nil {
		return ports.PostRecord{}, err
	}
	record, err := scanPostRecord(execer.QueryRowContext(ctx, updateDraftBodySQL,
		input.NewDraftBodyID,
		input.NewDraftBodyHash,
		input.NewDraftSizeBytes,
		input.NewDraftPlainTextLen,
		input.PublicID,
		input.OwnerID,
		input.BasePostVersion,
		input.BaseDraftBodyID,
		input.BaseDraftBodyHash,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return ports.PostRecord{}, classifyMutationMiss(ctx, execer, input.PublicID, input.OwnerID, false)
	}
	if err != nil {
		return ports.PostRecord{}, fmt.Errorf("update content post draft body: %w", err)
	}
	return record, nil
}

func (s *Store) Publish(ctx context.Context, tx ports.Tx, input ports.PublishPostUpdate) (ports.PostRecord, error) {
	execer, err := s.execer(tx)
	if err != nil {
		return ports.PostRecord{}, err
	}
	record, err := scanPostRecord(execer.QueryRowContext(ctx, publishPostSQL,
		input.NewPublishedBodyID,
		input.NewPublishedBodyHash,
		input.NewPublishedPlainTextLen,
		input.PublishedAt,
		input.PublicID,
		input.OwnerID,
		input.BasePostVersion,
		input.ExpectedDraftBodyID,
		input.ExpectedDraftBodyHash,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return ports.PostRecord{}, classifyMutationMiss(ctx, execer, input.PublicID, input.OwnerID, true)
	}
	if err != nil {
		return ports.PostRecord{}, fmt.Errorf("publish content post: %w", err)
	}
	return record, nil
}

func (s *Store) Unpublish(ctx context.Context, tx ports.Tx, input ports.PostLifecycleUpdate) (ports.PostRecord, error) {
	return s.mutatePostLifecycle(ctx, tx, unpublishPostSQL, input, "unpublish content post")
}

func (s *Store) DeletePost(ctx context.Context, tx ports.Tx, input ports.PostLifecycleUpdate) (ports.PostRecord, error) {
	execer, err := s.execer(tx)
	if err != nil {
		return ports.PostRecord{}, err
	}
	record, err := scanPostRecord(execer.QueryRowContext(ctx, deletePostSQL,
		input.PublicID,
		input.OwnerID,
		input.BasePostVersion,
		input.UpdatedAt,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return ports.PostRecord{}, classifyMutationMiss(ctx, execer, input.PublicID, input.OwnerID, false)
	}
	if err != nil {
		return ports.PostRecord{}, fmt.Errorf("delete content post: %w", err)
	}
	// A scheduled post can be deleted before its due time. Canceling the pending
	// schedule in the same transaction prevents a later scheduler from trying
	// to publish content that is no longer author-visible.
	if _, err := execer.ExecContext(ctx, cancelScheduledPublishEventSQL, record.ID, input.UpdatedAt); err != nil {
		return ports.PostRecord{}, fmt.Errorf("cancel scheduled publish event for deleted post: %w", err)
	}
	return record, nil
}

func (s *Store) RestorePost(ctx context.Context, tx ports.Tx, input ports.PostLifecycleUpdate) (ports.PostRecord, error) {
	return s.mutatePostLifecycle(ctx, tx, restorePostSQL, input, "restore content post")
}

func (s *Store) SchedulePost(ctx context.Context, tx ports.Tx, input ports.SchedulePostUpdate) (ports.PostRecord, error) {
	execer, err := s.execer(tx)
	if err != nil {
		return ports.PostRecord{}, err
	}
	record, err := scanPostRecord(execer.QueryRowContext(ctx, schedulePostSQL,
		input.PublicID,
		input.OwnerID,
		input.BasePostVersion,
		input.DraftBodyID,
		input.DraftBodyHash,
		input.UpdatedAt,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return ports.PostRecord{}, classifyMutationMiss(ctx, execer, input.PublicID, input.OwnerID, false)
	}
	if err != nil {
		return ports.PostRecord{}, fmt.Errorf("schedule content post: %w", err)
	}
	if _, err := execer.ExecContext(ctx, upsertScheduledPublishEventSQL,
		record.ID,
		record.PublicID,
		record.OwnerID,
		input.DraftBodyID,
		input.DraftBodyHash,
		input.ScheduledAt,
		input.UpdatedAt,
	); err != nil {
		return ports.PostRecord{}, fmt.Errorf("upsert scheduled publish event: %w", err)
	}
	return record, nil
}

func (s *Store) CancelSchedule(ctx context.Context, tx ports.Tx, input ports.PostLifecycleUpdate) (ports.PostRecord, error) {
	execer, err := s.execer(tx)
	if err != nil {
		return ports.PostRecord{}, err
	}
	record, err := scanPostRecord(execer.QueryRowContext(ctx, cancelScheduleSQL,
		input.PublicID,
		input.OwnerID,
		input.BasePostVersion,
		input.UpdatedAt,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return ports.PostRecord{}, classifyMutationMiss(ctx, execer, input.PublicID, input.OwnerID, false)
	}
	if err != nil {
		return ports.PostRecord{}, fmt.Errorf("cancel scheduled content post: %w", err)
	}
	if _, err := execer.ExecContext(ctx, cancelScheduledPublishEventSQL, record.ID, input.UpdatedAt); err != nil {
		return ports.PostRecord{}, fmt.Errorf("cancel scheduled publish event: %w", err)
	}
	return record, nil
}

func (s *Store) mutatePostLifecycle(ctx context.Context, tx ports.Tx, query string, input ports.PostLifecycleUpdate, operation string) (ports.PostRecord, error) {
	execer, err := s.execer(tx)
	if err != nil {
		return ports.PostRecord{}, err
	}
	record, err := scanPostRecord(execer.QueryRowContext(ctx, query,
		input.PublicID,
		input.OwnerID,
		input.BasePostVersion,
		input.UpdatedAt,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return ports.PostRecord{}, classifyMutationMiss(ctx, execer, input.PublicID, input.OwnerID, false)
	}
	if err != nil {
		return ports.PostRecord{}, fmt.Errorf("%s: %w", operation, err)
	}
	return record, nil
}

func (s *Store) GetPublishedBodyPointer(ctx context.Context, publicID string) (ports.PublishedBodyPointer, error) {
	var pointer ports.PublishedBodyPointer
	var status string
	var bodyID, bodyHash sql.NullString
	var plainTextLen sql.NullInt64

	err := s.db.QueryRowContext(ctx, selectPublishedBodyPointerSQL, publicID).Scan(
		&pointer.PostID,
		&pointer.PublicID,
		&status,
		&bodyID,
		&bodyHash,
		&plainTextLen,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ports.PublishedBodyPointer{}, domain.ErrPostNotFound
	}
	if err != nil {
		return ports.PublishedBodyPointer{}, fmt.Errorf("select published body pointer: %w", err)
	}

	pointer.Status = domain.PostStatus(status)
	pointer.PublishedBodyID = bodyID.String
	pointer.PublishedBodyHash = bodyHash.String
	pointer.PublishedPlainTextLen = int(plainTextLen.Int64)
	return pointer, nil
}

func (s *Store) IsBodyReferenced(ctx context.Context, bodyID string) (bool, error) {
	var referenced bool
	if err := s.db.QueryRowContext(ctx, selectBodyReferencedSQL, bodyID).Scan(&referenced); err != nil {
		return false, fmt.Errorf("check content body reference: %w", err)
	}
	return referenced, nil
}

func (s *Store) Append(ctx context.Context, tx ports.Tx, event ports.OutboxEvent) error {
	execer, err := s.execer(tx)
	if err != nil {
		return err
	}
	eventID, err := s.eventIDs.NewID()
	if err != nil {
		return fmt.Errorf("generate content outbox event id: %w", err)
	}
	version := event.PayloadVersion
	if version == 0 {
		version = 1
	}
	if _, err := execer.ExecContext(ctx, insertOutboxEventSQL,
		eventID,
		event.EventType,
		version,
		event.AggregateType,
		event.AggregateID,
		event.AggregateVersion,
		event.PayloadJSON,
		event.OccurredAt,
	); err != nil {
		return fmt.Errorf("insert content outbox event: %w", err)
	}
	return nil
}

func appendCleanupTask(ctx context.Context, execer sqlExecutor, task ports.BodyCleanupTask) error {
	if _, err := execer.ExecContext(ctx, upsertCleanupTaskSQL,
		sqlarg.Int64(task.PostID),
		task.BodyID,
		task.TaskType,
		task.Reason,
		task.CreatedAt,
	); err != nil {
		return fmt.Errorf("upsert content body cleanup task: %w", err)
	}
	return nil
}

func appendRepairTask(ctx context.Context, execer sqlExecutor, task ports.BodyRepairTask) error {
	if task.BodyID == "" {
		return domain.ErrBodyRequired
	}
	if _, err := execer.ExecContext(ctx, upsertRepairTaskSQL,
		task.PostID,
		task.BodyID,
		task.TaskType,
		sqlarg.String(task.ExpectedHash),
		sqlarg.String(task.ObservedHash),
		task.CreatedAt,
	); err != nil {
		return fmt.Errorf("upsert content body repair task: %w", err)
	}
	return nil
}

func (s *Store) execer(tx ports.Tx) (sqlExecutor, error) {
	if tx == nil {
		return s.db, nil
	}
	sqlTx, ok := tx.(*sql.Tx)
	if !ok {
		return nil, fmt.Errorf("content postgres: unsupported transaction %T", tx)
	}
	return sqlTx, nil
}

func scanPostRecord(row *sql.Row) (ports.PostRecord, error) {
	var record ports.PostRecord
	var status string
	var draftTitle, draftSummary, draftCover, draftBodyID, draftBodyHash sql.NullString
	var draftSize, draftPlainTextLen sql.NullInt64
	var publishedTitle, publishedSummary, publishedCover, publishedBodyID, publishedBodyHash sql.NullString
	var publishedPlainTextLen sql.NullInt64
	var publishedAt sql.NullTime

	if err := row.Scan(
		&record.ID,
		&record.PublicID,
		&record.OwnerID,
		&status,
		&record.PostVersion,
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
	); err != nil {
		return ports.PostRecord{}, err
	}

	record.Status = domain.PostStatus(status)
	record.DraftTitle = draftTitle.String
	record.DraftSummary = draftSummary.String
	record.DraftCoverFileID = draftCover.String
	record.DraftBodyID = draftBodyID.String
	record.DraftBodyHash = draftBodyHash.String
	record.DraftSizeBytes = int(draftSize.Int64)
	record.DraftPlainTextLength = int(draftPlainTextLen.Int64)
	record.PublishedTitle = publishedTitle.String
	record.PublishedSummary = publishedSummary.String
	record.PublishedCoverFileID = publishedCover.String
	record.PublishedBodyID = publishedBodyID.String
	record.PublishedBodyHash = publishedBodyHash.String
	record.PublishedPlainTextLen = int(publishedPlainTextLen.Int64)
	record.PublishedAt = publishedAt.Time
	return record, nil
}

func classifyMutationMiss(ctx context.Context, execer sqlExecutor, publicID string, ownerID int64, publishing bool) error {
	var actualOwner int64
	var status string
	var postVersion int64
	var draftBodyID, draftBodyHash sql.NullString
	err := execer.QueryRowContext(ctx, classifyPostMutationMissSQL, publicID).Scan(
		&actualOwner,
		&status,
		&postVersion,
		&draftBodyID,
		&draftBodyHash,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ErrPostNotFound
	}
	if err != nil {
		return fmt.Errorf("classify content post mutation miss: %w", err)
	}
	if actualOwner != ownerID {
		return domain.ErrForbidden
	}
	switch domain.PostStatus(status) {
	case domain.PostStatusDeleted:
		return domain.ErrPostDeleted
	case domain.PostStatusPublished:
		if publishing {
			return domain.ErrPostAlreadyPublished
		}
	}
	_ = postVersion
	_ = draftBodyID
	_ = draftBodyHash
	return domain.ErrDraftConflict
}

func isUniqueViolation(err error, constraint string) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		if string(pqErr.Code) == "23505" && (constraint == "" || pqErr.Constraint == constraint) {
			return true
		}
	}
	return constraint != "" && strings.Contains(err.Error(), constraint)
}

type randomIDGenerator struct {
	prefix string
}

func (g randomIDGenerator) NewID() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return g.prefix + hex.EncodeToString(buf[:]), nil
}

var _ ports.PostRepository = (*Store)(nil)
var _ ports.PostQueryRepository = (*Store)(nil)
var _ ports.BodyReferenceChecker = (*Store)(nil)
var _ ports.TransactionRunner = (*TransactionRunner)(nil)
var _ ports.OutboxPublisher = (*Store)(nil)
