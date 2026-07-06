// Package outbox contains reusable PostgreSQL transactional outbox primitives.
//
// DispatchRepository is intentionally limited to claim-based dispatcher tables.
// The table must expose the common outbox columns plus claimed_by,
// claim_started_at, updated_at, next_retry_at, published_at, last_error, and
// support PENDING, FAILED, CLAIMING, PUBLISHED, and DEAD statuses. Services with
// simpler schemas can still reuse InsertPublisher without adopting the dispatch
// state machine.
package outbox

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"time"
)

var ErrClaimLost = errors.New("outbox claim lost")

type DB interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type Execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type Config struct {
	Table                  string
	VersionColumn          string
	AggregateVersionColumn string
}

type Event struct {
	ID               int64
	EventID          string
	EventType        string
	PayloadVersion   int
	AggregateType    string
	AggregateID      string
	AggregateVersion *int64
	Payload          []byte
	OccurredAt       time.Time
	AttemptCount     int
}

type ClaimOptions struct {
	DispatcherID string
	BatchSize    int
	StaleAfter   time.Duration
	Now          time.Time
}

type Published struct {
	ID           int64
	DispatcherID string
	PublishedAt  time.Time
}

type Failure struct {
	ID           int64
	DispatcherID string
	AttemptCount int
	NextRetryAt  *time.Time
	Dead         bool
	LastError    string
	FailedAt     time.Time
}

type Message struct {
	EventType      string
	PayloadVersion int
	AggregateType  string
	AggregateID    string
	Payload        []byte
	OccurredAt     time.Time
}

type EventIDGenerator interface {
	NewEventID() (string, error)
}

type DispatchRepository struct {
	db                    DB
	claimPendingSQL       string
	claimAggregateVersion bool
	markPublishedSQL      string
	markFailedSQL         string
}

func NewDispatchRepository(db DB, config Config) *DispatchRepository {
	sqls := buildSQL(config)
	return &DispatchRepository{
		db:                    db,
		claimPendingSQL:       sqls.claimPending,
		claimAggregateVersion: sqls.claimAggregateVersion,
		markPublishedSQL:      sqls.markPublished,
		markFailedSQL:         sqls.markFailed,
	}
}

func (r *DispatchRepository) ClaimPending(ctx context.Context, options ClaimOptions) ([]Event, error) {
	staleBefore := options.Now.Add(-options.StaleAfter)
	rows, err := r.db.QueryContext(ctx, r.claimPendingSQL, options.Now, staleBefore, options.DispatcherID, options.BatchSize)
	if err != nil {
		return nil, fmt.Errorf("claim outbox events: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		dest := []any{
			&event.ID,
			&event.EventID,
			&event.EventType,
			&event.PayloadVersion,
			&event.AggregateType,
			&event.AggregateID,
		}
		var aggregateVersion sql.NullInt64
		if r.claimAggregateVersion {
			dest = append(dest, &aggregateVersion)
		}
		dest = append(dest,
			&event.Payload,
			&event.OccurredAt,
			&event.AttemptCount,
		)
		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("scan claimed outbox event: %w", err)
		}
		if aggregateVersion.Valid {
			value := aggregateVersion.Int64
			event.AggregateVersion = &value
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate claimed outbox events: %w", err)
	}
	return events, nil
}

func (r *DispatchRepository) MarkPublished(ctx context.Context, published Published) error {
	result, err := r.db.ExecContext(ctx, r.markPublishedSQL, published.PublishedAt, published.ID, published.DispatcherID)
	if err != nil {
		return fmt.Errorf("mark outbox published: %w", err)
	}
	return requireClaimedRow(result)
}

func (r *DispatchRepository) MarkFailed(ctx context.Context, failure Failure) error {
	status := "FAILED"
	var nextRetryAt any = failure.NextRetryAt
	if failure.Dead {
		status = "DEAD"
		nextRetryAt = nil
	}
	result, err := r.db.ExecContext(ctx, r.markFailedSQL,
		status,
		failure.AttemptCount,
		nextRetryAt,
		failure.LastError,
		failure.FailedAt,
		failure.ID,
		failure.DispatcherID,
	)
	if err != nil {
		return fmt.Errorf("mark outbox failed: %w", err)
	}
	return requireClaimedRow(result)
}

type InsertPublisher struct {
	ids       EventIDGenerator
	insertSQL string
}

func NewInsertPublisher(config Config, ids EventIDGenerator) *InsertPublisher {
	if ids == nil {
		panic("outbox: nil event id generator")
	}
	sqls := buildSQL(config)
	return &InsertPublisher{ids: ids, insertSQL: sqls.insert}
}

func (p *InsertPublisher) Publish(ctx context.Context, execer Execer, message Message) error {
	eventID, err := p.ids.NewEventID()
	if err != nil {
		return err
	}
	version := message.PayloadVersion
	if version == 0 {
		version = 1
	}
	if _, err := execer.ExecContext(ctx, p.insertSQL,
		eventID,
		message.EventType,
		version,
		message.AggregateType,
		message.AggregateID,
		message.Payload,
		message.OccurredAt,
	); err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}
	return nil
}

func requireClaimedRow(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read outbox affected rows: %w", err)
	}
	if affected == 0 {
		// Conditional updates only succeed for the current claim owner. A zero
		// row result means another dispatcher finished or reclaimed the event.
		return ErrClaimLost
	}
	return nil
}

type sqlSet struct {
	claimPending          string
	claimAggregateVersion bool
	markPublished         string
	markFailed            string
	insert                string
}

func buildSQL(config Config) sqlSet {
	table := identifier(config.Table, "outbox_events")
	versionColumn := identifier(config.VersionColumn, "payload_version")
	// Keep the optional aggregate-version projection in SQL files so Go only
	// validates identifiers and selects the repository contract variant.
	claimTemplate := claimPendingTemplate
	claimArgs := []any{table, table, versionColumn}
	claimAggregateVersion := false
	if config.AggregateVersionColumn != "" {
		aggregateVersionColumn := identifier(config.AggregateVersionColumn, "")
		claimTemplate = claimPendingWithAggregateVersionTemplate
		claimArgs = []any{table, table, versionColumn, aggregateVersionColumn}
		claimAggregateVersion = true
	}
	return sqlSet{
		claimPending:          fmt.Sprintf(claimTemplate, claimArgs...),
		claimAggregateVersion: claimAggregateVersion,
		markPublished:         fmt.Sprintf(markPublishedTemplate, table),
		markFailed:            fmt.Sprintf(markFailedTemplate, table),
		insert:                fmt.Sprintf(insertTemplate, table, versionColumn),
	}
}

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func identifier(value, fallback string) string {
	if value == "" {
		return fallback
	}
	if identifierPattern.MatchString(value) {
		return value
	}
	panic(fmt.Sprintf("outbox: invalid SQL identifier %q", value))
}
