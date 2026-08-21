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

	var now time.Time
	if err := tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&now); err != nil {
		t.Fatal(err)
	}
	now = now.UTC().Truncate(time.Millisecond)
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

func TestListOpenProviderPeriodSnapshotsReturnsPreloadedCurrentPeriod(t *testing.T) {
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

	var now time.Time
	if err := tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&now); err != nil {
		t.Fatal(err)
	}
	now = now.UTC().Truncate(time.Millisecond)
	lotteryCode := fmt.Sprintf("__preloaded_%d", now.UnixNano())
	if _, err := tx.Exec(ctx, `
INSERT INTO provider_period_snapshots
    (lottery_code, period_no, open_at, close_at, observed_at, source, snapshot_hash, raw_payload)
VALUES ($1, 'preloaded-period', $2, $3, $4, 'test', 'preloaded', '{}'::jsonb)`,
		lotteryCode, now, now.Add(6*time.Second), now.Add(-7*time.Second)); err != nil {
		t.Fatal(err)
	}

	rows, err := sqlcdb.New(tx).ListOpenProviderPeriodSnapshots(ctx, lotteryCode, "source-period", now, now.Add(-6*time.Second), 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].PeriodNo != "preloaded-period" {
		t.Fatalf("rows=%+v want preloaded current period", rows)
	}
}

func TestListOpenProviderPeriodSnapshotsUsesDatabaseClock(t *testing.T) {
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

	var databaseNow time.Time
	if err := tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&databaseNow); err != nil {
		t.Fatal(err)
	}
	databaseNow = databaseNow.UTC()
	lotteryCode := fmt.Sprintf("__db_clock_%d", databaseNow.UnixNano())
	if _, err := tx.Exec(ctx, `
INSERT INTO provider_period_snapshots
    (lottery_code, period_no, open_at, close_at, observed_at, source, snapshot_hash, raw_payload)
VALUES ($1, 'db-current-period', $2, $3, $4, 'test', 'db-clock', '{}'::jsonb)`,
		lotteryCode, databaseNow.Add(-time.Second), databaseNow.Add(5*time.Second), databaseNow.Add(-7*time.Second)); err != nil {
		t.Fatal(err)
	}

	applicationNow := databaseNow.Add(-3 * time.Second)
	rows, err := sqlcdb.New(tx).ListOpenProviderPeriodSnapshots(
		ctx, lotteryCode, "source-period", applicationNow, applicationNow.Add(-6*time.Second), 8,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].PeriodNo != "db-current-period" {
		t.Fatalf("rows=%+v want database-current period despite application clock skew", rows)
	}
}

func TestPreSendRecoveryOnlyReturnsCurrentInstanceChain(t *testing.T) {
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
	account := fmt.Sprintf("chain%d", stamp%1e12)
	definitionID := fmt.Sprintf("chain-def-%d", stamp)
	instanceID := fmt.Sprintf("chain-inst-%d", stamp)
	lotteryCode := fmt.Sprintf("ch%d", stamp)
	var memberID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO members (account, password_hash, display_name, status)
VALUES ($1, 'test', 'chain recovery test', 'active') RETURNING id`, account).Scan(&memberID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO scheme_definitions (id, member_id, kind, scheme_name, lottery_code, lottery_label, share_status, config)
VALUES ($1, $2, 'custom', 'chain recovery test', $3, 'test', 'private', '{}'::jsonb)`, definitionID, memberID, lotteryCode); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO scheme_instances
    (id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label, status,
     sim_bet, betting_owner, strict_chain_state, chain_id, chain_seq)
