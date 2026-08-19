-- +goose Up
ALTER TABLE scheme_betting_shard_leases
    ADD COLUMN IF NOT EXISTS lease_kind VARCHAR(16) NOT NULL DEFAULT 'strategy';
ALTER TABLE scheme_betting_shard_leases
    DROP CONSTRAINT IF EXISTS scheme_betting_shard_leases_pkey;
ALTER TABLE scheme_betting_shard_leases
    ADD CONSTRAINT scheme_betting_shard_leases_pkey PRIMARY KEY (lease_kind, shard_no);
ALTER TABLE scheme_betting_shard_leases
    DROP CONSTRAINT IF EXISTS scheme_betting_shard_leases_kind_check;
ALTER TABLE scheme_betting_shard_leases
    ADD CONSTRAINT scheme_betting_shard_leases_kind_check
    CHECK (lease_kind IN ('strategy', 'dispatcher'));

CREATE TABLE IF NOT EXISTS scheme_betting_draw_leases (
    lottery_code VARCHAR(64) PRIMARY KEY,
    lease_owner VARCHAR(128) NOT NULL,
    lease_epoch BIGINT NOT NULL CHECK (lease_epoch > 0),
    lease_until TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS scheme_betting_dispatch_rate_buckets (
    scope_type VARCHAR(16) NOT NULL CHECK (scope_type IN ('global', 'lottery', 'account')),
    scope_key VARCHAR(128) NOT NULL,
    window_start TIMESTAMPTZ NOT NULL,
    dispatch_count INTEGER NOT NULL DEFAULT 0 CHECK (dispatch_count >= 0),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (scope_type, scope_key, window_start)
);

ALTER TABLE scheme_betting_capacity_limits
    ADD COLUMN IF NOT EXISTS max_account_dispatch_per_second INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS max_global_dispatch_per_second INTEGER NOT NULL DEFAULT 100;
ALTER TABLE scheme_betting_capacity_limits
    DROP CONSTRAINT IF EXISTS scheme_betting_capacity_account_rate_check;
ALTER TABLE scheme_betting_capacity_limits
    ADD CONSTRAINT scheme_betting_capacity_account_rate_check
    CHECK (max_account_dispatch_per_second > 0);
ALTER TABLE scheme_betting_capacity_limits
    DROP CONSTRAINT IF EXISTS scheme_betting_capacity_global_rate_check;
ALTER TABLE scheme_betting_capacity_limits
    ADD CONSTRAINT scheme_betting_capacity_global_rate_check
    CHECK (max_global_dispatch_per_second > 0);

CREATE INDEX IF NOT EXISTS idx_scheme_betting_rate_buckets_cleanup
    ON scheme_betting_dispatch_rate_buckets (window_start);

-- +goose Down
DROP INDEX IF EXISTS idx_scheme_betting_rate_buckets_cleanup;
ALTER TABLE scheme_betting_capacity_limits
    DROP CONSTRAINT IF EXISTS scheme_betting_capacity_global_rate_check,
    DROP CONSTRAINT IF EXISTS scheme_betting_capacity_account_rate_check,
    DROP COLUMN IF EXISTS max_global_dispatch_per_second,
    DROP COLUMN IF EXISTS max_account_dispatch_per_second;
DROP TABLE IF EXISTS scheme_betting_dispatch_rate_buckets;
DROP TABLE IF EXISTS scheme_betting_draw_leases;
DELETE FROM scheme_betting_shard_leases WHERE lease_kind <> 'strategy';
ALTER TABLE scheme_betting_shard_leases
    DROP CONSTRAINT IF EXISTS scheme_betting_shard_leases_kind_check;
ALTER TABLE scheme_betting_shard_leases
    DROP CONSTRAINT IF EXISTS scheme_betting_shard_leases_pkey;
ALTER TABLE scheme_betting_shard_leases
    DROP COLUMN IF EXISTS lease_kind;
ALTER TABLE scheme_betting_shard_leases
    ADD CONSTRAINT scheme_betting_shard_leases_pkey PRIMARY KEY (shard_no);
