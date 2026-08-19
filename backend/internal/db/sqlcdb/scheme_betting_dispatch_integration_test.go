package sqlcdb_test

import (
	"context"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
)

func TestDispatchQueriesAgainstMigratedSchema(t *testing.T) {
	_ = godotenv.Load("../../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	pool, err := db.Connect(context.Background(), cfg.DatabaseURL, 2, 0)
	if err != nil {
		t.Skipf("database unavailable: %v", err)
	}
	defer pool.Close()

	q := sqlcdb.New(pool)
	now := time.Now().UTC()
	commands, err := q.LeaseFormalSchemeBetOutbox(context.Background(), sqlcdb.LeaseFormalOutboxParams{
		Mode: "gray", LeaseOwner: "schema-probe", ShardNo: -1, Limit: 1,
		Now: now, LeaseUntil: now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("lease formal outbox query: %v", err)
	}
	if len(commands) != 0 {
		t.Fatalf("schema probe unexpectedly leased %d rows", len(commands))
	}
	for _, table := range []string{"scheme_betting_shard_leases", "scheme_betting_admin_actions", "scheme_betting_capacity_limits"} {
		var exists bool
		if err := pool.QueryRow(context.Background(), `SELECT to_regclass('public.' || $1) IS NOT NULL`, table).Scan(&exists); err != nil || !exists {
			t.Fatalf("table %s exists=%v err=%v", table, exists, err)
		}
	}
}
