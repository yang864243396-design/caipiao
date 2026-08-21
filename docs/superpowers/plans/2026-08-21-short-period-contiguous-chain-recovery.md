# Short-Period Contiguous Chain Recovery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让正式短周期方案只依据已接受来源期 `N` 的本地开奖结果推进一次策略，并且只向紧邻期 `N+1` 下单；若错过 `N+1`，停止当前 chain，禁止自动跨期恢复。

**Architecture:** 保留现有 JetStream、按实例分片、租约、PostgreSQL Outbox 和第三方精确指纹对账。把“策略判定”与“目标期派单”拆成两个持久化阶段：第一阶段提交策略状态和 `awaiting_target` 决策，第二阶段由彩票级 WebSocket 边界事件解析唯一 `N+1`；数据库时钟与条件更新裁决解析/过期竞态。

**Tech Stack:** Go 1.x、PostgreSQL/pgx、Goose migrations、NATS JetStream、Gorilla WebSocket、项目现有 `schemes`、`schemeeventbus`、`providerperiodtarget` 与 `sqlcdb` 包。

**Spec:** `docs/superpowers/specs/2026-08-21-short-period-contiguous-chain-recovery-design.md`

## Global Constraints

- 正式短周期来源期 `N` 只能生成紧邻目标期 `N+1`，不能使用 REST 当前期授权下单。
- 上一期策略判定必须先持久化；第三方财务结算延迟不能阻塞倍率、轮次和选号推进。
- 错过紧邻期后写入 `missed_contiguous_period`，暂停实例并禁止启动接管或自动 rearm。
- 用户手工重新开启必须创建新 chain，并把倍投轮次、选号轮次及游标重置到第一轮。
- 重复开奖、JetStream 重投、后端重启和并发 worker 不得重复推进策略、重复建 Outbox 或重复下单。
- WebSocket、REST 快照和 stale 检测按彩票共享，不增加逐用户、逐方案或逐注单第三方轮询。
- 单彩票或单实例异常不得持有全局锁，不得阻塞其他彩票、用户、实例或分片。
- 热路径必须使用有界批量与索引查询，保持日均至少 500 万注的扩展模型。
- 不修改玩法规则、金额、单挑、第三方 payload 和财务结算口径。
- 日志和诊断不得包含 token、密码或完整凭证。

## File Map

- `backend/migrations/00177_scheme_contiguous_target_wait.sql`：增加等待目标、错期及 chain 阻断持久化字段和索引。
- `backend/internal/db/sqlcdb/scheme_contiguous_target_ext.go`：封装等待决策的锁定、解析、过期和恢复查询。
- `backend/internal/guaji/draw_ws_liveness.go`：连接级 ping/pong、读截止和单次重连信号。
- `backend/internal/guaji/draw_boundary_health.go`：按彩票维护边界接收时间和 stale generation。
- `backend/internal/schemeeventbus/period_boundary.go`：定义、发布和消费彩票级边界事件。
- `backend/internal/schemes/strategy_formal.go`：第一阶段只持久化策略结果与等待决策。
- `backend/internal/schemes/contiguous_target_resolver.go`：第二阶段生成唯一 Outbox 或停止 chain。
- `backend/internal/schemes/contiguous_target_recovery.go`：有界数据库恢复，不调用第三方。
- `backend/internal/server/scheme_period_boundary_bus.go`：连接边界事件、分片处理器和恢复任务。
- `backend/internal/handler/admin_runtime_diagnostics.go`：暴露开发者诊断状态。

---

### Task 1: Persist Waiting and Terminal Gap State

**Files:**
- Create: `backend/migrations/00177_scheme_contiguous_target_wait.sql`
- Create: `backend/internal/db/contiguous_target_migration_test.go`
- Create: `backend/internal/db/sqlcdb/scheme_contiguous_target_ext.go`
- Create: `backend/internal/db/sqlcdb/scheme_contiguous_target_integration_test.go`
- Modify: `backend/internal/db/sqlcdb/scheme_betting_rearm_ext.go`

**Interfaces:**
- Produces: `AwaitingContiguousTargetRow`, `GetAwaitingContiguousTargetForUpdate(ctx, decisionID)`, `ListAwaitingContiguousTargets(ctx, lotteryCodes, shards, cursor, limit)`, `CompleteAwaitingContiguousTarget(ctx, params)`, `MissAwaitingContiguousTarget(ctx, params)`.
- Produces: decision statuses `awaiting_target` and `missed_contiguous_period`; instance field `chain_block_reason`.
- Consumes: existing `(scheme_id, source_period_no)` uniqueness, `strict_chain_state`, `chain_id`, `chain_seq` and shard columns.
- Test support: `newAwaitingDecisionDBFixture(t) *awaitingDecisionDBFixture` owns one rollback-only transaction and provides `DatabaseNow()`, `Seed(deadline)`, `SeedMany(count)`, `Complete(decisionID)`, `Miss(decisionID)`, `List(cursor, limit)` and `AssertSingleTerminal(decisionID)`; each method uses the production query object rather than duplicate SQL.

- [ ] **Step 1: Write migration contract tests**

```go
func TestContiguousTargetMigrationContracts(t *testing.T) {
    sql := readContiguousTargetMigration(t)
    require.Contains(t, sql, "awaiting_target")
    require.Contains(t, sql, "missed_contiguous_period")
    require.Contains(t, sql, "target_deadline_at")
    require.Contains(t, sql, "target_period_no")
    require.Contains(t, sql, "failure_reason")
    require.Contains(t, sql, "chain_block_reason")
    require.Contains(t, sql, "lottery_code, status, target_deadline_at, id")
}
```

Define `readContiguousTargetMigration(t *testing.T) string` in the same test file using `os.ReadFile("../../migrations/00177_scheme_contiguous_target_wait.sql")`; call `t.Helper()` and fail the test with the path and read error.

- [ ] **Step 2: Run the migration contract test and confirm RED**

Run: `cd backend; go test ./internal/db -run TestContiguousTargetMigrationContracts -count=1`

Expected: FAIL because migration `00177_scheme_contiguous_target_wait.sql` does not exist.

- [ ] **Step 3: Add the schema migration**

