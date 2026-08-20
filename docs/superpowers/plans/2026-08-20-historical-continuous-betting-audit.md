# Historical Continuous Betting Branch Audit Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Determine whether every patch-unique commit on `feature/all-lotteries-continuous-betting` is already represented by the current event-driven formal betting architecture, and port only genuinely missing compatible behavior.

**Architecture:** Treat current `master` as authoritative: formal strategy progression is persisted through `scheme_strategy_evaluations`, period decisions through `scheme_period_decisions`, and provider submission through `scheme_bet_outbox`. Historical worker-loop code and migration `00145` must not be merged as a parallel state machine; useful behavior must be mapped to the current bounded, idempotent, lease-fenced components.

**Tech Stack:** Go, PostgreSQL, pgx/sqlc extensions, Git worktrees, Go tests.

**Spec:** `docs/superpowers/specs/2026-08-13-all-lotteries-continuous-betting-design.md`, reconciled with `docs/superpowers/specs/2026-08-20-formal-dispatch-prewrite-lease-safety-design.md`

## Global Constraints

- Do not merge `feature/all-lotteries-continuous-betting` wholesale.
- Do not restore the retired worker-owned formal betting state machine.
- Preserve one formal order per scheme and target period through the current outbox uniqueness and lease fencing.
- Do not execute live betting, restart services, modify production data, or rearm blocked instances during this audit.
- Generated `.cache/` files remain local-only and must never be committed.

---

### Task 1: Remove generated worktree cache safely

**Files:**
- Local only: `.runtime/git-excludes`
- Remove generated directory: `.worktrees/cloud-center-nats-realtime/.cache`

**Interfaces:**
- Consumes: existing Git worktree layout.
- Produces: a clean auxiliary worktree and a repository-local ignore rule for `.cache/`.

- [x] **Step 1: Resolve and validate the exact cache target**

Run `Resolve-Path` for the cache and its parent worktree and verify the target starts with the resolved worktree path.

- [x] **Step 2: Delete only the validated generated cache directory**

Run `Remove-Item -LiteralPath <validated-cache-path> -Recurse -Force` and verify `Test-Path` returns `False`.

- [x] **Step 3: Configure the local exclude file**

Create `.runtime/git-excludes` containing `.cache/`, configure `core.excludesFile`, and verify with `git check-ignore -v .cache/probe.tmp`.

### Task 2: Classify every patch-unique historical commit

**Files:**
- Read: historical commits in `master..feature/all-lotteries-continuous-betting`
- Read: `backend/internal/schemes/strategy_processor.go`
- Read: `backend/internal/schemebetting/dispatcher.go`
- Read: `backend/internal/schemebettingdispatch/runtime.go`
- Read: `backend/internal/schemebettingdispatch/finalizer.go`
- Read: `backend/migrations/00149_play_rule_specs.sql`
- Read: `backend/migrations/00154_event_driven_scheme_betting_shadow.sql`
- Read: `backend/migrations/00155_event_driven_scheme_betting_dispatch.sql`

**Interfaces:**
- Consumes: historical commit patches and current architecture.
- Produces: one disposition per commit: equivalent, superseded, documentation-only, or port-required.

- [x] **Step 1: Enumerate patch-unique commits**

Run `git cherry master feature/all-lotteries-continuous-betting` and exclude entries marked `-` because their patches already exist on `master`.

- [x] **Step 2: Compare persisted state models**

Compare historical migration `00145` and its query extensions with current migrations `00149`, `00154`, and `00155`; reject any port that creates a second strategy or dispatch authority.

- [x] **Step 3: Compare behavioral guarantees**

Map immediate-previous-draw gating, payout-independent strategy progression, period deduplication, advisory locking, missed-period diagnostics, and provider-call lease reuse to current code and tests.

- [x] **Step 4: Record the disposition**

Append an audit table to this document naming each patch-unique commit, its current equivalent, and whether a port is required.

#### Audit result

`git cherry master feature/all-lotteries-continuous-betting` reports two patch-equivalent documentation commits (`20caa6b`, `1c4748d`) and the following sixteen patch-unique commits. None may be cherry-picked into the current formal path because their worker-loop ownership has been retired.

