package schemebettingdispatch

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/guajibet"
	"caipiao/backend/internal/schemebetting"
)

func TestDispatchRateLimiterEnforcesAccountLotteryAndGlobalQuota(t *testing.T) {
	_ = godotenv.Load("../../.env")
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
	lottery := "rate_limit_test_lottery"
	now := time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err = pool.Exec(ctx, `
INSERT INTO scheme_betting_capacity_limits
    (lottery_code, max_due_outbox, max_active_schemes, max_dispatch_per_second,
     max_account_dispatch_per_second, max_global_dispatch_per_second, enabled)
VALUES ($1, 10, 10, 2, 1, 2, true)
ON CONFLICT (lottery_code) DO UPDATE
SET max_dispatch_per_second = 2,
    max_account_dispatch_per_second = 1,
    max_global_dispatch_per_second = 2,
    enabled = true`, lottery)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Exec(ctx, `DELETE FROM scheme_betting_dispatch_rate_buckets WHERE window_start = $1 AND scope_key IN ('global', $2, 'rate-account-a', 'rate-account-b')`, now, lottery)
	defer pool.Exec(ctx, `DELETE FROM scheme_betting_capacity_limits WHERE lottery_code = $1`, lottery)

	commandFor := func(account string) schemebetting.LeasedCommand {
		frozen, err := json.Marshal(FrozenGuajiRequest{
			MemberAccount: account,
			Request:       guajibet.Request{LotteryCode: lottery, IssueNo: "T", Amount: 2, Currency: "CNY"},
		})
		if err != nil {
			t.Fatal(err)
		}
		return schemebetting.LeasedCommand{FrozenRequest: frozen}
	}
	limiter := NewDispatchRateLimiter(pool)
	allowed, err := limiter.Allow(ctx, commandFor("rate-account-a"), now)
	if err != nil || !allowed {
		t.Fatalf("first account dispatch allowed=%v err=%v", allowed, err)
	}
	allowed, err = limiter.Allow(ctx, commandFor("rate-account-a"), now)
	if err != nil || allowed {
		t.Fatalf("same account second dispatch allowed=%v err=%v", allowed, err)
	}
	allowed, err = limiter.Allow(ctx, commandFor("rate-account-b"), now)
	if err != nil || !allowed {
		t.Fatalf("second account dispatch allowed=%v err=%v", allowed, err)
	}
	allowed, err = limiter.Allow(ctx, commandFor("rate-account-b"), now)
	if err != nil || allowed {
		t.Fatalf("global or lottery third dispatch allowed=%v err=%v", allowed, err)
	}
}
