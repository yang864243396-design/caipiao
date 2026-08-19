-- +goose Up
-- +goose StatementBegin
ALTER TABLE scheme_bet_outbox
    ADD COLUMN IF NOT EXISTS ready_published_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS ready_publish_attempts INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS reconcile_published_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS reconcile_published_state VARCHAR(32),
    ADD COLUMN IF NOT EXISTS reconcile_publish_attempts INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_scheme_bet_outbox_ready_publish
    ON scheme_bet_outbox (shard_no, safe_deadline_at, id)
    WHERE mode IN ('gray', 'production') AND ready_published_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_scheme_bet_outbox_reconcile_publish
    ON scheme_bet_outbox (updated_at, id)
    WHERE mode IN ('gray', 'production')
      AND state NOT IN ('pending', 'leased')
      AND reconcile_published_state IS DISTINCT FROM state;
CREATE INDEX IF NOT EXISTS idx_scheme_bet_outbox_scheme_detail
    ON scheme_bet_outbox (scheme_id, created_at DESC, id DESC)
    WHERE scheme_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_wallet_ledger_scheme_bet_audit
    ON wallet_ledger (order_ref, created_at DESC, id DESC)
    WHERE txn_type IN ('bet_debit', 'payout') AND order_ref IS NOT NULL;

CREATE TABLE scheme_betting_retention_policies (
    data_kind VARCHAR(32) PRIMARY KEY,
    online_days INTEGER NOT NULL CHECK (online_days > 0),
    archive_days INTEGER NOT NULL CHECK (archive_days >= online_days),
    delete_allowed BOOLEAN NOT NULL DEFAULT false,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO scheme_betting_retention_policies (data_kind, online_days, archive_days, delete_allowed)
VALUES
    ('outbox_audit', 90, 2555, false),
    ('financial_ledger', 2555, 2555, false)
ON CONFLICT (data_kind) DO NOTHING;

CREATE TABLE scheme_bet_outbox_archive (
    archive_month DATE NOT NULL,
    outbox_id BIGINT NOT NULL,
    scheme_id VARCHAR(64),
    origin VARCHAR(16) NOT NULL,
    terminal_state VARCHAR(32) NOT NULL,
    terminal_at TIMESTAMPTZ NOT NULL,
    snapshot JSONB NOT NULL,
    archived_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (archive_month, outbox_id)
) PARTITION BY RANGE (archive_month);

CREATE TABLE scheme_bet_outbox_archive_2026
    PARTITION OF scheme_bet_outbox_archive
    FOR VALUES FROM ('2026-01-01') TO ('2027-01-01');
CREATE TABLE scheme_bet_outbox_archive_2027
    PARTITION OF scheme_bet_outbox_archive
    FOR VALUES FROM ('2027-01-01') TO ('2028-01-01');

CREATE INDEX idx_scheme_bet_outbox_archive_scheme
    ON scheme_bet_outbox_archive (scheme_id, terminal_at DESC, outbox_id DESC);

CREATE OR REPLACE FUNCTION ensure_scheme_bet_outbox_archive_year(target_year INTEGER)
RETURNS VOID
LANGUAGE plpgsql
AS $$
DECLARE
    start_date DATE := make_date(target_year, 1, 1);
    end_date DATE := make_date(target_year + 1, 1, 1);
    partition_name TEXT := format('scheme_bet_outbox_archive_%s', target_year);
BEGIN
    IF target_year < 2026 OR target_year > 2200 THEN
        RAISE EXCEPTION 'archive year is outside the supported range';
    END IF;
    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS %I PARTITION OF scheme_bet_outbox_archive FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date
    );
END;
$$;

CREATE OR REPLACE FUNCTION archive_terminal_scheme_bets(cutoff TIMESTAMPTZ, row_limit INTEGER DEFAULT 1000)
RETURNS INTEGER
LANGUAGE plpgsql
AS $$
DECLARE
    archived_count INTEGER;
BEGIN
    IF row_limit < 1 OR row_limit > 10000 THEN
        RAISE EXCEPTION 'row_limit must be between 1 and 10000';
    END IF;

    WITH candidates AS (
        SELECT o.*
        FROM scheme_bet_outbox o
        WHERE o.terminal_at IS NOT NULL
          AND o.terminal_at < cutoff
          AND o.state NOT IN ('pending', 'leased', 'sent_unknown', 'external_acceptance_unknown')
          AND (o.state <> 'accepted' OR o.financial_finalized_at IS NOT NULL)
        ORDER BY o.terminal_at, o.id
        LIMIT row_limit
    )
    INSERT INTO scheme_bet_outbox_archive
        (archive_month, outbox_id, scheme_id, origin, terminal_state, terminal_at, snapshot)
    SELECT date_trunc('month', c.terminal_at)::date,
           c.id,
           c.scheme_id,
           c.origin,
           c.state,
           c.terminal_at,
           to_jsonb(c) || jsonb_build_object(
               'attempts',
               COALESCE((
                   SELECT jsonb_agg(to_jsonb(a) ORDER BY a.attempt_no)
                   FROM scheme_bet_attempts a
                   WHERE a.outbox_id = c.id
               ), '[]'::jsonb)
           )
    FROM candidates c
    ON CONFLICT (archive_month, outbox_id) DO NOTHING;

    GET DIAGNOSTICS archived_count = ROW_COUNT;
    RETURN archived_count;
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION IF EXISTS archive_terminal_scheme_bets(TIMESTAMPTZ, INTEGER);
DROP FUNCTION IF EXISTS ensure_scheme_bet_outbox_archive_year(INTEGER);
DROP TABLE IF EXISTS scheme_bet_outbox_archive CASCADE;
DROP TABLE IF EXISTS scheme_betting_retention_policies;
DROP INDEX IF EXISTS idx_wallet_ledger_scheme_bet_audit;
DROP INDEX IF EXISTS idx_scheme_bet_outbox_scheme_detail;
DROP INDEX IF EXISTS idx_scheme_bet_outbox_reconcile_publish;
DROP INDEX IF EXISTS idx_scheme_bet_outbox_ready_publish;
ALTER TABLE scheme_bet_outbox
    DROP COLUMN IF EXISTS reconcile_publish_attempts,
    DROP COLUMN IF EXISTS reconcile_published_state,
    DROP COLUMN IF EXISTS reconcile_published_at,
    DROP COLUMN IF EXISTS ready_publish_attempts,
    DROP COLUMN IF EXISTS ready_published_at;
-- +goose StatementEnd
