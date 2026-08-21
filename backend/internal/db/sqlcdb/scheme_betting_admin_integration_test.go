package sqlcdb_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
)

func TestSchemeBettingAdminQueriesAgainstMigratedSchema(t *testing.T) {
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
	if err := q.ActivateSchemeBettingChain(ctx, "__missing_scheme__", "probe-chain", true); err == nil {
		t.Fatal("missing scheme owner transition unexpectedly succeeded")
	}
	if _, err := q.CancelSchemeBetOutbox(ctx, -1, time.Now().UTC()); err == nil {
		t.Fatal("missing outbox cancellation unexpectedly succeeded")
	}
	if _, err := q.ListStrategyReadyCandidates(ctx, "__schema_probe__", "period", 0, 10); err != nil {
		t.Fatalf("strategy ready expansion query: %v", err)
	}
	if _, found, err := q.PendingFormalStrategyRowForSchemeDraw(ctx, -1, "__missing_scheme__", "__schema_probe__", "period", 0); err != nil || found {
		t.Fatalf("strategy ready exact query found=%v err=%v", found, err)
	}

	var triggerExists bool
	if err := pool.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1 FROM pg_trigger
    WHERE tgname = 'trg_scheme_betting_admin_actions_append_only' AND NOT tgisinternal
)`).Scan(&triggerExists); err != nil || !triggerExists {
		t.Fatalf("append-only trigger exists=%v err=%v", triggerExists, err)
	}
	var constraintDefinition string
	if err := pool.QueryRow(ctx, `
SELECT pg_get_constraintdef(oid)
FROM pg_constraint
WHERE conname = 'scheme_betting_admin_actions_action_check'`).Scan(&constraintDefinition); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(constraintDefinition, "enable_event") {
		t.Fatalf("action constraint = %s", constraintDefinition)
	}
}

func TestSettledAcceptedBetRemainsStrategyReadyUntilEvaluated(t *testing.T) {
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

	stamp := time.Now().UnixNano()
	account := fmt.Sprintf("strategy%d", stamp%1e12)
	definitionID := fmt.Sprintf("strategy-def-%d", stamp)
	instanceID := fmt.Sprintf("strategy-inst-%d", stamp)
	lotteryCode := fmt.Sprintf("st%d", stamp)
	periodNo := fmt.Sprintf("period-%d", stamp)
	var memberID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO members (account, password_hash, display_name, status)
VALUES ($1, 'test', 'strategy ready test', 'active') RETURNING id`, account).Scan(&memberID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO scheme_definitions (id, member_id, kind, scheme_name, lottery_code, lottery_label, share_status, config)
VALUES ($1, $2, 'custom', 'strategy ready test', $3, 'test', 'private', '{}'::jsonb)`, definitionID, memberID, lotteryCode); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO scheme_instances
    (id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label, status,
     sim_bet, betting_owner, strict_chain_state, chain_id, state_version)
VALUES ($1, $2, $3, 'custom', 'strategy ready test', $4, 'test', 'running',
        false, 'event', 'active', 'strategy-chain', 7)`, instanceID, definitionID, memberID, lotteryCode); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO lottery_draws (lottery_code, issue_no, period_short, balls, sum_value, drawn_at)
