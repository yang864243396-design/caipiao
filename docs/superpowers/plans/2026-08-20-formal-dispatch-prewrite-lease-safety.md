# Formal Dispatch Prewrite And Lease Safety Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent pre-write network failures from blocking formal schemes while preserving duplicate-bet safety, and make every formal-dispatch lease decision use PostgreSQL time.

**Architecture:** The Guaji HTTP adapter records whether `httptrace.WroteRequest` completed without error and exposes that evidence through wrapped errors. The formal dispatcher derives its provider-operation timeout from a database-computed safe window, while leasing, attempt start, heartbeat, sweeping, and API-specific acquisition all compare timestamps with `clock_timestamp()` inside PostgreSQL.

**Tech Stack:** Go, `net/http/httptrace`, PostgreSQL, pgx/sqlc extensions, existing `schemebetting`, `schemebettingdispatch`, and `guaji` packages.

**Spec:** `docs/superpowers/specs/2026-08-20-formal-dispatch-prewrite-lease-safety-design.md`

## Global Constraints

- Never retry a provider placement after the request may have been written.
- A DNS, TCP, proxy-connect, or TLS failure before a successful `WroteRequest` callback must finish as a definite pre-send rejection and must not block the scheme chain.
- An error after a successful request write remains acceptance-unknown and blocks the strict chain.
- Formal dispatch must stop no later than the database-computed `safe_deadline_at` window or the earlier parent/global HTTP deadline.
- Lease acquisition, validity checks, attempt start, renewal, recovery, and expiry use PostgreSQL `clock_timestamp()`; application wall-clock timestamps are diagnostic only.
- Shared lottery-level provider-period snapshots remain unchanged; do not reintroduce per-scheme period polling.
- One failed outbox may consume only its own dispatcher concurrency slot and must not stop other shards, lotteries, schemes, or users.
- Do not change bet content, unit count, amount, multiplier, solo rules, lottery mapping, schema, or indexes.

---

### Task 1: Correct Guaji Request-Write Evidence

**Files:**
- Modify: `backend/internal/guaji/client.go`
- Test: `backend/internal/guaji/client_test.go`

**Interfaces:**
- Consumes: `httptrace.ClientTrace.WroteRequest(httptrace.WroteRequestInfo)`.
- Produces: wrapped errors implementing `interface { DefinitelyNotSent() bool }` only when no successful request-write callback occurred.

- [ ] **Step 1: Add a failing test for a failed write callback**

Add a package-local `RoundTripper` that calls the request trace with a non-nil `WroteRequestInfo.Err`, then returns a sentinel transport error. Assert `errors.As(err, &definitelyNotSent)` succeeds and `DefinitelyNotSent()` is true.

```go
func TestDoJSONRawWroteRequestCallbackErrorIsDefinitelyNotSent(t *testing.T) {
	transportErr := errors.New("tls prewrite failure")
	client := newTraceTestClient(roundTripFunc(func(req *http.Request) (*http.Response, error) {
		httptrace.ContextClientTrace(req.Context()).WroteRequest(httptrace.WroteRequestInfo{Err: transportErr})
		return nil, transportErr
	}))
	_, _, err := client.doJSONRaw(context.Background(), http.MethodPost, client.cfg.HTTPBase, "/api/web_bets/lott", "token", map[string]int{"game_id": 74})
	var marker definitelyNotSent
	if !errors.As(err, &marker) || !marker.DefinitelyNotSent() {
		t.Fatalf("expected definitely-not-sent marker, got %T %v", err, err)
	}
}
```

- [ ] **Step 2: Add a passing-write ambiguity test**

Call `WroteRequest` with `Err:nil`, return a timeout sentinel, and assert the error does not implement `DefinitelyNotSent`. This guards against unsafe retries after the transport accepted the write.

- [ ] **Step 3: Run the focused tests and confirm RED**

Run: `go test ./internal/guaji -run 'TestDoJSONRawWroteRequest' -count=1`

