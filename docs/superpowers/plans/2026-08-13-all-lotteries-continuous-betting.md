# All-Lotteries Continuous Betting Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make every running scheme bet each eligible open period without waiting for third-party payout settlement, while always using the immediately previous draw and recording developer-only root-cause diagnostics for missed periods.

**Architecture:** Split scheme progression into a draw-driven transaction and a financial-settlement transaction. A draw-driven worker advances round/pick state exactly once from the previous period's actual draw before the next bet; payout synchronization only persists third-party money. Per-scheme period claims prevent duplicate outbound bets, while structured diagnostics and a persistent table identify third-party versus platform failures.

**Tech Stack:** Go, PostgreSQL, pgx/sqlc-compatible handwritten query extensions, existing scheme Worker, Guaji period/history clients, Go `testing`.

## Global Constraints

- The behavior applies to every lottery using the shared scheme Worker; no lottery-name exception is allowed.
- A prior third-party order with status `pending` must never block the next period.
- The current bet must use the immediately previous draw; an older draw is never an acceptable fallback.
- If the previous draw remains unavailable through close, skip that period and persist a developer diagnostic.
- Financial settlement remains authoritative for order money, PnL, payout, wallet ledger, stop-profit, and stop-loss.
- Draw-driven progression controls round index, pick index, current pick, direction, and lookback state and is idempotent per scheme bet period.
- No client or admin UI changes are included.
- Existing unrelated working-tree changes must not be staged or committed.

---

## File Structure

- Create `backend/migrations/00145_scheme_draw_progress_and_diagnostics.sql`: strategy-progress columns, cross-server period claims, and persistent diagnostic table.
- Create `backend/internal/db/sqlcdb/scheme_progress_ext.go`: transaction-safe strategy advancement claims and effective-period claims.
- Create `backend/internal/db/sqlcdb/scheme_run_diagnostics_ext.go`: diagnostic upsert/list methods and row types.
- Create `backend/internal/schemes/worker_draw_progress.go`: adjacent-draw resolution and result-driven scheme-state advancement.
- Create `backend/internal/schemes/worker_draw_progress_test.go`: pure and DB-backed progression regression tests.
- Create `backend/internal/schemes/worker_continuous_bet_test.go`: pending-order, missing-draw, and one-period-one-bet integration tests.
- Create `backend/internal/schemes/worker_diagnostics.go`: diagnostic classification, persistence, and structured logging.
- Create `backend/internal/schemes/worker_diagnostics_test.go`: developer diagnostic classification tests.
- Create `backend/cmd/diag-scheme-runs/main.go`: developer CLI for querying recent missed-period diagnostics by scheme ID.
- Modify `backend/internal/schemes/worker.go`: remove the global pending gate and call draw progression before bet generation.
- Modify `backend/internal/schemes/worker_bet_dedup.go`: remove prior-settlement blocking while retaining current/effective-period deduplication.
- Modify `backend/internal/schemes/worker_trigger_repick.go`: remove the “use older draw near close” fallback.
- Modify `backend/internal/schemes/hot_cold_prev_draw.go`: expose strict adjacent-draw resolution for every run type.
- Modify `backend/internal/db/sqlcdb/worker_bet_ext.go`: stop treating unsettled bets as a global gate and add effective-period record lookups.
- Modify `backend/internal/cloud/schemestate/settle.go`: make the progression operation callable from draw processing without applying financial totals.
- Modify `backend/internal/guaji/accountsvc/payout_sync.go`: make formal payout commit financial-only and add local/third-party mismatch diagnostics callback.
- Modify `backend/internal/schemes/worker_sim_settle.go`: settle simulated money without advancing scheme state a second time.
- Modify `backend/internal/server/server.go`: wire the developer diagnostic store/probe into scheme and payout workers.

---

### Task 1: Persist strategy advancement, period claims, and diagnostics

