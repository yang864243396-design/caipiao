# Short-Period Settlement and Strategy Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the per-bet balance request from formal scheme execution, freeze the correct published play rule, and expose enough payout-sync evidence to distinguish provider delay from local processing failure without blocking other schemes.

**Architecture:** Scheme start keeps the existing synchronous third-party balance preflight, while each running bet goes directly to `PlaceLottBet` and treats only an explicit provider balance message as insufficient funds. Published-rule lookup uses catalogue sub-play ID first and semantic sub-play ID only as fallback; payout synchronization records a concurrency-safe in-memory account snapshot that the existing admin runtime diagnostic endpoint reads without making a new provider request.

**Tech Stack:** Go 1.24+, PostgreSQL/pgx/sqlc, Guaji REST/WebSocket, standard-library `sync.RWMutex`, existing admin diagnostics API.

## Global Constraints

- Query third-party available balance exactly once in the existing scheme-start preflight; do not query balance before each running bet.
- Keep third-party `settled`, `net_amount`, and `payout_amount` as the only authority for formal financial settlement.
- Use WebSocket draw plus the immutable frozen rule only for local strategy progression; do not synthesize provider financial results.
- Never backfill historical formal orders whose `rule_snapshot` is `NULL` with the current rule version.
- Do not add per-scheme period or payout polling; retain account-batched payout synchronization and lottery-level period snapshots.
- Do not force a bet into a closed or different period; accepted-period mismatch remains visible and pauses the affected scheme.
- Diagnostics are admin/developer-only, read-only, in-memory, and must not trigger a provider request.
- Add no database migration and make no client/admin UI change for this fix.
- Every behavior change starts with a failing test, then the smallest implementation, targeted verification, and a focused commit.

---

## File Map

- `backend/internal/schemes/worker.go` — resolve and freeze published rules using catalogue ID before semantic mode ID.
- `backend/internal/schemes/rule_evaluator_test.go` — cover catalogue-first resolution and semantic fallback.
- `backend/internal/guaji/errclass.go` — identify explicit insufficient-balance provider errors without relying on generic code `40000`.
- `backend/internal/guaji/errclass_test.go` — cover positive and negative balance-error messages.
- `backend/internal/guaji/accountsvc/place_bet.go` — remove running-bet `UserInfo` preflight and map provider balance rejection to `guajibet.ErrInsufficient`.
- `backend/internal/guaji/accountsvc/place_bet_policy_test.go` — verify provider rejection mapping for insufficient balance, period closure, and generic amount errors.
- `backend/internal/guaji/accountsvc/payout_diagnostics.go` — own the concurrency-safe account-level payout diagnostic snapshots.
- `backend/internal/guaji/accountsvc/payout_diagnostics_test.go` — verify snapshot isolation, success/error transitions, and concurrent access.
- `backend/internal/guaji/accountsvc/service.go` — initialize the payout diagnostic store and expose a read-only snapshot getter.
- `backend/internal/guaji/accountsvc/payout_sync.go` — return list-fetch errors, record each account attempt, and count settled/unsettled rows.
- `backend/internal/guaji/accountsvc/payout_batch_test.go` — verify batch failures are not silently converted to success and counters are correct.
- `backend/internal/handler/admin_runtime_diagnostics.go` — attach the scheme account's in-memory payout snapshot to the existing runtime response.
- `backend/internal/handler/admin_runtime_diagnostics_test.go` — verify account selection and diagnostic response shaping remain local/read-only.

### Task 1: Resolve and Freeze the Correct Published Rule

**Files:**
- Modify: `backend/internal/schemes/worker.go:101-116`
- Modify: `backend/internal/schemes/rule_evaluator_test.go:89-114`

**Interfaces:**
- Consumes: `playRule.CatalogSubID`, `playRule.SubPlayID`, and `playrules.RegistryStore.Resolve(playrules.Locator)`.
- Produces: `publishedRuleSubIDs(rule playRule) []string`, returning trimmed unique candidates in catalogue-first order.
- Preserves: `freezePublishedRule` returns without mutation when no published candidate exists.

- [ ] **Step 1: Write failing catalogue-first and fallback tests**