| Historical commit | Historical responsibility | Current authoritative equivalent | Disposition |
| --- | --- | --- | --- |
| `6947ec7` | Ignore local worktrees | Root `.gitignore` already contains `.worktrees/`; generated `.cache/` is covered by the repository-local exclude file | Equivalent; no port |
| `d4cebdd` | Persist draw progression and run diagnostics in migration `00145` | `scheme_strategy_evaluations` (`00149`), `scheme_period_decisions` and `scheme_bet_outbox` (`00154`) | Superseded schema; no port |
| `e825595` | Add old diagnostics to generated sqlc model | Current event tables use focused sqlc extensions and admin read models | Superseded data access; no port |
| `65051f5` | Require the immediately previous draw in the worker loop | Accepted order period is joined to the exact `lottery_draws.issue_no`; formal progression consumes that frozen order and draw in `strategy_processor.go` | Superseded execution owner; no port |
| `a944fcd` | Avoid using an older draw when an expected draw is missing | Scoped strategy recovery only processes the exact notified lottery and period; target selection requires a fresh open provider snapshot | Superseded execution owner; no port |
| `33fba86` | Advance scheme state from a draw independently of payout | `StrategyProcessor.process` evaluates the frozen rule and calls `ProcessStrategyAfterDraw` inside an idempotent strategy evaluation transaction | Equivalent; no port |
| `febc542` | Preserve accepted bet content for later strategy evaluation | `PendingFormalStrategyRow` reads the accepted record's `bet_content`, frozen rule snapshot, version and hash | Equivalent; no port |
| `ca343ab` | Permit next-period strategy work while provider payout is pending | Pending accepted formal rows are eligible for strategy processing once their exact draw exists; payout status is not the strategy gate | Equivalent; no port |
| `941eab3` | Keep payout settlement financial-only | `StrategyProcessor` owns strategy state; acceptance/finalization and payout recovery own order and financial persistence | Equivalent separation; no port |
| `22f554f` | Persist missed-period diagnostics | Admin runtime, strategy and event diagnostics expose period refresh, previous draw, queue, attempt, deadline and strategy timing evidence | Superseded diagnostics; no port |
| `75c7678` | Harden old diagnostic and lease interactions | Current diagnostics query immutable decision/outbox/attempt records rather than sharing a worker advisory-lock session | Superseded diagnostics; no port |
| `ffc21d8` | Prevent duplicate bets during period rollover | Unique decision/request/target-period identities plus outbox state CAS prevent duplicate formal dispatch | Stronger current invariant; no port |
| `248f10b` | Harden worker advisory-lock cleanup | Formal dispatch uses bounded row leases, fencing tokens and database-clock expiry instead of a session advisory lock | Superseded concurrency model; no port |
| `751131b` | Keep all worker queries on the advisory-lock lease | Strategy shard leases and dispatcher outbox leases fence each current authority independently | Superseded concurrency model; no port |
| `6b63b8f` | Reuse the worker scheme lease during provider placement | Dispatcher starts one persisted attempt, renews its fenced lease during I/O and performs exactly one provider call | Stronger current invariant; no port |
| `d677fbf` | Add a CLI for the old run-diagnostics table | Admin runtime, strategy, event-list and event-summary endpoints query current authoritative tables | Superseded operator interface; no port |

The historical arithmetic-based adjacent-period helper is intentionally not restored. Current formal dispatch treats provider period snapshots as authoritative because provider period identifiers are not universally safe to increment or decrement. A stale or unsafe provider target is rejected or recorded through the current strict-chain/outbox diagnostics rather than reviving the old worker state machine.

### Task 3: Port only confirmed gaps with tests first

**Files:**
- Modify only files identified by Task 2.
- Test in the owning Go package for every ported behavior.

**Interfaces:**
- Consumes: Task 2 entries marked `port-required`.
- Produces: event-driven implementations with regression coverage, or no source changes when no compatible gap exists.

- [x] **Step 1: Write the smallest failing regression test for each confirmed gap**

Run the package test and verify it fails for the missing behavior rather than for setup or database connectivity.

No entry was classified `port-required`, so no new regression test is needed.

- [x] **Step 2: Implement the behavior in the current authority**

Use current strategy evaluation, outbox, dispatcher, and finalizer boundaries. Do not call the historical worker-loop implementation.

No source implementation is required.

- [x] **Step 3: Run the targeted test to green**

Run `go test ./internal/<owning-package> -count=1` and verify exit code `0`.

Task 4 verifies the existing authoritative packages because Task 3 makes no source-code change.

### Task 4: Verify and prepare delivery

**Files:**
- Verify all changed files and this audit document.

**Interfaces:**
- Consumes: completed audit and any selective ports.
- Produces: a clean, evidence-backed delivery candidate.

- [x] **Step 1: Run focused event-driven tests**

Run `go test ./internal/schemes ./internal/schemebetting ./internal/schemebettingdispatch ./internal/db/sqlcdb -count=1`.

- [x] **Step 2: Run the complete backend test suite**

Run `go test ./... -count=1` from `backend` and verify exit code `0`.

- [x] **Step 3: Build the server and inspect the diff**

Run `go build ./cmd/server`, `git diff --check`, and `git status --short`.

- [x] **Step 4: Commit only after user-visible audit results are available**

Stage exact paths, commit with a message describing the audit or selective port, then push only when requested or already authorized.
