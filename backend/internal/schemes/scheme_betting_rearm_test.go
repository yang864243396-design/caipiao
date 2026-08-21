package schemes

import (
	"testing"
	"time"

	"caipiao/backend/internal/db/sqlcdb"
)

func TestManualRestartCreatesNewChainAndResetsRound(t *testing.T) {
	f := newFormalStrategyEntryFixture(t, "15s", fakedPastDrawTime())
	q := sqlcdb.New(f.pool)
	oldChain := "manual-rearm-old-chain"
	if _, err := f.pool.Exec(f.ctx, `
UPDATE scheme_instances
SET state_version = 7,
    round_index = 2,
    pick_index = 3,
    current_pick = '1,2,3',
    last_direction = 'neg',
    lookback_round_reset_pending = TRUE,
    strict_chain_state = 'blocked_requires_rearm',
    chain_block_reason = 'missed_contiguous_period',
    chain_id = $2,
    chain_seq = 4
WHERE id = $1`, f.schemeID, oldChain); err != nil {
		t.Fatal(err)
	}

	version, err := q.ResetSchemeStrategyForNewChain(f.ctx, f.schemeID, 7)
	if err != nil {
		t.Fatal(err)
	}
	if version != 8 {
		t.Fatalf("state version=%d, want 8", version)
	}
	if err := q.ActivateSchemeBettingChain(f.ctx, f.schemeID, "manual-rearm-new-chain", false); err != nil {
		t.Fatal(err)
	}

	var stateVersion int64
	var roundIndex, pickIndex int32
	var currentPick, lastDirection, chainID string
	var resetPending bool
	var chainBlockReason *string
	if err := f.pool.QueryRow(f.ctx, `
SELECT state_version, round_index, pick_index, current_pick, last_direction,
       lookback_round_reset_pending, chain_id, chain_block_reason
FROM scheme_instances WHERE id = $1`, f.schemeID).Scan(
		&stateVersion, &roundIndex, &pickIndex, &currentPick, &lastDirection,
		&resetPending, &chainID, &chainBlockReason,
	); err != nil {
		t.Fatal(err)
	}
	if stateVersion != 8 || roundIndex != 0 || pickIndex != 0 || currentPick != "" || lastDirection != "" || resetPending {
		t.Fatalf("manual restart state was not reset: version=%d round=%d pick=%d current=%q direction=%q reset=%v", stateVersion, roundIndex, pickIndex, currentPick, lastDirection, resetPending)
	}
	if chainID == oldChain || chainID != "manual-rearm-new-chain" || chainBlockReason != nil {
		t.Fatalf("manual restart chain was not replaced and unblocked: chain=%q reason=%v", chainID, chainBlockReason)
	}

	_, found, err := q.PendingFormalStrategyRowForSchemeDraw(f.ctx, f.recordID, f.schemeID, f.lotteryCode, f.periodNo, 7)
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("stale strategy-ready event matched reset state")
	}
}

func TestManualRestartBreaksAwaitingTargetWithoutCreatingOrder(t *testing.T) {
	f := newFormalStrategyEntryFixture(t, "15s", fakedPastDrawTime())
	f.preseedIncompleteDecision()
	q := sqlcdb.New(f.pool)

	if err := q.BreakAwaitingTargetRowsForNewChain(f.ctx, f.schemeID); err != nil {
		t.Fatal(err)
	}

	var status, reason string
	if err := f.pool.QueryRow(f.ctx, `
SELECT status, COALESCE(failure_reason, '')
FROM scheme_period_decisions
WHERE scheme_id = $1 AND source_period_no = $2`, f.schemeID, f.periodNo).Scan(&status, &reason); err != nil {
		t.Fatal(err)
	}
	if status != "chain_broken" || reason != "manual_rearm_replaced_chain" {
		t.Fatalf("awaiting target was not broken: status=%q reason=%q", status, reason)
	}
	var outboxCount int
	if err := f.pool.QueryRow(f.ctx, `SELECT COUNT(*)::int FROM scheme_bet_outbox WHERE scheme_id = $1`, f.schemeID).Scan(&outboxCount); err != nil {
		t.Fatal(err)
	}
	if outboxCount != 0 {
		t.Fatalf("breaking old awaiting target created %d outbox rows", outboxCount)
	}
}

func fakedPastDrawTime() time.Time { return time.Now().UTC() }