```go
func TestWorkerResolvesPublishedRuleByCatalogSubIDBeforeSemanticMode(t *testing.T) {
    registry, err := playrules.NewRegistry([]playrules.PublishedSpec{
        {Locator: playrules.Locator{TemplateCode: "fast_ssc_std", TypeID: "g017", SubID: "390"}, RuleVersion: 7, EvaluatorVersion: 1, EvaluatorKey: "ssc.attribute", EvaluationSpec: []byte(`{"mode":"attribute","catalogSubId":"390"}`), StrategyEnabled: true},
        {Locator: playrules.Locator{TemplateCode: "fast_ssc_std", TypeID: "g017", SubID: "daxiao"}, RuleVersion: 3, EvaluatorVersion: 1, EvaluatorKey: "ssc.attribute", EvaluationSpec: []byte(`{"mode":"attribute","catalogSubId":"legacy"}`), StrategyEnabled: true},
    })
    if err != nil { t.Fatal(err) }
    worker := &Worker{ruleRegistry: playrules.NewRegistryStore(registry)}
    got, ok := worker.resolvePublishedRule("tron_ffc_3s", playRule{PlayTemplate: "fast_ssc_std", PlayTypeID: "g017", CatalogSubID: "390", SubPlayID: "daxiao"})
    if !ok || got.Locator.SubID != "390" || got.RuleVersion != 7 {
        t.Fatalf("snapshot=%+v ok=%v, want catalogue rule 390 version 7", got, ok)
    }
}

func TestWorkerResolvesPublishedRuleBySemanticFallbackWithoutCatalogID(t *testing.T) {
    registry, err := playrules.NewRegistry([]playrules.PublishedSpec{
        {Locator: playrules.Locator{TemplateCode: "fast_ssc_std", TypeID: "g017", SubID: "daxiao"}, RuleVersion: 3, EvaluatorVersion: 1, EvaluatorKey: "ssc.attribute", EvaluationSpec: []byte(`{"mode":"attribute","catalogSubId":"390"}`), StrategyEnabled: true},
    })
    if err != nil { t.Fatal(err) }
    worker := &Worker{ruleRegistry: playrules.NewRegistryStore(registry)}
    got, ok := worker.resolvePublishedRule("tron_ffc_3s", playRule{PlayTemplate: "fast_ssc_std", PlayTypeID: "g017", SubPlayID: "daxiao"})
    if !ok || got.Locator.SubID != "daxiao" || got.RuleVersion != 3 {
        t.Fatalf("snapshot=%+v ok=%v, want semantic fallback daxiao version 3", got, ok)
    }
}
```

- [ ] **Step 2: Run the focused test and verify RED**

Run from `backend`: `go test ./internal/schemes -run "TestWorkerResolvesPublishedRuleBy(CatalogSubIDBeforeSemanticMode|SemanticFallbackWithoutCatalogID)" -count=1`

Expected: the first test fails because the current resolver asks only for `SubPlayID=daxiao`.

- [ ] **Step 3: Implement catalogue-first unique candidate resolution**

```go
func publishedRuleSubIDs(rule playRule) []string {
    seen := make(map[string]struct{}, 2)
    out := make([]string, 0, 2)
    for _, raw := range []string{rule.CatalogSubID, rule.SubPlayID} {
        id := strings.TrimSpace(raw)
        if id == "" { continue }
        if _, exists := seen[id]; exists { continue }
        seen[id] = struct{}{}
        out = append(out, id)
    }
    return out
}
```

Loop over these candidates in `resolvePublishedRule`, keeping the same template, type, and lottery locator fields. Return immediately on the first successful resolve and return `(playrules.Snapshot{}, false)` only after all candidates fail.

- [ ] **Step 4: Verify snapshot resolution and commit**

Run from `backend`: `go test ./internal/schemes -run "TestWorkerResolvesPublishedRule|TestFrozenFastSSCHashTailBigSmall" -count=1`

Expected: PASS, with catalogue rule `390` selected before semantic `daxiao`.

Commit: `git add -- backend/internal/schemes/worker.go backend/internal/schemes/rule_evaluator_test.go && git commit -m "fix: resolve frozen rules by catalog subplay"`

### Task 2: Remove Runtime Balance Preflight and Preserve Provider Error Semantics

