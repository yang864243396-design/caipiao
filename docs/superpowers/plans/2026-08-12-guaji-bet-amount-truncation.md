# Guaji Bet Amount Truncation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Use Guaji's two-decimal truncation as the single source of truth for every actual betting amount calculation, persistence path, and customer-facing display.

**Architecture:** Keep the rule close to each runtime boundary: Go owns actual scheme/order amounts and Guaji wire payloads; TypeScript owns client preflight and display formatting. Both sides calculate the raw product once and truncate toward zero at two decimals before any threshold check, comparison, persistence, aggregation, or displayed order amount.

**Tech Stack:** Go, PostgreSQL-backed scheme worker, Vue 3, TypeScript, Vitest.

## Global Constraints

- Actual bet amount is `trunc(注数 × 投注单位 × 倍数, 2)`; never round it.
- Apply the rule only to betting amounts, not balances, payouts, prizes, or profit/loss.
- Preserve the user's `0.001` unit configuration; only the computed monetary order amount is truncated.
- Do not alter existing historical records while implementing this rule.

---

### Task 1: Prove the backend amount boundary

**Files:**
- Modify: `backend/internal/guaji/accountsvc/place_bet_amount_test.go`
- Modify: `backend/internal/schemes/worker_play_test.go`
- Modify: `backend/internal/guaji/accountsvc/place_bet.go:15-42`
- Modify: `backend/internal/schemes/worker_config.go:1257-1270`

**Interfaces:**
- Produces: `roundLottBetAmount(unit float64, betsNums, mult int) float64` and `calcBetAmount(betUnits int, mult float64, unitYuan float64) float64`, both returning the actual two-decimal truncated amount.

- [ ] **Step 1: Write failing backend tests**

```go
if got := roundLottBetAmount(0.001, 176, 1); got != 0.17 {
    t.Fatalf("got %v want 0.17", got)
}
if got := calcBetAmount(179, 1, 0.001); got != 0.17 {
    t.Fatalf("got %v want 0.17", got)
}
```

- [ ] **Step 2: Run the focused tests and verify they fail because existing code rounds/preserves precision**

Run: `go test ./internal/guaji/accountsvc ./internal/schemes -run 'Test(RoundLottBetAmount|CalcBetAmount)' -count=1`

- [ ] **Step 3: Implement the minimum shared calculation behavior in each boundary**

```go
raw := unit * float64(units) * mult
return math.Floor((raw+1e-9)*100) / 100
```

Use the existing defaults before applying this calculation.

- [ ] **Step 4: Re-run focused backend tests**

Run: `go test ./internal/guaji/accountsvc ./internal/schemes -run 'Test(RoundLottBetAmount|CalcBetAmount)' -count=1`

### Task 2: Use the truncated backend amount end-to-end

**Files:**
- Modify: `backend/internal/schemes/worker_guaji.go:136-172`
- Modify: `backend/internal/schemes/worker.go:459-460,665-666`
- Modify: `backend/internal/schemes/detail_preview.go:139-140`
- Test: existing focused scheme/Guaji package tests

**Interfaces:**
- Consumes: the Task 1 amount functions.
- Produces: a single truncated `amount` for preflight, Guaji request data, order persistence, preview and P/L calculations that depend on the stake.

- [ ] **Step 1: Add a failing worker-level regression case**

```go
// A 176 × 0.001 order must enter the Guaji request and local order path as 0.17.
if meta.Amount != 0.17 { t.Fatalf("amount=%v want 0.17", meta.Amount) }
```

- [ ] **Step 2: Run the focused regression and verify failure**

Run: `go test ./internal/schemes ./internal/guaji/accountsvc -run 'Test.*(Trunc|Amount)' -count=1`

- [ ] **Step 3: Route all worker/order paths through the Task 1 calculation**

Do not add a second arithmetic implementation; use the calculated `amount` for max/min checks, Guaji payload input, persistence and preview/P/L stake calculations.

