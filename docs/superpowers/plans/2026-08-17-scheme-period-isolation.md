# 方案期号同步隔离 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让第三方 periods 请求的慢速或失败仅影响对应彩种，并在第三方跨期接单时自动暂停方案。

**Architecture:** `periodsync.Worker` 负责所有第三方 periods 网络访问，并扩展为有界、按彩种 single-flight 的异步刷新调度器。`schemes.Worker` 仅向该调度器请求刷新并读取严格 periods 缓存，永不等待刷新完成。跨期接单在本地流水持久化后进入 `bet_failed` 自动暂停分支。

**Tech Stack:** Go、context、sync、atomic、PostgreSQL/sqlc、现有 Guaji periods 与方案 worker。

## Global Constraints

- 同一彩种最多一个在途 periods 刷新；总刷新并发必须有上限。
- 单个第三方请求、彩种或方案不能阻塞其它彩种下单、结算或方案 tick。
- 下单期号继续只使用第三方 periods 缓存，不能使用 WebSocket 替代。
- 第三方接受期号与目标期号不一致时自动暂停实例；不补投、不重投。
- 保留既有 `placeSem` 下单并发限制、按期去重与结算语义。

---

### Task 1: 有界异步 periods 刷新调度器

**Files:**

- Modify: `backend/internal/guaji/periodsync/worker.go`
- Modify: `backend/internal/guaji/periodsync/worker_test.go`
- Create: `backend/internal/guaji/periodsync/request_queue_test.go`

**Interfaces:**

- Produces `func (w *Worker) RequestRefresh(lotteryCode string)` as nonblocking request entry.
- Produces `func (w *Worker) Diagnostics(lotteryCode string) (RefreshDiagnostics, bool)` for admin diagnostics.
- `Run(context.Context)` maintains fixed refresh slots; every HTTP request has a timeout context.

- [ ] **Step 1: Write the failing tests**

Add tests that inject a blocking refresh function and prove one-code single-flight, unrelated-code progress while a slow code is blocked, and global concurrency peak bounded by the configured maximum.

- [ ] **Step 2: Run the tests red**

Run: `go test ./internal/guaji/periodsync -run 'TestWorkerRequestRefresh' -count=1`

Expected: FAIL because `RequestRefresh` and the injectable scheduler do not exist yet.

- [ ] **Step 3: Implement the scheduler**

Add per-code queued/in-flight/backoff state, high and normal queues, fixed worker slots, `RequestRefresh`, timeout-controlled execution, and immutable diagnostics snapshots. Existing periodic targets enqueue work instead of calling HTTP serially. A current periods snapshot with duration at most 15 seconds receives high priority.

- [ ] **Step 4: Run focused verification**

Run: `go test ./internal/guaji/periodsync -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

Run: `git add backend/internal/guaji/periodsync && git commit -m "feat: isolate async period refreshes"`

### Task 2: Make scheme ticks only queue refreshes

**Files:**

- Modify: `backend/internal/schemes/worker.go`
- Modify: `backend/internal/schemes/worker_parallel.go`
- Modify: `backend/internal/schemes/worker_parallel_test.go`
- Modify: `backend/internal/server/server.go`

**Interfaces:**

- Consumes a `RequestRefresh(string)` interface injected using `SetPeriodRefreshRequester`.
- Produces nonblocking `prefetchPeriodSync`, `betWindowGate.ensureOpen`, and activation refresh behavior.

- [ ] **Step 1: Write the failing tests**

Add a fake requester. Assert prefetch returns immediately and queues each unique lottery once. Assert a missing cache makes `ensureOpen` return false while queuing a request, without a `ForceRefresh` call.

- [ ] **Step 2: Run the tests red**

Run: `go test ./internal/schemes -run 'TestPrefetchPeriodSyncOnlyQueuesRequests|TestBetWindowGateQueuesRefreshWhenCacheMissing' -count=1`

Expected: FAIL because the worker still calls `EnsureFreshIfStale` and `ForceRefresh` synchronously.

- [ ] **Step 3: Implement minimal wiring**

Replace the prefetch wait group with deduplicated requests. When cache is unavailable, queue then skip only the affected instance. Create the periods worker before the schemes worker in `server.go`, inject it, and start both with the same service context. Keep `Syncer` for start snapshots, game detail and existing non-scheme callers.

- [ ] **Step 4: Run focused verification**

Run: `go test ./internal/schemes ./internal/guaji/periodsync -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

Run: `git add backend/internal/schemes backend/internal/server/server.go && git commit -m "fix: keep scheme ticks independent of period refresh"`

### Task 3: Persist cross-period result, then pause safely

**Files:**

- Modify: `backend/internal/schemes/worker.go`
- Create: `backend/internal/schemes/worker_period_mismatch_test.go`
- Modify: `backend/internal/handler/admin_scheme_runtime_diagnostics.go`
- Modify: `backend/internal/handler/admin_scheme_runtime_diagnostics_test.go`

**Interfaces:**

- Produces `isAcceptedPeriodMismatch(targetPeriod, acceptedPeriod string) bool`.
- Admin runtime diagnostics include per-lottery periods refresh status; client endpoints stay unchanged.

- [ ] **Step 1: Write the failing tests**

Test empty/equal periods as non-mismatches and distinct target/accepted periods as mismatch. Add an integration-style worker test proving an accepted next-period response persists `third_party_period`, commits the record, pauses the instance as `pending/bet_failed`, and does not call Place again. Add admin diagnostic response coverage for last attempt, last success, failures, next allowed time, and error summary.

- [ ] **Step 2: Run the tests red**

Run: `go test ./internal/schemes ./internal/handler -run 'TestIsAcceptedPeriodMismatch|TestRuntimeDiagnosticsIncludesPeriodRefreshStatus' -count=1`

Expected: FAIL because the mismatch helper and diagnostic fields are absent.

- [ ] **Step 3: Implement the safety path**

After Guaji acceptance and local metadata persistence, retain a mismatch flag when accepted period differs from target. Commit the existing transaction, then call `pauseRunningInstance` with target and accepted period in the failure detail and return `errSchemeBetStopped`. Do not retry, move the request forward, or delete the accepted record. Extend only the admin diagnostic endpoint with the scheduler snapshot.

- [ ] **Step 4: Run focused regression verification**

Run: `go test ./internal/schemes ./internal/guaji/periodsync ./internal/handler -count=1`

Expected: PASS.

- [ ] **Step 5: Commit**

Run: `git add backend/internal/schemes backend/internal/handler && git commit -m "fix: pause schemes after third-party period mismatch"`

### Task 4: Document and verify

**Files:**

- Modify: `backend/docs/modules/schemes.md`

- [ ] **Step 1: Document operations**

Document short-period priority, bounded retries, admin-only refresh diagnostics, cross-period pause behavior, and the fact that WebSocket is not a placement-period source.

- [ ] **Step 2: Run full backend tests**

Run: `go test ./...`

Expected: exit code 0; if pre-existing unrelated failures remain, report package, test and error without claiming all green.

- [ ] **Step 3: Compile the production server**

Run: `go build ./cmd/server`

Expected: exit code 0.

- [ ] **Step 4: Commit**

Run: `git add backend/docs/modules/schemes.md && git commit -m "docs: describe isolated scheme period refresh"`