**Files:**
- Modify: `backend/internal/guaji/errclass.go`
- Modify: `backend/internal/guaji/errclass_test.go`
- Modify: `backend/internal/guaji/accountsvc/place_bet.go:132-215`
- Create: `backend/internal/guaji/accountsvc/place_bet_policy_test.go`
- Verify unchanged: `backend/internal/schemes/instance_start.go:203-230`
- Verify unchanged: `backend/internal/schemes/instance_start_balance_test.go`

**Interfaces:**
- Produces: `guaji.IsInsufficientBalanceError(err error) bool`.
- Produces: `placeLottBetBusinessError(err error) error`, returning `guajibet.ErrInsufficient`, `guajibet.ErrPeriodClosed`, or `nil` for handling by existing token/generic-error logic.
- Preserves: `schemes.Service.ensureUsableBalanceForStart` remains the one synchronous balance check before a scheme becomes running.

- [ ] **Step 1: Write failing provider-error classification tests**

```go
func TestIsInsufficientBalanceErrorRequiresExplicitBalanceMeaning(t *testing.T) {
    cases := []struct { name string; err error; want bool }{
        {"chinese", &APIError{Code: 40000, Message: "可用余额不足"}, true},
        {"english", &APIError{Code: 40000, Message: "insufficient balance"}, true},
        {"same code amount error", &APIError{Code: 40000, Message: "投注金额不正确"}, false},
        {"same code count error", &APIError{Code: 40000, Message: "投注注数不正确"}, false},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            if got := IsInsufficientBalanceError(tc.err); got != tc.want { t.Fatalf("got %v want %v", got, tc.want) }
        })
    }
}
```

Add this test in `place_bet_policy_test.go`:

```go
func TestPlaceLottBetBusinessError(t *testing.T) {
    cases := []struct { name string; err error; want error }{
        {"insufficient", &guaji.APIError{Code: 40000, Message: "余额不足"}, guajibet.ErrInsufficient},
        {"closed", &guaji.APIError{Code: 40000, Message: "已过投注截止时间"}, guajibet.ErrPeriodClosed},
        {"amount", &guaji.APIError{Code: 40000, Message: "投注金额不正确"}, nil},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            got := placeLottBetBusinessError(tc.err)
            if tc.want == nil && got != nil { t.Fatalf("got %v want nil", got) }
            if tc.want != nil && !errors.Is(got, tc.want) { t.Fatalf("got %v want %v", got, tc.want) }
        })
    }
}
```

- [ ] **Step 2: Run the focused tests and verify RED**

Run from `backend`: `go test ./internal/guaji ./internal/guaji/accountsvc -run "TestIsInsufficientBalanceError|TestPlaceLottBetBusinessError" -count=1`

Expected: FAIL because both helpers are absent.

- [ ] **Step 3: Implement strict message-based balance detection**

```go
func IsInsufficientBalanceError(err error) bool {
    if err == nil { return false }
    msg := err.Error()
    var api *APIError
    if errors.As(err, &api) { msg = api.Message }
    lower := strings.ToLower(strings.TrimSpace(msg))
    for _, phrase := range []string{"余额不足", "可用余额不足", "insufficient balance", "balance insufficient"} {
        if strings.Contains(lower, strings.ToLower(phrase)) { return true }
    }
    return false
}
```

Do not check `APIError.Code` in this helper.

- [ ] **Step 4: Remove only the per-bet `UserInfo` call and map the place response**

Delete the `UserInfo`/`BalanceByCurrency` block from `placeRealBetWithRow`. Keep token decryption, game/rule/content validation, one `PlaceLottBet` request, token invalidation, period mismatch recording, and third-party bet-ID validation unchanged. Before generic error classification, apply:

```go
func placeLottBetBusinessError(err error) error {
    switch {
    case guaji.IsInsufficientBalanceError(err):
        return guajibet.ErrInsufficient
    case guaji.IsPeriodClosedError(err):
        return guajibet.ErrPeriodClosed
    default:
        return nil
    }
}
```

The post-settlement `UserInfo` refresh in `payout_sync.go` remains because it is outside the outbound betting critical path.

- [ ] **Step 5: Verify runtime and start behavior, then commit**

Run from `backend`: `go test ./internal/guaji ./internal/guaji/accountsvc ./internal/schemes -run "TestIsInsufficientBalanceError|TestPlaceLottBetBusinessError|TestEnsureUsableBalanceForStart|TestShouldDeferGuajiPlaceError" -count=1`