- [ ] **Step 4: Run focused backend packages**

Run: `go test ./internal/schemes ./internal/guaji ./internal/guaji/accountsvc -count=1`

### Task 3: Prove client preflight and display behavior

**Files:**
- Modify: `client/src/api/cloud/center.format.spec.ts`
- Create or modify: closest existing `client/src/utils/*` Vitest spec for bet amount formatting/calculation
- Modify: `client/src/utils/betPayload.ts:3259-3264`
- Modify: `client/src/utils/schemeMinBet.ts:153-173`
- Modify: `client/src/api/cloud/center.ts:118-121`

**Interfaces:**
- Produces: shared client functions that return two-decimal truncated numeric stakes and fixed-two-decimal display strings.

- [ ] **Step 1: Write failing client tests**

```ts
expect(calcBetAmount(176, 1, 0.001)).toBe(0.17)
expect(formatCloudSchemeTurnover(0.179)).toBe('0.17')
expect(formatCloudSchemeTurnover(0.1)).toBe('0.10')
```

- [ ] **Step 2: Run the focused tests and verify failure**

Run: `npm.cmd test -- client/src/api/cloud/center.format.spec.ts`

- [ ] **Step 3: Implement shared truncation before preflight or display formatting**

```ts
export function truncateBetAmount(n: number): number {
  return Number.isFinite(n) ? Math.floor((n + Number.EPSILON) * 100) / 100 : 0
}
```

Use it in `calcBetAmount`, `schemeMinSingleBetAmount`, and the card turnover formatter.

- [ ] **Step 4: Re-run focused client tests**

Run: `npm.cmd test -- client/src/api/cloud/center.format.spec.ts`

### Task 4: Replace every customer-facing actual bet amount formatter

**Files:**
- Modify: `client/src/api/orders/bets.ts:59-95`
- Modify: `client/src/api/cloud/betRecords.ts:186-197`
- Modify: `client/src/views/cloud/BetDetailView.vue:38-42`
- Modify: `client/src/views/play/GameDetailView.vue:909-919,946-948,1496-1499`
- Modify: `client/src/views/play/SchemeDetailView.vue:242-256`

**Interfaces:**
- Consumes: the Task 3 truncation/display functions.
- Produces: card, detail, cloud record, member record and preview amounts that match the actual Guaji stake.

- [ ] **Step 1: Add failing formatter assertions for 0.176 and 0.179**

```ts
expect(formatBetAmount(0.176)).toBe('0.17')
expect(toDisplayRow({ amount: 0.179, /* required fields */ }).amount).toBe('0.17')
```

- [ ] **Step 2: Run the relevant Vitest files and verify failure**

Run: `npm.cmd test -- client/src/api/cloud/center.format.spec.ts`

- [ ] **Step 3: Replace only actual-bet amount formatters/calculators**

Use the shared truncation helper before fixed-two-decimal rendering. Leave payout, P/L, wallet balances and prize formatting unchanged.

- [ ] **Step 4: Run focused client tests**

Run: `npm.cmd test -- client/src/api/cloud/center.format.spec.ts`

### Task 5: Full verification and commit

**Files:**
- Verify all files modified by Tasks 1-4.

- [ ] **Step 1: Format Go files**

Run: `gofmt -w internal/guaji/accountsvc/place_bet.go internal/schemes/worker_config.go`

- [ ] **Step 2: Run backend verification**

Run: `go test ./internal/guaji ./internal/guaji/accountsvc ./internal/schemes -count=1`

- [ ] **Step 3: Run client verification**

Run: `npm.cmd test -- client/src/api/cloud/center.format.spec.ts` and `npm.cmd run build`

- [ ] **Step 4: Inspect the staged diff and commit only intentional files**

Run: `git diff --check` then `git status --short`.

Commit only the implementation/tests/docs; keep the pre-existing generated `client/components.d.ts` change out of the commit.
