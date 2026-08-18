# Short-Period Phase-Lock Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent 3-second lottery schemes from permanently skipping every period when fresh periods and WebSocket snapshots already agree on a safe target period.

**Architecture:** Add one pure shared-snapshot predicate beside the existing member-level period verification decision. For periods no longer than 15 seconds, matching fresh periods and WS snapshots with more than the existing 1.2-second safety margin bypass the extra synchronous member verification; all existing final pre-place checks remain unchanged.

**Tech Stack:** Go 1.24+, standard library `time`, existing in-memory `lottery.PeriodsSchedule` and `lottery.PeriodState` snapshots.

## Global Constraints

- Keep `guajiPlaceCloseSafety` at 1.2 seconds.
- Do not use WS alone to select a target period; periods and WS must both match the requested target.
- Do not add per-scheme polling, HTTP requests, database migrations, or UI changes.
- Stale, missing, mismatched, or unsafe snapshots must retain the existing synchronous verification/safe-skip behavior.
- Ordinary periods longer than 15 seconds retain the current near-close verification behavior.
- Third-party accepted-period mismatch handling remains unchanged.

---

## File Map

- `backend/internal/schemes/worker_guaji_period_verify.go` — decide whether shared short-period snapshots can safely replace member-level synchronous verification.
- `backend/internal/schemes/worker_bet_dedup_test.go` — reproduce the 3-second phase lock and protect mismatch, unsafe-window, and normal-period boundaries.

### Task 1: Bypass Redundant Short-Period Member Verification

**Files:**
- Modify: `backend/internal/schemes/worker_guaji_period_verify.go:17-33`
- Modify: `backend/internal/schemes/worker_bet_dedup_test.go:159-181`

**Interfaces:**
- Produces: `guajiShortPeriodSharedSnapshotsAllowDirectPlaceAt(lotteryCode, targetPeriod string, now time.Time) bool`.
- Changes: `guajiPrePlaceVerificationNeeded` returns false when that predicate succeeds.
- Preserves: `guajiBetPeriodMatches`, `guajiBetPeriodHasSafeWindowAt`, `guajiShortPeriodWSWindowAllowsAt`, and accepted-period mismatch handling.

- [ ] **Step 1: Write the failing phase-lock regression test**

Add this test with unique lottery state:

```go
func TestGuajiPrePlaceVerificationSkipsMemberProbeWhenShortPeriodSnapshotsAgree(t *testing.T) {
    code := "guaji_preplace_short_period_phase_lock_test"
    now := time.Now().UTC()
    lottery.ClearPeriodsSchedule(code)
    t.Cleanup(func() { lottery.ClearPeriodsSchedule(code) })

    closeAt := now.Add(1800 * time.Millisecond)
    lottery.UpdatePeriodsScheduleFullWithDuration(code, "P2", "P2", now, closeAt, 3, "", now)
    lottery.UpdatePeriodState(code, "P1", "P2", closeAt.Add(-3*time.Second), 3)

    if guajiPrePlaceVerificationNeeded(code, "P2", now) {
        t.Fatal("matching fresh short-period snapshots must not spend the remaining window on member verification")
    }
}
```

- [ ] **Step 2: Run the regression test and verify RED**

Run from `backend`:

```powershell
go test ./internal/schemes -run '^TestGuajiPrePlaceVerificationSkipsMemberProbeWhenShortPeriodSnapshotsAgree$' -count=1
```

Expected: FAIL because the current `guajiPrePlaceVerificationNeeded` returns true whenever 1.8 seconds is below the 2.4-second verification lead.

- [ ] **Step 3: Add mismatch, unsafe-window, and ordinary-period safety tests**

Use table-driven setup or three focused subtests that assert verification remains required when:

```go
// WS points to P3 while periods target P2.
lottery.UpdatePeriodState(code, "P1", "P3", closeAt.Add(-3*time.Second), 3)

// Both sources target P2 but only the exact 1.2-second safety boundary remains.
closeAt = now.Add(guajiPlaceCloseSafety)

// A 60-second period is near the existing 2.4-second verification lead.
lottery.UpdatePeriodsScheduleFullWithDuration(code, "P2", "P2", now, now.Add(guajiPrePlaceVerificationLead), 60, "", now)
```

Each case must assert `guajiPrePlaceVerificationNeeded(...) == true`.

- [ ] **Step 4: Implement the minimal shared-snapshot predicate**

Add this function and call it before the near-close return in `guajiPrePlaceVerificationNeeded`:

```go
func guajiShortPeriodSharedSnapshotsAllowDirectPlaceAt(lotteryCode, targetPeriod string, now time.Time) bool {
    ps, periodsOK := lottery.PeriodsScheduleFor(lotteryCode)
    state, stateOK := lottery.PeriodStateFor(lotteryCode)
    if !periodsOK || !stateOK || !guajiBetSnapshotFreshAt(ps, now) {
        return false
    }
    intervalSec := state.IntervalSec
    if intervalSec <= 0 || intervalSec > 15 {
        return false
    }
    targetPeriod = strings.TrimSpace(targetPeriod)
    if strings.TrimSpace(ps.CurrentPeriod) != targetPeriod || strings.TrimSpace(state.NextIssue) != targetPeriod {
        return false
    }
    if ps.CloseAt.UTC().Sub(now.UTC()) <= guajiPlaceCloseSafety {
        return false
    }
    return guajiShortPeriodWSWindowAllowsAt(lotteryCode, targetPeriod, now)
}
```

In `guajiPrePlaceVerificationNeeded`, after the existing fresh/matching periods checks and before the near-close decision:

```go
if guajiShortPeriodSharedSnapshotsAllowDirectPlaceAt(lotteryCode, targetPeriod, now) {
    return false
}
```

- [ ] **Step 5: Run focused tests and verify GREEN**

Run from `backend`:

```powershell
go test ./internal/schemes -run 'TestGuaji(PrePlaceVerification|ShortPeriodWSWindow|BetPeriodHasSafeWindow)' -count=1
```

Expected: PASS.

- [ ] **Step 6: Run regression verification**

Run from `backend`:

```powershell
go test ./internal/schemes ./internal/guaji ./internal/guaji/accountsvc ./internal/handler -count=1
go test ./... -skip '^TestGetCloudCenterStatsSeededE2E$' -count=1
go build ./cmd/server
```

Expected: PASS. The shared-data-mutating `TestGetCloudCenterStatsSeededE2E` remains explicitly skipped.

- [ ] **Step 7: Check patch hygiene and commit**

Run from the repository root:

```powershell
git diff --check
git status --short
git add -- backend/internal/schemes/worker_guaji_period_verify.go backend/internal/schemes/worker_bet_dedup_test.go docs/superpowers/plans/2026-08-18-short-period-phase-lock-fix.md
git commit -m "fix: avoid short-period verification phase lock"
```

The six unrelated untracked 2026-08-12 documents must remain unstaged and unchanged.

- [ ] **Step 8: Controlled live verification after deployment authorization**

After the updated backend is deployed and restarted, call the existing admin runtime diagnostic for `inst-1-1786600039439` and inspect new records. Confirm that at least one eligible period places without `period_no != third_party_period`; never force a bet into a closed period.
