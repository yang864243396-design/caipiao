package db

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
)

func TestCoreOnlinePartitionMirrorsAgainstMigratedSchema(t *testing.T) {
	_ = godotenv.Load("../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := Connect(context.Background(), cfg.DatabaseURL, 2, 0)
	if err != nil {
		t.Skipf("database unavailable: %v", err)
	}
	defer pool.Close()

	var phase string
	if err := pool.QueryRow(
		context.Background(),
		`SELECT phase FROM core_partition_migration_state WHERE id = 1`,
	).Scan(&phase); err != nil {
		t.Fatal(err)
	}
	validationQuery := `SELECT validate_core_online_partitions()`
	partitionTables := []string{
		"bet_orders_partitioned",
		"cloud_bet_records_partitioned",
		"wallet_ledger_partitioned",
	}
	if phase == "cutover" || phase == "rollback_ready" {
		validationQuery = `SELECT validate_core_online_partition_cutover()`
		partitionTables = []string{"bet_orders", "cloud_bet_records", "wallet_ledger"}
	}
	var validationBytes []byte
	if err := pool.QueryRow(
		context.Background(),
		validationQuery,
	).Scan(&validationBytes); err != nil {
		t.Fatal(err)
	}
	var validation struct {
		Valid bool `json:"valid"`
	}
	if err := json.Unmarshal(validationBytes, &validation); err != nil {
		t.Fatal(err)
	}
	if !validation.Valid {
		t.Fatalf("partition mirrors are not exact: %s", validationBytes)
	}

	for _, table := range partitionTables {
		var relkind string
		if err := pool.QueryRow(
			context.Background(),
			`SELECT relkind::text FROM pg_class WHERE oid = $1::regclass`,
			table,
		).Scan(&relkind); err != nil {
			t.Fatal(err)
		}
		if relkind != "p" {
			t.Fatalf("%s relkind=%s, want p", table, relkind)
		}
	}

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())
	var id int64
	var placedAt any
	if err := tx.QueryRow(
		context.Background(),
		`UPDATE bet_orders
		 SET updated_at = updated_at
		 WHERE id = (SELECT min(id) FROM bet_orders)
		 RETURNING id, placed_at`,
	).Scan(&id, &placedAt); err != nil {
		t.Fatal(err)
	}
	var mirrored bool
	if phase == "cutover" || phase == "rollback_ready" {
		if err := tx.QueryRow(
			context.Background(),
			`SELECT EXISTS(
			    SELECT 1 FROM bet_orders_legacy_unpartitioned WHERE id = $1
			)`,
			id,
		).Scan(&mirrored); err != nil {
			t.Fatal(err)
		}
	} else {
		if err := tx.QueryRow(
			context.Background(),
			`SELECT EXISTS(
			    SELECT 1 FROM bet_orders_partitioned
			    WHERE id = $1 AND placed_at = $2
			)`,
			id, placedAt,
		).Scan(&mirrored); err != nil {
			t.Fatal(err)
		}
	}
	if !mirrored {
		t.Fatal("core table sync did not preserve the updated bet order")
	}
}

func TestCoreOnlinePartitionCutoverAndRollbackInTransaction(t *testing.T) {
	_ = godotenv.Load("../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := Connect(context.Background(), cfg.DatabaseURL, 1, 0)
	if err != nil {
		t.Skipf("database unavailable: %v", err)
	}
	defer pool.Close()

	var phase string
	if err := pool.QueryRow(
		context.Background(),
		`SELECT phase FROM core_partition_migration_state WHERE id = 1`,
	).Scan(&phase); err != nil {
		t.Fatal(err)
	}
	if phase != "validated" {
		t.Skipf("cutover probe requires validated phase, got %s", phase)
	}

	tx, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(context.Background())

	var result []byte
	if err := tx.QueryRow(
		context.Background(),
		`SELECT cutover_core_online_partitions()`,
	).Scan(&result); err != nil {
		t.Fatalf("cutover: %v", err)
	}
	for _, table := range []string{"bet_orders", "cloud_bet_records", "wallet_ledger"} {
		var relkind string
		if err := tx.QueryRow(
			context.Background(),
			`SELECT relkind::text FROM pg_class WHERE oid = $1::regclass`,
			table,
		).Scan(&relkind); err != nil {
			t.Fatal(err)
		}
		if relkind != "p" {
			t.Fatalf("%s relkind=%s after cutover", table, relkind)
		}
	}

	if err := tx.QueryRow(
		context.Background(),
		`SELECT validate_core_online_partition_cutover()`,
	).Scan(&result); err != nil {
		t.Fatalf("post-cutover validation: %v", err)
	}
	var validation struct {
		Valid bool `json:"valid"`
	}
	if err := json.Unmarshal(result, &validation); err != nil {
		t.Fatal(err)
	}
	if !validation.Valid {
		t.Fatalf("post-cutover validation failed: %s", result)
	}

	if err := tx.QueryRow(
		context.Background(),
		`SELECT rollback_core_online_partitions()`,
	).Scan(&result); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	for _, table := range []string{"bet_orders", "cloud_bet_records", "wallet_ledger"} {
		var relkind string
		if err := tx.QueryRow(
			context.Background(),
			`SELECT relkind::text FROM pg_class WHERE oid = $1::regclass`,
			table,
		).Scan(&relkind); err != nil {
			t.Fatal(err)
		}
		if relkind == "p" {
			t.Fatalf("%s remained partitioned after rollback", table)
		}
	}
}
