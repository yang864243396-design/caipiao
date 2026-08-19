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

func TestEventDrivenQueriesAgainstMigratedSchema(t *testing.T) {
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
	if _, err := q.ListOpenProviderPeriodSnapshots(context.Background(), "__schema_probe__", "source", now, now.Add(-time.Minute), 1); err != nil {
		t.Fatalf("provider snapshot query: %v", err)
	}
	for _, table := range []string{"provider_period_snapshots", "scheme_period_decisions", "scheme_bet_outbox", "scheme_bet_attempts"} {
		var exists bool
		if err := pool.QueryRow(context.Background(), `SELECT to_regclass('public.' || $1) IS NOT NULL`, table).Scan(&exists); err != nil || !exists {
			t.Fatalf("table %s exists=%v err=%v", table, exists, err)
		}
	}
}
