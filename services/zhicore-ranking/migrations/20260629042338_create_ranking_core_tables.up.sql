BEGIN;

CREATE TABLE ranking_event_ledger (
  event_id VARCHAR(128) PRIMARY KEY,
  event_type VARCHAR(128) NOT NULL,
  post_id BIGINT NOT NULL,
  public_post_id VARCHAR(32) NULL,
  bucket_start TIMESTAMPTZ NOT NULL,
  actor_id BIGINT NULL,
  author_id BIGINT NULL,
  metric_type VARCHAR(32) NOT NULL,
  delta INTEGER NOT NULL,
  occurred_at TIMESTAMPTZ NOT NULL,
  published_at TIMESTAMPTZ NULL,
  partition_key VARCHAR(64) NOT NULL,
  source_service VARCHAR(64) NULL,
  source_op_id VARCHAR(128) NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT ck_ranking_ledger_delta_nonzero CHECK (delta <> 0),
  CONSTRAINT ck_ranking_ledger_metric_type CHECK (
    metric_type IN ('VIEW', 'LIKE', 'FAVORITE', 'COMMENT')
  )
);

CREATE INDEX idx_ranking_ledger_post_occurred
  ON ranking_event_ledger(post_id, occurred_at, event_id);

CREATE INDEX idx_ranking_ledger_occurred_event
  ON ranking_event_ledger(occurred_at, event_id);

CREATE INDEX idx_ranking_ledger_bucket
  ON ranking_event_ledger(bucket_start, post_id);

CREATE TABLE ranking_delta_bucket (
  bucket_start TIMESTAMPTZ NOT NULL,
  post_id BIGINT NOT NULL,
  view_delta BIGINT NOT NULL DEFAULT 0,
  like_delta INTEGER NOT NULL DEFAULT 0,
  favorite_delta INTEGER NOT NULL DEFAULT 0,
  comment_delta INTEGER NOT NULL DEFAULT 0,
  applied_view_delta BIGINT NOT NULL DEFAULT 0,
  applied_like_delta INTEGER NOT NULL DEFAULT 0,
  applied_favorite_delta INTEGER NOT NULL DEFAULT 0,
  applied_comment_delta INTEGER NOT NULL DEFAULT 0,
  flush_owner VARCHAR(128) NULL,
  flush_started_at TIMESTAMPTZ NULL,
  flushed BOOLEAN NOT NULL DEFAULT FALSE,
  flushed_at TIMESTAMPTZ NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (bucket_start, post_id),
  CONSTRAINT ck_ranking_bucket_flushed_applied CHECK (
    flushed = FALSE OR (
      applied_view_delta = view_delta
      AND applied_like_delta = like_delta
      AND applied_favorite_delta = favorite_delta
      AND applied_comment_delta = comment_delta
    )
  )
);

CREATE INDEX idx_ranking_bucket_flush
  ON ranking_delta_bucket(flushed, bucket_start, updated_at);

CREATE TABLE ranking_post_state (
  post_id BIGINT PRIMARY KEY,
  public_post_id VARCHAR(32) NULL,
  author_id BIGINT NULL,
  published_at TIMESTAMPTZ NULL,
  topic_ids JSONB NOT NULL DEFAULT '[]'::JSONB,
  public_visible BOOLEAN NOT NULL DEFAULT FALSE,
  content_status VARCHAR(32) NULL,
  visibility_reason VARCHAR(64) NULL,
  visibility_updated_at TIMESTAMPTZ NULL,
  view_count BIGINT NOT NULL DEFAULT 0,
  like_count INTEGER NOT NULL DEFAULT 0,
  favorite_count INTEGER NOT NULL DEFAULT 0,
  comment_count INTEGER NOT NULL DEFAULT 0,
  raw_score DOUBLE PRECISION NOT NULL DEFAULT 0,
  hot_score DOUBLE PRECISION NOT NULL DEFAULT 0,
  version BIGINT NOT NULL DEFAULT 0,
  last_bucket_start TIMESTAMPTZ NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT ck_ranking_post_topic_ids_array CHECK (jsonb_typeof(topic_ids) = 'array'),
  CONSTRAINT ck_ranking_post_counts_nonnegative CHECK (
    view_count >= 0
    AND like_count >= 0
    AND favorite_count >= 0
    AND comment_count >= 0
  )
);

CREATE INDEX idx_ranking_post_state_hot_score
  ON ranking_post_state(public_visible, hot_score DESC, post_id);

CREATE UNIQUE INDEX idx_ranking_post_state_public_post
  ON ranking_post_state(public_post_id)
  WHERE public_post_id IS NOT NULL;

CREATE INDEX idx_ranking_post_state_author
  ON ranking_post_state(author_id)
  WHERE author_id IS NOT NULL;

CREATE INDEX idx_ranking_post_state_topic_ids
  ON ranking_post_state USING GIN (topic_ids);

CREATE TABLE ranking_projection_event_inbox (
  event_id VARCHAR(128) PRIMARY KEY,
  event_type VARCHAR(128) NOT NULL,
  post_id BIGINT NOT NULL,
  public_post_id VARCHAR(32) NULL,
  occurred_at TIMESTAMPTZ NOT NULL,
  aggregate_version BIGINT NULL,
  source_service VARCHAR(64) NULL,
  payload JSONB NOT NULL DEFAULT '{}'::JSONB,
  processed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_ranking_projection_inbox_post
  ON ranking_projection_event_inbox(post_id, occurred_at);

CREATE INDEX idx_ranking_projection_inbox_type_occurred
  ON ranking_projection_event_inbox(event_type, occurred_at);

CREATE TABLE ranking_period_score (
  period_type VARCHAR(16) NOT NULL,
  period_key VARCHAR(32) NOT NULL,
  post_id BIGINT NOT NULL,
  delta_score DOUBLE PRECISION NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (period_type, period_key, post_id),
  CONSTRAINT ck_ranking_period_type CHECK (period_type IN ('DAY', 'WEEK', 'MONTH'))
);

CREATE INDEX idx_ranking_period_score_lookup
  ON ranking_period_score(period_type, period_key, delta_score DESC, post_id);

COMMIT;