```sql
-- +goose Up
ALTER TABLE scheme_period_decisions
  ADD COLUMN IF NOT EXISTS target_deadline_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS target_period_no VARCHAR(64),
  ADD COLUMN IF NOT EXISTS failure_reason VARCHAR(64);

ALTER TABLE scheme_instances
  ADD COLUMN IF NOT EXISTS chain_block_reason VARCHAR(64);

ALTER TABLE scheme_period_decisions
  DROP CONSTRAINT IF EXISTS scheme_period_decisions_status_check;

ALTER TABLE scheme_period_decisions
  ADD CONSTRAINT chk_scheme_period_decisions_status CHECK (
    status IN ('awaiting_target', 'completed', 'missed_contiguous_period',
               'blocked', 'duplicate', 'chain_broken')
  );

CREATE INDEX IF NOT EXISTS idx_scheme_period_decisions_awaiting_target
  ON scheme_period_decisions (lottery_code, status, target_deadline_at, id)
  WHERE status = 'awaiting_target';

-- +goose Down
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM scheme_period_decisions
    WHERE status IN ('awaiting_target', 'missed_contiguous_period')
  ) THEN
    RAISE EXCEPTION 'cannot roll back migration 177 while contiguous decisions remain';
  END IF;
END $$;

DROP INDEX IF EXISTS idx_scheme_period_decisions_awaiting_target;
ALTER TABLE scheme_period_decisions
  DROP CONSTRAINT IF EXISTS scheme_period_decisions_status_check;
ALTER TABLE scheme_period_decisions
  ADD CONSTRAINT scheme_period_decisions_status_check CHECK (
    status IN ('completed', 'blocked', 'duplicate', 'chain_broken')
  );
ALTER TABLE scheme_period_decisions
  DROP COLUMN IF EXISTS failure_reason,
  DROP COLUMN IF EXISTS target_period_no,
  DROP COLUMN IF EXISTS target_deadline_at;
ALTER TABLE scheme_instances
  DROP COLUMN IF EXISTS chain_block_reason;
```

The Goose down section must first raise an exception when any row remains in `awaiting_target` or `missed_contiguous_period`; only then may it drop the partial index and new columns and restore `scheme_period_decisions_status_check` to the old four-status set. It must never silently discard live state.

- [ ] **Step 4: Write DB integration tests for atomic winner semantics**

```go
func TestAwaitingTargetResolverAndExpiryHaveOneWinner(t *testing.T) {
    f := newAwaitingDecisionDBFixture(t)
    decisionID := f.Seed(f.DatabaseNow().Add(time.Second))
    var resolved, missed atomic.Int32
    var wg sync.WaitGroup
    wg.Add(2)
    go func() { defer wg.Done(); if f.Complete(decisionID) { resolved.Add(1) } }()
    go func() { defer wg.Done(); if f.Miss(decisionID) { missed.Add(1) } }()
    wg.Wait()
    require.Equal(t, int32(1), resolved.Load()+missed.Load())
    f.AssertSingleTerminal(decisionID)
}

func TestAwaitingTargetQueryIsBoundedAndCursorOrdered(t *testing.T) {
    f := newAwaitingDecisionDBFixture(t)
    f.SeedMany(40)
    rows := f.List(0, 32)
    require.Len(t, rows, 32)
    for i := 1; i < len(rows); i++ {
        require.Greater(t, rows[i].DecisionID, rows[i-1].DecisionID)
    }
}
```

- [ ] **Step 5: Run the DB integration tests and confirm RED**

Run: `cd backend; go test ./internal/db/sqlcdb -run 'TestAwaitingTarget' -count=1`

Expected: FAIL because the new query types and methods do not exist.

- [ ] **Step 6: Implement the SQL boundary methods**

```go
type AwaitingContiguousTargetRow struct {
    DecisionID       int64
    SchemeID         string
    MemberID         int64
    LotteryCode      string
    SourcePeriodNo   string
    SourceBetRecordID int64
    TargetDeadlineAt time.Time
    StateVersionAfter int64
    ChainID          string
    ChainSeq         int64
    ShardNo          int32
    Mode             string
}

type CompleteAwaitingContiguousTargetParams struct {
    DecisionID      int64
    TargetPeriodNo  string
    Diagnostics     []byte
}

type MissAwaitingContiguousTargetParams struct {
    DecisionID      int64
    FailureReason   string
    Diagnostics     []byte
}
```

`CompleteAwaitingContiguousTarget` must update only when `status='awaiting_target' AND now() < target_deadline_at`. `MissAwaitingContiguousTarget` must update only when `status='awaiting_target' AND now() >= target_deadline_at`, then set only that instance to `paused / bet_failed / blocked_requires_rearm / missed_contiguous_period`. Both callers lock the decision and instance in one transaction and treat zero affected rows as an idempotent no-op.

- [ ] **Step 7: Make chain activation clear stale block metadata**

Add `chain_block_reason = NULL` to `ActivateSchemeBettingChain`, but do not permit that method to be called from automatic recovery for `missed_contiguous_period`.

- [ ] **Step 8: Run migration and DB tests GREEN**

Run: `cd backend; go test ./internal/db ./internal/db/sqlcdb -run 'ContiguousTarget|AwaitingTarget|SchemeBettingAdmin' -count=1`

Expected: PASS.

- [ ] **Step 9: Commit**

```powershell
git add -- backend/migrations/00177_scheme_contiguous_target_wait.sql backend/internal/db/contiguous_target_migration_test.go backend/internal/db/sqlcdb/scheme_contiguous_target_ext.go backend/internal/db/sqlcdb/scheme_contiguous_target_integration_test.go backend/internal/db/sqlcdb/scheme_betting_rearm_ext.go
git commit -m "feat: persist contiguous target decisions"
```

---

### Task 2: Supervise the Draw WebSocket Transport

**Files:**
- Create: `backend/internal/guaji/draw_ws_liveness.go`
- Create: `backend/internal/guaji/draw_ws_liveness_test.go`
- Modify: `backend/internal/guaji/draws.go`
- Modify: `backend/internal/guaji/ws.go`
- Modify: `backend/internal/guaji/drawsync/worker.go`

**Interfaces:**
- Produces: `type DrawWSHealthSnapshot`, `newDrawWSLiveness(conn drawWSConn, now func() time.Time)`, `Run(ctx) error`, `MarkFrame()`, `Snapshot()`.
- Produces: one reconnect request per connection/stale generation; reconnect backoff `1s,2s,4s,8s,16s,30s` with jitter.
- Consumes: one existing shared Guaji draw connection and existing `drawsync.Worker.Run` reconnect loop.
- Test support: `newFakeDrawWSConn() *fakeDrawWSConn` records read deadlines, pong handler, control writes and close state behind a mutex; `newFakeClock() *fakeClock` exposes `Now()` and `Advance(duration)` and wakes registered deadline waiters.

- [ ] **Step 1: Write deterministic liveness tests with a fake connection and clock**

