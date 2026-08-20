# Event Bet-Ready Consumer and Safe Recovery Implementation Plan

> **For Codex:** Execute this plan task by task with test-driven development and verify every boundary before claiming completion.

**Goal:** Make the formal event-driven scheme chain place orders through JetStream without high-frequency 64-shard scans, and automatically recover only commands proven not to have reached the provider.

**Architecture:** `scheme_bet_outbox` remains the source of truth. A small `bet.ready.<shard>` JetStream event wakes an exact-ID fenced lease and dispatch; a low-frequency database sweep remains disaster recovery only. Expected period rotation and expired commands with no send attempt are rescheduled, while ambiguous post-write outcomes continue to block the chain.

**Tech Stack:** Go, PostgreSQL/pgx, NATS JetStream, existing scheme betting outbox and fencing model.

---

### Task 1: Add JetStream bet-ready consumption

- [x] Add failing tests for bet-ready identity and shard validation; reuse the established JetStream ack/retry wrapper.
- [x] Implement a durable queue consumer per shard in `internal/schemeeventbus`.
- [x] Verify the scheme event bus package tests.

### Task 2: Dispatch one exact outbox from an event

- [x] Add failing SQL/runtime tests for exact-ID leasing and stale/duplicate event idempotency.
- [x] Implement fenced exact-ID leasing and runtime event handling.
- [x] Return an explicit deferred result when capacity limiting releases a lease so JetStream retries instead of losing the wake-up.
- [x] Verify dispatcher and SQL query tests.

### Task 3: Separate hot event publication from low-frequency recovery scans

- [x] Add failing runtime cadence tests.
- [x] Keep unpublished-event discovery fast and bounded.
- [x] Move all-shard recovery dispatch to a low-frequency safety loop so normal throughput is event-driven.
- [x] Verify no per-scheme polling or sequential 64-shard hot loop remains.

### Task 4: Recover commands proven not sent

- [x] Extend the migrated-schema integration probe to cover the recoverable-expiry query chain.
- [x] Extend replacement recovery to safe-deadline expiry and dispatcher-loss-before-start outcomes.
- [x] Keep `sent_unknown` and other possible-write outcomes blocked for manual/provider reconciliation.
- [x] Verify period rollover creates at most one replacement command.

### Task 5: Recover old blocked event-owned instances safely

- [x] Add failing startup takeover tests for event-owned `blocked_requires_rearm` instances.
- [x] Automatically rearm only when the existing unresolved-bet guard proves no ambiguous order exists.
- [x] Keep unresolved/possibly-sent instances blocked.

### Task 6: Wire, verify, and hand off

- [x] Start one bet-ready consumer for each configured shard.
- [x] Run targeted package tests and build the backend server.
- [x] Review the diff for migration safety, concurrency bounds, and accidental unrelated changes.
- [x] Report required migration/restart steps and live verification evidence still needed on `192.168.20.2`.