**Files:**
- Create: `backend/migrations/00145_scheme_draw_progress_and_diagnostics.sql`
- Create: `backend/internal/db/sqlcdb/scheme_progress_ext.go`
- Create: `backend/internal/db/sqlcdb/scheme_run_diagnostics_ext.go`
- Test: `backend/internal/db/sqlcdb/scheme_progress_ext_test.go`

**Interfaces:**
- Produces: `TryAdvanceCloudBetStrategy(ctx, recordID, drawIssue string, hit bool) (bool, error)`.
- Produces: `TryClaimSchemeEffectivePeriod(ctx, schemeID, period string, recordID int64) (bool, error)`.
- Produces: `UpsertSchemeRunDiagnostic(ctx, UpsertSchemeRunDiagnosticParams) error`.
- Produces: `ListSchemeRunDiagnostics(ctx, schemeID string, limit int32) ([]SchemeRunDiagnostic, error)`.

- [ ] **Step 1: Write DB-backed failing tests**

Create tests that run in a rollback transaction and prove both claims are idempotent:

```go
func TestTryAdvanceCloudBetStrategyIsIdempotent(t *testing.T) {
    q, recordID, rollback := seedPendingCloudBet(t, "scheme-progress", "101")
    defer rollback()
    first, err := q.TryAdvanceCloudBetStrategy(context.Background(), recordID, "101", true)
    if err != nil || !first { t.Fatalf("first=%v err=%v", first, err) }
    second, err := q.TryAdvanceCloudBetStrategy(context.Background(), recordID, "101", true)
    if err != nil || second { t.Fatalf("second=%v err=%v", second, err) }
}

func TestTryClaimSchemeEffectivePeriodRejectsDuplicate(t *testing.T) {
    q, rollback := testQueries(t)
    defer rollback()
    ok, err := q.TryClaimSchemeEffectivePeriod(context.Background(), "scheme-1", "102", 1)
    if err != nil || !ok { t.Fatalf("first=%v err=%v", ok, err) }
    ok, err = q.TryClaimSchemeEffectivePeriod(context.Background(), "scheme-1", "102", 2)
    if err != nil || ok { t.Fatalf("duplicate=%v err=%v", ok, err) }
}
```

- [ ] **Step 2: Run the tests and verify RED**

Run: `cd backend && go test ./internal/db/sqlcdb -run 'TestTryAdvanceCloudBetStrategyIsIdempotent|TestTryClaimSchemeEffectivePeriodRejectsDuplicate' -count=1`

Expected: compilation fails because the new query methods do not exist.

- [ ] **Step 3: Add migration 00145**

Add these forward-schema elements and matching down statements:

```sql
ALTER TABLE cloud_bet_records
    ADD COLUMN IF NOT EXISTS strategy_advanced_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS strategy_hit BOOLEAN,
    ADD COLUMN IF NOT EXISTS strategy_draw_issue VARCHAR(32);

CREATE TABLE scheme_effective_period_claims (
    scheme_id VARCHAR(64) NOT NULL,
    period_no VARCHAR(64) NOT NULL,
    cloud_bet_record_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (scheme_id, period_no)
);

CREATE TABLE scheme_run_diagnostics (
    id BIGSERIAL PRIMARY KEY,
    scheme_id VARCHAR(64) NOT NULL,
    definition_id VARCHAR(64) NOT NULL DEFAULT '',
    member_id BIGINT NOT NULL,
    lottery_code VARCHAR(64) NOT NULL,
    target_period VARCHAR(64) NOT NULL,
    previous_period VARCHAR(64) NOT NULL DEFAULT '',
    actual_period VARCHAR(64) NOT NULL DEFAULT '',
    code VARCHAR(64) NOT NULL,
    responsibility VARCHAR(16) NOT NULL CHECK (responsibility IN ('third_party','platform','unknown')),
    stage VARCHAR(32) NOT NULL,
    detail TEXT NOT NULL DEFAULT '',
    draw_received BOOLEAN NOT NULL DEFAULT FALSE,
    draw_persisted BOOLEAN NOT NULL DEFAULT FALSE,
    settlement_status VARCHAR(32) NOT NULL DEFAULT '',
    first_occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    occurrence_count INTEGER NOT NULL DEFAULT 1,
    UNIQUE (scheme_id, target_period, code)
);
CREATE INDEX idx_scheme_run_diagnostics_scheme_last
    ON scheme_run_diagnostics (scheme_id, last_occurred_at DESC);
```

