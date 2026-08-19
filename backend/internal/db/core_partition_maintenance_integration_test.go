package db

import (
	"context"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
)

func TestEnsureCoreOnlinePartitionsAfterCutover(t *testing.T) {
	_ = godotenv.Load("../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	pool, err := Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		t.Skip(err)
	}
	t.Cleanup(pool.Close)

	if _, err := EnsureCoreOnlinePartitions(ctx, pool, CorePartitionMonthsAhead); err != nil {
		t.Fatal(err)
	}

	const cutoverProbeMonth = "2040-01-01"
	if _, err := pool.Exec(ctx, `SELECT ensure_core_online_partitions($1::date, 0)`, cutoverProbeMonth); err != nil {
		t.Fatalf("cutover parent maintenance: %v", err)
	}

	future := time.Now().UTC().AddDate(0, CorePartitionMonthsAhead, 0).Format("200601")
	for _, prefix := range []string{
		"bet_orders_partitioned_",
		"cloud_bet_records_partitioned_",
		"wallet_ledger_partitioned_",
	} {
		var exists bool
		if err := pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, prefix+future).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Errorf("future partition %s%s is missing", prefix, future)
		}
		if err := pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, prefix+"204001").Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Errorf("cutover probe partition %s204001 is missing", prefix)
		}
	}
}