```go
func TestDrawWSLivenessTimesOutSilentHalfOpenConnection(t *testing.T) {
    conn, clock := newFakeDrawWSConn(), newFakeClock()
    live := newDrawWSLiveness(conn, clock.Now)
    go func() { _ = live.Run(context.Background()) }()
    clock.Advance(drawWSReadIdleTimeout + time.Millisecond)
    require.Eventually(t, conn.WasClosed, time.Second, time.Millisecond)
}

func TestDrawWSLivenessPongExtendsReadDeadline(t *testing.T) {
    conn, clock := newFakeDrawWSConn(), newFakeClock()
    live := newDrawWSLiveness(conn, clock.Now)
    conn.EmitPong()
    require.Equal(t, clock.Now().Add(drawWSReadIdleTimeout), conn.LastReadDeadline())
}

func TestDrawWSLivenessStopsAllGoroutinesOnCancel(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    live := newDrawWSLiveness(newFakeDrawWSConn(), time.Now)
    done := make(chan error, 1)
    go func() { done <- live.Run(ctx) }()
    cancel()
    require.NoError(t, <-done)
}
```

- [ ] **Step 2: Run liveness tests and confirm RED**

Run: `cd backend; go test ./internal/guaji -run TestDrawWSLiveness -count=1`

Expected: FAIL because the liveness supervisor does not exist.

- [ ] **Step 3: Add a narrow connection abstraction and supervisor**

```go
type drawWSConn interface {
    ReadMessage() (int, []byte, error)
    WriteControl(int, []byte, time.Time) error
    SetReadDeadline(time.Time) error
    SetPongHandler(func(string) error)
    Close() error
}

type DrawWSHealthSnapshot struct {
    ConnectedAt time.Time
    LastFrameAt time.Time
    LastPongAt  time.Time
    Reconnects  uint64
    LastError   string
}
```

Use one ping writer per connection, refresh the read deadline on every valid frame and pong, close the socket when the read deadline expires, and stop the ping goroutine when the context is canceled or `ReadMessage` returns. Do not create a connection per lottery, account or scheme.

- [ ] **Step 4: Preserve bounded reconnect behavior**

In `drawsync.Worker.Run`, retain one reconnect loop and replace unbounded/implicit retries with capped exponential backoff plus jitter. Reset the backoff after the first valid parsed draw frame. Context cancellation must return without an extra reconnect.

- [ ] **Step 5: Run transport tests GREEN**

Run: `cd backend; go test ./internal/guaji ./internal/guaji/drawsync -run 'DrawWS|SubscribeDraws|Reconnect' -count=1`

Expected: PASS with no leaked goroutines under `-race` in the targeted packages.

- [ ] **Step 6: Commit**

```powershell
git add -- backend/internal/guaji/draw_ws_liveness.go backend/internal/guaji/draw_ws_liveness_test.go backend/internal/guaji/draws.go backend/internal/guaji/ws.go backend/internal/guaji/drawsync/worker.go
git commit -m "fix: supervise draw websocket liveness"
```

---

### Task 3: Detect Lottery-Specific Stale Boundaries

**Files:**
- Create: `backend/internal/guaji/draw_boundary_health.go`
- Create: `backend/internal/guaji/draw_boundary_health_test.go`
- Modify: `backend/internal/guaji/drawsync/worker.go`
- Modify: `backend/internal/lottery/period_state.go`
- Modify: `backend/internal/lottery/period_state_test.go`

**Interfaces:**
- Produces: `NewBoundaryHealth(monitored []string) *BoundaryHealth`.
- Produces: `BoundaryHealth.Observe(lotteryCode, currentIssue, nextIssue string, receivedMono time.Time, interval time.Duration)`.
- Produces: `BoundaryHealth.Stale(now time.Time) []StaleLottery` and `BoundaryHealth.Snapshot(lotteryCode string) LotteryBoundaryHealthSnapshot`.
- Consumes: `lottery.DrawIntervalSecForLottery`; only configured formal short lotteries are monitored.
- Test support: `lotteryCodes([]StaleLottery) []string` returns codes in emitted order; tests never depend on map iteration order.

- [ ] **Step 1: Write per-lottery stale tests**

```go
func TestBoundaryHealthMarksOnlySilentLotteryStale(t *testing.T) {
    h := NewBoundaryHealth([]string{"tron_ffc_3s", "tron_ffc_6s"})
    base := time.Unix(100, 0)
    h.Observe("tron_ffc_3s", "10", "11", base, 3*time.Second)
    h.Observe("tron_ffc_6s", "20", "21", base.Add(5*time.Second), 6*time.Second)
    stale := h.Stale(base.Add(3501 * time.Millisecond))
    require.Equal(t, []string{"tron_ffc_3s"}, lotteryCodes(stale))
}

func TestBoundaryHealthEmitsOneReconnectPerStaleGeneration(t *testing.T) {
    h := NewBoundaryHealth([]string{"tron_ffc_6s"})
    h.Observe("tron_ffc_6s", "20", "21", time.Unix(100, 0), 6*time.Second)
    require.Len(t, h.Stale(time.Unix(106, 0).Add(time.Second)), 1)
    require.Empty(t, h.Stale(time.Unix(107, 0)))
    h.Observe("tron_ffc_6s", "21", "22", time.Unix(108, 0), 6*time.Second)
    require.Len(t, h.Stale(time.Unix(115, 0)), 1)
}
```

- [ ] **Step 2: Run stale tests and confirm RED**

Run: `cd backend; go test ./internal/guaji -run TestBoundaryHealth -count=1`

Expected: FAIL because lottery-level health tracking does not exist.

- [ ] **Step 3: Implement monotonic boundary health**

For each lottery, calculate the next expected boundary as:

```go
grace := min(500*time.Millisecond, interval/6)
staleAt := lastReceivedMono.Add(interval).Add(grace)
```

Store local receipt time, not provider wall-clock time. A frame for another lottery must not refresh this lottery. Emit at most one reconnect signal for a stale generation, and clear the generation only after a newer valid boundary for that lottery.

- [ ] **Step 4: Wire health into the shared draw worker**

After a valid `DrawEvent` is parsed, update the shared period state and call `BoundaryHealth.Observe`. A small supervisor ticker checks only the configured short-lottery set and requests a single connection close/reconnect when any lottery becomes stale. It must never call REST or query schemes.

- [ ] **Step 5: Keep formal target authorization strict**

Add/retain tests proving `providerperiodtarget.Current` for 3s/6s/15s requires `WS.CurrentIssue == sourcePeriod` and never falls back to REST periods. The health component diagnoses and reconnects; it does not relax authorization.

- [ ] **Step 6: Run health and period-state tests GREEN**

Run: `cd backend; go test ./internal/guaji ./internal/guaji/drawsync ./internal/lottery ./internal/providerperiodtarget -run 'BoundaryHealth|FreshShort|PeriodState' -count=1`