- [ ] **Step 4: Implement the DB helpers**

Use atomic `UPDATE ... WHERE strategy_advanced_at IS NULL` for advancement and `INSERT ... ON CONFLICT DO NOTHING RETURNING` for claims. Diagnostic upsert must update `last_occurred_at`, increment `occurrence_count`, and retain the latest detail/snapshot fields.

- [ ] **Step 5: Run migration and focused tests**

Run: `cd backend && go run ./cmd/migrate`

Run: `cd backend && go test ./internal/db/sqlcdb -run 'TestTryAdvanceCloudBetStrategyIsIdempotent|TestTryClaimSchemeEffectivePeriodRejectsDuplicate' -count=1`

Expected: migration succeeds and both tests pass.

- [ ] **Step 6: Commit**

```bash
git add backend/migrations/00145_scheme_draw_progress_and_diagnostics.sql backend/internal/db/sqlcdb/scheme_progress_ext.go backend/internal/db/sqlcdb/scheme_run_diagnostics_ext.go backend/internal/db/sqlcdb/scheme_progress_ext_test.go
git commit -m "feat: persist scheme draw progression diagnostics"
```

---

### Task 2: Resolve the strict immediately previous draw

**Files:**
- Modify: `backend/internal/schemes/hot_cold_prev_draw.go`
- Create: `backend/internal/schemes/worker_draw_progress.go`
- Create: `backend/internal/schemes/worker_draw_progress_test.go`

**Interfaces:**
- Produces: `type adjacentDrawResult struct { CurrentPeriod string; PreviousPeriod string; Draw sqlcdb.LotteryDraw; Ready bool; Missing bool }`.
- Produces: `loadAdjacentDraw(ctx context.Context, q *sqlcdb.Queries, lotteryCode, currentPeriod string) (adjacentDrawResult, error)`.
- Consumes: existing `prevIssueNo`, `hotColdAdjacentPrevMissing`, and lottery draw queries.

- [ ] **Step 1: Write failing table tests for adjacency**

Cover sequential issues, missing one issue, and rollover series:

```go
func TestResolveAdjacentDrawIssue(t *testing.T) {
    cases := []struct{ current, latest string; want string; ready bool }{
        {"85309831", "85309830", "85309830", true},
        {"85309831", "85309829", "85309830", false},
        {"6800040", "6791200", "6791200", true},
    }
    // Assert that a same-series gap never falls back to latest,
    // while a proven series rollover uses the latest real predecessor.
}
```

- [ ] **Step 2: Run and verify RED**

Run: `cd backend && go test ./internal/schemes -run TestResolveAdjacentDrawIssue -count=1`

Expected: compilation fails because strict adjacent resolution is not implemented.

- [ ] **Step 3: Implement strict resolution**

Return the exact numeric predecessor when it exists. If it does not exist, query `GetPreviousLotteryDrawByIssue`; accept that row only when `hotColdAdjacentPrevMissing(expected, latest)` proves a series rollover. Return `Ready=false, Missing=true` for a same-series gap. Database errors must return an error rather than being treated as ready.

- [ ] **Step 4: Remove the near-close fallback**

Change both `worker.go` and `worker_trigger_repick.go` callers so no run type can log `proceed without previous draw`. Missing adjacency returns a retry decision while the period remains open and a missed-period decision after it closes.

- [ ] **Step 5: Run focused tests**

Run: `cd backend && go test ./internal/schemes -run 'TestResolveAdjacentDrawIssue|TestHotColdAdjacentPrevMissing|TestHotColdPreviousDraw' -count=1`