Expected: PASS; the existing start-balance tests still prove start preflight behavior.

Commit: `git add -- backend/internal/guaji/errclass.go backend/internal/guaji/errclass_test.go backend/internal/guaji/accountsvc/place_bet.go backend/internal/guaji/accountsvc/place_bet_policy_test.go && git commit -m "fix: remove per-bet guaji balance preflight"`

### Task 3: Record Account-Level Payout Synchronization Diagnostics

**Files:**
- Create: `backend/internal/guaji/accountsvc/payout_diagnostics.go`
- Create: `backend/internal/guaji/accountsvc/payout_diagnostics_test.go`
- Modify: `backend/internal/guaji/accountsvc/service.go:26-42`
- Modify: `backend/internal/guaji/accountsvc/payout_sync.go:50-212,339-362`
- Modify: `backend/internal/guaji/accountsvc/payout_batch_test.go`

**Interfaces:**
- Produces: exported immutable `PayoutSyncDiagnostics` JSON model.
- Produces: `func (s *Service) PayoutSyncDiagnostics(accountID int64) (PayoutSyncDiagnostics, bool)`.
- Internal store methods: `begin(accountID int64, pending int, at time.Time)`, `succeed(accountID, providerList, settled, unresolved int, at time.Time)`, and `fail(accountID int64, err error, at time.Time)`.
- Changes: `syncHistoricalOne(...) (settled bool, err error)` so the batch can count successful historical settlements.

- [ ] **Step 1: Write failing diagnostic state tests**

```go
func TestPayoutDiagnosticStoreTracksFailureThenRecovery(t *testing.T) {
    store := newPayoutDiagnosticStore()
    t1 := time.Unix(100, 0).UTC()
    store.begin(9, 4, t1)
    store.fail(9, errors.New("tls handshake timeout"), t1.Add(time.Second))
    failed, ok := store.snapshot(9)
    if !ok || failed.PendingCount != 4 || failed.LastError == "" || failed.LastErrorAt == nil { t.Fatalf("failed=%+v", failed) }

    t2 := t1.Add(10 * time.Second)
    store.begin(9, 4, t2)
    store.succeed(9, 50, 2, 2, t2.Add(time.Second))
    recovered, _ := store.snapshot(9)
    if recovered.LastError != "" || recovered.LastSuccessAt == nil || recovered.SettledCount != 2 || recovered.ProviderUnsettledCount != 2 { t.Fatalf("recovered=%+v", recovered) }
}
```

Add a concurrent reader/writer test using 100 goroutines:

```go
func TestPayoutDiagnosticStoreConcurrentSnapshots(t *testing.T) {
    store := newPayoutDiagnosticStore()
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(n int) {
            defer wg.Done()
            now := time.Unix(int64(n+1), 0).UTC()
            store.begin(9, n, now)
            store.succeed(9, n, n/2, n-n/2, now)
            if got, ok := store.snapshot(9); ok && (got.PendingCount < 0 || got.SettledCount < 0 || got.ProviderUnsettledCount < 0) {
                t.Errorf("invalid snapshot: %+v", got)
            }
        }(i)
    }
    wg.Wait()
}
```

- [ ] **Step 2: Run diagnostic tests and verify RED**

Run from `backend`: `go test ./internal/guaji/accountsvc -run "TestPayoutDiagnosticStore" -count=1`

Expected: FAIL because the diagnostic store does not exist.

- [ ] **Step 3: Implement the bounded in-memory snapshot store**

```go
type PayoutSyncDiagnostics struct {
    AccountID               int64      `json:"accountId"`
    LastAttemptAt           *time.Time `json:"lastAttemptAt,omitempty"`
    LastSuccessAt           *time.Time `json:"lastSuccessAt,omitempty"`
    LastError               string     `json:"lastError,omitempty"`
    LastErrorAt             *time.Time `json:"lastErrorAt,omitempty"`
    PendingCount            int        `json:"pendingCount"`
    ProviderListCount       int        `json:"providerListCount"`
    SettledCount            int        `json:"settledCount"`
    ProviderUnsettledCount  int        `json:"providerUnsettledCount"`
}
```