Expected: PASS.

- [ ] **Step 7: Commit**

```powershell
git add -- backend/internal/guaji/draw_boundary_health.go backend/internal/guaji/draw_boundary_health_test.go backend/internal/guaji/drawsync/worker.go backend/internal/lottery/period_state.go backend/internal/lottery/period_state_test.go
git commit -m "fix: detect stale lottery draw boundaries"
```

---

### Task 4: Publish Durable Period-Boundary Wakeups

**Files:**
- Create: `backend/internal/schemeeventbus/period_boundary.go`
- Create: `backend/internal/schemeeventbus/period_boundary_test.go`
- Modify: `backend/internal/schemeeventbus/bus.go`
- Modify: `backend/internal/guaji/drawsync/worker.go`

**Interfaces:**
- Produces: `schemeeventbus.PeriodBoundary{LotteryCode, CurrentIssue, NextIssue, ReceivedAt, Generation}`.
- Produces: `Bus.PublishPeriodBoundary(ctx, event)` and `Bus.ConsumePeriodBoundaries(ctx, durable, handler)`.
- Produces: `schemeeventbus.ContiguousTargetReady{DecisionID, SchemeID, LotteryCode, SourcePeriod, BoundaryGeneration}` plus shard-aware publish/consume methods.
- Consumes: every valid WS boundary, including a draw whose DB insert is deduplicated because REST inserted it first.
- Test support: `fakeBoundaryPublisher` stores published events under a mutex; `wsDraw(lotteryCode, currentIssue, nextIssue string) guaji.DrawEvent` builds a valid WS boundary event; `newDrawWorkerForTest(store fakeDrawStore, publisher *fakeBoundaryPublisher) *Worker` injects both dependencies without opening network connections.

- [ ] **Step 1: Write event contract and duplicate-boundary tests**

```go
func TestPeriodBoundaryMessageIDIncludesLotteryAndGeneration(t *testing.T) {
    event := PeriodBoundary{LotteryCode: "tron_ffc_6s", CurrentIssue: "100", NextIssue: "101", Generation: 7}
    require.Equal(t, "period-boundary:tron_ffc_6s:100:101:7", event.MessageID())
}

func TestContiguousTargetReadyRoutesBySchemeShard(t *testing.T) {
    event := ContiguousTargetReady{DecisionID: 9, SchemeID: "inst-9", LotteryCode: "tron_ffc_6s", SourcePeriod: "100", BoundaryGeneration: 7}
    require.Equal(t, schemebetting.ShardForScheme("inst-9", 64), event.Shard(64))
    require.Equal(t, "contiguous-target:9:7", event.MessageID())
}

func TestDrawWorkerPublishesBoundaryWhenDrawInsertIsDuplicate(t *testing.T) {
    store := fakeDrawStore{inserted: false}
    publisher := &fakeBoundaryPublisher{}
    worker := newDrawWorkerForTest(store, publisher)
    worker.Ingest(context.Background(), wsDraw("tron_ffc_6s", "100", "101"))
    require.Equal(t, 1, publisher.Count())
}
```

- [ ] **Step 2: Run event tests and confirm RED**

Run: `cd backend; go test ./internal/schemeeventbus ./internal/guaji/drawsync -run 'PeriodBoundary|Duplicate' -count=1`

Expected: FAIL because the event and publisher are not defined.

- [ ] **Step 3: Add a separate JetStream subject**

```go
type PeriodBoundary struct {
    LotteryCode string    `json:"lotteryCode"`
    CurrentIssue string   `json:"currentIssue"`
    NextIssue    string   `json:"nextIssue"`
    ReceivedAt   time.Time `json:"receivedAt"`
    Generation   uint64   `json:"generation"`
}

type ContiguousTargetReady struct {
    DecisionID        int64  `json:"decisionId"`
    SchemeID          string `json:"schemeId"`
    LotteryCode       string `json:"lotteryCode"`
    SourcePeriod      string `json:"sourcePeriod"`
    BoundaryGeneration uint64 `json:"boundaryGeneration"`
}
```

Use a global boundary subject such as `<prefix>.period.boundary.<lottery>` with durable expander `scheme-contiguous-target-expander`, then publish each candidate to `<prefix>.target.ready.<shard>`. Do not reuse `draw.confirmed`: draw hash deduplication intentionally suppresses duplicate draw insertion, while target resolution needs every newer WS boundary generation.

- [ ] **Step 4: Publish after period-state update, regardless of draw insertion result**

The draw worker order must be:

1. parse and validate the WS frame;
2. update shared `lottery.PeriodState`;
3. update `BoundaryHealth`;
4. publish `PeriodBoundary`;
5. persist/merge the draw and publish `DrawConfirmed` only under its existing persistence rules.

Publishing failure must be logged and recovered by Task 6's bounded DB scan; it must not roll back the shared period snapshot or draw persistence.

- [ ] **Step 5: Run event tests GREEN**

Run: `cd backend; go test ./internal/schemeeventbus ./internal/guaji/drawsync -run 'PeriodBoundary|Duplicate' -count=1`

Expected: PASS.

- [ ] **Step 6: Commit**

```powershell
git add -- backend/internal/schemeeventbus/period_boundary.go backend/internal/schemeeventbus/period_boundary_test.go backend/internal/schemeeventbus/bus.go backend/internal/guaji/drawsync/worker.go
git commit -m "feat: publish durable period boundaries"
```

---

### Task 5: Commit Strategy Evaluation Before Target Selection

**Files:**
- Modify: `backend/internal/db/sqlcdb/worker_bet_ext.go`
- Modify: `backend/internal/db/sqlcdb/strategy_event_ext.go`
- Modify: `backend/internal/db/sqlcdb/scheme_betting_outbox_ext.go`
- Modify: `backend/internal/schemes/strategy_processor.go`
- Modify: `backend/internal/schemes/strategy_formal.go`
- Modify: `backend/internal/schemes/strategy_processor_test.go`
- Create: `backend/internal/schemes/strategy_formal_phase_test.go`

**Interfaces:**
- Produces: `persistFormalAwaitingTarget(ctx, q, row, inst, config, stateVersionBefore, result) (decisionID int64, created bool, err error)`.
- Produces: `contiguousTargetDeadline(drawnAt time.Time, intervalSec int, safety time.Duration) (time.Time, error)`; non-positive interval returns a typed configuration error.
- Produces: a committed `awaiting_target` decision with advanced strategy state, no target period and no Outbox.
- Consumes: `PendingFormalStrategyRow.DrawnAt`, immutable rule snapshot, accepted source bet and current chain state.
- Test support: `newFormalStrategyFixture(t) *formalStrategyFixture` creates one accepted formal bet, persisted rule snapshot and draw inside a rollback-only transaction; its methods read production rows to report state version, round advance count, decision count, strategy-evaluated flag and Outbox count.