Expected: all selected tests pass.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/schemes/hot_cold_prev_draw.go backend/internal/schemes/worker_draw_progress.go backend/internal/schemes/worker_draw_progress_test.go backend/internal/schemes/worker_trigger_repick.go backend/internal/schemes/worker.go
git commit -m "fix: require the immediately previous draw"
```

---

### Task 3: Advance scheme strategy from draws before the next bet

**Files:**
- Modify: `backend/internal/cloud/schemestate/settle.go`
- Modify: `backend/internal/schemes/schemestate_wire.go`
- Modify: `backend/internal/schemes/worker_draw_progress.go`
- Modify: `backend/internal/db/sqlcdb/scheme_progress_ext.go`
- Test: `backend/internal/schemes/worker_draw_progress_test.go`

**Interfaces:**
- Produces: `AdvanceFromDraw(ctx, q, inst, periodNo string, hit bool, localPnL float64, definitionConfig []byte, numericFromFloat func(float64) pgtype.Numeric) error` in `schemestate`.
- Produces: `advancePreviousBetFromDraw(ctx context.Context, q *sqlcdb.Queries, inst sqlcdb.SchemeInstance, def sqlcdb.SchemeDefinition, draw sqlcdb.LotteryDraw) (advanced bool, err error)`.
- Consumes: cloud bet content for the draw issue and `TryAdvanceCloudBetStrategy`.

- [ ] **Step 1: Write a failing progression test**

Seed a running scheme at round zero, a pending previous-period cloud bet, and a losing draw. Assert that draw progression moves round index once even though the cloud bet remains `pending`:

```go
func TestAdvancePreviousBetFromDrawDoesNotRequirePayout(t *testing.T) {
    fx := seedSchemeProgressFixture(t, "101", "102")
    advanced, err := fx.worker.advancePreviousBetFromDraw(fx.ctx, fx.q, fx.inst, fx.def, fx.draw101)
    if err != nil || !advanced { t.Fatalf("advanced=%v err=%v", advanced, err) }
    got := mustLoadInstance(t, fx.q, fx.inst.ID)
    if got.RoundIndex != 1 { t.Fatalf("round=%d want=1", got.RoundIndex) }
    if status := mustCloudStatus(t, fx.q, fx.recordID); status != "pending" {
        t.Fatalf("financial status changed to %s", status)
    }
    advanced, err = fx.worker.advancePreviousBetFromDraw(fx.ctx, fx.q, got, fx.def, fx.draw101)
    if err != nil || advanced { t.Fatalf("duplicate advanced=%v err=%v", advanced, err) }
}
```

- [ ] **Step 2: Run and verify RED**

Run: `cd backend && go test ./internal/schemes -run TestAdvancePreviousBetFromDrawDoesNotRequirePayout -count=1`

Expected: fails because progression still occurs only from payout synchronization.

- [ ] **Step 3: Separate strategy progression from financial statistics**

Extract the round/pick/lookback portion of `ProcessAfterSettlement` into `AdvanceFromDraw`. It must write zero turnover and zero financial PnL, use the local hit result, and retain the existing lookback reset behavior. Keep `ProcessAfterSettlement` temporarily as a compatibility wrapper until Task 5 removes its payout callers.

- [ ] **Step 4: Implement transactional previous-bet advancement**

Within one transaction: lock the scheme instance, find the cloud bet whose effective period equals the draw issue, evaluate its stored `bet_content` using `evaluatePlayHit`, call `TryAdvanceCloudBetStrategy`, then call `schemestate.AdvanceFromDraw`. If no scheme bet exists for the previous draw, return `advanced=false` without changing round or pick state.

- [ ] **Step 5: Verify RED-to-GREEN and existing state tests**

Run: `cd backend && go test ./internal/schemes -run 'TestAdvancePreviousBetFromDraw|TestSimRandomDrawAllFamiliesMultiPeriod' -count=1`

Run: `cd backend && go test ./internal/cloud/schemestate -count=1`

Expected: all tests pass; the previous cloud record remains financially pending.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/cloud/schemestate/settle.go backend/internal/schemes/schemestate_wire.go backend/internal/schemes/worker_draw_progress.go backend/internal/schemes/worker_draw_progress_test.go backend/internal/db/sqlcdb/scheme_progress_ext.go
git commit -m "feat: advance scheme state from lottery draws"
```