VALUES ($1, $2, 'test', '["1","2","3","4","5"]'::jsonb, 15, clock_timestamp())`, lotteryCode, periodNo); err != nil {
		t.Fatal(err)
	}
	var recordID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO cloud_bet_records
    (record_no, member_id, scheme_id, scheme_name, period_no, play_type, multiplier,
     round_label, amount, pnl, status, bet_content, sim_bet, currency, lottery_code,
     lottery_label, definition_id, bet_units, third_party_bet_id, rule_snapshot,
     rule_version, rule_snapshot_hash)
VALUES ($1, $2, $3, 'strategy ready test', $4, 'test', '1',
        '1/1', 1, 1, 'hit', '1', false, 'CNY', $5,
        'test', $6, 1, $7, '{"evaluatorKey":"test"}'::jsonb, 1, 'snapshot-hash')
RETURNING id`, fmt.Sprintf("sr%d", stamp), memberID, instanceID, periodNo, lotteryCode, definitionID, fmt.Sprintf("third-party-%d", stamp)).Scan(&recordID); err != nil {
		t.Fatal(err)
	}

	q := sqlcdb.New(tx)
	candidates, err := q.ListStrategyReadyCandidates(ctx, lotteryCode, periodNo, 0, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 1 || candidates[0].RecordID != recordID {
		t.Fatalf("strategy candidates=%+v want settled record %d", candidates, recordID)
	}
	row, found, err := q.PendingFormalStrategyRowForSchemeDraw(ctx, recordID, instanceID, lotteryCode, periodNo, 7)
	if err != nil || !found || row.RecordID != recordID {
		t.Fatalf("exact strategy row=%+v found=%v err=%v", row, found, err)
	}
	if !row.ProviderHit.Valid || !row.ProviderHit.Bool {
		t.Fatalf("provider hit=%+v want settled hit", row.ProviderHit)
	}
	rows, err := q.ListPendingFormalStrategyRowsForDraw(ctx, lotteryCode, periodNo, 10)
	if err != nil || len(rows) != 1 || rows[0].RecordID != recordID {
		t.Fatalf("draw strategy rows=%+v err=%v", rows, err)
	}
}

func TestFinancialSettlementUpdateDoesNotOverwriteStrategyState(t *testing.T) {
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
	// Keep this regression isolated from the developer database migration
	// level. PostgreSQL rolls the test-only DDL back with the transaction.
	if _, err := tx.Exec(ctx, `
ALTER TABLE scheme_instances
ADD COLUMN IF NOT EXISTS lookback_round_reset_pending BOOLEAN NOT NULL DEFAULT FALSE`); err != nil {
		t.Fatal(err)
	}

	stamp := time.Now().UnixNano()
	account := fmt.Sprintf("finance%d", stamp%1e12)
	definitionID := fmt.Sprintf("finance-def-%d", stamp)
	instanceID := fmt.Sprintf("finance-inst-%d", stamp)
	lotteryCode := fmt.Sprintf("fn%d", stamp)
	var memberID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO members (account, password_hash, display_name, status)
VALUES ($1, 'test', 'financial state test', 'active') RETURNING id`, account).Scan(&memberID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO scheme_definitions (id, member_id, kind, scheme_name, lottery_code, lottery_label, share_status, config)
VALUES ($1, $2, 'custom', 'financial state test', $3, 'test', 'private', '{}'::jsonb)`, definitionID, memberID, lotteryCode); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO scheme_instances
    (id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label, status,
     round_index, pick_index, current_pick, last_direction, lookback_pnl, betting_owner,
     strict_chain_state, chain_id)
VALUES ($1, $2, $3, 'custom', 'financial state test', $4, 'test', 'running',
        5, 6, 'new-pick', 'neg', 2, 'event', 'active', 'financial-test-chain')`, instanceID, definitionID, memberID, lotteryCode); err != nil {
		t.Fatal(err)
	}
	q := sqlcdb.New(tx)
	if err := q.ApplySchemeInstanceFinancialAfterSettlement(ctx, instanceID, 3); err != nil {
		t.Fatal(err)
	}
	var roundIndex, pickIndex int32
	var currentPick, lastDirection string
	var lookback float64
	var resetPending bool
	if err := tx.QueryRow(ctx, `
SELECT round_index, pick_index, current_pick, last_direction, lookback_pnl::float8,
       lookback_round_reset_pending
FROM scheme_instances WHERE id=$1`, instanceID).Scan(&roundIndex, &pickIndex, &currentPick, &lastDirection, &lookback, &resetPending); err != nil {
		t.Fatal(err)
	}
	if roundIndex != 5 || pickIndex != 6 || currentPick != "new-pick" || lastDirection != "neg" {
		t.Fatalf("strategy overwritten: round=%d pick=%d current=%q direction=%q", roundIndex, pickIndex, currentPick, lastDirection)
	}
	if lookback != 5 {
		t.Fatalf("lookback=%v want=5", lookback)
	}
	if err := q.ResetSchemeInstanceLookbackOnly(ctx, instanceID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `
SELECT round_index, pick_index, current_pick, last_direction, lookback_pnl::float8,
       lookback_round_reset_pending