- [ ] **Step 1: Add `DrawnAt` to the pending formal row contract**

Extend `PendingFormalStrategyRow` and both pending-row SELECTs so the target deadline is derived from the persisted source draw rather than process wall time.

```go
type PendingFormalStrategyRow struct {
    RecordID         int64
    SchemeID         string
    LotteryCode      string
    PeriodNo         string
    BetContent       string
    RuleSnapshot     []byte
    RuleVersion      pgtype.Int4
    RuleSnapshotHash pgtype.Text
    Balls            []string
    ProviderHit      pgtype.Bool
    DrawnAt          time.Time
}
```

Both SELECTs must append `d.drawn_at`, and both `rows.Scan` calls must scan it into `DrawnAt`. Extend `InsertSchemePeriodDecisionParams` with `TargetDeadlineAt pgtype.Timestamptz`, and write it to `scheme_period_decisions.target_deadline_at`; older callers pass an invalid/null `pgtype.Timestamptz`.

- [ ] **Step 2: Write phase-one transaction tests**

```go
func TestFormalEvaluationCommitsWhenTargetIsTemporarilyUnavailable(t *testing.T) {
    fixture := newFormalStrategyFixture(t)
    fixture.ProviderPeriodState.Clear()
    require.NoError(t, fixture.Processor.ProcessStrategyReady(fixture.Context(), fixture.RecordID, fixture.SchemeID, fixture.Lottery, fixture.SourcePeriod, fixture.StateVersion))
    require.Equal(t, int64(1), fixture.StateVersionAfter())
    require.Equal(t, "awaiting_target", fixture.DecisionStatus())
    require.True(t, fixture.CloudRecordStrategyEvaluated())
    require.Equal(t, 0, fixture.OutboxCount())
}

func TestDuplicateFormalEvaluationDoesNotAdvanceRoundTwice(t *testing.T) {
    fixture := newFormalStrategyFixture(t)
    fixture.ProcessSameDrawTwice()
    require.Equal(t, 1, fixture.RoundAdvanceCount())
    require.Equal(t, 1, fixture.DecisionCount())
}
```

- [ ] **Step 3: Run phase-one tests and confirm RED**

Run: `cd backend; go test ./internal/schemes -run 'FormalEvaluationCommits|DuplicateFormalEvaluation' -count=1`

Expected: FAIL because `tryProcessFormalCandidate` still resolves the provider target inside the evaluation transaction and rolls back when it is unavailable.

- [ ] **Step 4: Split formal phase one from target resolution**

Within the existing strategy transaction:

```go
if err := schemestate.ProcessStrategyAfterDraw(ctx, q, inst, row.PeriodNo, result.Hit, definitionConfig); err != nil {
    return 0, false, err
}
deadline, err := contiguousTargetDeadline(row.DrawnAt, lottery.DrawIntervalSecForLottery(ctx, q, row.LotteryCode), guajiPlaceCloseSafety)
if err != nil {
    return 0, false, err
}
decisionID, created, err := q.InsertSchemePeriodDecision(ctx, sqlcdb.InsertSchemePeriodDecisionParams{
    SchemeID: row.SchemeID,
    LotteryCode: row.LotteryCode,
    SourcePeriodNo: row.PeriodNo,
    SourceBetRecordID: row.RecordID,
    StateVersionBefore: stateVersionBefore,
    StateVersionAfter: stateVersionBefore + 1,
    Status: "awaiting_target",
    TargetDeadlineAt: pgtype.Timestamptz{Time: deadline, Valid: true},
})
```

Complete `scheme_strategy_evaluations` and `cloud_bet_records.strategy_evaluated_at` in the same transaction. Do not call `providerperiodtarget.Current`, `BuildShadowCommand`, `buildFormalFrozenRequest` or `InsertFormalSchemeBetOutbox` in phase one.

- [ ] **Step 5: Handle already-expired deadlines without rollback**

If `DrawnAt + interval - 1.2s` is not later than the database clock, still commit the local strategy evaluation exactly once, then transition the decision to `missed_contiguous_period` in a second transaction. An unknown/non-positive interval is a terminal configuration failure for that chain, not permission to choose a REST period.

- [ ] **Step 6: Run phase-one tests GREEN**

Run: `cd backend; go test ./internal/schemes -run 'FormalEvaluation|DuplicateFormal|StrategyProcessor' -count=1`

Expected: PASS; target unavailability no longer erases strategy progress.

- [ ] **Step 7: Commit**

```powershell
git add -- backend/internal/db/sqlcdb/worker_bet_ext.go backend/internal/db/sqlcdb/strategy_event_ext.go backend/internal/db/sqlcdb/scheme_betting_outbox_ext.go backend/internal/schemes/strategy_processor.go backend/internal/schemes/strategy_formal.go backend/internal/schemes/strategy_processor_test.go backend/internal/schemes/strategy_formal_phase_test.go
git commit -m "fix: commit formal strategy before target wait"
```

---

### Task 6: Resolve Exactly N+1 or Stop the Chain

**Files:**
- Create: `backend/internal/schemes/contiguous_target_resolver.go`
- Create: `backend/internal/schemes/contiguous_target_resolver_test.go`
- Create: `backend/internal/schemes/contiguous_target_recovery.go`
- Create: `backend/internal/schemes/contiguous_target_recovery_test.go`
- Modify: `backend/internal/schemes/worker.go`
- Modify: `backend/internal/schemes/strategy_processor.go`
- Modify: `backend/internal/schemes/formal_command.go`
- Create: `backend/internal/server/scheme_period_boundary_bus.go`
- Create: `backend/internal/server/scheme_period_boundary_bus_test.go`
- Modify: `backend/internal/server/server.go`

**Interfaces:**
- Produces: `Worker.ProcessContiguousTargetReady(ctx, schemeeventbus.ContiguousTargetReady) error`.
- Produces: `StrategyProcessor.ResolveAwaitingTarget(ctx, decisionID int64) error`.
- Produces: `Worker.RunContiguousTargetRecovery(ctx, lotteries, shards, batch, concurrency, interval)`.
- Consumes: `providerperiodtarget.Current(ctx, q, lotteryCode, sourcePeriod, dbNow)` and Task 1's locked decision methods.
- Test support: `newAwaitingTargetFixture(t, sourcePeriod string, deadlineOffset time.Duration) *awaitingTargetFixture` seeds a committed phase-one row whose deadline is database-now plus the offset, plus shared WS period state; concurrent methods use `sync.WaitGroup`, and all assertions read decisions, Outbox, instance state and frozen payload from the database.