---

### Task 4: Remove the global unsettled-order gate for every lottery

**Files:**
- Modify: `backend/internal/schemes/worker.go`
- Modify: `backend/internal/schemes/worker_bet_dedup.go`
- Modify: `backend/internal/db/sqlcdb/worker_bet_ext.go`
- Modify: `backend/internal/schemes/worker_bet_dedup_claim_test.go`
- Create: `backend/internal/schemes/worker_continuous_bet_test.go`

**Interfaces:**
- Consumes: `loadAdjacentDraw` and `advancePreviousBetFromDraw` from Tasks 2–3.
- Produces: shared Worker behavior where only current/effective-period claims deduplicate; prior pending orders do not.

- [ ] **Step 1: Reverse the old blocking regression test**

Change the accepted-pending section of `TestHasUnsettledGuajiBetIgnoresHistoricalUnconfirmedClaim` into a new test named `TestAcceptedUnsettledBetDoesNotBlockNextPeriod`. It must assert:

```go
if w.hasUnsettledGuajiBet(ctx, inst) {
    t.Fatal("prior accepted pending bet must not block a different open period")
}
if dedup.Skip {
    t.Fatalf("prior pending period must not dedup current period: %+v", dedup)
}
```

- [ ] **Step 2: Add an end-to-end Worker test**

Seed period 101 as an accepted pending order, insert draw 101, expose open period 102, and run one Worker tick. Assert exactly one period-102 cloud record is created with round/pick state derived from draw 101.

- [ ] **Step 3: Run and verify RED**

Run: `cd backend && go test ./internal/schemes -run 'TestAcceptedUnsettledBetDoesNotBlockNextPeriod|TestWorkerBetsNextPeriodWhilePreviousPayoutPending' -count=1`

Expected: fails at the two `hasUnsettledGuajiBet` gates or `prior_third_party_pending` dedup branch.

- [ ] **Step 4: Remove only the global pending behavior**

Delete the calls at both the outer `tickInstance` gate and inner `placePeriodBet` gate. Remove `prior_third_party_pending` from `evaluateGuajiBetDedup`. Keep same-current-period checks, local claim checks, preflight claim handling, third-party bet-ID recovery, and effective-period checks.

- [ ] **Step 5: Insert strict draw progression before pick generation**

After final current-period dedup and before parsing the active round, load the adjacent previous draw. If ready, advance the previous bet in a transaction, reload the instance, and only then compute `roundIdx`, `betMult`, and pick content. If not ready, return without claiming the period so the next tick can retry.

- [ ] **Step 6: Run focused and package tests**

Run: `cd backend && go test ./internal/schemes -run 'TestAcceptedUnsettledBetDoesNotBlockNextPeriod|TestWorkerBetsNextPeriodWhilePreviousPayoutPending|TestSchemeUnsettledGuajiPeriodIncludesUnconfirmedClaim' -count=1`

Run: `cd backend && go test ./internal/schemes -count=1`

