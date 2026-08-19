# Core Online Partition Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move the three high-volume financial tables to monthly PostgreSQL partitions with online backfill, global uniqueness, validation, cutover, and rollback.

**Architecture:** Stable identity registries preserve global keys while partitioned mirrors use date-inclusive primary keys. Trigger-based synchronization makes batch backfill online; a guarded table-name swap preserves every existing SQL query.

**Tech Stack:** Go 1.x, PostgreSQL, Goose, pgx, Vue 3, TypeScript.

**Spec:** `docs/superpowers/specs/2026-08-19-core-online-partition-design.md`

## Global Constraints

- Existing application SQL table names must not change.
- Wallet ledger remains append-only.
- Cutover is forbidden unless exact validation passes.
- Rollback remains available while reverse synchronization is enabled.

---

### Task 1: Migration contract

**Files:**
- Create: `backend/internal/db/core_online_partition_migration_test.go`
- Create: `backend/migrations/00165_core_online_partition_prepare.sql`

- [ ] Write a static test requiring three identity registries, three partition parents,
  monthly/default partitions, forward synchronization, batch backfill, and validation.
- [ ] Run `go test ./internal/db -run TestCoreOnlinePartitionMigration -count=1`
  and confirm it fails because migration 165 does not exist.
- [ ] Implement migration 165 and rerun the test to green.

### Task 2: Guarded cutover and rollback

**Files:**
- Modify: `backend/internal/db/core_online_partition_migration_test.go`
- Create: `backend/migrations/00166_core_online_partition_cutover.sql`

- [ ] Extend the failing contract test with cutover, reverse sync, prepared-statement
  restart guard, and rollback requirements.
- [ ] Implement `cutover_core_online_partitions()`,
  `validate_core_online_partitions()`, and `rollback_core_online_partitions()`.
- [ ] Re-run the migration contract tests.

### Task 3: Operational command

**Files:**
- Create: `backend/cmd/core-partition/main.go`
- Create: `backend/cmd/core-partition/main_test.go`

- [ ] Write failing tests for command validation and safe state transitions.
- [ ] Implement `status`, `backfill`, `validate`, `cutover`, and `rollback`.
- [ ] Require `--confirm-cutover` and `--confirm-rollback` for table swaps.
- [ ] Run `go test ./cmd/core-partition -count=1`.

### Task 4: Database integration

**Files:**
- Create: `backend/internal/db/core_online_partition_integration_test.go`

- [ ] Test catalog shape and current migration state against the configured database.
- [ ] Test idempotent backfill and exact validation without mutating business facts.
- [ ] Test uniqueness and append-only constraints inside rollback-only transactions.
- [ ] Run the integration test after applying migrations 165 and 166.

### Task 5: Admin diagnostics

**Files:**
- Create: `backend/internal/handler/admin_core_partition.go`
- Modify: `backend/internal/handler/handler.go`
- Modify: `backend/internal/server/server.go`
- Modify: `admin/src/api/schemeBettingDiagnostics.ts`
- Modify: `admin/src/components/schemes/SchemeBettingEventsPanel.vue`

- [ ] Write a failing handler integration assertion for partition status.
- [ ] Add a read-only admin endpoint with phase, counts, missing rows, amount deltas,
  partition coverage, and last validation time.
- [ ] Display the status in the existing scheme betting diagnostics panel.
- [ ] Run focused handler tests and the admin production build.

### Task 6: Apply and verify

- [ ] Apply Goose migrations and confirm version 166.
- [ ] Run bounded backfill until all three tables report zero missing rows.
- [ ] Stop the backend, validate, cut over, restart, and verify health.
- [ ] Verify reverse synchronization and mark rollback ready.
- [ ] Run focused Go tests, full Go compile, vet, admin build, client tests, and
  `git diff --check`.