Expected: the callback-error test fails because the current callback sets `requestWritten=true` regardless of `WroteRequestInfo.Err`.

- [ ] **Step 4: Implement successful-write-only tracking**

Change the callback to:

```go
WroteRequest: func(info httptrace.WroteRequestInfo) {
	if info.Err == nil {
		requestWritten.Store(true)
	}
},
```

Keep `%w` wrapping and the existing `DefinitelyNotSent()` marker so `errors.Is` and `errors.As` continue to work through account-service wrappers.

- [ ] **Step 5: Run the focused package tests and confirm GREEN**

Run: `go test ./internal/guaji -count=1`

- [ ] **Step 6: Commit the task**

```powershell
git add -- backend/internal/guaji/client.go backend/internal/guaji/client_test.go
git commit -m "fix: preserve guaji prewrite evidence"
```

### Task 2: Propagate Pre-Send Evidence Through Formal Transport

**Files:**
- Modify: `backend/internal/schemebettingdispatch/runtime.go`
- Test: `backend/internal/schemebettingdispatch/runtime_test.go`

**Interfaces:**
- Consumes: placer errors implementing `interface { DefinitelyNotSent() bool }` through any `%w` wrapper.
- Produces: `Transport.PlaceOnce` errors that retain both phase timing and the same `DefinitelyNotSent` classification.

- [ ] **Step 1: Add a failing wrapped-marker test**

Configure the existing fake placer to return `fmt.Errorf("account place: %w", fakeDefinitelyNotSentError{err: sentinel})`. Call `Transport.PlaceOnce`, assert the returned text includes `provider placement failed`, and assert `errors.As` finds a true `DefinitelyNotSent` marker.

- [ ] **Step 2: Run the focused test and confirm RED**

Run: `go test ./internal/schemebettingdispatch -run 'TestTransport.*DefinitelyNotSent' -count=1`

Expected: marker assertion fails because the transport currently only promotes text-classified business rejections.

- [ ] **Step 3: Preserve marker classification before business classification**

Add a shared local interface and classify the wrapped placer error before `isDefinitiveProviderReject`:

```go
var preSend interface{ DefinitelyNotSent() bool }
if errors.As(err, &preSend) && preSend.DefinitelyNotSent() {
	return schemebetting.ProviderAcceptance{}, definitelyNotSentError{err: phaseErr}
}
if isDefinitiveProviderReject(err) {
	return schemebetting.ProviderAcceptance{}, definitelyNotSentError{err: phaseErr}
}
```

- [ ] **Step 4: Run focused and package tests and confirm GREEN**

Run: `go test ./internal/schemebettingdispatch -count=1`

- [ ] **Step 5: Commit the task**

```powershell
git add -- backend/internal/schemebettingdispatch/runtime.go backend/internal/schemebettingdispatch/runtime_test.go
git commit -m "fix: propagate formal pre-send failures"
```

### Task 3: Make Attempt Start Return A Database Safe Window

**Files:**
- Modify: `backend/internal/schemebetting/dispatcher.go`
- Modify: `backend/internal/db/sqlcdb/scheme_betting_dispatch_ext.go`
- Test: `backend/internal/schemebetting/dispatcher_test.go`
- Test: `backend/internal/db/sqlcdb/scheme_betting_dispatch_integration_test.go`

**Interfaces:**
- Produces: `schemebetting.AttemptStart{Started bool, SafeWindow time.Duration}`.
- Changes: `DispatchStore.StartAttempt(ctx context.Context, command LeasedCommand, leaseDuration time.Duration) (AttemptStart, error)`.
- Consumes: `SafeWindow` as a relative monotonic timeout for provider verification and placement.

- [ ] **Step 1: Add failing dispatcher safe-window tests**

Change the fake store to return a configurable `AttemptStart`. Add one test where `SafeWindow=25*time.Millisecond` and the fake transport blocks until `ctx.Done()`. Assert transport observes a deadline and dispatch completes as unknown, not by the host-clock `command.SafeDeadline` calculation.

- [ ] **Step 2: Add a failing database integration test**