Expected: continuous-bet tests pass; same-period request-in-flight protection remains green.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/schemes/worker.go backend/internal/schemes/worker_bet_dedup.go backend/internal/db/sqlcdb/worker_bet_ext.go backend/internal/schemes/worker_bet_dedup_claim_test.go backend/internal/schemes/worker_continuous_bet_test.go
git commit -m "fix: allow continuous betting before payout settlement"
```

---

### Task 5: Make formal and simulated settlement financial-only

**Files:**
- Modify: `backend/internal/guaji/accountsvc/payout_sync.go`
- Modify: `backend/internal/schemes/worker_sim_settle.go`
- Modify: `backend/internal/cloud/schemestate/settle.go`
- Test: `backend/internal/guaji/accountsvc/payout_sync_test.go`
- Test: `backend/internal/schemes/worker_sim_settle_test.go`

**Interfaces:**
- Consumes: `strategy_hit` written by draw progression.
- Produces: payout commits that update order/cloud money exactly once without changing round/pick state.

- [ ] **Step 1: Write failing financial-only tests**

For formal and simulated settlement, snapshot `round_index`, `pick_index`, `current_pick`, and `last_direction`, run settlement, and assert all four values are unchanged while cloud status/PnL/payout are updated.

```go
before := mustLoadInstance(t, q, schemeID)
mustCommitThirdPartySettlement(t, worker, order)
after := mustLoadInstance(t, q, schemeID)
if before.RoundIndex != after.RoundIndex || before.PickIndex != after.PickIndex ||
   before.CurrentPick != after.CurrentPick || before.LastDirection != after.LastDirection {
    t.Fatalf("payout advanced strategy twice: before=%+v after=%+v", before, after)
}
```

- [ ] **Step 2: Run and verify RED**

Run: `cd backend && go test ./internal/guaji/accountsvc ./internal/schemes -run 'Test.*SettlementFinancialOnly' -count=1`

Expected: fails because both payout paths currently call `schemestate.ProcessAfterSettlement`.

- [ ] **Step 3: Remove strategy advancement from payout paths**

Delete `ProcessFormalAfterSettlement` from `PayoutSyncWorker.commitSettlement` and `ProcessAfterSettlement` from `settleSimCloudBet`. Retain cloud/bet-order status, actual PnL, payout, wallet mirror, cumulative scheme financial statistics, notifications, and stop-limit checks.

- [ ] **Step 4: Detect result disagreement**

When `strategy_hit` is present and differs from third-party `status == "win"`, call the diagnostic callback with code `platform_settlement_mismatch`, responsibility `unknown`, and include local hit plus third-party status. Do not rewrite already-used strategy state.

- [ ] **Step 5: Run focused tests**

Run: `cd backend && go test ./internal/guaji/accountsvc ./internal/schemes -run 'Test.*SettlementFinancialOnly|TestUpdateCloudBetRecordFromSettlementByID_idempotent' -count=1`

Expected: tests pass and money still settles exactly once.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/guaji/accountsvc/payout_sync.go backend/internal/guaji/accountsvc/payout_sync_test.go backend/internal/schemes/worker_sim_settle.go backend/internal/schemes/worker_sim_settle_test.go backend/internal/cloud/schemestate/settle.go
git commit -m "fix: keep payout settlement financial only"
```

---

### Task 6: Persist and classify developer-only missed-period diagnostics

**Files:**
- Create: `backend/internal/schemes/worker_diagnostics.go`
- Create: `backend/internal/schemes/worker_diagnostics_test.go`
- Modify: `backend/internal/schemes/worker.go`
- Modify: `backend/internal/guaji/historysync/worker.go`
- Modify: `backend/internal/server/server.go`

**Interfaces:**
- Produces: `type DrawIssueProbe interface { ProbeIssue(context.Context, string, string) (found bool, err error) }`.
- Produces: `recordMissedPeriod(ctx, inst, targetPeriod, previousPeriod string, localErr error)`.
- Consumes: `UpsertSchemeRunDiagnostic` from Task 1.

- [ ] **Step 1: Write failing classification tests**

Use a fake probe and assert exact codes/responsibilities:

```go
func TestClassifyMissingPreviousDraw(t *testing.T) {
    cases := []struct{
        name string; upstreamFound bool; upstreamErr error
        wantCode, wantOwner string
    }{
        {"upstream also missing", false, nil, "third_party_draw_missing", "third_party"},
        {"upstream has draw", true, nil, "platform_draw_sync_failed", "platform"},
        {"probe timeout", false, context.DeadlineExceeded, "third_party_draw_missing", "unknown"},
    }
    // Call classifyMissingPreviousDraw and assert exact code and owner.
}
```

- [ ] **Step 2: Run and verify RED**

Run: `cd backend && go test ./internal/schemes -run TestClassifyMissingPreviousDraw -count=1`

Expected: compilation fails because diagnostic classification is absent.