- [ ] **Step 1: Write resolver success and strict-gap tests**

```go
func TestResolveAwaitingTargetCreatesOneOutboxForNPlusOne(t *testing.T) {
    f := newAwaitingTargetFixture(t, "100", 2*time.Second)
    f.SetWSBoundary("100", "101")
    f.ResolveConcurrently(8)
    require.Equal(t, "completed", f.DecisionStatus())
    require.Equal(t, "101", f.TargetPeriod())
    require.Equal(t, 1, f.OutboxCount())
    require.Equal(t, f.AdvancedMultiplier(), f.FrozenRequestMultiplier())
}

func TestResolveAwaitingTargetNeverSkipsToCurrentProviderPeriod(t *testing.T) {
    f := newAwaitingTargetFixture(t, "100", 2*time.Second)
    f.SetWSBoundary("101", "102")
    f.Resolve()
    require.Equal(t, "missed_contiguous_period", f.DecisionStatus())
    require.Equal(t, 0, f.OutboxCount())
    require.Equal(t, "blocked_requires_rearm", f.ChainState())
}

func TestResolverLosesToDeadlineWithoutCreatingOutbox(t *testing.T) {
    f := newAwaitingTargetFixture(t, "100", -time.Millisecond)
    f.SetWSBoundary("100", "101")
    f.ResolveAndExpireConcurrently()
    require.Equal(t, "missed_contiguous_period", f.DecisionStatus())
    require.Equal(t, 0, f.OutboxCount())
}
```

- [ ] **Step 2: Run resolver tests and confirm RED**

Run: `cd backend; go test ./internal/schemes -run 'ResolveAwaitingTarget|ResolverLoses' -count=1`

Expected: FAIL because the two-stage resolver does not exist.

- [ ] **Step 3: Implement one-decision resolution transaction**

Inside one database transaction:

1. assert the existing strategy shard lease fence;
2. lock the `awaiting_target` decision and instance;
3. read database `now()`;
4. if expired, atomically miss the decision and block only this chain;
5. call the shared in-memory `providerperiodtarget.Current` using the decision's exact source period;
6. if WS current is already after the source or target is not its immediate successor, atomically mark the gap missed;
7. build the frozen request from the already advanced instance state;
8. conditionally mark the decision completed and insert exactly one Outbox with `chain_seq + 1`.

The resolver must not use REST periods, must not advance strategy state, and must not modify financial settlement fields.

- [ ] **Step 4: Expand a lottery boundary into shard-ready events**

`runSchemePeriodBoundaryExpander` queries only `awaiting_target` rows for `event.LotteryCode`, ordered by `id`, with page size 32. For each row it publishes one `ContiguousTargetReady` to the existing scheme shard subject. Its durable message ID is `(decision_id, boundary_generation)`, so redelivery is harmless. The expander performs no target resolution, no third-party request and no unbounded goroutine creation.

- [ ] **Step 5: Wire the durable consumer and close the early-event race**

Start one `runSchemeContiguousTargetConsumer` per configured strategy shard. Each consumer acquires/asserts the same strategy shard lease fence used by `StrategyReady`, then passes the event to `Worker.ProcessContiguousTargetReady`. After Task 5's phase-one transaction commits, `StrategyProcessor.process` immediately calls `ResolveAwaitingTarget(decisionID)` once; this handles the case where the matching WS boundary arrived before the waiting row existed. A transient resolution error is returned for JetStream retry, while the committed waiting row remains recoverable.

- [ ] **Step 6: Add the database safety net**

`RunContiguousTargetRecovery` runs at the existing bounded recovery cadence, not once per scheme. Each iteration lists at most `SCHEME_BETTING_BATCH` rows across configured formal lotteries/shards and calls the same resolver. It must execute once on startup after WS and consumers are ready, then on the configured recovery interval. It must not perform third-party HTTP calls.

- [ ] **Step 7: Test restart and redelivery idempotency**

```go
func TestAwaitingTargetRecoveryAfterRestartIsIdempotent(t *testing.T) {
    f := newAwaitingTargetFixture(t, "100", 2*time.Second)
    f.SetWSBoundary("100", "101")
    f.RunRecoveryTwice()
    require.Equal(t, 1, f.OutboxCount())
    require.Equal(t, int64(1), f.ChainSequenceIncrement())
}
```

- [ ] **Step 8: Run resolver and recovery tests GREEN**

Run: `cd backend; go test ./internal/schemes ./internal/server -run 'AwaitingTarget|PeriodBoundary|ContiguousTarget' -count=1`

Expected: PASS.

- [ ] **Step 9: Commit**

```powershell
git add -- backend/internal/schemes/contiguous_target_resolver.go backend/internal/schemes/contiguous_target_resolver_test.go backend/internal/schemes/contiguous_target_recovery.go backend/internal/schemes/contiguous_target_recovery_test.go backend/internal/schemes/worker.go backend/internal/schemes/strategy_processor.go backend/internal/schemes/formal_command.go backend/internal/server/scheme_period_boundary_bus.go backend/internal/server/scheme_period_boundary_bus_test.go backend/internal/server/server.go
git commit -m "feat: resolve contiguous targets atomically"
```

---

### Task 7: Enforce Restart, Rearm and Round-Reset Policy

**Files:**
- Modify: `backend/internal/schemes/scheme_betting_rearm_recovery.go`
- Modify: `backend/internal/schemes/scheme_betting_rearm_recovery_test.go`
- Modify: `backend/internal/schemes/scheme_betting_startup_takeover.go`
- Modify: `backend/internal/schemes/worker_formal_takeover_test.go`
- Modify: `backend/internal/schemes/scheme_betting_rearm.go`
- Modify: `backend/internal/schemes/scheme_betting_rearm_test.go`
- Modify: `backend/internal/db/sqlcdb/scheme_betting_rearm_ext.go`

**Interfaces:**
- Produces: `isAutomaticRearmAllowed(chainBlockReason, outboxState, outboxReason string) bool`.
- Produces: `ResetSchemeStrategyForNewChain(ctx, schemeID string, expectedStateVersion int64) (newStateVersion int64, err error)`.
- Produces: manual restart semantics: new `chain_id`, `chain_seq=1` for the new initial instruction, round/pick cursors reset to initial configuration.
- Consumes: `scheme_instances.chain_block_reason` and existing authoritative unsent-outbox allowlist.
- Test support: `newBlockedSchemeFixture(t, reason string) *blockedSchemeFixture` seeds a formal event-owned blocked instance with nonzero round/pick/state version and exposes only production `RearmEventScheme` plus read-only state accessors.

