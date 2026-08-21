package sqlcdb_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"

	"caipiao/backend/internal/config"
	"caipiao/backend/internal/db"
	"caipiao/backend/internal/db/sqlcdb"
)

func TestAwaitingTargetResolverAndExpiryHaveOneWinner(t *testing.T) {
	f := newAwaitingDecisionDBFixture(t)
	decisionID := f.Seed(f.DatabaseNow().Add(time.Second))
	var resolved, missed atomic.Int32
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if f.Complete(decisionID) {
			resolved.Add(1)
		}
	}()
	go func() {
		defer wg.Done()
		if f.Miss(decisionID) {
			missed.Add(1)
		}
	}()
	wg.Wait()
	if got := resolved.Load() + missed.Load(); got != 1 {
		t.Fatalf("terminal winners=%d want=1", got)
	}
	f.AssertSingleTerminal(decisionID)
}

func TestAwaitingTargetQueryIsBoundedAndCursorOrdered(t *testing.T) {
	f := newAwaitingDecisionDBFixture(t)
	f.SeedMany(40)
	rows := f.List(0, 32)
	if len(rows) != 32 {
		t.Fatalf("rows=%d want=32", len(rows))
	}
	for i := 1; i < len(rows); i++ {
		if rows[i].DecisionID <= rows[i-1].DecisionID {
			t.Fatalf("rows not cursor ordered: id[%d]=%d id[%d]=%d", i-1, rows[i-1].DecisionID, i, rows[i].DecisionID)
		}
	}
	if rows := f.List(rows[len(rows)-1].DecisionID, 32); len(rows) != 8 {
		t.Fatalf("second page rows=%d want=8", len(rows))
	}
}

func TestGetAwaitingContiguousTargetForUpdateReturnsPersistedShard(t *testing.T) {
	f := newAwaitingDecisionDBFixture(t)
	decisionID := f.Seed(f.DatabaseNow().Add(time.Minute))
	row, found, err := f.q.GetAwaitingContiguousTargetForUpdate(f.ctx, decisionID)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("awaiting decision was not found")
	}
	if row.DecisionID != decisionID || row.SchemeID != f.schemeID || row.LotteryCode != f.lotteryCode || row.ShardNo != f.shardNo {
		t.Fatalf("row=%+v", row)
	}
}

func TestActivateSchemeBettingChainClearsContiguousBlockReason(t *testing.T) {
	f := newAwaitingDecisionDBFixture(t)
	if _, err := f.tx.Exec(f.ctx, `UPDATE scheme_instances
SET strict_chain_state = 'blocked_requires_rearm', chain_block_reason = 'missed_contiguous_period'
WHERE id = $1`, f.schemeID); err != nil {
		t.Fatal(err)
	}
	if err := f.q.ActivateSchemeBettingChain(f.ctx, f.schemeID, "replacement-chain", false); err != nil {
		t.Fatal(err)
	}
	var reason *string
	if err := f.tx.QueryRow(f.ctx, `SELECT chain_block_reason FROM scheme_instances WHERE id = $1`, f.schemeID).Scan(&reason); err != nil {
		t.Fatal(err)
	}
	if reason != nil {
		t.Fatalf("chain block reason=%q want NULL", *reason)
	}
}

type awaitingDecisionDBFixture struct {
	t           *testing.T
	ctx         context.Context
	tx          pgx.Tx
	q           *sqlcdb.Queries
	schemeID    string
	lotteryCode string
	shardNo     int32
	nextSeed    int
}

func newAwaitingDecisionDBFixture(t *testing.T) *awaitingDecisionDBFixture {
	t.Helper()
	_ = godotenv.Load("../../../.env")
	cfg := config.Load()
	if cfg.DatabaseURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL, 2, 0)
	if err != nil {
		t.Skipf("database unavailable: %v", err)
	}
	t.Cleanup(pool.Close)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(ctx) })

	if _, err := tx.Exec(ctx, `
SELECT target_deadline_at, target_period_no, failure_reason, shard_no
FROM scheme_period_decisions
WHERE FALSE`); err != nil {
		t.Skipf("migration 177 not applied: %v", err)
	}
	if _, err := tx.Exec(ctx, `SELECT chain_block_reason FROM scheme_instances WHERE FALSE`); err != nil {
		t.Skipf("migration 177 not applied: %v", err)
	}

	stamp := time.Now().UnixNano()
	f := &awaitingDecisionDBFixture{
		t:           t,
		ctx:         ctx,
		tx:          tx,
		q:           sqlcdb.New(tx),
		schemeID:    fmt.Sprintf("awaiting-inst-%d", stamp),
		lotteryCode: fmt.Sprintf("awaiting-lottery-%d", stamp),
		shardNo:     7,
	}
	account := fmt.Sprintf("awaiting-account-%d", stamp)
	definitionID := fmt.Sprintf("awaiting-definition-%d", stamp)
	var memberID int64
	if err := tx.QueryRow(ctx, `
INSERT INTO members (account, password_hash, display_name, status)
VALUES ($1, 'test', 'awaiting target test', 'active')
RETURNING id`, account).Scan(&memberID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO scheme_definitions (id, member_id, kind, scheme_name, lottery_code, lottery_label, share_status, config)
VALUES ($1, $2, 'custom', 'awaiting target test', $3, 'test', 'private', '{}'::jsonb)`, definitionID, memberID, f.lotteryCode); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO scheme_instances
    (id, definition_id, member_id, kind, scheme_name, lottery_code, lottery_label, status,
     sim_bet, betting_owner, strict_chain_state, chain_id, chain_seq, state_version)
VALUES ($1, $2, $3, 'custom', 'awaiting target test', $4, 'test', 'running',
        false, 'event', 'active', 'awaiting-chain', 4, 8)`, f.schemeID, definitionID, memberID, f.lotteryCode); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *awaitingDecisionDBFixture) DatabaseNow() time.Time {
	f.t.Helper()
	var now time.Time
	if err := f.tx.QueryRow(f.ctx, `SELECT clock_timestamp()`).Scan(&now); err != nil {
		f.t.Fatal(err)
	}
	return now
}