FROM scheme_instances WHERE id=$1`, instanceID).Scan(&roundIndex, &pickIndex, &currentPick, &lastDirection, &lookback, &resetPending); err != nil {
		t.Fatal(err)
	}
	if roundIndex != 5 || pickIndex != 6 || currentPick != "new-pick" || lastDirection != "neg" {
		t.Fatalf("financial reset overwrote strategy: round=%d pick=%d current=%q direction=%q", roundIndex, pickIndex, currentPick, lastDirection)
	}
	if lookback != 0 {
		t.Fatalf("reset lookback=%v want=0", lookback)
	}
	if !resetPending {
		t.Fatal("event-owned financial reset must schedule a round reset")
	}
	if err := q.ApplySchemeInstanceStrategyAfterDraw(ctx, instanceID, 6, 9, "advanced-pick", "pos"); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `
SELECT round_index, pick_index, current_pick, last_direction, lookback_pnl::float8,
       lookback_round_reset_pending
FROM scheme_instances WHERE id=$1`, instanceID).Scan(&roundIndex, &pickIndex, &currentPick, &lastDirection, &lookback, &resetPending); err != nil {
		t.Fatal(err)
	}
	if roundIndex != 0 || resetPending {
		t.Fatalf("consumed reset round=%d pending=%v, want round=0 pending=false", roundIndex, resetPending)
	}
	if pickIndex != 9 || currentPick != "advanced-pick" || lastDirection != "pos" {
		t.Fatalf("reset changed pick advancement: pick=%d current=%q direction=%q", pickIndex, currentPick, lastDirection)
	}

	if _, err := tx.Exec(ctx, `
UPDATE scheme_instances
SET betting_owner='legacy', round_index=4, lookback_pnl=7,
    lookback_round_reset_pending=FALSE
WHERE id=$1`, instanceID); err != nil {
		t.Fatal(err)
	}
	if err := q.ResetSchemeInstanceLookbackOnly(ctx, instanceID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `
SELECT round_index, lookback_pnl::float8, lookback_round_reset_pending
FROM scheme_instances WHERE id=$1`, instanceID).Scan(&roundIndex, &lookback, &resetPending); err != nil {
		t.Fatal(err)
	}
	if roundIndex != 0 || lookback != 0 || resetPending {
		t.Fatalf("legacy reset round=%d lookback=%v pending=%v, want immediate reset", roundIndex, lookback, resetPending)
	}

	if _, err := tx.Exec(ctx, `
UPDATE scheme_instances
SET betting_owner='event', strict_chain_state='blocked_requires_rearm',
    round_index=3, lookback_pnl=8, lookback_round_reset_pending=FALSE
WHERE id=$1`, instanceID); err != nil {
		t.Fatal(err)
	}
	if err := q.ResetSchemeInstanceLookbackOnly(ctx, instanceID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `
SELECT round_index, lookback_pnl::float8, lookback_round_reset_pending
FROM scheme_instances WHERE id=$1`, instanceID).Scan(&roundIndex, &lookback, &resetPending); err != nil {
		t.Fatal(err)
	}
	if roundIndex != 0 || lookback != 0 || resetPending {
		t.Fatalf("inactive event reset round=%d lookback=%v pending=%v, want immediate reset", roundIndex, lookback, resetPending)
	}

	if _, err := tx.Exec(ctx, `
UPDATE scheme_instances
SET sim_bet=TRUE, betting_owner='event', strict_chain_state='active',
    round_index=2, lookback_pnl=9, lookback_round_reset_pending=FALSE
WHERE id=$1`, instanceID); err != nil {
		t.Fatal(err)
	}
	if err := q.ResetSchemeInstanceLookbackOnly(ctx, instanceID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `
SELECT round_index, lookback_pnl::float8, lookback_round_reset_pending
FROM scheme_instances WHERE id=$1`, instanceID).Scan(&roundIndex, &lookback, &resetPending); err != nil {
		t.Fatal(err)
	}
	if roundIndex != 0 || lookback != 0 || resetPending {
		t.Fatalf("simulation event reset round=%d lookback=%v pending=%v, want immediate reset", roundIndex, lookback, resetPending)
	}
}