Use `sync.RWMutex` plus `map[int64]PayoutSyncDiagnostics`; return value copies only. Initialize it in `NewService`. A successful batch clears `LastError` and `LastErrorAt`. No diagnostic update may return an error to or delay the payout worker.

- [ ] **Step 4: Make payout-list failure observable and count each batch**

At the start of `syncAccountPending`, record `begin(accountID, len(rows), time.Now())`. When `fetchRecentAccountSettlements` fails, record `fail`, then return the original error instead of `nil`; `tick` already emits the account ID, pending-order count, and error in its warning log.

Count a row as settled only after `commitResolvedSettlement` succeeds. Change historical lookup to return `true` only after settlement commit and recovery-cursor cleanup succeed. On a complete batch, set:

```go
providerListCount := len(itemsByID)
providerUnsettledCount := len(rows) - settledCount
```

Then record success. A provider record present with `Settled=false`, a record outside the recent list, or a historical lookup with no settlement all remain in `ProviderUnsettledCount`.

- [ ] **Step 5: Add batch error/count tests and verify**

Extend `payout_batch_test.go` with exact failure and counter tests:

```go
func TestPayoutSyncFetchErrorIsReturned(t *testing.T) {
    want := errors.New("tls handshake timeout")
    _, err := fetchRecentAccountSettlements(context.Background(), func(context.Context, int, int) ([]guaji.WebBetRecord, error) { return nil, want })
    if !errors.Is(err, want) { t.Fatalf("got %v want %v", err, want) }
}

func TestPayoutBatchCountsUnresolvedRows(t *testing.T) {
    settled, unresolved := payoutBatchCounts(3, 1)
    if settled != 1 || unresolved != 2 { t.Fatalf("settled=%d unresolved=%d", settled, unresolved) }
}
```

Implement and use this clamped helper in `syncAccountPending`:

```go
func payoutBatchCounts(total, settled int) (int, int) {
    if total < 0 { total = 0 }
    if settled < 0 { settled = 0 }
    if settled > total { settled = total }
    return settled, total - settled
}
```

Run from `backend`: `go test -race ./internal/guaji/accountsvc -run "TestPayout(DiagnosticStore|SyncFetches|Batch)" -count=1`

Expected: PASS with no race report.

Commit: `git add -- backend/internal/guaji/accountsvc/service.go backend/internal/guaji/accountsvc/payout_sync.go backend/internal/guaji/accountsvc/payout_batch_test.go backend/internal/guaji/accountsvc/payout_diagnostics.go backend/internal/guaji/accountsvc/payout_diagnostics_test.go && git commit -m "feat: expose guaji payout sync diagnostics"`

### Task 4: Attach Payout Evidence to the Existing Admin Runtime Diagnostic

**Files:**
- Modify: `backend/internal/handler/admin_runtime_diagnostics.go:14-94`
- Modify: `backend/internal/handler/admin_runtime_diagnostics_test.go`

**Interfaces:**
- Consumes: `accountsvc.Service.PayoutSyncDiagnostics(accountID)` only.
- Produces: existing `GET /api/v1/admin/diagnostics/schemes/{instanceId}/runtime` response field `payoutSync`, containing a snapshot or `null`.
- Preserves: no new route, no client route, no provider request, and no database write.

- [ ] **Step 1: Write failing account-selection and payload tests**

Extract a pure helper and test it directly:

```go
func TestRuntimePayoutDiagnosticsUsesResolvedAccountSnapshot(t *testing.T) {
    want := accountsvc.PayoutSyncDiagnostics{AccountID: 17, PendingCount: 3, ProviderUnsettledCount: 3}
    got := runtimePayoutDiagnostics(17, func(id int64) (accountsvc.PayoutSyncDiagnostics, bool) {
        return want, id == 17
    })
    if got == nil || got.AccountID != 17 || got.PendingCount != 3 { t.Fatalf("got=%+v", got) }
}

func TestRuntimePayoutDiagnosticsMissingAccountDoesNotProbe(t *testing.T) {
    calls := 0
    got := runtimePayoutDiagnostics(0, func(int64) (accountsvc.PayoutSyncDiagnostics, bool) { calls++; return accountsvc.PayoutSyncDiagnostics{}, false })
    if got != nil || calls != 0 { t.Fatalf("got=%+v calls=%d", got, calls) }
}
```