func (f *awaitingDecisionDBFixture) Seed(deadline time.Time) int64 {
	f.t.Helper()
	f.nextSeed++
	var decisionID int64
	if err := f.tx.QueryRow(f.ctx, `
INSERT INTO scheme_period_decisions
    (scheme_id, lottery_code, source_period_no, state_version_before, state_version_after,
     status, target_deadline_at, shard_no)
VALUES ($1, $2, $3, 7, 8, 'awaiting_target', $4, $5)
RETURNING id`, f.schemeID, f.lotteryCode, fmt.Sprintf("source-%d", f.nextSeed), deadline, f.shardNo).Scan(&decisionID); err != nil {
		f.t.Fatal(err)
	}
	return decisionID
}

func (f *awaitingDecisionDBFixture) SeedMany(count int) {
	f.t.Helper()
	deadline := f.DatabaseNow().Add(time.Minute)
	for range count {
		f.Seed(deadline)
	}
}

func (f *awaitingDecisionDBFixture) Complete(decisionID int64) bool {
	f.t.Helper()
	completed, err := f.q.CompleteAwaitingContiguousTarget(f.ctx, sqlcdb.CompleteAwaitingContiguousTargetParams{
		DecisionID:     decisionID,
		TargetPeriodNo: fmt.Sprintf("target-%d", decisionID),
		Diagnostics:    []byte(`{"source":"test"}`),
	})
	if err != nil {
		f.t.Fatal(err)
	}
	return completed
}

func (f *awaitingDecisionDBFixture) Miss(decisionID int64) bool {
	f.t.Helper()
	missed, err := f.q.MissAwaitingContiguousTarget(f.ctx, sqlcdb.MissAwaitingContiguousTargetParams{
		DecisionID:    decisionID,
		FailureReason: "missed_contiguous_period",
		Diagnostics:   []byte(`{"source":"test"}`),
	})
	if err != nil {
		f.t.Fatal(err)
	}
	return missed
}

func (f *awaitingDecisionDBFixture) List(cursor int64, limit int32) []sqlcdb.AwaitingContiguousTargetRow {
	f.t.Helper()
	rows, err := f.q.ListAwaitingContiguousTargets(f.ctx, []string{f.lotteryCode}, []int32{f.shardNo}, cursor, limit)
	if err != nil {
		f.t.Fatal(err)
	}
	return rows
}

func (f *awaitingDecisionDBFixture) AssertSingleTerminal(decisionID int64) {
	f.t.Helper()
	var decisionStatus, instanceStatus, statusReason, chainState string
	var blockReason *string
	if err := f.tx.QueryRow(f.ctx, `
SELECT d.status, i.status, i.status_reason, i.strict_chain_state, i.chain_block_reason
FROM scheme_period_decisions d
JOIN scheme_instances i ON i.id = d.scheme_id
WHERE d.id = $1`, decisionID).Scan(&decisionStatus, &instanceStatus, &statusReason, &chainState, &blockReason); err != nil {
		f.t.Fatal(err)
	}
	switch decisionStatus {
	case "completed":
		if instanceStatus != "running" || chainState != "active" {
			f.t.Fatalf("completed instance status=%q chain=%q", instanceStatus, chainState)
		}
	case "missed_contiguous_period":
		if instanceStatus != "paused" || statusReason != "bet_failed" || chainState != "blocked_requires_rearm" {
			f.t.Fatalf("missed instance status=%q reason=%q chain=%q", instanceStatus, statusReason, chainState)
		}
		if blockReason == nil || *blockReason != "missed_contiguous_period" {
			f.t.Fatalf("missed instance block reason=%v", blockReason)
		}
	default:
		f.t.Fatalf("decision status=%q is not terminal", decisionStatus)
	}
}
