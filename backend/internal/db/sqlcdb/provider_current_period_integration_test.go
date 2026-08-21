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

func TestRecordCurrentProviderPeriodSnapshotAgainstCurrentSchema(t *testing.T) {
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
	defer tx.Rollback(ctx)

	now := time.Now().UTC().Truncate(time.Millisecond)
	params := sqlcdb.RecordCurrentProviderPeriodSnapshotParams{
		LotteryCode: fmt.Sprintf("__current_provider_%d", now.UnixNano()),
		PeriodNo:    "P-current", OpenAt: now, CloseAt: now.Add(6 * time.Second), ObservedAt: now,
		Source: "guaji_periods_current", SnapshotHash: fmt.Sprintf("hash-%d", now.UnixNano()),
		RawPayload: []byte(`{"source":"test"}`),
	}
	q := sqlcdb.New(tx)
	first, err := q.RecordCurrentProviderPeriodSnapshot(ctx, params)
	if err != nil {
		t.Fatal(err)
	}
	second, err := q.RecordCurrentProviderPeriodSnapshot(ctx, params)
	if err != nil {
		t.Fatal(err)
	}
	if first <= 0 || second != first {
		t.Fatalf("snapshot ids first=%d second=%d", first, second)
	}
}
