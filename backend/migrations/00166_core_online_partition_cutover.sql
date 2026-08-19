-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION sync_core_online_legacy_row()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_ARGV[0] = 'bet_orders' THEN
        IF TG_OP = 'DELETE' THEN
            DELETE FROM bet_orders_legacy_unpartitioned WHERE id = OLD.id;
            RETURN OLD;
        END IF;
        DELETE FROM bet_orders_legacy_unpartitioned WHERE id = NEW.id;
        INSERT INTO bet_orders_legacy_unpartitioned SELECT NEW.*;
    ELSIF TG_ARGV[0] = 'cloud_bet_records' THEN
        IF TG_OP = 'DELETE' THEN
            DELETE FROM cloud_bet_records_legacy_unpartitioned WHERE id = OLD.id;
            RETURN OLD;
        END IF;
        DELETE FROM cloud_bet_records_legacy_unpartitioned WHERE id = NEW.id;
        INSERT INTO cloud_bet_records_legacy_unpartitioned SELECT NEW.*;
    ELSIF TG_ARGV[0] = 'wallet_ledger' THEN
        IF TG_OP <> 'INSERT' THEN
            RAISE EXCEPTION 'wallet_ledger is append-only';
        END IF;
        INSERT INTO wallet_ledger_legacy_unpartitioned SELECT NEW.*
        ON CONFLICT (id) DO NOTHING;
    ELSE
        RAISE EXCEPTION 'unsupported reverse sync table: %', TG_ARGV[0];
    END IF;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION validate_core_online_partition_cutover()
RETURNS JSONB
LANGUAGE plpgsql
AS $$
DECLARE
    bet_active BIGINT;
    bet_legacy BIGINT;
    bet_missing BIGINT;
    bet_amount_delta NUMERIC;
    cloud_active BIGINT;
    cloud_legacy BIGINT;
    cloud_missing BIGINT;
    cloud_amount_delta NUMERIC;
    ledger_active BIGINT;
    ledger_legacy BIGINT;
    ledger_missing BIGINT;
    ledger_amount_delta NUMERIC;
    valid BOOLEAN;
    result JSONB;
BEGIN
    SELECT count(*), COALESCE(sum(amount), 0) INTO bet_active, bet_amount_delta FROM bet_orders;
    SELECT count(*), bet_amount_delta - COALESCE(sum(amount), 0)
      INTO bet_legacy, bet_amount_delta FROM bet_orders_legacy_unpartitioned;
    SELECT count(*) INTO bet_missing FROM bet_orders o
    WHERE NOT EXISTS (SELECT 1 FROM bet_orders_legacy_unpartitioned l WHERE l.id = o.id);

    SELECT count(*), COALESCE(sum(amount), 0) INTO cloud_active, cloud_amount_delta FROM cloud_bet_records;
    SELECT count(*), cloud_amount_delta - COALESCE(sum(amount), 0)
      INTO cloud_legacy, cloud_amount_delta FROM cloud_bet_records_legacy_unpartitioned;
    SELECT count(*) INTO cloud_missing FROM cloud_bet_records o
    WHERE NOT EXISTS (SELECT 1 FROM cloud_bet_records_legacy_unpartitioned l WHERE l.id = o.id);

    SELECT count(*), COALESCE(sum(delta_amount), 0) INTO ledger_active, ledger_amount_delta FROM wallet_ledger;
    SELECT count(*), ledger_amount_delta - COALESCE(sum(delta_amount), 0)
      INTO ledger_legacy, ledger_amount_delta FROM wallet_ledger_legacy_unpartitioned;
    SELECT count(*) INTO ledger_missing FROM wallet_ledger o
    WHERE NOT EXISTS (SELECT 1 FROM wallet_ledger_legacy_unpartitioned l WHERE l.id = o.id);

    valid := bet_active = bet_legacy AND bet_missing = 0 AND bet_amount_delta = 0
        AND cloud_active = cloud_legacy AND cloud_missing = 0 AND cloud_amount_delta = 0
        AND ledger_active = ledger_legacy AND ledger_missing = 0 AND ledger_amount_delta = 0;

    result := jsonb_build_object(
        'valid', valid,
        'betOrders', jsonb_build_object(
            'active', bet_active, 'legacy', bet_legacy, 'missing', bet_missing,
            'amountDelta', bet_amount_delta
        ),
        'cloudBetRecords', jsonb_build_object(
            'active', cloud_active, 'legacy', cloud_legacy, 'missing', cloud_missing,
            'amountDelta', cloud_amount_delta
        ),
        'walletLedger', jsonb_build_object(
            'active', ledger_active, 'legacy', ledger_legacy, 'missing', ledger_missing,
            'amountDelta', ledger_amount_delta
        )
    );

    UPDATE core_partition_migration_state
    SET phase = CASE WHEN valid THEN 'rollback_ready' ELSE 'cutover' END,
        last_validation = result,
        last_validated_at = now(),
        updated_at = now()
    WHERE id = 1 AND phase IN ('cutover', 'rollback_ready');
    RETURN result;
END;
$$;

