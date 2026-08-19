# Event-driven Scheme Betting Phase 3-5 Implementation Plan

> Status: approved by `docs/event-driven-scheme-betting-architecture.md`. Production activation remains configuration-gated until the gray-lottery acceptance suite passes.

**Goal:** Complete the durable event bus, real outbox dispatcher, gray ownership switch, capacity controls, recovery and diagnostics without allowing an ambiguous third-party result to be retried.

**Safety boundary:** The current third-party bet API has no idempotency key and no reliable client correlation lookup. A request that may have reached the provider but lacks a confirmed unique order id becomes `external_acceptance_unknown`; it blocks the strict chain and is never sent again automatically.

### Task 1: Freeze production invariants

1. Test legal terminal outcomes including wrong-period acceptance and external acceptance unknown.
2. Test lease fencing, stale owner rejection, immutable request hash, and one-attempt-only dispatch.
3. Test strict chain interruption and explicit rearm semantics.

### Task 2: Extend the authoritative schema

1. Add chain state and betting owner to scheme instances.
2. Add frozen request, request metadata, external outcome and dispatch timestamps to outbox.
3. Add shard leases, immutable admin actions, capacity limits and required recovery indexes.
4. Keep formal rows impossible to dispatch unless their full request is frozen.

### Task 3: Add JetStream and database recovery

1. Publish draw and strategy-ready envelopes to fixed shard subjects with message ids.
2. Use durable consumers and explicit acknowledgements when JetStream is enabled.
3. Keep PostgreSQL as authority and scan due outbox rows after bus loss or restart.
4. Treat NATS as an optional accelerator; unavailable NATS must not fabricate progress.

### Task 4: Implement the fenced dispatcher

1. Lease by EDF with `FOR UPDATE SKIP LOCKED`, owner, fencing token and expiry.
2. Revalidate provider target and safe deadline immediately before send.
3. Persist a started attempt before I/O and perform exactly one provider call.
4. Record accepted, accepted-wrong-period, rejected, expired, or external-acceptance-unknown using fencing CAS.
5. Only confirmed accepted rows may enter the existing order/ledger finalization path.

### Task 5: Gray ownership, admission and operations

1. A lottery has exactly one real-bet owner: legacy or event.
2. Gray allowlists are explicit; shadow is the default and production cannot start from an empty allowlist.
3. Reject new event-owned starts when due capacity, account quota or worker availability is unsafe.
4. Add read-only diagnostics plus reason-required audit actions for rearm/cancel; no client changes.

### Task 6: Acceptance

1. Unit test duplicate/late/stale-fence/timeout/wrong-period cases.
2. Run database migration contract and integration tests.
3. Run focused packages, full compile, admin build and `git diff --check`.
4. Leave real dispatch disabled by default and document the external protocol limitation.
