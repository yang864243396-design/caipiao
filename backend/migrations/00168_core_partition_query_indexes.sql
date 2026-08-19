-- +goose Up
-- +goose StatementBegin
DO $$
DECLARE
    bet_table TEXT;
    cloud_table TEXT;
    ledger_table TEXT;
BEGIN
    SELECT CASE WHEN c.relkind = 'p' THEN 'bet_orders' ELSE 'bet_orders_partitioned' END
    INTO bet_table FROM pg_class c WHERE c.oid = to_regclass('bet_orders');
    SELECT CASE WHEN c.relkind = 'p' THEN 'cloud_bet_records' ELSE 'cloud_bet_records_partitioned' END
    INTO cloud_table FROM pg_class c WHERE c.oid = to_regclass('cloud_bet_records');
    SELECT CASE WHEN c.relkind = 'p' THEN 'wallet_ledger' ELSE 'wallet_ledger_partitioned' END
    INTO ledger_table FROM pg_class c WHERE c.oid = to_regclass('wallet_ledger');

    IF bet_table IS NULL OR cloud_table IS NULL OR ledger_table IS NULL THEN
        RAISE EXCEPTION 'core partition index target is missing';
    END IF;

    EXECUTE format('CREATE INDEX IF NOT EXISTS bet_orders_partitioned_id_idx ON %I (id)', bet_table);
    EXECUTE format('CREATE INDEX IF NOT EXISTS bet_orders_partitioned_order_no_idx ON %I (order_no)', bet_table);
    EXECUTE format('CREATE INDEX IF NOT EXISTS bet_orders_partitioned_member_status_placed_idx ON %I (member_id, status, placed_at DESC, id DESC)', bet_table);
    EXECUTE format('CREATE INDEX IF NOT EXISTS bet_orders_partitioned_member_category_placed_idx ON %I (member_id, lottery_category, placed_at DESC, id DESC)', bet_table);
    EXECUTE format('CREATE INDEX IF NOT EXISTS bet_orders_partitioned_pending_member_idx ON %I (member_id, placed_at DESC, id DESC) WHERE status = ''pending''', bet_table);
    EXECUTE format('CREATE INDEX IF NOT EXISTS bet_orders_partitioned_guaji_account_idx ON %I (guaji_account_id, placed_at DESC, id DESC) WHERE guaji_account_id IS NOT NULL', bet_table);
    EXECUTE format('CREATE INDEX IF NOT EXISTS bet_orders_partitioned_member_third_party_idx ON %I (member_id, third_party_bet_id) WHERE third_party_bet_id IS NOT NULL AND third_party_bet_id <> ''''', bet_table);

    EXECUTE format('CREATE INDEX IF NOT EXISTS cloud_bet_records_partitioned_id_idx ON %I (id)', cloud_table);
    EXECUTE format('CREATE INDEX IF NOT EXISTS cloud_bet_records_partitioned_record_no_idx ON %I (record_no)', cloud_table);
    EXECUTE format('CREATE INDEX IF NOT EXISTS cloud_bet_records_partitioned_scheme_period_idx ON %I (scheme_id, period_no)', cloud_table);
    EXECUTE format('CREATE INDEX IF NOT EXISTS cloud_bet_records_partitioned_member_scheme_placed_idx ON %I (member_id, scheme_id, sim_bet, placed_at DESC, id DESC)', cloud_table);
    EXECUTE format('CREATE INDEX IF NOT EXISTS cloud_bet_records_partitioned_member_guaji_placed_idx ON %I (member_id, guaji_account_id, placed_at DESC, id DESC) WHERE guaji_account_id IS NOT NULL', cloud_table);
    EXECUTE format('CREATE INDEX IF NOT EXISTS cloud_bet_records_partitioned_third_party_idx ON %I (third_party_bet_id) WHERE third_party_bet_id IS NOT NULL AND third_party_bet_id <> ''''', cloud_table);
    EXECUTE format('CREATE INDEX IF NOT EXISTS cloud_bet_records_partitioned_member_definition_idx ON %I (member_id, definition_id, sim_bet, placed_at DESC, id DESC) WHERE definition_id <> ''''', cloud_table);
    EXECUTE format('CREATE INDEX IF NOT EXISTS cloud_bet_records_partitioned_member_lottery_idx ON %I (member_id, lottery_code, sim_bet, placed_at DESC, id DESC) WHERE lottery_code <> ''''', cloud_table);
    EXECUTE format('CREATE INDEX IF NOT EXISTS cloud_bet_records_partitioned_member_order_idx ON %I (member_id, bet_order_no) INCLUDE (scheme_name) WHERE bet_order_no IS NOT NULL AND bet_order_no <> ''''', cloud_table);
    EXECUTE format('CREATE INDEX IF NOT EXISTS cloud_bet_records_partitioned_member_guaji_lookup_idx ON %I (member_id, guaji_account_id, placed_at DESC) INCLUDE (amount, scheme_name)', cloud_table);
    EXECUTE format('CREATE INDEX IF NOT EXISTS cloud_bet_records_partitioned_scheme_stats_idx ON %I (scheme_id, placed_at DESC) INCLUDE (status, currency)', cloud_table);

    EXECUTE format('CREATE INDEX IF NOT EXISTS wallet_ledger_partitioned_id_idx ON %I (id)', ledger_table);
    EXECUTE format('CREATE INDEX IF NOT EXISTS wallet_ledger_partitioned_ledger_no_idx ON %I (ledger_no)', ledger_table);
    EXECUTE format('CREATE INDEX IF NOT EXISTS wallet_ledger_partitioned_member_type_created_idx ON %I (member_id, txn_type, created_at DESC, id DESC)', ledger_table);
    EXECUTE format('CREATE INDEX IF NOT EXISTS wallet_ledger_partitioned_member_guaji_idx ON %I (member_id, guaji_account_id, created_at DESC, id DESC) WHERE guaji_account_id IS NOT NULL', ledger_table);
    EXECUTE format('CREATE INDEX IF NOT EXISTS wallet_ledger_partitioned_member_guaji_lottery_idx ON %I (member_id, guaji_account_id, lottery_code, created_at DESC, id DESC) WHERE txn_type IN (''bet_debit'', ''payout'')', ledger_table);
    EXECUTE format('CREATE INDEX IF NOT EXISTS wallet_ledger_partitioned_scheme_audit_idx ON %I (order_ref, created_at DESC, id DESC) WHERE txn_type IN (''bet_debit'', ''payout'') AND order_ref IS NOT NULL', ledger_table);
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS wallet_ledger_partitioned_scheme_audit_idx;
DROP INDEX IF EXISTS wallet_ledger_partitioned_member_guaji_lottery_idx;
DROP INDEX IF EXISTS wallet_ledger_partitioned_member_guaji_idx;
DROP INDEX IF EXISTS wallet_ledger_partitioned_member_type_created_idx;
DROP INDEX IF EXISTS wallet_ledger_partitioned_ledger_no_idx;
DROP INDEX IF EXISTS wallet_ledger_partitioned_id_idx;
DROP INDEX IF EXISTS cloud_bet_records_partitioned_scheme_stats_idx;
DROP INDEX IF EXISTS cloud_bet_records_partitioned_member_guaji_lookup_idx;
DROP INDEX IF EXISTS cloud_bet_records_partitioned_member_order_idx;
DROP INDEX IF EXISTS cloud_bet_records_partitioned_member_lottery_idx;
DROP INDEX IF EXISTS cloud_bet_records_partitioned_member_definition_idx;
DROP INDEX IF EXISTS cloud_bet_records_partitioned_third_party_idx;
DROP INDEX IF EXISTS cloud_bet_records_partitioned_member_guaji_placed_idx;
DROP INDEX IF EXISTS cloud_bet_records_partitioned_member_scheme_placed_idx;
DROP INDEX IF EXISTS cloud_bet_records_partitioned_scheme_period_idx;
DROP INDEX IF EXISTS cloud_bet_records_partitioned_record_no_idx;
DROP INDEX IF EXISTS cloud_bet_records_partitioned_id_idx;
DROP INDEX IF EXISTS bet_orders_partitioned_member_third_party_idx;
DROP INDEX IF EXISTS bet_orders_partitioned_guaji_account_idx;
DROP INDEX IF EXISTS bet_orders_partitioned_pending_member_idx;
DROP INDEX IF EXISTS bet_orders_partitioned_member_category_placed_idx;
DROP INDEX IF EXISTS bet_orders_partitioned_member_status_placed_idx;
DROP INDEX IF EXISTS bet_orders_partitioned_order_no_idx;
DROP INDEX IF EXISTS bet_orders_partitioned_id_idx;
-- +goose StatementEnd