- [ ] **Step 3: Add an issue probe to history sync**

Implement `ProbeIssue(ctx, lotteryCode, issueNo)` using the lottery's configured REST history path and `FetchHistoryDrawLogs`. Return `found=true` only for an exact period match. For WS-only lotteries or network failures, return the original error so responsibility is `unknown` rather than falsely blaming the platform.

- [ ] **Step 4: Persist one diagnostic at period rollover**

Track the previous observed open period per scheme in Worker memory. When open period changes and the old period has no cloud bet record, classify the old period using its required previous draw, upsert one diagnostic, and emit a structured log containing `schemeId`, `lottery`, `targetPeriod`, `previousPeriod`, `code`, `responsibility`, `stage`, `drawPersisted`, `settlementStatus`, and `err`.

- [ ] **Step 5: Record internal and outbound failures at their source**

Map errors as follows:

```go
var diagnosticCodeByStage = map[string]string{
    "draw_read":     "platform_draw_sync_failed",
    "state_advance": "platform_state_advance_failed",
    "place_reject":  "third_party_place_rejected",
    "period_change": "third_party_period_rollover",
    "place_persist": "platform_place_persist_failed",
}
```

Persist diagnostics without pausing a scheme unless the existing business rule already requires a pause.

- [ ] **Step 6: Run tests**

Run: `cd backend && go test ./internal/schemes ./internal/guaji/historysync -run 'TestClassifyMissingPreviousDraw|Test.*ProbeIssue|Test.*MissedPeriodDiagnostic' -count=1`

Expected: classification, deduplication, and exact-period probing pass.

- [ ] **Step 7: Commit**

```bash
git add backend/internal/schemes/worker_diagnostics.go backend/internal/schemes/worker_diagnostics_test.go backend/internal/schemes/worker.go backend/internal/guaji/historysync/worker.go backend/internal/server/server.go
git commit -m "feat: record scheme missed-period diagnostics"
```

---

### Task 7: Harden cross-period and cross-server duplicate protection

**Files:**
- Modify: `backend/internal/schemes/worker.go`
- Modify: `backend/internal/schemes/worker_bet_dedup.go`
- Modify: `backend/internal/db/sqlcdb/scheme_progress_ext.go`
- Modify: `backend/internal/schemes/worker_guaji_meta_test.go`
- Modify: `backend/internal/schemes/worker_parallel_test.go`

**Interfaces:**
- Consumes: `TryClaimSchemeEffectivePeriod` from Task 1.
- Produces: one outbound attempt per scheme/effective third-party period across concurrent workers.

- [ ] **Step 1: Write failing concurrent and rollover tests**

Run two workers for one scheme/current period and assert only one effective-period claim succeeds. Add a rollover case where a target-N request is accepted as N+1; the later N+1 tick must not call the fake Guaji placer.

- [ ] **Step 2: Run and verify RED**

Run: `cd backend && go test ./internal/schemes -run 'TestConcurrentWorkersClaimOneEffectivePeriod|TestAcceptedRolloverPreventsNextPeriodDuplicate' -count=1`

Expected: the rollover case exposes duplicate risk after removing the pending gate.

- [ ] **Step 3: Claim target and accepted periods**

Before outbound, claim the current target period in `scheme_effective_period_claims`. After a successful response, claim the returned period before final metadata persistence. If the returned period is already claimed, preserve the real accepted order, attach it to its original target record, emit `third_party_period_rollover`, and ensure later ticks see the effective claim.

- [ ] **Step 4: Serialize one scheme's outbound request across servers**

Acquire a PostgreSQL session advisory lock derived from `scheme_id` before final dedup and hold it through claim, outbound request, and accepted metadata persistence. Release it in `defer` on the same acquired connection. A worker that cannot acquire the lock returns and retries; it does not create a second claim.

- [ ] **Step 5: Run concurrency tests repeatedly**

Run: `cd backend && go test ./internal/schemes -run 'TestConcurrentWorkersClaimOneEffectivePeriod|TestAcceptedRolloverPreventsNextPeriodDuplicate|TestParallel' -count=20`

