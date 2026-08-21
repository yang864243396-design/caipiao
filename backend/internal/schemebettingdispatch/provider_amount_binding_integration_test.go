package schemebettingdispatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
	"caipiao/backend/internal/guajibet"
	"caipiao/backend/internal/schemebetting"
)

type providerAmountTestPlacer struct {
	result guajibet.Result
}

func (providerAmountTestPlacer) Enabled() bool { return true }
func (providerAmountTestPlacer) PlaceRealBet(context.Context, string, guajibet.Request) (guajibet.Result, error) {
	return guajibet.Result{}, errors.New("not used")
}
func (providerAmountTestPlacer) MirrorBetDebitLedger(context.Context, *sqlcdb.Queries, int64, string, float64, int64, string) error {
	return nil
}
func (placer providerAmountTestPlacer) ResolveAcceptedBet(context.Context, string, guajibet.Request) (guajibet.Result, error) {
	return placer.result, nil
}
func (placer providerAmountTestPlacer) ResolveAcceptedBets(_ context.Context, _ string, requests []guajibet.Request) []guajibet.AcceptanceLookup {
	lookups := make([]guajibet.AcceptanceLookup, len(requests))
	for i := range lookups {
		lookups[i].Result = placer.result
	}
	return lookups
}

func TestResolveUnknownPersistsFloatProviderAmount(t *testing.T) {
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
	var memberID, accountID, snapshotID int64
	var lotteryCode string
	if err := pool.QueryRow(ctx, `
SELECT member_id, id
FROM member_guaji_accounts
WHERE is_active = TRUE
ORDER BY id
LIMIT 1`).Scan(&memberID, &accountID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			t.Skip("database has no active guaji account fixture")
		}
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `
SELECT id, lottery_code
FROM provider_period_snapshots
ORDER BY id DESC
LIMIT 1`).Scan(&snapshotID, &lotteryCode); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			t.Skip("database has no provider period snapshot fixture")
		}
		t.Fatal(err)
	}

	unique := fmt.Sprintf("pa%d", time.Now().UnixNano())
	targetPeriod := unique + "t"
	frozen, err := json.Marshal(FrozenGuajiRequest{
		RequestID:       unique,
		MemberAccount:   "integration-test",
		LotteryLabel:    "integration lottery",
		LotteryCategory: "ssc",
		LocalBetPayload: json.RawMessage(`{}`),
		Request: guajibet.Request{
			LotteryCode: lotteryCode,
			IssueNo:     targetPeriod,
			PlayMethod:  "integration play",
			Amount:      0.2,
			Currency:    "USDT",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var outboxID int64
	err = pool.QueryRow(ctx, `
INSERT INTO scheme_bet_outbox
    (origin, local_order_no, member_id, lottery_code, source_period_no, target_period_no,
     mode, state, request_id, payload_hash, payload, frozen_request, frozen_request_hash,
     command_frozen_at, provider_snapshot_id, close_at, safe_deadline_at, shard_no)
VALUES
    ('api', $4, $1, $2, $3, $3,
     'gray', 'external_acceptance_unknown', $4, $5, '{}'::jsonb, $6, $5,
     clock_timestamp(), $7, clock_timestamp() + interval '10 seconds',
     clock_timestamp() + interval '5 seconds', 0)
RETURNING id`, memberID, lotteryCode, targetPeriod, unique, unique+"-hash", frozen, snapshotID).Scan(&outboxID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM scheme_betting_admin_actions WHERE outbox_id=$1`, outboxID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM bet_orders WHERE order_no=$1`, unique)
		_, _ = pool.Exec(context.Background(), `DELETE FROM scheme_bet_outbox WHERE id=$1`, outboxID)
	})

	finalizer := NewAcceptanceFinalizer(pool, providerAmountTestPlacer{result: guajibet.Result{
		ThirdPartyBetID: unique + "-order", Periods: unique + "p", Amount: 0.2,
		GuajiAccountID: accountID, Currency: "USDT",
	}})
	err = finalizer.RecoverUnknown(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	var state, storedAmount string
	var finalized bool
	if err := pool.QueryRow(ctx, `
SELECT state, COALESCE(provider_amount::text, ''), financial_finalized_at IS NOT NULL
FROM scheme_bet_outbox
WHERE id=$1`, outboxID).Scan(&state, &storedAmount, &finalized); err != nil {
		t.Fatal(err)
	}
	if state != string(schemebetting.OutboxAcceptedWrongPeriod) {
		t.Fatalf("state=%s, want %s", state, schemebetting.OutboxAcceptedWrongPeriod)
	}
	if storedAmount != "0.200" {
		t.Fatalf("provider_amount=%q, want exact numeric text 0.200", storedAmount)
	}
	if !finalized {
		t.Fatal("accepted wrong-period provider order was not financially finalized")
	}
	var storedPeriod string
	if err := pool.QueryRow(ctx, `SELECT issue_no FROM bet_orders WHERE order_no=$1`, unique).Scan(&storedPeriod); err != nil {
		t.Fatal(err)
	}
	if storedPeriod != unique+"p" {
		t.Fatalf("bet order issue=%q, want provider accepted period", storedPeriod)
	}
}
