-- +goose Up
-- Phase 3-5: authoritative chain ownership and fenced real-bet dispatch.

ALTER TABLE scheme_instances
    ADD COLUMN IF NOT EXISTS betting_owner VARCHAR(16) NOT NULL DEFAULT 'legacy',
    ADD COLUMN IF NOT EXISTS strict_chain_state VARCHAR(32) NOT NULL DEFAULT 'idle',
    ADD COLUMN IF NOT EXISTS chain_id VARCHAR(80),
    ADD COLUMN IF NOT EXISTS chain_seq BIGINT NOT NULL DEFAULT 0;

ALTER TABLE scheme_instances DROP CONSTRAINT IF EXISTS scheme_instances_betting_owner_check;
ALTER TABLE scheme_instances ADD CONSTRAINT scheme_instances_betting_owner_check
    CHECK (betting_owner IN ('legacy', 'event'));
ALTER TABLE scheme_instances DROP CONSTRAINT IF EXISTS scheme_instances_strict_chain_state_check;
ALTER TABLE scheme_instances ADD CONSTRAINT scheme_instances_strict_chain_state_check
    CHECK (strict_chain_state IN ('idle', 'active', 'blocked_requires_rearm'));

ALTER TABLE scheme_bet_outbox
    ADD COLUMN IF NOT EXISTS member_id BIGINT REFERENCES members(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS source_state_version BIGINT,
    ADD COLUMN IF NOT EXISTS initial_bet BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS chain_id VARCHAR(80),
    ADD COLUMN IF NOT EXISTS chain_seq BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS frozen_request JSONB,
    ADD COLUMN IF NOT EXISTS frozen_request_hash TEXT,
    ADD COLUMN IF NOT EXISTS command_frozen_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS dispatch_started_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS outcome_reason VARCHAR(64),
    ADD COLUMN IF NOT EXISTS provider_response_digest TEXT;

ALTER TABLE scheme_bet_outbox DROP CONSTRAINT IF EXISTS scheme_bet_outbox_state_check;
ALTER TABLE scheme_bet_outbox ADD CONSTRAINT scheme_bet_outbox_state_check
    CHECK (state IN (
        'pending', 'leased', 'sent_unknown', 'accepted', 'rejected', 'expired', 'cancelled',
        'accepted_wrong_period', 'external_acceptance_unknown'
    ));
ALTER TABLE scheme_bet_outbox DROP CONSTRAINT IF EXISTS scheme_bet_outbox_check;
ALTER TABLE scheme_bet_outbox ADD CONSTRAINT scheme_bet_outbox_frozen_formal_check
    CHECK (mode = 'shadow' OR (
        frozen_request IS NOT NULL AND frozen_request_hash IS NOT NULL AND command_frozen_at IS NOT NULL
    ));

ALTER TABLE scheme_bet_attempts DROP CONSTRAINT IF EXISTS scheme_bet_attempts_outcome_check;
ALTER TABLE scheme_bet_attempts ADD CONSTRAINT scheme_bet_attempts_outcome_check
    CHECK (outcome IN (
        'started', 'sent_unknown', 'accepted', 'rejected', 'expired', 'cancelled',
        'accepted_wrong_period', 'external_acceptance_unknown'
    ));

CREATE TABLE IF NOT EXISTS scheme_betting_shard_leases (
    shard_no INTEGER PRIMARY KEY,
    lease_owner VARCHAR(128) NOT NULL,
    lease_epoch BIGINT NOT NULL,
    lease_until TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (shard_no >= 0),
    CHECK (lease_epoch > 0)
);

CREATE TABLE IF NOT EXISTS scheme_betting_admin_actions (
    id BIGSERIAL PRIMARY KEY,
    scheme_id VARCHAR(64) NOT NULL REFERENCES scheme_instances(id) ON DELETE CASCADE,
    outbox_id BIGINT REFERENCES scheme_bet_outbox(id) ON DELETE SET NULL,
    action VARCHAR(32) NOT NULL CHECK (action IN ('rearm', 'cancel', 'resolve_unknown')),
    actor_admin_id BIGINT,
    reason TEXT NOT NULL CHECK (length(btrim(reason)) >= 4),
    before_state JSONB NOT NULL,
    after_state JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS scheme_betting_capacity_limits (
    lottery_code VARCHAR(64) PRIMARY KEY,
    max_due_outbox INTEGER NOT NULL,
    max_active_schemes INTEGER NOT NULL,
    max_dispatch_per_second INTEGER NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (max_due_outbox > 0),
    CHECK (max_active_schemes > 0),
    CHECK (max_dispatch_per_second > 0)
);

CREATE INDEX IF NOT EXISTS idx_scheme_bet_outbox_edf_recovery
    ON scheme_bet_outbox (shard_no, safe_deadline_at, id)
    WHERE state IN ('pending', 'leased');
CREATE INDEX IF NOT EXISTS idx_scheme_instances_event_owner
    ON scheme_instances (lottery_code, status, id)
    WHERE betting_owner = 'event';
CREATE INDEX IF NOT EXISTS idx_scheme_betting_admin_actions_scheme
    ON scheme_betting_admin_actions (scheme_id, created_at DESC, id DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_scheme_betting_admin_actions_scheme;
DROP INDEX IF EXISTS idx_scheme_instances_event_owner;
DROP INDEX IF EXISTS idx_scheme_bet_outbox_edf_recovery;
DROP TABLE IF EXISTS scheme_betting_capacity_limits;
DROP TABLE IF EXISTS scheme_betting_admin_actions;
DROP TABLE IF EXISTS scheme_betting_shard_leases;
ALTER TABLE scheme_bet_attempts DROP CONSTRAINT IF EXISTS scheme_bet_attempts_outcome_check;
ALTER TABLE scheme_bet_attempts ADD CONSTRAINT scheme_bet_attempts_outcome_check
    CHECK (outcome IN ('started', 'sent_unknown', 'accepted', 'rejected', 'expired', 'cancelled'));
ALTER TABLE scheme_bet_outbox DROP CONSTRAINT IF EXISTS scheme_bet_outbox_frozen_formal_check;
ALTER TABLE scheme_bet_outbox DROP CONSTRAINT IF EXISTS scheme_bet_outbox_state_check;
ALTER TABLE scheme_bet_outbox ADD CONSTRAINT scheme_bet_outbox_state_check
    CHECK (state IN ('pending', 'leased', 'sent_unknown', 'accepted', 'rejected', 'expired', 'cancelled'));
ALTER TABLE scheme_bet_outbox ADD CONSTRAINT scheme_bet_outbox_check CHECK (mode = 'shadow' OR state <> 'pending');
ALTER TABLE scheme_bet_outbox
    DROP COLUMN IF EXISTS provider_response_digest,
    DROP COLUMN IF EXISTS outcome_reason,
    DROP COLUMN IF EXISTS dispatch_started_at,
    DROP COLUMN IF EXISTS command_frozen_at,
    DROP COLUMN IF EXISTS frozen_request_hash,
    DROP COLUMN IF EXISTS frozen_request,
    DROP COLUMN IF EXISTS chain_seq,
    DROP COLUMN IF EXISTS chain_id,
    DROP COLUMN IF EXISTS initial_bet,
    DROP COLUMN IF EXISTS source_state_version,
    DROP COLUMN IF EXISTS member_id;
ALTER TABLE scheme_instances DROP CONSTRAINT IF EXISTS scheme_instances_strict_chain_state_check;
ALTER TABLE scheme_instances DROP CONSTRAINT IF EXISTS scheme_instances_betting_owner_check;
ALTER TABLE scheme_instances
    DROP COLUMN IF EXISTS chain_seq,
    DROP COLUMN IF EXISTS chain_id,
    DROP COLUMN IF EXISTS strict_chain_state,
    DROP COLUMN IF EXISTS betting_owner;