Within a transaction, lease a fixture outbox and call `StartAttempt` with a deliberately irrelevant host timestamp removed from the API. Assert `Started=true`, `SafeWindow>0`, and `lease_until > clock_timestamp()`. Skip only when the test database or required fixture is unavailable; rollback all inserted/updated data.

- [ ] **Step 3: Run the focused tests and confirm RED/compile failure**

Run: `go test ./internal/schemebetting ./internal/db/sqlcdb -run 'Test.*(SafeWindow|StartAttempt)' -count=1`

Expected: compile failure until the new interface and return type exist.

- [ ] **Step 4: Add `AttemptStart` and use its timeout**

```go
type AttemptStart struct {
	Started    bool
	SafeWindow time.Duration
}
```

After a successful DB start, create `placeCtx` with `context.WithTimeout(ctx, start.SafeWindow)` and pass that context to `Transport.PlaceOnce`. The database CAS becomes authoritative for current lease and safe-deadline validity; retain command identity/hash checks before outbound work.

- [ ] **Step 5: Replace `StartAttempt` SQL with one database-clock statement**

Use a `db_now` CTE and a single fenced update/attempt insert. The statement must enforce:

```sql
lease_until > db_now.value
AND safe_deadline_at > db_now.value
```

and write:

```sql
dispatch_started_at = db_now.value,
lease_until = db_now.value + $lease_duration::interval
```

Return milliseconds from `safe_deadline_at - db_now.value`, convert to `time.Duration`, and return `AttemptStart{Started:true, SafeWindow:...}` only when the fenced statement succeeds.

- [ ] **Step 6: Run focused and package tests and confirm GREEN**

Run: `go test ./internal/schemebetting ./internal/db/sqlcdb -count=1`

- [ ] **Step 7: Commit the task**

```powershell
git add -- backend/internal/schemebetting/dispatcher.go backend/internal/schemebetting/dispatcher_test.go backend/internal/db/sqlcdb/scheme_betting_dispatch_ext.go backend/internal/db/sqlcdb/scheme_betting_dispatch_integration_test.go
git commit -m "fix: derive formal safe window from database"
```

### Task 4: Move Lease Acquisition, Heartbeat, And Sweeps To Database Time

**Files:**
- Modify: `backend/internal/schemebetting/dispatcher.go`
- Modify: `backend/internal/schemebetting/dispatcher_test.go`
- Modify: `backend/internal/schemebettingdispatch/runtime.go`
- Modify: `backend/internal/schemebettingdispatch/api_submit.go`
- Modify: `backend/internal/schemebettingdispatch/runtime_test.go`
- Modify: `backend/internal/db/sqlcdb/scheme_betting_dispatch_ext.go`
- Modify: `backend/internal/db/sqlcdb/scheme_betting_api_ext.go`
- Modify: `backend/internal/db/sqlcdb/scheme_betting_dispatch_integration_test.go`

**Interfaces:**
- Changes: `LeaseRenewStore.RenewLease(ctx context.Context, command LeasedCommand, leaseDuration time.Duration) (bool, error)`.
- Changes: `LeaseFormalOutboxParams` carries `LeaseDuration time.Duration`, not `Now`/`LeaseUntil`.
- Changes: `LeaseFormalOutboxByID(ctx, id, owner string, leaseDuration time.Duration)`.
- Changes: sweep/recovery methods accept only `rowLimit`; SQL obtains its own current time.
- Produces: heartbeat stop result containing the last renewal error and cancellation of the provider context on lease loss.

- [ ] **Step 1: Add failing lease-clock and fencing tests**

Add transaction-scoped integration assertions that acquisition and renewal set `lease_until` relative to `clock_timestamp()`, despite no host absolute timestamp argument. Renew with the correct token and assert success; renew with `token+1` and assert false.

- [ ] **Step 2: Add a failing heartbeat-loss test**

Configure the fake renewer to return false while a fake transport waits on context cancellation. Assert dispatch cancels the transport, preserves `lease heartbeat lost owner=... token=...` in `FinishDispatch.ErrorDetail`, and never performs another placement.

