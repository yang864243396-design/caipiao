# Event-driven Scheme Betting Phase 1/2 Implementation Plan

> Status: approved by the repository architecture document. This plan deliberately stops before production bet dispatch.

**Goal:** Replace draw-triggered global polling with persisted provider facts and scoped strategy evaluation, then atomically create a shadow bet outbox command that can be observed and reconciled without calling the third party.

**Architecture:** PostgreSQL remains authoritative. Period API responses and draw ingestion are persisted as immutable facts. A draw notification scopes recovery to `lottery_code + period_no`; the existing frozen-rule evaluator advances scheme strategy and writes a decision plus a shadow outbox row in the same transaction. Existing real-bet polling remains active until a later single-lottery gray release explicitly enables dispatch.

**Tech Stack:** Go 1.25, pgx, PostgreSQL/goose, Vue 3/Vite admin.

---

### Task 1: Lock domain invariants with tests

**Files:**
- Create: `backend/internal/schemebetting/domain_test.go`
- Create: `backend/internal/schemebetting/domain.go`
- Create: `backend/internal/db/event_driven_migration_test.go`

1. Add failing tests for provider-snapshot target selection, safe deadline budget, deterministic request id/payload hash, shard stability, and legal outbox state transitions.
2. Add a migration contract test for decisions, outbox, attempts, provider snapshots, immutable draw metadata, unique keys, leases, and shadow mode.
3. Run `go test ./internal/schemebetting ./internal/db` and confirm the new tests fail for missing implementation/migration.
4. Implement only enough pure domain logic and schema to pass.

### Task 2: Persist provider period and draw facts

**Files:**
- Create: `backend/migrations/00154_event_driven_scheme_betting_shadow.sql`
- Create: `backend/internal/guaji/periodsync/persist.go`
- Create: `backend/internal/guaji/periodsync/persist_test.go`
- Modify: `backend/internal/guaji/periodsync/worker.go`
- Modify: `backend/internal/guaji/periodsync/syncer.go`
- Modify: `backend/internal/lottery/draw_persist.go`
- Modify: `backend/internal/guaji/drawsync/worker.go`
- Modify: `backend/internal/guaji/historysync/worker.go`

1. Test canonical snapshot construction and hashes from provider period rows.
2. Persist every valid provider period row as an append-only snapshot keyed by content hash.
3. Persist draw source, provider event id, receive/confirm timestamps, and canonical draw hash without overwriting the first accepted draw.
4. Record conflicting draw payloads as corrections for diagnosis; do not re-run an already completed decision.

### Task 3: Scope draw-triggered strategy processing

**Files:**
- Modify: `backend/internal/schemes/strategy_processor.go`
- Modify: `backend/internal/schemes/strategy_processor_test.go`
- Modify: `backend/internal/db/sqlcdb/worker_bet_ext.go`
- Modify: `backend/internal/guaji/historysync/worker.go`
- Modify: `backend/internal/server/server.go`

1. Add a failing test proving `NotifyDraw(lottery, period)` passes that exact scope to recovery.
2. Add a targeted pending-row query with a bounded recovery fallback for startup only.
3. Wire both WS and REST draw insertion paths to the notifier.
4. Keep notifications non-blocking and preserve orderly shutdown.

### Task 4: Atomically write decisions and shadow outbox

**Files:**
- Create: `backend/internal/db/sqlcdb/scheme_betting_outbox_ext.go`
- Modify: `backend/internal/schemes/strategy_processor.go`
- Modify: `backend/internal/schemes/strategy_processor_test.go`

1. Select the target only from the latest persisted provider snapshot whose open/close window is valid.
2. Calculate `safe_deadline_at` from explicit budgets; if no safe window exists, write a blocked/expired decision and no sendable command.
3. In the existing strategy transaction, advance state, write an immutable decision, and insert one `mode=shadow` outbox row with deterministic `request_id` and `payload_hash`.
4. Enforce unique `(scheme_id, source_period_no)` and `(scheme_id, target_period_no)` boundaries in PostgreSQL.

### Task 5: Expose read-only admin diagnostics

**Files:**
- Create: `backend/internal/handler/admin_scheme_betting_events.go`
- Modify: `backend/internal/server/server.go`
- Modify: admin API and scheme-monitor view files discovered during implementation.

1. Add a paged admin endpoint for recent decisions/outbox state, deadlines, age, source/target period, mode, and diagnostics.
2. Add a compact read-only diagnostics area to the existing all-site scheme monitor; do not add client changes.
3. Verify Chinese literals remain UTF-8.

### Task 6: Verify without enabling production dispatch

1. Run focused Go tests, then `go test ./...` under `backend`.
2. Run the admin type/build command and distinguish pre-existing errors from new errors.
3. Run migration/static checks and inspect `git diff --check` plus `git status`.
4. Confirm no production outbox dispatcher exists and no third-party bet API can consume `mode=shadow` rows.