CREATE OR REPLACE FUNCTION cutover_core_online_partitions()
RETURNS JSONB
LANGUAGE plpgsql
AS $$
DECLARE
    phase_value TEXT;
    batch_result JSONB;
    validation JSONB;
BEGIN
    SELECT phase INTO phase_value
    FROM core_partition_migration_state
    WHERE id = 1
    FOR UPDATE;
    IF phase_value <> 'validated' THEN
        RAISE EXCEPTION 'core partition cutover requires validated phase, got %', phase_value;
    END IF;

    LOCK TABLE bet_orders IN ACCESS EXCLUSIVE MODE;
    LOCK TABLE cloud_bet_records IN ACCESS EXCLUSIVE MODE;
    LOCK TABLE wallet_ledger IN ACCESS EXCLUSIVE MODE;

    LOOP
        batch_result := backfill_core_online_partitions(100000);
        EXIT WHEN (batch_result->>'total')::integer = 0;
    END LOOP;
    validation := validate_core_online_partitions();
    IF NOT COALESCE((validation->>'valid')::boolean, false) THEN
        RAISE EXCEPTION 'core partition validation failed: %', validation;
    END IF;

    DROP TRIGGER trg_bet_orders_forward_partition_sync ON bet_orders;
    DROP TRIGGER trg_cloud_bet_records_forward_partition_sync ON cloud_bet_records;
    DROP TRIGGER trg_wallet_ledger_forward_partition_sync ON wallet_ledger;

    ALTER TABLE scheme_period_decisions
        DROP CONSTRAINT IF EXISTS scheme_period_decisions_source_bet_record_id_fkey;
    ALTER TABLE scheme_period_decisions
        ADD CONSTRAINT scheme_period_decisions_source_bet_record_id_fkey
        FOREIGN KEY (source_bet_record_id)
        REFERENCES cloud_bet_record_identity(id) ON DELETE SET NULL;

    ALTER TABLE scheme_bet_outbox
        DROP CONSTRAINT IF EXISTS scheme_bet_outbox_local_cloud_record_id_fkey;
    ALTER TABLE scheme_bet_outbox
        ADD CONSTRAINT scheme_bet_outbox_local_cloud_record_id_fkey
        FOREIGN KEY (local_cloud_record_id)
        REFERENCES cloud_bet_record_identity(id) ON DELETE SET NULL;

    ALTER TABLE bet_orders RENAME TO bet_orders_legacy_unpartitioned;
    ALTER TABLE bet_orders_partitioned RENAME TO bet_orders;
    ALTER TABLE cloud_bet_records RENAME TO cloud_bet_records_legacy_unpartitioned;
    ALTER TABLE cloud_bet_records_partitioned RENAME TO cloud_bet_records;
    ALTER TABLE wallet_ledger RENAME TO wallet_ledger_legacy_unpartitioned;
    ALTER TABLE wallet_ledger_partitioned RENAME TO wallet_ledger;

    ALTER SEQUENCE bet_orders_id_seq OWNED BY bet_orders.id;
    ALTER SEQUENCE cloud_bet_records_id_seq OWNED BY cloud_bet_records.id;
    ALTER SEQUENCE wallet_ledger_id_seq OWNED BY wallet_ledger.id;

    CREATE TRIGGER trg_bet_orders_reverse_sync
    AFTER INSERT OR UPDATE OR DELETE ON bet_orders
    FOR EACH ROW EXECUTE FUNCTION sync_core_online_legacy_row('bet_orders');
    CREATE TRIGGER trg_cloud_bet_records_reverse_sync
    AFTER INSERT OR UPDATE OR DELETE ON cloud_bet_records
    FOR EACH ROW EXECUTE FUNCTION sync_core_online_legacy_row('cloud_bet_records');
    CREATE TRIGGER trg_wallet_ledger_reverse_sync
    AFTER INSERT ON wallet_ledger
    FOR EACH ROW EXECUTE FUNCTION sync_core_online_legacy_row('wallet_ledger');

    CREATE TRIGGER trg_wallet_ledger_append_only
    BEFORE UPDATE OR DELETE ON wallet_ledger
    FOR EACH ROW EXECUTE FUNCTION reject_wallet_ledger_mutation();

    UPDATE core_partition_migration_state
    SET phase = 'cutover',
        forward_sync = false,
        reverse_sync = true,
        restart_required = true,
        cutover_at = now(),
        updated_at = now()
    WHERE id = 1;

    RETURN jsonb_build_object(
        'phase', 'cutover',
        'restart_required', true,
        'reverse_sync', true
    );
END;
$$;

CREATE OR REPLACE FUNCTION rollback_core_online_partitions()
RETURNS JSONB
LANGUAGE plpgsql
AS $$
DECLARE
    phase_value TEXT;
    validation JSONB;
