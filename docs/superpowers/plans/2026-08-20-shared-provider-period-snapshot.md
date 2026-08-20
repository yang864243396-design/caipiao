# Shared Provider Period Snapshot Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove per-scheme synchronous provider-period refreshes from the formal betting critical path while preserving strict target-period validation and actionable failure evidence.

**Architecture:** Formal dispatch first reads the provider-derived in-memory schedule shared by lottery. A snapshot is eligible only while it is fresh, open, and later passes the dispatch safety-margin comparison; stale or missing snapshots fall back to one coalesced refresh keyed only by lottery. Ambiguous placement errors retain their phase-qualified original detail while the outbox state machine remains conservative.

**Tech Stack:** Go, PostgreSQL/pgx, existing `periodsync`, `lottery`, and `schemebetting` packages.

**Spec:** Confirmed in conversation on 2026-08-20: lottery-scoped snapshot, singleflight refresh, per-lottery isolation, no cross-period betting, and developer diagnostics.

## Global Constraints

- Never retry an outbound placement after the request may have been written.
- Never accept a cached target after its close time or beyond the existing dispatch safety margin.
- One lottery's refresh failure must not block other lotteries.
- Hot-path snapshot reads remain in memory; database writes stay at provider refresh cadence.
- Existing amount, multiplier, strategy, and third-party payload rules remain unchanged.

---

### Task 1: Lottery-scoped pre-place verification

**Files:**
- Modify: `backend/internal/guaji/periodsync/syncer.go`
- Modify: `backend/internal/guaji/periodsync/preplace_verify.go`
- Test: `backend/internal/guaji/periodsync/preplace_verify_test.go`

**Interfaces:**
- Consumes: `lottery.PeriodsScheduleFor`, `lottery.PeriodsScheduleFresh`, `lottery.PeriodsFallbackStaleAge`.
- Produces: `freshSharedOpenPeriod(lotteryCode string, now time.Time) (prePlaceVerifyResult, bool)` and lottery-keyed `prePlaceVerifyCache.getOrRefresh` use.

- [x] Add a failing test proving a fresh provider schedule returns immediately without a refresh callback.
- [x] Add a failing test proving callers for different members of one lottery share one in-flight refresh.
- [x] Run `go test ./internal/guaji/periodsync -run 'Test.*PrePlace' -count=1` and confirm RED.
- [x] Implement fresh-snapshot selection and change the refresh key from lottery-plus-member to lottery only.
- [x] Run the targeted tests and confirm GREEN.

### Task 2: Preserve placement failure evidence

**Files:**
- Modify: `backend/internal/schemebetting/dispatcher.go`
- Modify: `backend/internal/db/sqlcdb/scheme_betting_dispatch_ext.go`
- Modify: `backend/internal/schemebettingdispatch/runtime.go`
- Test: `backend/internal/schemebetting/dispatcher_test.go`
- Test: `backend/internal/schemebettingdispatch/runtime_test.go`

**Interfaces:**
- Extends: `FinishDispatch.ErrorDetail string`.
- Produces: phase-qualified `provider period verification:` and `provider placement:` errors; `scheme_bet_outbox.last_error` retains the original error after reconciliation deadline expiry.

- [x] Add a failing dispatcher test proving an ambiguous transport error is stored separately from the state-machine reason.
- [x] Add a failing transport test proving placement failures carry the placement phase while preserving `errors.Is`/`errors.As` behavior.
- [x] Run targeted tests and confirm RED.
- [x] Populate bounded error detail, persist it in outbox/attempt rows, and prevent deadline reconciliation from overwriting it.
- [x] Run targeted tests and confirm GREEN.

### Task 3: Verification

**Files:**
- Verify all files changed by Tasks 1 and 2.

- [x] Run `gofmt` on changed Go files.
- [x] Run `go test ./internal/guaji/periodsync ./internal/schemebetting ./internal/schemebettingdispatch -count=1`.
- [x] Run `go test ./... -count=1`.
- [x] Run `go build ./cmd/server`.
- [x] Run `go vet ./internal/guaji/periodsync ./internal/schemebetting ./internal/schemebettingdispatch`.
- [x] Run `git diff --check` and review the final diff for duplicate-bet and cross-period risks.