- [ ] **Step 3: Run focused tests and confirm RED/compile failure**

Run: `go test ./internal/schemebetting ./internal/schemebettingdispatch ./internal/db/sqlcdb -run 'Test.*(Lease|Heartbeat|Fencing|Clock)' -count=1`

- [ ] **Step 4: Convert acquisition and API leasing SQL**

Use `clock_timestamp()` for pending-candidate checks, safe-deadline checks, expired-lease checks, and both lease timestamps. Convert `time.Duration` to PostgreSQL interval text or microseconds in one helper so all SQL paths use the same conversion.

- [ ] **Step 5: Convert heartbeat and sweep SQL**

Renew only when owner/token match and `lease_until > clock_timestamp()`, then set `lease_until=clock_timestamp()+duration`. Make abandoned-start, expired-unstarted, and due-expiry queries compare and persist `clock_timestamp()` values internally, removing app `now` parameters.

- [ ] **Step 6: Cancel provider work on lost lease without weakening duplicate safety**

Run heartbeat with a child context passed to `Transport.PlaceOnce`. Return structured heartbeat evidence from the stop function. If cancellation happens before a successful request write, finish rejected; if the request may have been written, finish unknown. Append heartbeat evidence to the bounded original error instead of replacing it.

- [ ] **Step 7: Remove the run-loop cached timestamp as a lease authority**

Keep app `Now` only for diagnostics, limiter accounting, and finished-at metadata. Do not pass a single `now := time.Now()` through all 64 shard lease calls.

- [ ] **Step 8: Run package tests and confirm GREEN**

Run: `go test ./internal/schemebetting ./internal/schemebettingdispatch ./internal/db/sqlcdb -count=1`

- [ ] **Step 9: Commit the task**

```powershell
git add -- backend/internal/schemebetting backend/internal/schemebettingdispatch backend/internal/db/sqlcdb/scheme_betting_dispatch_ext.go backend/internal/db/sqlcdb/scheme_betting_api_ext.go backend/internal/db/sqlcdb/scheme_betting_dispatch_integration_test.go
git commit -m "fix: use database clock for formal dispatch leases"
```

### Task 5: Regression And Deployment Gate

**Files:**
- Verify: all files changed by Tasks 1-4
- Update checkboxes: `docs/superpowers/plans/2026-08-20-formal-dispatch-prewrite-lease-safety.md`

**Interfaces:**
- Verifies normal accepted, explicit business reject, wrong-period accept, pre-write failure, post-write ambiguity, heartbeat loss, and fencing behavior.

- [ ] **Step 1: Format all changed Go files**

Run `gofmt -w` on the exact changed `.go` files reported by `git diff --name-only`.

- [ ] **Step 2: Run targeted race-sensitive tests repeatedly**

```powershell
go test ./internal/guaji ./internal/schemebetting ./internal/schemebettingdispatch ./internal/db/sqlcdb -count=5
```

- [ ] **Step 3: Run full backend verification**

```powershell
go test ./... -count=1
go build ./cmd/server
go vet ./internal/guaji ./internal/schemebetting ./internal/schemebettingdispatch ./internal/db/sqlcdb
```

- [ ] **Step 4: Review repository safety checks**

Run `git diff --check`, `git status --short`, and review every changed hunk for accidental retries, cross-period acceptance, amount/rule changes, or per-scheme polling.

- [ ] **Step 5: Commit the verified implementation and plan state**

```powershell
git add -- docs/superpowers/plans/2026-08-20-formal-dispatch-prewrite-lease-safety.md backend
git commit -m "fix: harden formal dispatch transport boundary"
```

- [ ] **Step 6: Prepare controlled rollout commands without executing real-money recovery**

Document the exact checks for commit hash, one port-8080 process, NATS connectivity, dispatcher owner, and outbox `606`. Do not reject/rearm `606` or restart a live scheme until the user authorizes that state-changing rollout after deployment.
