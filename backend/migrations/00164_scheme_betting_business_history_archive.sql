-- +goose Up
-- +goose StatementBegin
INSERT INTO scheme_betting_retention_policies (data_kind, online_days, archive_days, delete_allowed)
VALUES
    ('bet_orders', 365, 2555, false),
    ('cloud_bet_records', 365, 2555, false)
ON CONFLICT (data_kind) DO UPDATE
SET online_days = EXCLUDED.online_days,
    archive_days = EXCLUDED.archive_days,
    delete_allowed = false,
    updated_at = now();

CREATE TABLE scheme_betting_business_history_archive (
    archive_month DATE NOT NULL,
    data_kind VARCHAR(32) NOT NULL,
    source_id BIGINT NOT NULL,
    member_id BIGINT NOT NULL,
    business_no VARCHAR(80),
    occurred_at TIMESTAMPTZ NOT NULL,
    snapshot JSONB NOT NULL,
    archived_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (archive_month, data_kind, source_id)
) PARTITION BY RANGE (archive_month);

CREATE TABLE scheme_betting_business_history_archive_legacy
    PARTITION OF scheme_betting_business_history_archive
    FOR VALUES FROM ('2000-01-01') TO ('2026-01-01');
CREATE TABLE scheme_betting_business_history_archive_2026
    PARTITION OF scheme_betting_business_history_archive
    FOR VALUES FROM ('2026-01-01') TO ('2027-01-01');
CREATE TABLE scheme_betting_business_history_archive_2027
    PARTITION OF scheme_betting_business_history_archive
    FOR VALUES FROM ('2027-01-01') TO ('2028-01-01');

CREATE INDEX idx_scheme_betting_business_archive_member
    ON scheme_betting_business_history_archive
    (member_id, occurred_at DESC, data_kind, source_id DESC);
CREATE INDEX idx_scheme_betting_business_archive_no
    ON scheme_betting_business_history_archive
    (data_kind, business_no)
    WHERE business_no IS NOT NULL;

CREATE OR REPLACE FUNCTION ensure_scheme_betting_business_archive_year(target_year INTEGER)
RETURNS VOID
LANGUAGE plpgsql
AS $$
DECLARE
    start_date DATE := make_date(target_year, 1, 1);
    end_date DATE := make_date(target_year + 1, 1, 1);
    partition_name TEXT := format('scheme_betting_business_history_archive_%s', target_year);
BEGIN
    IF target_year < 2026 OR target_year > 2200 THEN
        RAISE EXCEPTION 'archive year is outside the supported range';
    END IF;
    EXECUTE format(
        'CREATE TABLE IF NOT EXISTS %I PARTITION OF scheme_betting_business_history_archive FOR VALUES FROM (%L) TO (%L)',
        partition_name, start_date, end_date
    );
END;
$$;

CREATE OR REPLACE FUNCTION archive_scheme_betting_business_history(
    cutoff TIMESTAMPTZ,
    row_limit INTEGER DEFAULT 1000
)
RETURNS INTEGER
LANGUAGE plpgsql
AS $$
DECLARE
    inserted_rows INTEGER;
    total_rows INTEGER := 0;
BEGIN
    IF row_limit < 1 OR row_limit > 10000 THEN
        RAISE EXCEPTION 'row_limit must be between 1 and 10000';
    END IF;

    INSERT INTO scheme_betting_business_history_archive
        (archive_month, data_kind, source_id, member_id, business_no, occurred_at, snapshot)
    SELECT date_trunc('month', o.placed_at)::date,
           'bet_orders',
           o.id,
           o.member_id,
           o.order_no,
           o.placed_at,
           to_jsonb(o)
    FROM bet_orders o
    WHERE o.placed_at < cutoff
      AND o.status IN ('win', 'lose', 'cancel')
    ORDER BY o.placed_at, o.id
    LIMIT row_limit
    ON CONFLICT (archive_month, data_kind, source_id) DO NOTHING;
    GET DIAGNOSTICS inserted_rows = ROW_COUNT;
    total_rows := total_rows + inserted_rows;

    INSERT INTO scheme_betting_business_history_archive
        (archive_month, data_kind, source_id, member_id, business_no, occurred_at, snapshot)
    SELECT date_trunc('month', c.placed_at)::date,
           'cloud_bet_records',
           c.id,
           c.member_id,
           c.record_no,
           c.placed_at,
           to_jsonb(c)
    FROM cloud_bet_records c
    WHERE c.placed_at < cutoff
      AND c.status IN ('hit', 'miss')
    ORDER BY c.placed_at, c.id
    LIMIT row_limit
    ON CONFLICT (archive_month, data_kind, source_id) DO NOTHING;
    GET DIAGNOSTICS inserted_rows = ROW_COUNT;
    total_rows := total_rows + inserted_rows;

    INSERT INTO scheme_betting_business_history_archive
        (archive_month, data_kind, source_id, member_id, business_no, occurred_at, snapshot)
    SELECT date_trunc('month', l.created_at)::date,
           'wallet_ledger',
           l.id,
           l.member_id,
           l.ledger_no,
           l.created_at,
           to_jsonb(l)
    FROM wallet_ledger l
    WHERE l.created_at < cutoff
    ORDER BY l.created_at, l.id
    LIMIT row_limit
    ON CONFLICT (archive_month, data_kind, source_id) DO NOTHING;
    GET DIAGNOSTICS inserted_rows = ROW_COUNT;
    total_rows := total_rows + inserted_rows;

    RETURN total_rows;
END;
$$;

CREATE OR REPLACE FUNCTION reject_wallet_ledger_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'wallet_ledger is append-only';
END;
$$;

DROP TRIGGER IF EXISTS trg_wallet_ledger_append_only ON wallet_ledger;
CREATE TRIGGER trg_wallet_ledger_append_only
BEFORE UPDATE OR DELETE ON wallet_ledger
FOR EACH ROW EXECUTE FUNCTION reject_wallet_ledger_mutation();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trg_wallet_ledger_append_only ON wallet_ledger;
DROP FUNCTION IF EXISTS reject_wallet_ledger_mutation();
DROP FUNCTION IF EXISTS archive_scheme_betting_business_history(TIMESTAMPTZ, INTEGER);
DROP FUNCTION IF EXISTS ensure_scheme_betting_business_archive_year(INTEGER);
DROP TABLE IF EXISTS scheme_betting_business_history_archive CASCADE;
DELETE FROM scheme_betting_retention_policies
WHERE data_kind IN ('bet_orders', 'cloud_bet_records');
-- +goose StatementEnd