BEGIN
    SELECT phase INTO phase_value
    FROM core_partition_migration_state
    WHERE id = 1
    FOR UPDATE;
    IF phase_value NOT IN ('cutover', 'rollback_ready') THEN
        RAISE EXCEPTION 'core partition rollback requires cutover phase, got %', phase_value;
    END IF;

    LOCK TABLE bet_orders IN ACCESS EXCLUSIVE MODE;
    LOCK TABLE cloud_bet_records IN ACCESS EXCLUSIVE MODE;
    LOCK TABLE wallet_ledger IN ACCESS EXCLUSIVE MODE;
    LOCK TABLE bet_orders_legacy_unpartitioned IN ACCESS EXCLUSIVE MODE;
    LOCK TABLE cloud_bet_records_legacy_unpartitioned IN ACCESS EXCLUSIVE MODE;
    LOCK TABLE wallet_ledger_legacy_unpartitioned IN ACCESS EXCLUSIVE MODE;

    validation := validate_core_online_partition_cutover();
    IF NOT COALESCE((validation->>'valid')::boolean, false) THEN
        RAISE EXCEPTION 'core partition rollback validation failed: %', validation;
    END IF;

    DROP TRIGGER trg_bet_orders_reverse_sync ON bet_orders;
    DROP TRIGGER trg_cloud_bet_records_reverse_sync ON cloud_bet_records;
    DROP TRIGGER trg_wallet_ledger_reverse_sync ON wallet_ledger;
    DROP TRIGGER trg_wallet_ledger_append_only ON wallet_ledger;

    ALTER TABLE scheme_period_decisions
        DROP CONSTRAINT IF EXISTS scheme_period_decisions_source_bet_record_id_fkey;
    ALTER TABLE scheme_bet_outbox
        DROP CONSTRAINT IF EXISTS scheme_bet_outbox_local_cloud_record_id_fkey;

    ALTER TABLE bet_orders RENAME TO bet_orders_partitioned;
    ALTER TABLE bet_orders_legacy_unpartitioned RENAME TO bet_orders;
    ALTER TABLE cloud_bet_records RENAME TO cloud_bet_records_partitioned;
    ALTER TABLE cloud_bet_records_legacy_unpartitioned RENAME TO cloud_bet_records;
    ALTER TABLE wallet_ledger RENAME TO wallet_ledger_partitioned;
    ALTER TABLE wallet_ledger_legacy_unpartitioned RENAME TO wallet_ledger;

    ALTER TABLE scheme_period_decisions
        ADD CONSTRAINT scheme_period_decisions_source_bet_record_id_fkey
        FOREIGN KEY (source_bet_record_id)
        REFERENCES cloud_bet_records(id) ON DELETE SET NULL;
    ALTER TABLE scheme_bet_outbox
        ADD CONSTRAINT scheme_bet_outbox_local_cloud_record_id_fkey
        FOREIGN KEY (local_cloud_record_id)
        REFERENCES cloud_bet_records(id) ON DELETE SET NULL;

    ALTER SEQUENCE bet_orders_id_seq OWNED BY bet_orders.id;
    ALTER SEQUENCE cloud_bet_records_id_seq OWNED BY cloud_bet_records.id;
    ALTER SEQUENCE wallet_ledger_id_seq OWNED BY wallet_ledger.id;

    CREATE TRIGGER trg_bet_orders_forward_partition_sync
    AFTER INSERT OR UPDATE OR DELETE ON bet_orders
    FOR EACH ROW EXECUTE FUNCTION sync_core_online_partition_row();
    CREATE TRIGGER trg_cloud_bet_records_forward_partition_sync
    AFTER INSERT OR UPDATE OR DELETE ON cloud_bet_records
    FOR EACH ROW EXECUTE FUNCTION sync_core_online_partition_row();
    CREATE TRIGGER trg_wallet_ledger_forward_partition_sync
    AFTER INSERT OR UPDATE OR DELETE ON wallet_ledger
    FOR EACH ROW EXECUTE FUNCTION sync_core_online_partition_row();

    UPDATE core_partition_migration_state
    SET phase = 'mirroring',
        forward_sync = true,
        reverse_sync = false,
        restart_required = true,
        rollback_at = now(),
        updated_at = now()
    WHERE id = 1;

    RETURN jsonb_build_object(
        'phase', 'mirroring',
        'restart_required', true,
        'forward_sync', true
    );
END;
$$;

CREATE OR REPLACE FUNCTION acknowledge_core_partition_restart()
RETURNS VOID
LANGUAGE sql
AS $$
    UPDATE core_partition_migration_state
    SET restart_required = false, updated_at = now()
    WHERE id = 1;
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
DECLARE
    phase_value TEXT;
BEGIN
    SELECT phase INTO phase_value FROM core_partition_migration_state WHERE id = 1;
    IF phase_value IN ('cutover', 'rollback_ready') THEN
        RAISE EXCEPTION 'rollback the core online partition cutover before migrating down';
    END IF;
END;
$$;
DROP FUNCTION IF EXISTS acknowledge_core_partition_restart();
DROP FUNCTION IF EXISTS rollback_core_online_partitions();
DROP FUNCTION IF EXISTS cutover_core_online_partitions();
DROP FUNCTION IF EXISTS validate_core_online_partition_cutover();
DROP FUNCTION IF EXISTS sync_core_online_legacy_row();
-- +goose StatementEnd