VALUES ($1, $2, $3, 'custom', 'chain recovery test', $4, 'test', 'running',
        false, 'event', 'active', 'current-chain', 4)`, instanceID, definitionID, memberID, lotteryCode); err != nil {
		t.Fatal(err)
	}
	var snapshotID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO provider_period_snapshots
    (lottery_code, period_no, open_at, close_at, observed_at, source, snapshot_hash, raw_payload)
VALUES ($1, 'provider-period', clock_timestamp(), clock_timestamp() + interval '1 minute',
        clock_timestamp(), 'test', $2, '{}'::jsonb) RETURNING id`, lotteryCode, fmt.Sprintf("chain-snapshot-%d", stamp)).Scan(&snapshotID); err != nil {
		t.Fatal(err)
	}

	insertOutbox := func(sourcePeriod, targetPeriod, requestID, chainID string) int64 {
		t.Helper()
		var decisionID, outboxID int64
		if err := tx.QueryRow(ctx, `
INSERT INTO scheme_period_decisions
    (scheme_id, lottery_code, source_period_no, state_version_before, state_version_after, status)
VALUES ($1, $2, $3, 1, 1, 'completed') RETURNING id`, instanceID, lotteryCode, sourcePeriod).Scan(&decisionID); err != nil {
			t.Fatal(err)
		}
		if err := tx.QueryRow(ctx, `
INSERT INTO scheme_bet_outbox
    (decision_id, scheme_id, member_id, lottery_code, source_period_no, target_period_no,
     mode, state, request_id, payload_hash, payload, provider_snapshot_id, close_at,
     safe_deadline_at, shard_no, outcome_reason, chain_id, chain_seq)
VALUES ($1, $2, $3, $4, $5, $6,
        'shadow', 'rejected', $7, $10, '{}'::jsonb, $8, clock_timestamp() + interval '1 minute',
        clock_timestamp() + interval '30 seconds', 0, 'provider_pre_send_failed', $9, 1)
RETURNING id`, decisionID, instanceID, memberID, lotteryCode, sourcePeriod, targetPeriod, requestID, snapshotID, chainID, "hash-"+requestID).Scan(&outboxID); err != nil {
			t.Fatal(err)
		}
		return outboxID
	}

	oldID := insertOutbox("old-source", "old-target", fmt.Sprintf("old-request-%d", stamp), "old-chain")
	currentID := insertOutbox("current-source", "current-target", fmt.Sprintf("current-request-%d", stamp), "current-chain")
	q := sqlcdb.New(tx)
	if _, found, err := q.GetPreSendFailureOutbox(ctx, oldID); err != nil || found {
		t.Fatalf("old chain found=%v err=%v", found, err)
	}
	if row, found, err := q.GetPreSendFailureOutbox(ctx, currentID); err != nil || !found || row.ID != currentID {
		t.Fatalf("current chain row=%+v found=%v err=%v", row, found, err)
	} else if row.ChainID != "current-chain" {
		t.Fatalf("current chain id=%q want current-chain", row.ChainID)
	}
	ids, err := q.ListPendingPreSendFailureOutboxIDs(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		if id == oldID {
			t.Fatalf("old chain outbox %d was returned for recovery", oldID)
		}
	}
	foundCurrent := false
	for _, id := range ids {
		foundCurrent = foundCurrent || id == currentID
	}
	if !foundCurrent {
		t.Fatalf("current chain outbox %d missing from recovery ids=%v", currentID, ids)
	}
	blocked, err := q.BlockSchemeBettingChainIfCurrent(ctx, instanceID, "old-chain", "old failure", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if blocked {
		t.Fatal("old chain failure blocked the current execution chain")
	}
	var chainState string
	if err := tx.QueryRow(ctx, `SELECT strict_chain_state FROM scheme_instances WHERE id=$1`, instanceID).Scan(&chainState); err != nil {
		t.Fatal(err)
	}
	if chainState != "active" {
		t.Fatalf("chain state=%q want active", chainState)
	}
	blocked, err = q.BlockSchemeBettingChainIfCurrent(ctx, instanceID, "current-chain", "current failure", time.Now().UTC())
	if err != nil || !blocked {
		t.Fatalf("current chain blocked=%v err=%v", blocked, err)
	}
}