Expected: all repetitions pass with one outbound call per effective period.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/schemes/worker.go backend/internal/schemes/worker_bet_dedup.go backend/internal/db/sqlcdb/scheme_progress_ext.go backend/internal/schemes/worker_guaji_meta_test.go backend/internal/schemes/worker_parallel_test.go
git commit -m "fix: prevent duplicate bets across period rollover"
```

---

### Task 8: Add the developer diagnostic command and run full verification

**Files:**
- Create: `backend/cmd/diag-scheme-runs/main.go`
- Test: `backend/internal/db/sqlcdb/scheme_run_diagnostics_ext_test.go`
- Modify: `docs/superpowers/specs/2026-08-13-all-lotteries-continuous-betting-design.md` only if implementation names differ, without changing approved behavior.

**Interfaces:**
- Consumes: `ListSchemeRunDiagnostics` from Task 1.
- Produces: `go run ./cmd/diag-scheme-runs -scheme <instance-id> -limit 20`.

- [ ] **Step 1: Write the diagnostic query test**

Insert diagnostics for two schemes and assert filtering, newest-first ordering, and occurrence-count upsert behavior.

- [ ] **Step 2: Run and verify RED**

Run: `cd backend && go test ./internal/db/sqlcdb -run TestListSchemeRunDiagnostics -count=1`

Expected: fails until list/query behavior and fixtures are complete.

- [ ] **Step 3: Implement the CLI**

Parse `-scheme` as required and `-limit` defaulting to 20. Print one line per event:

```text
2026-08-13T08:15:30Z scheme=def... target=85309831 previous=85309830 owner=platform code=platform_draw_sync_failed stage=draw_read count=2 detail="..."
```

Exit non-zero for a missing scheme argument or database error. The command is developer-only and registers no HTTP route.

- [ ] **Step 4: Run focused package tests**

Run: `cd backend && go test ./internal/db/sqlcdb ./internal/cloud/schemestate ./internal/guaji/historysync ./internal/guaji/accountsvc ./internal/schemes -count=1`

Expected: all selected packages pass.

- [ ] **Step 5: Run full backend tests and build**

Run: `cd backend && go test ./... -count=1`

Run: `cd backend && go build ./cmd/server ./cmd/diag-scheme-runs`

Expected: both commands exit 0. If unrelated pre-existing failures remain, record exact package/test names and do not claim full green.

- [ ] **Step 6: Verify migration and live-safe diagnostics**

Run: `cd backend && go run ./cmd/migrate`

Run: `cd backend && go run ./cmd/diag-scheme-runs -scheme def-1-1786596387063 -limit 20`

Expected: migration is idempotent; the command either lists persisted diagnostics or prints an empty result without modifying the scheme.

- [ ] **Step 7: Review diff hygiene**

Run: `git diff --check`

Run: `git status --short`

Expected: no whitespace errors; only files from this plan plus pre-existing unrelated dirty files are present.

- [ ] **Step 8: Commit**

```bash
git add backend/cmd/diag-scheme-runs/main.go backend/internal/db/sqlcdb/scheme_run_diagnostics_ext_test.go docs/superpowers/specs/2026-08-13-all-lotteries-continuous-betting-design.md
git commit -m "chore: add scheme run diagnostic command"
```

---

## Final Acceptance Checklist

- [ ] A real pending order from period N does not block the same scheme from betting N+1.
- [ ] N+1 uses draw N and the round/pick state advanced from N exactly once.
- [ ] A missing draw N never falls back to N-1 or an older draw.
- [ ] Missing draw, platform processing, third-party reject, and accepted-period rollover each produce a developer diagnostic with an explicit responsibility.
- [ ] Multiple pending real orders can settle out of order without advancing strategy twice.
- [ ] One scheme cannot place two bets in one effective third-party period, including multiple-server workers.
- [ ] No client/admin UI files changed.
- [ ] Focused tests, full backend tests, migration, build, and diff checks have fresh recorded results.
