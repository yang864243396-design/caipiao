package sqlcdb_test

import (
	"context"
	"fmt"
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

func TestListOpenProviderPeriodSnapshotsUsesLatestPeriodFact(t *testing.T) {
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

	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := time.Now().UTC().Truncate(time.Millisecond)
	lotteryCode := fmt.Sprintf("__latest_fact_%d", now.UnixNano())
	periodNo := "future-period"
	for _, snapshot := range []struct {
		openAt       any
		observedAt   time.Time
		snapshotHash string
	}{
		{openAt: nil, observedAt: now.Add(-2 * time.Second), snapshotHash: "old-current"},
		{openAt: now.Add(time.Minute), observedAt: now.Add(-time.Second), snapshotHash: "new-future"},
	} {
		if _, err := tx.Exec(ctx, `
INSERT INTO provider_period_snapshots
    (lottery_code, period_no, open_at, close_at, observed_at, source, snapshot_hash, raw_payload)
VALUES ($1, $2, $3, $4, $5, 'test', $6, '{}'::jsonb)`,
			lotteryCode, periodNo, snapshot.openAt, now.Add(2*time.Minute), snapshot.observedAt, snapshot.snapshotHash); err != nil {
			t.Fatal(err)
		}
	}

	rows, err := sqlcdb.New(tx).ListOpenProviderPeriodSnapshots(ctx, lotteryCode, "source-period", now, now.Add(-10*time.Second), 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("open snapshots=%d want=0; an older eligible fact must not override the latest future fact", len(rows))
	}
}
