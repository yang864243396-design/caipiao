# Stale Pending Settlement Recovery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Keep historical accepted pending third-party bets recoverable without allowing them to block later scheme periods.

**Architecture:** The scheme worker keeps duplicate prevention scoped to its current strict third-party period. The payout synchronizer keeps historical pending records and searches the third-party list beyond the recent three-page window using a bounded, rotating page cursor. The admin runtime diagnostic exposes accepted pending records and their current blocking classification.

**Tech Stack:** Go, PostgreSQL, pgx, existing Guaji HTTP client, Go testing.

## Global Constraints

- Real-money settlement and payout are sourced only from confirmed third-party responses.
- Accepted third-party bets are never deleted, recreated, or placed again.
- Same-current-period duplicate prevention remains mandatory.
- Existing untracked process documents are not included in this feature commit.

---

### Task 1: Period-scoped pending blocker

**Files:**
- Modify: `backend/internal/schemes/worker_bet_dedup.go`
- Test: `backend/internal/schemes/worker_bet_dedup_claim_test.go`

**Interfaces:**
- Consumes: `thirdPartyOpenPeriod`, `SchemeUnsettledGuajiPeriod`, `SchemeHasAcceptedUnsettledGuajiBet`.
- Produces: `hasUnsettledGuajiBet` that blocks same or ambiguous periods but releases a strictly older accepted period.

- [ ] Write failing tests for a strictly earlier accepted period and an ambiguous accepted period.
- [ ] Run `go test ./internal/schemes -run TestHasUnsettledGuajiBet -count=1` and observe Red.
- [ ] Implement the minimal period comparison and rerun the focused test to Green.

### Task 2: Historical third-party settlement lookup

**Files:**
- Modify: `backend/internal/guaji/bets.go`
- Test: `backend/internal/guaji/bets_test.go`

**Interfaces:**
- Consumes: `ListWebBets(ctx, token, limit, page)`.
- Produces: a bounded lookup that finds an accepted historical ID on a page after the three recent pages.

- [ ] Write a failing HTTP-client test with target ID only on page four.
- [ ] Run `go test ./internal/guaji -run TestGetWebBetFindsAcceptedHistoricalBetWithinBoundedPages -count=1` and observe Red.
- [ ] Implement the bounded historical lookup and rerun the focused test to Green.

### Task 3: Recovery cursor and diagnostics

**Files:**
- Modify: `backend/internal/guaji/accountsvc/payout_sync.go`
- Modify: `backend/internal/handler/admin_runtime_diagnostics.go`
- Add: `backend/migrations/00148_guaji_settlement_recovery.sql`
- Test: `backend/internal/guaji/accountsvc/payout_sync_test.go`
- Test: `backend/internal/handler/admin_runtime_diagnostics_test.go`

**Interfaces:**
- Consumes: bounded historical lookup and pending order rows.
- Produces: rotating recovery state and diagnostic `acceptedPending` entries.

- [ ] Write failing tests for cursor advancement after a not-found search and for non-blocking older pending metadata.
- [ ] Run the two focused tests and observe Red.
- [ ] Implement bounded recovery and read-only diagnostic fields, then rerun the focused tests to Green.

### Task 4: Verification and delivery

- [ ] Run `go test ./internal/schemes ./internal/guaji ./internal/guaji/accountsvc ./internal/handler -count=1`.
- [ ] Run `go build ./cmd/server`.
- [ ] Run `git diff --check`.
- [ ] Commit only feature code, regression tests, and the two design documents.