- [ ] **Step 1: Write exclusion tests for missed gaps**

```go
func TestAutomaticRearmRejectsMissedContiguousPeriod(t *testing.T) {
    require.False(t, isAutomaticRearmAllowed("missed_contiguous_period", "blocked", "safe_deadline_elapsed"))
}

func TestStartupTakeoverDoesNotRearmMissedContiguousPeriod(t *testing.T) {
    source := fakeTakeoverSource{candidate: blockedCandidate("missed_contiguous_period")}
    result := runStartupTakeover(context.Background(), source, fakeEnabler{})
    require.Zero(t, result.Rearmed)
}
```

- [ ] **Step 2: Write manual restart reset tests**

```go
func TestManualRestartCreatesNewChainAndResetsRound(t *testing.T) {
    f := newBlockedSchemeFixture(t, "missed_contiguous_period")
    oldChain := f.ChainID()
    f.ManualRestart()
    require.NotEqual(t, oldChain, f.ChainID())
    require.Equal(t, int32(0), f.RoundIndex())
    require.Equal(t, int32(0), f.PickIndex())
    require.Empty(t, f.CurrentPick())
    require.Empty(t, f.LastDirection())
    require.Equal(t, f.OldStateVersion()+1, f.StateVersion())
    require.Empty(t, f.ChainBlockReason())
}
```

- [ ] **Step 3: Run policy tests and confirm RED**

Run: `cd backend; go test ./internal/schemes -run 'AutomaticRearmRejects|StartupTakeoverDoesNot|ManualRestartCreates' -count=1`

Expected: FAIL because startup takeover currently treats every `blocked_requires_rearm` chain as recoverable and manual restart does not enforce all reset fields.

- [ ] **Step 4: Apply one shared automatic-rearm predicate**

Both JetStream automatic recovery and startup takeover must call the same predicate. Return false for:

```text
missed_contiguous_period
provider_accepted_wrong_period
provider_acceptance_unknown
```

Keep the current authoritative allowlist only for outcomes proven not to have left the platform. Do not infer safety from a generic `bet_failed` status.

- [ ] **Step 5: Make manual restart a new chain boundary**

Within one transaction:

1. lock the instance and record the old `state_version` and `chain_id`;
2. prove a fresh safe provider target before mutating the instance;
3. update old-chain `awaiting_target` rows to `chain_broken` with `failure_reason='manual_rearm_replaced_chain'`, without creating orders;
4. call `ResetSchemeStrategyForNewChain`, which conditionally sets `round_index=0`, `pick_index=0`, `current_pick=''`, `last_direction=''`, `lookback_round_reset_pending=false` and increments `state_version` by one;
5. reload the reset instance and build the initial frozen request from that state;
6. create a new chain ID, clear `chain_block_reason` and failure detail, and insert the initial decision/Outbox with the new state version and `chain_seq=1`.

If no fresh target exists or frozen-request construction fails, roll back the entire transaction. A stale strategy-ready event carrying the old version must fail its existing state-version fence.

- [ ] **Step 6: Run policy tests GREEN**

Run: `cd backend; go test ./internal/schemes ./internal/db/sqlcdb -run 'Rearm|Takeover|ManualRestart|ActivateSchemeBettingChain' -count=1`

Expected: PASS.

- [ ] **Step 7: Commit**

```powershell
git add -- backend/internal/schemes/scheme_betting_rearm_recovery.go backend/internal/schemes/scheme_betting_rearm_recovery_test.go backend/internal/schemes/scheme_betting_startup_takeover.go backend/internal/schemes/worker_formal_takeover_test.go backend/internal/schemes/scheme_betting_rearm.go backend/internal/schemes/scheme_betting_rearm_test.go backend/internal/db/sqlcdb/scheme_betting_rearm_ext.go
git commit -m "fix: enforce contiguous chain rearm policy"
```

---

### Task 8: Expose Developer Diagnostics Without Mutating Runtime

**Files:**
- Modify: `backend/internal/handler/admin_runtime_diagnostics.go`
- Modify: `backend/internal/handler/admin_runtime_diagnostics_test.go`
- Modify: `backend/internal/server/server.go`
- Modify: `backend/internal/guaji/draw_boundary_health.go`

**Interfaces:**
- Produces: admin-only diagnostic fields `drawWS`, `periodBoundary`, `awaitingTarget`, `chainBlockReason`, and a single precedence-ordered `blockReason`.
- Consumes: read-only WebSocket health snapshot, period state, latest decision, Outbox and chain state.

- [ ] **Step 1: Write blocker precedence tests**

```go
func TestSchemeRuntimeBlockReasonPrefersTerminalGap(t *testing.T) {
    input := runtimeBlockInput{
        ChainBlockReason: "missed_contiguous_period",
        DrawWSStale: true,
        AwaitingTarget: true,
    }
    require.Equal(t, "missed_contiguous_period", schemeRuntimeBlockReason(input))
}

func TestSchemeRuntimeBlockReasonReportsWaitingBeforeDeadline(t *testing.T) {
    input := runtimeBlockInput{AwaitingTarget: true, DeadlineExpired: false, DrawWSStale: false}
    require.Equal(t, "next_period_unavailable", schemeRuntimeBlockReason(input))
}
```

- [ ] **Step 2: Run diagnostics tests and confirm RED**

Run: `cd backend; go test ./internal/handler -run TestSchemeRuntimeBlockReason -count=1`

Expected: FAIL because current diagnostics inspect only local preflight state.

- [ ] **Step 3: Return a complete read-only timeline**

For the requested instance, return:

```json
{
  "blockReason": "missed_contiguous_period",
  "chainBlockReason": "missed_contiguous_period",
  "sourcePeriod": "100",
  "targetPeriod": "101",
  "decisionStatus": "missed_contiguous_period",
  "targetDeadlineAt": "2026-08-21T10:00:05.000+08:00",
  "drawWS": {
    "connected": true,
    "lastFrameAt": "2026-08-21T10:00:04.100+08:00",
    "lastPongAt": "2026-08-21T10:00:04.000+08:00",
    "reconnects": 2
  },
  "periodBoundary": {
    "currentIssue": "101",
    "nextIssue": "102",
    "receivedAt": "2026-08-21T10:00:06.050+08:00",
    "wsRestLagPeriods": 0
  }
}
```

The values are examples of the response shape. The implementation must source them from current rows/snapshots and redact credentials and provider response bodies.

- [ ] **Step 4: Apply stable blocker precedence**

Use this order:

