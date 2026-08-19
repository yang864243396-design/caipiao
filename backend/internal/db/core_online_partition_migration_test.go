package db

import (
	"os"
	"strings"
	"testing"
)

func TestCoreOnlinePartitionPrepareMigration(t *testing.T) {
	sql := readMigration(t, "../../migrations/00165_core_online_partition_prepare.sql")
	for _, want := range []string{
		"bet_order_identity",
		"cloud_bet_record_identity",
		"wallet_ledger_identity",
		"bet_orders_partitioned",
		"cloud_bet_records_partitioned",
		"wallet_ledger_partitioned",
		"PARTITION BY RANGE (placed_at)",
		"PARTITION BY RANGE (created_at)",
		"ensure_core_online_partitions",
		"sync_core_online_partition_row",
		"backfill_core_online_partitions",
		"validate_core_online_partitions",
		"core_partition_migration_state",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("prepare migration missing %q", want)
		}
	}
}

func TestCoreOnlinePartitionCutoverMigration(t *testing.T) {
	sql := readMigration(t, "../../migrations/00166_core_online_partition_cutover.sql")
	for _, want := range []string{
		"cutover_core_online_partitions",
		"rollback_core_online_partitions",
		"sync_core_online_legacy_row",
		"reverse_sync",
		"restart_required",
		"scheme_period_decisions_source_bet_record_id_fkey",
		"scheme_bet_outbox_local_cloud_record_id_fkey",
		"LOCK TABLE bet_orders",
		"LOCK TABLE cloud_bet_records",
		"LOCK TABLE wallet_ledger",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("cutover migration missing %q", want)
		}
	}
}

func TestCoreOnlinePartitionReverseSyncFixMigration(t *testing.T) {
	sql := readMigration(t, "../../migrations/00167_core_partition_reverse_sync_fix.sql")
	for _, want := range []string{
		"TG_ARGV[0]",
		"sync_core_online_legacy_row('bet_orders')",
		"sync_core_online_legacy_row('cloud_bet_records')",
		"sync_core_online_legacy_row('wallet_ledger')",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("reverse sync fix migration missing %q", want)
		}
	}
}

func TestCoreOnlinePartitionQueryIndexesMigration(t *testing.T) {
	sql := readMigration(t, "../../migrations/00168_core_partition_query_indexes.sql")
	for _, want := range []string{
		"bet_orders_partitioned_id_idx",
		"bet_orders_partitioned_order_no_idx",
		"bet_orders_partitioned_member_status_placed_idx",
		"bet_orders_partitioned_member_category_placed_idx",
		"bet_orders_partitioned_pending_member_idx",
		"bet_orders_partitioned_guaji_account_idx",
		"bet_orders_partitioned_member_third_party_idx",
		"cloud_bet_records_partitioned_id_idx",
		"cloud_bet_records_partitioned_record_no_idx",
		"cloud_bet_records_partitioned_scheme_period_idx",
		"cloud_bet_records_partitioned_member_scheme_placed_idx",
		"cloud_bet_records_partitioned_member_guaji_placed_idx",
		"cloud_bet_records_partitioned_member_definition_idx",
		"cloud_bet_records_partitioned_member_lottery_idx",
		"cloud_bet_records_partitioned_member_order_idx",
		"cloud_bet_records_partitioned_member_guaji_lookup_idx",
		"cloud_bet_records_partitioned_scheme_stats_idx",
		"wallet_ledger_partitioned_id_idx",
		"wallet_ledger_partitioned_ledger_no_idx",
		"wallet_ledger_partitioned_member_type_created_idx",
		"wallet_ledger_partitioned_member_guaji_idx",
		"wallet_ledger_partitioned_member_guaji_lottery_idx",
		"wallet_ledger_partitioned_scheme_audit_idx",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("query index migration missing %q", want)
		}
	}
}

func TestCoreOnlinePartitionMaintenanceMigration(t *testing.T) {
	sql := readMigration(t, "../../migrations/00169_core_partition_maintenance.sql")
	for _, want := range []string{
		"ensure_core_online_partitions",
		"relkind = 'p'",
		"bet_orders_partitioned",
		"cloud_bet_records_partitioned",
		"wallet_ledger_partitioned",
		"parent_names",
	} {
		if !strings.Contains(sql, want) {
			t.Errorf("partition maintenance migration missing %q", want)
		}
	}
}

func readMigration(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
