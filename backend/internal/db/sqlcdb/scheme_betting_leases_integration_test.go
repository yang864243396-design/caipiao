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

func TestSchemeBettingLeaseFencingAgainstMigratedSchema(t *testing.T) {
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
	q := sqlcdb.New(pool)
	shard := int32(2147483000)
	_, _ = pool.Exec(ctx, `DELETE FROM scheme_betting_shard_leases WHERE lease_kind = 'dispatcher' AND shard_no = $1`, shard)
	defer pool.Exec(ctx, `DELETE FROM scheme_betting_shard_leases WHERE lease_kind = 'dispatcher' AND shard_no = $1`, shard)
	now := time.Now().UTC()
	epochA, acquired, err := q.AcquireSchemeBettingShardLease(ctx, "dispatcher", shard, "lease-test-a", now, now.Add(time.Second))
	if err != nil || !acquired || epochA <= 0 {
		t.Fatalf("first shard lease epoch=%d acquired=%v err=%v", epochA, acquired, err)
	}
	if _, acquired, err := q.AcquireSchemeBettingShardLease(ctx, "dispatcher", shard, "lease-test-b", now.Add(100*time.Millisecond), now.Add(time.Second)); err != nil || acquired {
		t.Fatalf("competing shard lease acquired=%v err=%v", acquired, err)
	}
	if err := q.AssertSchemeBettingShardLease(ctx, "dispatcher", shard, "lease-test-a", epochA, now.Add(200*time.Millisecond)); err != nil {
		t.Fatalf("current shard fence rejected: %v", err)
	}
	epochB, acquired, err := q.AcquireSchemeBettingShardLease(ctx, "dispatcher", shard, "lease-test-b", now.Add(2*time.Second), now.Add(3*time.Second))
	if err != nil || !acquired || epochB <= epochA {
		t.Fatalf("takeover shard lease epochA=%d epochB=%d acquired=%v err=%v", epochA, epochB, acquired, err)
	}
	if err := q.AssertSchemeBettingShardLease(ctx, "dispatcher", shard, "lease-test-a", epochA, now.Add(2*time.Second)); err == nil {
		t.Fatal("stale shard epoch must be rejected")
	}

	lottery := "lease_test_lottery"
	_, _ = pool.Exec(ctx, `DELETE FROM scheme_betting_draw_leases WHERE lottery_code = $1`, lottery)
	defer pool.Exec(ctx, `DELETE FROM scheme_betting_draw_leases WHERE lottery_code = $1`, lottery)
	drawEpochA, acquired, err := q.AcquireSchemeBettingDrawLease(ctx, lottery, "lease-test-a", now, now.Add(time.Second))
	if err != nil || !acquired {
		t.Fatalf("first draw lease epoch=%d acquired=%v err=%v", drawEpochA, acquired, err)
	}
	if _, acquired, err := q.AcquireSchemeBettingDrawLease(ctx, lottery, "lease-test-b", now.Add(100*time.Millisecond), now.Add(time.Second)); err != nil || acquired {
		t.Fatalf("competing draw lease acquired=%v err=%v", acquired, err)
	}
}