1. `provider_accepted_wrong_period`
2. `provider_acceptance_unknown`
3. `missed_contiguous_period`
4. `strategy_evaluation_failed`
5. `draw_missing`
6. `draw_ws_stale`
7. `next_period_unavailable`
8. existing preflight reasons

- [ ] **Step 5: Run diagnostics tests GREEN**

Run: `cd backend; go test ./internal/handler ./internal/server -run 'RuntimeDiagnostic|BlockReason|DrawWSHealth' -count=1`

Expected: PASS; the endpoint remains read-only.

- [ ] **Step 6: Commit**

```powershell
git add -- backend/internal/handler/admin_runtime_diagnostics.go backend/internal/handler/admin_runtime_diagnostics_test.go backend/internal/server/server.go backend/internal/guaji/draw_boundary_health.go
git commit -m "feat: diagnose contiguous betting stalls"
```

---

### Task 9: End-to-End, Concurrency and Performance Verification

**Files:**
- Create: `backend/internal/schemes/contiguous_chain_e2e_test.go`
- Create: `backend/internal/schemes/contiguous_chain_fault_test.go`
- Modify: `backend/internal/schemes/strategy_processor_lifecycle_test.go`
- Modify: `docs/superpowers/specs/2026-08-21-short-period-contiguous-chain-recovery-design.md` only if verification discovers a factual mismatch; do not weaken accepted requirements.

**Interfaces:**
- Verifies: `source N -> target N+1`, one strategy advance, one decision, one Outbox, one external request ID.
- Verifies: no automatic order for `N+2` after missing `N+1`; manual restart begins a new chain at round one.
- Verifies: one lottery failure does not block unrelated lotteries or shards.
- Test support: `newContiguousChainE2EFixture(t, lotteryCode string) *contiguousChainE2EFixture` composes the production strategy processor, in-memory period state, fake boundary bus and fake dispatcher over a rollback-only database transaction; it never calls Guaji.

- [ ] **Step 1: Write the complete lifecycle test**

```go
func TestFormalShortPeriodContiguousLifecycle(t *testing.T) {
    f := newContiguousChainE2EFixture(t, "tron_ffc_6s")
    accepted := f.PlaceInitialBet("100")
    f.IngestDraw(accepted.Period, []string{"1", "2", "3", "4", "5"})
    f.PublishBoundary("100", "101")
    next := f.DispatchOne()
    require.Equal(t, "101", next.TargetPeriod)
    require.Equal(t, accepted.MultiplierAfterLoss(), next.Multiplier)
    require.Equal(t, accepted.RoundAfterLoss(), next.Round)
    require.Equal(t, 1, f.DecisionCount("100"))
    require.Equal(t, 1, f.OutboxCountForSource("100"))
}
```

- [ ] **Step 2: Write fault-injection tests**

Cover these exact failures:

- REST inserts draw before WS duplicate arrives;
- boundary event arrives before phase-one commit;
- JetStream redelivers draw, strategy-ready and boundary events;
- process restarts with an unexpired waiting decision;
- process restarts with an expired waiting decision;
- WS remains connected but one lottery becomes stale;
- resolver and expiry worker race;
- third party accepts a period different from frozen target;
- provider response is unknown and exact-fingerprint reconciliation is pending;
- one shard's database call fails while another shard continues.

- [ ] **Step 3: Run targeted tests with race detection**

Run:

```powershell
cd backend
go test -race ./internal/guaji ./internal/guaji/drawsync ./internal/schemeeventbus ./internal/schemes ./internal/server -run 'Contiguous|Boundary|DrawWS|FormalStrategy|Rearm' -count=1
```

Expected: PASS with no data races or goroutine leaks.

- [ ] **Step 4: Verify query plans and bounded work**

On a migrated development database, run `EXPLAIN (ANALYZE, BUFFERS)` for:

```sql
SELECT id
FROM scheme_period_decisions
WHERE lottery_code = 'tron_ffc_6s'
  AND status = 'awaiting_target'
  AND id > 0
ORDER BY id
LIMIT 32;
```

Expected: the partial waiting-target index is used; no sequential scan of all decisions or schemes. Verify one boundary produces bounded page queries and zero third-party HTTP requests per scheme.

- [ ] **Step 5: Run backend-wide verification**

Run:

```powershell
cd backend
go test ./... -count=1
go vet ./...
go build ./cmd/server
```

Expected: all commands exit 0. If an unrelated pre-existing failure remains, capture the exact package/test and prove it reproduces on the plan's parent commit before excluding it.

- [ ] **Step 6: Apply migration and perform a read-only deployment preflight**

Run:

```powershell
cd backend
go run ./cmd/migrate status
go run ./cmd/migrate up
go run ./cmd/migrate status
```

Expected: database reaches version 177 and the server starts without unknown decision-status or column errors.

- [ ] **Step 7: Perform the authorized gray canary**

With `SCHEME_BETTING_MODE=gray` and only `tron_ffc_6s` enabled, manually start a fresh or explicitly restarted instance and observe at least 100 physical periods. Acceptance requires:

- every order has `source N -> target N+1`;
- multiplier, round and pick derive from source `N`'s local verdict;
- no duplicate decision, Outbox, request ID or provider order;
- strategy evaluation plus target Outbox is within 3 seconds under a healthy WS;
- forced WS interruption reconnects; if `N+1` is missed, the chain pauses with `missed_contiguous_period` and never orders `N+2`;
- manual restart creates a new chain and starts at round one;
- other lotteries and shards continue processing during the fault.

Do not reconstruct missing provider orders, replay historical physical periods or restart a production instance without explicit authorization.

- [ ] **Step 8: Commit final verification artifacts**

```powershell
git add -- backend/internal/schemes/contiguous_chain_e2e_test.go backend/internal/schemes/contiguous_chain_fault_test.go backend/internal/schemes/strategy_processor_lifecycle_test.go docs/superpowers/specs/2026-08-21-short-period-contiguous-chain-recovery-design.md
git commit -m "test: verify contiguous short-period chains"
```

---

## Release Order and Rollback Gate

1. Apply migration 177 before deploying code that writes new statuses.
2. Start the shared draw WS, JetStream consumers and shard leases before the one-time waiting-decision recovery scan.
3. Keep gray scope limited to `tron_ffc_6s` until the 100-period canary passes.
4. Expand lotteries only after WS health, strategy latency, waiting depth, Outbox latency and JetStream pending metrics remain stable.
5. Before application rollback, drain every `awaiting_target` decision to `completed` or `missed_contiguous_period`; old code must never encounter an unknown live state.
6. A rollback must preserve accepted bets, immutable rule snapshots, provider identifiers and diagnostic records; it must not fabricate or replay orders.
