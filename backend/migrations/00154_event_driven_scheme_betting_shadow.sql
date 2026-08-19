-- +goose Up
-- Phase 1/2 only: persist provider facts and shadow commands. No production
-- dispatcher is allowed to consume these rows until a later gray migration.

ALTER TABLE lottery_draws
    ADD COLUMN IF NOT EXISTS source VARCHAR(32) NOT NULL DEFAULT 'legacy',
    ADD COLUMN IF NOT EXISTS provider_event_id TEXT,
    ADD COLUMN IF NOT EXISTS draw_hash TEXT,
    ADD COLUMN IF NOT EXISTS raw_payload_digest TEXT,
    ADD COLUMN IF NOT EXISTS received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN IF NOT EXISTS confirmed_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS lottery_draw_corrections (
    id BIGSERIAL PRIMARY KEY,
    lottery_code VARCHAR(64) NOT NULL,
    issue_no VARCHAR(32) NOT NULL,
    existing_draw_hash TEXT NOT NULL,
    corrected_draw_hash TEXT NOT NULL,
    source VARCHAR(32) NOT NULL,
    provider_event_id TEXT,
    balls JSONB NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    diagnostics JSONB NOT NULL DEFAULT '{}'::jsonb,
    UNIQUE (lottery_code, issue_no, corrected_draw_hash)
);

CREATE TABLE IF NOT EXISTS provider_period_snapshots (
    id BIGSERIAL PRIMARY KEY,
    lottery_code VARCHAR(64) NOT NULL,
    period_no VARCHAR(64) NOT NULL,
    open_at TIMESTAMPTZ,
    close_at TIMESTAMPTZ NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL,
    source VARCHAR(32) NOT NULL DEFAULT 'guaji_periods',
    snapshot_hash TEXT NOT NULL,
    raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (lottery_code, period_no, snapshot_hash)
);

CREATE TABLE IF NOT EXISTS scheme_period_decisions (
    id BIGSERIAL PRIMARY KEY,
    scheme_id VARCHAR(64) NOT NULL REFERENCES scheme_instances(id) ON DELETE CASCADE,
    lottery_code VARCHAR(64) NOT NULL,
    source_period_no VARCHAR(64) NOT NULL,
    source_bet_record_id BIGINT REFERENCES cloud_bet_records(id) ON DELETE SET NULL,
    draw_hash TEXT,
    state_version_before BIGINT NOT NULL,
    state_version_after BIGINT NOT NULL,
    rule_version INTEGER,
    rule_snapshot_hash TEXT,
    local_hit BOOLEAN,
    winning_units INTEGER,
    status VARCHAR(32) NOT NULL CHECK (status IN ('completed', 'blocked', 'duplicate', 'chain_broken')),
    diagnostics JSONB NOT NULL DEFAULT '{}'::jsonb,
    decided_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (scheme_id, source_period_no)
);

CREATE TABLE IF NOT EXISTS scheme_bet_outbox (
    id BIGSERIAL PRIMARY KEY,
    decision_id BIGINT NOT NULL UNIQUE REFERENCES scheme_period_decisions(id) ON DELETE CASCADE,
    scheme_id VARCHAR(64) NOT NULL REFERENCES scheme_instances(id) ON DELETE CASCADE,
    lottery_code VARCHAR(64) NOT NULL,
    source_period_no VARCHAR(64) NOT NULL,
    target_period_no VARCHAR(64) NOT NULL,
    mode VARCHAR(16) NOT NULL DEFAULT 'shadow' CHECK (mode IN ('shadow', 'gray', 'production')),
    state VARCHAR(24) NOT NULL DEFAULT 'pending'
        CHECK (state IN ('pending', 'leased', 'sent_unknown', 'accepted', 'rejected', 'expired', 'cancelled')),
    request_id VARCHAR(80) NOT NULL UNIQUE,
    payload_hash TEXT NOT NULL,
    payload JSONB NOT NULL,
    provider_snapshot_id BIGINT NOT NULL REFERENCES provider_period_snapshots(id),
    close_at TIMESTAMPTZ NOT NULL,
    safe_deadline_at TIMESTAMPTZ NOT NULL,
    shard_no INTEGER NOT NULL,
    lease_owner VARCHAR(128),
    lease_fencing_token BIGINT NOT NULL DEFAULT 0,
    lease_until TIMESTAMPTZ,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    accepted_period_no VARCHAR(64),
    provider_order_no VARCHAR(128),
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    terminal_at TIMESTAMPTZ,
    UNIQUE (scheme_id, target_period_no),
    CHECK (safe_deadline_at < close_at),
    CHECK (mode = 'shadow' OR state <> 'pending')
);

CREATE TABLE IF NOT EXISTS scheme_bet_attempts (
    id BIGSERIAL PRIMARY KEY,
    outbox_id BIGINT NOT NULL REFERENCES scheme_bet_outbox(id) ON DELETE CASCADE,
    attempt_no INTEGER NOT NULL CHECK (attempt_no > 0),
    request_id VARCHAR(80) NOT NULL,
    fencing_token BIGINT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,
    outcome VARCHAR(24) NOT NULL CHECK (outcome IN ('started', 'sent_unknown', 'accepted', 'rejected', 'expired', 'cancelled')),
    http_status INTEGER,
    provider_order_no VARCHAR(128),
    accepted_period_no VARCHAR(64),
    response_digest TEXT,
    error_message TEXT,
    UNIQUE (outbox_id, attempt_no)
);

ALTER TABLE scheme_instances
    ADD COLUMN IF NOT EXISTS state_version BIGINT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_provider_period_snapshots_current
    ON provider_period_snapshots (lottery_code, close_at, observed_at DESC);
CREATE INDEX IF NOT EXISTS idx_scheme_period_decisions_recent
    ON scheme_period_decisions (decided_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_scheme_bet_outbox_dispatch
    ON scheme_bet_outbox (mode, state, shard_no, safe_deadline_at, id)
    WHERE state IN ('pending', 'leased', 'sent_unknown');

-- +goose Down
DROP INDEX IF EXISTS idx_scheme_bet_outbox_dispatch;
DROP INDEX IF EXISTS idx_scheme_period_decisions_recent;
DROP INDEX IF EXISTS idx_provider_period_snapshots_current;
ALTER TABLE scheme_instances DROP COLUMN IF EXISTS state_version;
DROP TABLE IF EXISTS scheme_bet_attempts;
DROP TABLE IF EXISTS scheme_bet_outbox;
DROP TABLE IF EXISTS scheme_period_decisions;
DROP TABLE IF EXISTS provider_period_snapshots;
DROP TABLE IF EXISTS lottery_draw_corrections;
ALTER TABLE lottery_draws
    DROP COLUMN IF EXISTS confirmed_at,
    DROP COLUMN IF EXISTS received_at,
    DROP COLUMN IF EXISTS raw_payload_digest,
    DROP COLUMN IF EXISTS draw_hash,
    DROP COLUMN IF EXISTS provider_event_id,
    DROP COLUMN IF EXISTS source;

