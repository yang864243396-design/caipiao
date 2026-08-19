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

DO $$
DECLARE
    phase_value TEXT;
BEGIN
    SELECT phase INTO phase_value FROM core_partition_migration_state WHERE id = 1;
    IF phase_value IN ('cutover', 'rollback_ready') THEN
        DROP TRIGGER IF EXISTS trg_bet_orders_reverse_sync ON bet_orders;
        DROP TRIGGER IF EXISTS trg_cloud_bet_records_reverse_sync ON cloud_bet_records;
        DROP TRIGGER IF EXISTS trg_wallet_ledger_reverse_sync ON wallet_ledger;

        CREATE TRIGGER trg_bet_orders_reverse_sync
        AFTER INSERT OR UPDATE OR DELETE ON bet_orders
        FOR EACH ROW EXECUTE FUNCTION sync_core_online_legacy_row('bet_orders');
        CREATE TRIGGER trg_cloud_bet_records_reverse_sync
        AFTER INSERT OR UPDATE OR DELETE ON cloud_bet_records
        FOR EACH ROW EXECUTE FUNCTION sync_core_online_legacy_row('cloud_bet_records');
        CREATE TRIGGER trg_wallet_ledger_reverse_sync
        AFTER INSERT ON wallet_ledger
        FOR EACH ROW EXECUTE FUNCTION sync_core_online_legacy_row('wallet_ledger');
    END IF;
END;
$$;
-- +goose StatementEnd

-- +goose Down
SELECT 1;
