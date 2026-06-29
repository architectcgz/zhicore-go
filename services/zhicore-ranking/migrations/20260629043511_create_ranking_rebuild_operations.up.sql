BEGIN;

CREATE TABLE ranking_rebuild_operation (
  operation_id VARCHAR(64) PRIMARY KEY,
  status VARCHAR(32) NOT NULL,
  dry_run BOOLEAN NOT NULL DEFAULT FALSE,
  force BOOLEAN NOT NULL DEFAULT FALSE,
  requested_by BIGINT NOT NULL,
  reason VARCHAR(200) NULL,
  lock_key VARCHAR(128) NOT NULL,
  lock_owner VARCHAR(128) NOT NULL,
  lock_ttl_seconds INTEGER NOT NULL,
  failed_stage VARCHAR(64) NULL,
  error_code VARCHAR(64) NULL,
  error_message VARCHAR(500) NULL,
  replayed_events BIGINT NOT NULL DEFAULT 0,
  rebuilt_posts BIGINT NOT NULL DEFAULT 0,
  refreshed_snapshots INTEGER NOT NULL DEFAULT 0,
  refreshed_candidates BOOLEAN NOT NULL DEFAULT FALSE,
  accepted_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  started_at TIMESTAMPTZ NULL,
  completed_at TIMESTAMPTZ NULL,
  duration_ms BIGINT NULL,
  request_id VARCHAR(128) NULL,
  trace_id VARCHAR(128) NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT ck_ranking_rebuild_status CHECK (
    status IN ('ACCEPTED', 'RUNNING', 'SUCCEEDED', 'PARTIAL_FAILED', 'FAILED', 'CANCELED')
  ),
  CONSTRAINT ck_ranking_rebuild_lock_ttl_positive CHECK (lock_ttl_seconds > 0),
  CONSTRAINT ck_ranking_rebuild_counts_nonnegative CHECK (
    replayed_events >= 0
    AND rebuilt_posts >= 0
    AND refreshed_snapshots >= 0
  ),
  CONSTRAINT ck_ranking_rebuild_duration_nonnegative CHECK (
    duration_ms IS NULL OR duration_ms >= 0
  )
);

CREATE INDEX idx_ranking_rebuild_operation_status
  ON ranking_rebuild_operation(status, accepted_at DESC);

CREATE INDEX idx_ranking_rebuild_operation_requested_by
  ON ranking_rebuild_operation(requested_by, accepted_at DESC);

COMMIT;
