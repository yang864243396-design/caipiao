-- +goose Up
-- +goose StatementBegin
CREATE TABLE core_partition_migration_state (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    phase VARCHAR(24) NOT NULL DEFAULT 'mirroring'
        CHECK (phase IN ('mirroring', 'validated', 'cutover', 'rollback_ready')),
    forward_sync BOOLEAN NOT NULL DEFAULT true,
    reverse_sync BOOLEAN NOT NULL DEFAULT false,
    restart_required BOOLEAN NOT NULL DEFAULT false,
    last_validation JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_validated_at TIMESTAMPTZ,
    cutover_at TIMESTAMPTZ,
    rollback_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO core_partition_migration_state (id)
VALUES (1)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE bet_order_identity (
    id BIGINT PRIMARY KEY,
    order_no VARCHAR(64) NOT NULL UNIQUE,
    placed_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE cloud_bet_record_identity (
    id BIGINT PRIMARY KEY,
    record_no VARCHAR(32) NOT NULL UNIQUE,
    scheme_id VARCHAR(64) NOT NULL,
    period_no VARCHAR(32) NOT NULL,
    placed_at TIMESTAMPTZ NOT NULL,
    UNIQUE (scheme_id, period_no)
);

CREATE TABLE wallet_ledger_identity (
    id BIGINT PRIMARY KEY,
    ledger_no VARCHAR(32) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL
);

INSERT INTO bet_order_identity (id, order_no, placed_at)
SELECT id, order_no, placed_at FROM bet_orders;

INSERT INTO cloud_bet_record_identity (id, record_no, scheme_id, period_no, placed_at)
SELECT id, record_no, scheme_id, period_no, placed_at FROM cloud_bet_records;

INSERT INTO wallet_ledger_identity (id, ledger_no, created_at)
SELECT id, ledger_no, created_at FROM wallet_ledger;

CREATE TABLE bet_orders_partitioned (
    LIKE bet_orders
        INCLUDING DEFAULTS
        INCLUDING CONSTRAINTS
        INCLUDING GENERATED
        INCLUDING STORAGE
        INCLUDING COMMENTS
) PARTITION BY RANGE (placed_at);

CREATE TABLE cloud_bet_records_partitioned (
    LIKE cloud_bet_records
        INCLUDING DEFAULTS
        INCLUDING CONSTRAINTS
        INCLUDING GENERATED
        INCLUDING STORAGE
        INCLUDING COMMENTS
) PARTITION BY RANGE (placed_at);

CREATE TABLE wallet_ledger_partitioned (
    LIKE wallet_ledger
        INCLUDING DEFAULTS
        INCLUDING CONSTRAINTS
        INCLUDING GENERATED
        INCLUDING STORAGE
        INCLUDING COMMENTS
) PARTITION BY RANGE (created_at);

ALTER TABLE bet_orders_partitioned
    ADD CONSTRAINT bet_orders_partitioned_pkey PRIMARY KEY (placed_at, id),
    ADD CONSTRAINT bet_orders_partitioned_order_no_key UNIQUE (placed_at, order_no),
    ADD CONSTRAINT bet_orders_partitioned_identity_fk
        FOREIGN KEY (id) REFERENCES bet_order_identity(id) ON DELETE RESTRICT;

ALTER TABLE cloud_bet_records_partitioned
    ADD CONSTRAINT cloud_bet_records_partitioned_pkey PRIMARY KEY (placed_at, id),
    ADD CONSTRAINT cloud_bet_records_partitioned_record_no_key UNIQUE (placed_at, record_no),
    ADD CONSTRAINT cloud_bet_records_partitioned_scheme_period_key
        UNIQUE (placed_at, scheme_id, period_no),
    ADD CONSTRAINT cloud_bet_records_partitioned_identity_fk
        FOREIGN KEY (id) REFERENCES cloud_bet_record_identity(id) ON DELETE RESTRICT;

ALTER TABLE wallet_ledger_partitioned
    ADD CONSTRAINT wallet_ledger_partitioned_pkey PRIMARY KEY (created_at, id),
    ADD CONSTRAINT wallet_ledger_partitioned_ledger_no_key UNIQUE (created_at, ledger_no),
    ADD CONSTRAINT wallet_ledger_partitioned_identity_fk
        FOREIGN KEY (id) REFERENCES wallet_ledger_identity(id) ON DELETE RESTRICT;

CREATE OR REPLACE FUNCTION ensure_core_online_partitions(
    start_month DATE DEFAULT date_trunc('month', now())::date,
    months_ahead INTEGER DEFAULT 12
)
RETURNS INTEGER
LANGUAGE plpgsql
AS $$
DECLARE
    month_start DATE;
    month_end DATE;
    suffix TEXT;
    created_count INTEGER := 0;
    i INTEGER;
    table_name TEXT;
    partition_name TEXT;
BEGIN
    IF start_month <> date_trunc('month', start_month)::date THEN
        RAISE EXCEPTION 'start_month must be the first day of a month';
    END IF;
    IF months_ahead < 0 OR months_ahead > 240 THEN
        RAISE EXCEPTION 'months_ahead must be between 0 and 240';
    END IF;

    FOR i IN 0..months_ahead LOOP
        month_start := (start_month + make_interval(months => i))::date;
        month_end := (month_start + interval '1 month')::date;
        suffix := to_char(month_start, 'YYYYMM');
        FOREACH table_name IN ARRAY ARRAY[
            'bet_orders_partitioned',
            'cloud_bet_records_partitioned',
            'wallet_ledger_partitioned'
        ] LOOP
            partition_name := table_name || '_' || suffix;
            IF to_regclass(partition_name) IS NULL THEN
                EXECUTE format(
                    'CREATE TABLE %I PARTITION OF %I FOR VALUES FROM (%L) TO (%L)',
                    partition_name, table_name, month_start, month_end
                );
                created_count := created_count + 1;
            END IF;
        END LOOP;
    END LOOP;
    RETURN created_count;
END;
$$;

SELECT ensure_core_online_partitions('2020-01-01'::date, 143);

CREATE TABLE bet_orders_partitioned_default
    PARTITION OF bet_orders_partitioned DEFAULT;
CREATE TABLE cloud_bet_records_partitioned_default
    PARTITION OF cloud_bet_records_partitioned DEFAULT;
CREATE TABLE wallet_ledger_partitioned_default
    PARTITION OF wallet_ledger_partitioned DEFAULT;

CREATE INDEX bet_orders_partitioned_member_placed_idx
    ON bet_orders_partitioned (member_id, placed_at DESC, id DESC);
CREATE INDEX bet_orders_partitioned_pending_issue_idx
    ON bet_orders_partitioned (lottery_code, issue_no, placed_at)
    WHERE status = 'pending';
CREATE INDEX bet_orders_partitioned_pending_scan_idx
    ON bet_orders_partitioned (placed_at, id)
    WHERE status = 'pending';
CREATE INDEX bet_orders_partitioned_third_party_idx
    ON bet_orders_partitioned (third_party_bet_id, placed_at)
    WHERE third_party_bet_id IS NOT NULL AND third_party_bet_id <> '';

CREATE INDEX cloud_bet_records_partitioned_member_placed_idx
    ON cloud_bet_records_partitioned (member_id, sim_bet, placed_at DESC, id DESC);
CREATE INDEX cloud_bet_records_partitioned_scheme_placed_idx
    ON cloud_bet_records_partitioned (scheme_id, placed_at DESC, id DESC);
CREATE INDEX cloud_bet_records_partitioned_order_idx
    ON cloud_bet_records_partitioned (bet_order_no, placed_at)
    WHERE bet_order_no IS NOT NULL AND bet_order_no <> '';
CREATE INDEX cloud_bet_records_partitioned_strategy_pending_idx
    ON cloud_bet_records_partitioned (scheme_id, period_no, placed_at)
    WHERE third_party_bet_id IS NOT NULL AND strategy_evaluated_at IS NULL;

CREATE INDEX wallet_ledger_partitioned_member_created_idx
    ON wallet_ledger_partitioned (member_id, created_at DESC, id DESC);
CREATE INDEX wallet_ledger_partitioned_order_ref_idx
    ON wallet_ledger_partitioned (order_ref, created_at DESC, id DESC)
    WHERE order_ref IS NOT NULL;
CREATE INDEX wallet_ledger_partitioned_admin_bet_idx
    ON wallet_ledger_partitioned (created_at DESC, id DESC)
    WHERE txn_type IN ('bet_debit', 'payout');

CREATE OR REPLACE FUNCTION ensure_core_partition_identity()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    affected INTEGER;
BEGIN
    CASE TG_ARGV[0]
    WHEN 'bet_orders' THEN
        INSERT INTO bet_order_identity (id, order_no, placed_at)
        VALUES (NEW.id, NEW.order_no, NEW.placed_at)
        ON CONFLICT (id) DO UPDATE
        SET order_no = EXCLUDED.order_no,
            placed_at = EXCLUDED.placed_at
        WHERE bet_order_identity.order_no = EXCLUDED.order_no
          AND bet_order_identity.placed_at = EXCLUDED.placed_at;
    WHEN 'cloud_bet_records' THEN
        INSERT INTO cloud_bet_record_identity (id, record_no, scheme_id, period_no, placed_at)
        VALUES (NEW.id, NEW.record_no, NEW.scheme_id, NEW.period_no, NEW.placed_at)
        ON CONFLICT (id) DO UPDATE
        SET record_no = EXCLUDED.record_no,
            scheme_id = EXCLUDED.scheme_id,
            period_no = EXCLUDED.period_no,
            placed_at = EXCLUDED.placed_at
        WHERE cloud_bet_record_identity.record_no = EXCLUDED.record_no
          AND cloud_bet_record_identity.scheme_id = EXCLUDED.scheme_id
          AND cloud_bet_record_identity.period_no = EXCLUDED.period_no
          AND cloud_bet_record_identity.placed_at = EXCLUDED.placed_at;
    WHEN 'wallet_ledger' THEN
        INSERT INTO wallet_ledger_identity (id, ledger_no, created_at)
        VALUES (NEW.id, NEW.ledger_no, NEW.created_at)
        ON CONFLICT (id) DO UPDATE
        SET ledger_no = EXCLUDED.ledger_no,
            created_at = EXCLUDED.created_at
        WHERE wallet_ledger_identity.ledger_no = EXCLUDED.ledger_no
          AND wallet_ledger_identity.created_at = EXCLUDED.created_at;
    ELSE
        RAISE EXCEPTION 'unsupported core identity kind: %', TG_ARGV[0];
    END CASE;
    GET DIAGNOSTICS affected = ROW_COUNT;
    IF affected <> 1 THEN
        RAISE EXCEPTION 'immutable core identity mismatch for % id %', TG_ARGV[0], NEW.id;
    END IF;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION delete_core_partition_identity()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    CASE TG_ARGV[0]
    WHEN 'bet_orders' THEN
        DELETE FROM bet_order_identity WHERE id = OLD.id;
    WHEN 'cloud_bet_records' THEN
        DELETE FROM cloud_bet_record_identity WHERE id = OLD.id;
    WHEN 'wallet_ledger' THEN
        DELETE FROM wallet_ledger_identity WHERE id = OLD.id;
    ELSE
        RAISE EXCEPTION 'unsupported core identity kind: %', TG_ARGV[0];
    END CASE;
    RETURN OLD;
END;
$$;

CREATE TRIGGER trg_bet_orders_partitioned_identity
BEFORE INSERT OR UPDATE ON bet_orders_partitioned
FOR EACH ROW EXECUTE FUNCTION ensure_core_partition_identity('bet_orders');
CREATE TRIGGER trg_cloud_bet_records_partitioned_identity
BEFORE INSERT OR UPDATE ON cloud_bet_records_partitioned
FOR EACH ROW EXECUTE FUNCTION ensure_core_partition_identity('cloud_bet_records');
CREATE TRIGGER trg_wallet_ledger_partitioned_identity
BEFORE INSERT OR UPDATE ON wallet_ledger_partitioned
FOR EACH ROW EXECUTE FUNCTION ensure_core_partition_identity('wallet_ledger');

CREATE TRIGGER trg_bet_orders_partitioned_identity_delete
AFTER DELETE ON bet_orders_partitioned
FOR EACH ROW EXECUTE FUNCTION delete_core_partition_identity('bet_orders');
CREATE TRIGGER trg_cloud_bet_records_partitioned_identity_delete
AFTER DELETE ON cloud_bet_records_partitioned
FOR EACH ROW EXECUTE FUNCTION delete_core_partition_identity('cloud_bet_records');
CREATE TRIGGER trg_wallet_ledger_partitioned_identity_delete
AFTER DELETE ON wallet_ledger_partitioned
FOR EACH ROW EXECUTE FUNCTION delete_core_partition_identity('wallet_ledger');

CREATE OR REPLACE FUNCTION sync_core_online_partition_row()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_TABLE_NAME = 'bet_orders' THEN
        IF TG_OP = 'DELETE' THEN
            DELETE FROM bet_orders_partitioned
            WHERE placed_at = OLD.placed_at AND id = OLD.id;
            RETURN OLD;
        END IF;
        IF TG_OP = 'UPDATE' AND (OLD.placed_at, OLD.id) <> (NEW.placed_at, NEW.id) THEN
            DELETE FROM bet_orders_partitioned
            WHERE placed_at = OLD.placed_at AND id = OLD.id;
        END IF;
        DELETE FROM bet_orders_partitioned
        WHERE placed_at = NEW.placed_at AND id = NEW.id;
        INSERT INTO bet_orders_partitioned SELECT NEW.*;
    ELSIF TG_TABLE_NAME = 'cloud_bet_records' THEN
        IF TG_OP = 'DELETE' THEN
            DELETE FROM cloud_bet_records_partitioned
            WHERE placed_at = OLD.placed_at AND id = OLD.id;
            RETURN OLD;
        END IF;
        IF TG_OP = 'UPDATE' AND (OLD.placed_at, OLD.id) <> (NEW.placed_at, NEW.id) THEN
            DELETE FROM cloud_bet_records_partitioned
            WHERE placed_at = OLD.placed_at AND id = OLD.id;
        END IF;
        DELETE FROM cloud_bet_records_partitioned
        WHERE placed_at = NEW.placed_at AND id = NEW.id;
        INSERT INTO cloud_bet_records_partitioned SELECT NEW.*;
    ELSIF TG_TABLE_NAME = 'wallet_ledger' THEN
        IF TG_OP = 'DELETE' THEN
            DELETE FROM wallet_ledger_partitioned
            WHERE created_at = OLD.created_at AND id = OLD.id;
            RETURN OLD;
        END IF;
        IF TG_OP = 'UPDATE' AND (OLD.created_at, OLD.id) <> (NEW.created_at, NEW.id) THEN
            DELETE FROM wallet_ledger_partitioned
            WHERE created_at = OLD.created_at AND id = OLD.id;
        END IF;
        DELETE FROM wallet_ledger_partitioned
        WHERE created_at = NEW.created_at AND id = NEW.id;
        INSERT INTO wallet_ledger_partitioned SELECT NEW.*;
    ELSE
        RAISE EXCEPTION 'unsupported core sync table: %', TG_TABLE_NAME;
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_bet_orders_forward_partition_sync
AFTER INSERT OR UPDATE OR DELETE ON bet_orders
FOR EACH ROW EXECUTE FUNCTION sync_core_online_partition_row();
CREATE TRIGGER trg_cloud_bet_records_forward_partition_sync
AFTER INSERT OR UPDATE OR DELETE ON cloud_bet_records
FOR EACH ROW EXECUTE FUNCTION sync_core_online_partition_row();
CREATE TRIGGER trg_wallet_ledger_forward_partition_sync
AFTER INSERT OR UPDATE OR DELETE ON wallet_ledger
FOR EACH ROW EXECUTE FUNCTION sync_core_online_partition_row();

CREATE OR REPLACE FUNCTION backfill_core_online_partitions(row_limit INTEGER DEFAULT 5000)
RETURNS JSONB
LANGUAGE plpgsql
AS $$
DECLARE
    bet_count INTEGER;
    cloud_count INTEGER;
    ledger_count INTEGER;
BEGIN
    IF row_limit < 1 OR row_limit > 100000 THEN
        RAISE EXCEPTION 'row_limit must be between 1 and 100000';
    END IF;

    INSERT INTO bet_orders_partitioned
    SELECT o.*
    FROM bet_orders o
    WHERE NOT EXISTS (
        SELECT 1 FROM bet_orders_partitioned p
        WHERE p.placed_at = o.placed_at AND p.id = o.id
    )
    ORDER BY o.placed_at, o.id
    LIMIT row_limit
    ON CONFLICT (placed_at, id) DO NOTHING;
    GET DIAGNOSTICS bet_count = ROW_COUNT;

    INSERT INTO cloud_bet_records_partitioned
    SELECT c.*
    FROM cloud_bet_records c
    WHERE NOT EXISTS (
        SELECT 1 FROM cloud_bet_records_partitioned p
        WHERE p.placed_at = c.placed_at AND p.id = c.id
    )
    ORDER BY c.placed_at, c.id
    LIMIT row_limit
    ON CONFLICT (placed_at, id) DO NOTHING;
    GET DIAGNOSTICS cloud_count = ROW_COUNT;

    INSERT INTO wallet_ledger_partitioned
    SELECT l.*
    FROM wallet_ledger l
    WHERE NOT EXISTS (
        SELECT 1 FROM wallet_ledger_partitioned p
        WHERE p.created_at = l.created_at AND p.id = l.id
    )
    ORDER BY l.created_at, l.id
    LIMIT row_limit
    ON CONFLICT (created_at, id) DO NOTHING;
    GET DIAGNOSTICS ledger_count = ROW_COUNT;

    RETURN jsonb_build_object(
        'betOrders', bet_count,
        'cloudBetRecords', cloud_count,
        'walletLedger', ledger_count,
        'total', bet_count + cloud_count + ledger_count
    );
END;
$$;

CREATE OR REPLACE FUNCTION validate_core_online_partitions()
RETURNS JSONB
LANGUAGE plpgsql
AS $$
DECLARE
    bet_source BIGINT;
    bet_target BIGINT;
    bet_missing BIGINT;
    bet_extra BIGINT;
    bet_amount_delta NUMERIC;
    cloud_source BIGINT;
    cloud_target BIGINT;
    cloud_missing BIGINT;
    cloud_extra BIGINT;
    cloud_amount_delta NUMERIC;
    ledger_source BIGINT;
    ledger_target BIGINT;
    ledger_missing BIGINT;
    ledger_extra BIGINT;
    ledger_delta_delta NUMERIC;
    valid BOOLEAN;
    result JSONB;
BEGIN
    SELECT count(*), COALESCE(sum(amount), 0) INTO bet_source, bet_amount_delta FROM bet_orders;
    SELECT count(*), bet_amount_delta - COALESCE(sum(amount), 0)
      INTO bet_target, bet_amount_delta FROM bet_orders_partitioned;
    SELECT count(*) INTO bet_missing FROM bet_orders o
    WHERE NOT EXISTS (SELECT 1 FROM bet_orders_partitioned p WHERE p.placed_at = o.placed_at AND p.id = o.id);
    SELECT count(*) INTO bet_extra FROM bet_orders_partitioned p
    WHERE NOT EXISTS (SELECT 1 FROM bet_orders o WHERE o.placed_at = p.placed_at AND o.id = p.id);

    SELECT count(*), COALESCE(sum(amount), 0) INTO cloud_source, cloud_amount_delta FROM cloud_bet_records;
    SELECT count(*), cloud_amount_delta - COALESCE(sum(amount), 0)
      INTO cloud_target, cloud_amount_delta FROM cloud_bet_records_partitioned;
    SELECT count(*) INTO cloud_missing FROM cloud_bet_records o
    WHERE NOT EXISTS (SELECT 1 FROM cloud_bet_records_partitioned p WHERE p.placed_at = o.placed_at AND p.id = o.id);
    SELECT count(*) INTO cloud_extra FROM cloud_bet_records_partitioned p
    WHERE NOT EXISTS (SELECT 1 FROM cloud_bet_records o WHERE o.placed_at = p.placed_at AND o.id = p.id);

    SELECT count(*), COALESCE(sum(delta_amount), 0) INTO ledger_source, ledger_delta_delta FROM wallet_ledger;
    SELECT count(*), ledger_delta_delta - COALESCE(sum(delta_amount), 0)
      INTO ledger_target, ledger_delta_delta FROM wallet_ledger_partitioned;
    SELECT count(*) INTO ledger_missing FROM wallet_ledger o
    WHERE NOT EXISTS (SELECT 1 FROM wallet_ledger_partitioned p WHERE p.created_at = o.created_at AND p.id = o.id);
    SELECT count(*) INTO ledger_extra FROM wallet_ledger_partitioned p
    WHERE NOT EXISTS (SELECT 1 FROM wallet_ledger o WHERE o.created_at = p.created_at AND o.id = p.id);

    valid := bet_missing = 0 AND bet_extra = 0 AND bet_amount_delta = 0
        AND cloud_missing = 0 AND cloud_extra = 0 AND cloud_amount_delta = 0
        AND ledger_missing = 0 AND ledger_extra = 0 AND ledger_delta_delta = 0;

    result := jsonb_build_object(
        'valid', valid,
        'betOrders', jsonb_build_object(
            'source', bet_source, 'target', bet_target, 'missing', bet_missing,
            'extra', bet_extra, 'amountDelta', bet_amount_delta
        ),
        'cloudBetRecords', jsonb_build_object(
            'source', cloud_source, 'target', cloud_target, 'missing', cloud_missing,
            'extra', cloud_extra, 'amountDelta', cloud_amount_delta
        ),
        'walletLedger', jsonb_build_object(
            'source', ledger_source, 'target', ledger_target, 'missing', ledger_missing,
            'extra', ledger_extra, 'amountDelta', ledger_delta_delta
        )
    );

    UPDATE core_partition_migration_state
    SET phase = CASE WHEN valid THEN 'validated' ELSE 'mirroring' END,
        last_validation = result,
        last_validated_at = now(),
        updated_at = now()
    WHERE id = 1 AND phase IN ('mirroring', 'validated');
    RETURN result;
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM core_partition_migration_state
        WHERE id = 1 AND phase IN ('cutover', 'rollback_ready')
    ) THEN
        RAISE EXCEPTION 'rollback the core online partition cutover before migrating down';
    END IF;
END;
$$;

DROP TRIGGER IF EXISTS trg_wallet_ledger_forward_partition_sync ON wallet_ledger;
DROP TRIGGER IF EXISTS trg_cloud_bet_records_forward_partition_sync ON cloud_bet_records;
DROP TRIGGER IF EXISTS trg_bet_orders_forward_partition_sync ON bet_orders;
DROP FUNCTION IF EXISTS validate_core_online_partitions();
DROP FUNCTION IF EXISTS backfill_core_online_partitions(INTEGER);
DROP FUNCTION IF EXISTS sync_core_online_partition_row();
DROP FUNCTION IF EXISTS delete_core_partition_identity();
DROP FUNCTION IF EXISTS ensure_core_partition_identity();
DROP FUNCTION IF EXISTS ensure_core_online_partitions(DATE, INTEGER);
DROP TABLE IF EXISTS wallet_ledger_partitioned CASCADE;
DROP TABLE IF EXISTS cloud_bet_records_partitioned CASCADE;
DROP TABLE IF EXISTS bet_orders_partitioned CASCADE;
DROP TABLE IF EXISTS wallet_ledger_identity;
DROP TABLE IF EXISTS cloud_bet_record_identity;
DROP TABLE IF EXISTS bet_order_identity;
DROP TABLE IF EXISTS core_partition_migration_state;
-- +goose StatementEnd