- [ ] **Step 2: Run handler tests and verify RED**

Run from `backend`: `go test ./internal/handler -run "TestRuntimePayoutDiagnostics" -count=1`

Expected: FAIL because the response helper and snapshot field do not exist.

- [ ] **Step 3: Resolve the account locally and attach its snapshot**

Add an internal `GuajiAccountID int64` field to `adminSchemeRuntimeInstance` with `json:"-"`. Extend the instance query with a local-only account expression that prefers the latest formal cloud-bet account for this scheme and falls back to the member's active account:

```sql
COALESCE(
  (SELECT c.guaji_account_id
   FROM cloud_bet_records c
   WHERE c.scheme_id = si.id AND c.sim_bet = FALSE AND c.guaji_account_id IS NOT NULL
   ORDER BY c.placed_at DESC, c.id DESC LIMIT 1),
  (SELECT a.id
   FROM member_guaji_accounts a
   WHERE a.member_id = si.member_id AND a.is_active = TRUE
   ORDER BY a.bound_at DESC LIMIT 1),
  0
)
```

Alias the outer table as `scheme_instances si`, scan the result, then call only the in-memory getter:

```go
func runtimePayoutDiagnostics(accountID int64, read func(int64) (accountsvc.PayoutSyncDiagnostics, bool)) *accountsvc.PayoutSyncDiagnostics {
    if accountID <= 0 || read == nil { return nil }
    snapshot, ok := read(accountID)
    if !ok { return nil }
    return &snapshot
}
```

Add `"payoutSync": payoutSync` to the response. If `h.guajiAccounts` is nil, pass no reader and return `null`.

- [ ] **Step 4: Verify endpoint shaping and commit**

Run from `backend`: `go test ./internal/handler -run "Test(SchemeRuntime|AcceptedPending|RuntimePayoutDiagnostics)" -count=1`

Expected: PASS; the helper test proves the endpoint path reads local state only.

Commit: `git add -- backend/internal/handler/admin_runtime_diagnostics.go backend/internal/handler/admin_runtime_diagnostics_test.go && git commit -m "feat: add payout evidence to scheme diagnostics"`

### Task 5: Regression Verification and Controlled Live Check

**Files:**
- Verify only; no additional production file is expected.

**Interfaces:**
- Validates: rule snapshot selection, scheme-start balance preflight, one-call running bet path, payout diagnostic concurrency, admin runtime response, and full backend compatibility.

- [ ] **Step 1: Run focused package tests**

Run from `backend`:

```powershell
go test ./internal/schemes ./internal/guaji ./internal/guaji/accountsvc ./internal/handler -count=1
```

Expected: all four packages PASS.

- [ ] **Step 2: Run race coverage for new shared state**

Run from `backend`: `go test -race ./internal/guaji/accountsvc -run "TestPayoutDiagnosticStore" -count=1`

Expected: PASS with no race detector output.

- [ ] **Step 3: Run full backend verification**

Run from `backend`:

```powershell
go test ./... -count=1
go build ./cmd/server
```

Expected: PASS. If an unrelated pre-existing test fails, record its exact package, test name, and failure output; do not claim a full green run.

- [ ] **Step 4: Verify patch hygiene**

Run from repository root:

```powershell
git diff --check
git status --short
```

Expected: no whitespace errors; the six unrelated untracked 2026-08-12 documents remain unstaged and unchanged.

- [ ] **Step 5: Deploy only after user authorization and inspect one new formal order**

After the user authorizes restart/deployment, verify the admin runtime endpoint for the target instance and a newly accepted order. Confirm:

- `period_no` equals `third_party_period` for the new order.
- `rule_snapshot` is non-null and locator sub-ID is `390` for `fast_ssc_std/g017/390`.
- `payoutSync.lastAttemptAt` advances at the worker interval.
- A provider list timeout appears in `payoutSync.lastError` and server warning logs instead of silently looking successful.
- When the provider has not settled, `providerUnsettledCount` remains positive; after settlement, `settledCount`/`lastSuccessAt` update and the financial record uses provider values.

- [ ] **Step 6: Create the final focused implementation commit if verification required cleanup**

Stage only files listed in this plan and commit any final test-only or formatting correction with: `git commit -m "test: verify short-period payout strategy fix"`.
