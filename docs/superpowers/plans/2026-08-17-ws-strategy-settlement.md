# WebSocket 策略推进与数据库规则 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让已验证玩法在 WebSocket 开奖后独立完成本地策略推进，同时把玩法规则、版本、验证和灰度状态统一存入数据库，并保持第三方资金结算为最终账务来源。

**Architecture:** 现有玩法目录继续由 `play_templates → play_types → sub_plays` 表达；`play_rule_specs` 保存每个目录玩法当前发布的规则，`play_rule_spec_revisions` 保存草稿和审计。第三方接单成功时冻结可执行规则快照；持久化的策略判定记录在开奖后驱动每个实例串行推进，第三方批量结算只负责账务与结果对账。

**Tech Stack:** Go 1.24+, PostgreSQL 18, pgx/sqlc/goose, Vue 3/Vite 管理端，Guaji REST/WebSocket。

## Global Constraints

- Excel 仅用于一次性导入和人工核对；运行、测试与后台编辑只读取数据库规则。
- 不创建平行玩法目录；规则以 `(template_code, type_id, sub_id, lottery_code nullable)` 关联现有目录。
- 开奖、策略推进和下注热路径不得逐方案 JOIN 规则表；使用已发布规则缓存与下注时冻结快照。
- 不得产生 `target=N accepted=N+1`；临近封盘或队列超时只能跳过该期并记录开发诊断。
- 第三方 `settled`、`net_amount`、`payout_amount` 是资金唯一真值；本地策略只推进倍数、轮次和出号游标。
- 未验证规则、未知缓存版本和本地/第三方结果不一致时，只暂停受影响实例/玩法，不阻塞其他用户或实例。
- 每项行为变更先写失败测试并确认失败，再写最小实现；每个任务独立提交。

---

## File map

- `backend/migrations/00149_play_rule_specs.sql` — 规则当前版本、修订、策略判定持久化和快照列的 DDL/索引。
- `backend/sqlc.yaml` — 将 `00149` 纳入 sqlc schema 顺序。
- `backend/internal/db/queries/play_rule_specs.sql` — 规则加载、修订、发布、灰度开关与目录完整性查询。
- `backend/internal/db/queries/strategy_evaluations.sql` — 策略判定 claim、完成、跳过、恢复扫描与诊断查询。
- `backend/internal/db/queries/cloud_bet_records.sql` — 读写已接单注单的规则快照与策略处理状态。
- `backend/internal/db/sqlcdb/*.go` — `make sqlc` 生成的查询类型，禁止手改。
- `backend/internal/playrules/registry.go` — 已发布规则加载、版本缓存、作用域回退与失效。
- `backend/internal/playrules/spec.go` — `evaluation_spec` 的受控结构、`evaluator_key`、样例与快照类型。
- `backend/internal/playrules/registry_test.go` — 作用域、版本和未知缓存的单元测试。
- `backend/cmd/import-play-rules/main.go` — 只读 Excel 导入为 `draft` 修订的命令；不发布。
- `backend/cmd/import-play-rules/main_test.go` — Excel 行到 draft 修订的映射和歧义拒绝测试。
- `backend/internal/schemes/rule_evaluator.go` — 将已冻结规则规范化为现有受控判定器输入并调用共享判定入口。
- `backend/internal/schemes/rule_evaluator_test.go` — 数据库规则驱动的 SSC/LHC 判定、样例和未知判定器测试。
- `backend/internal/cloud/schemestate/strategy.go` — 只推进倍投轮次/出号游标、不记资金的事务函数。
- `backend/internal/cloud/schemestate/strategy_test.go` — 命中、未中、重复期号和不改账务的测试。
- `backend/internal/schemes/strategy_processor.go` — 分片、持久化 claim、恢复扫描、封盘过载跳过与实例隔离。
- `backend/internal/schemes/worker.go` — 注册策略处理器；用“前一期策略已完成”替换真实盘的“等待第三方 settled”拦截。
- `backend/internal/schemes/worker_guaji.go`、`worker_bet_dedup.go` — 接单成功时写入冻结快照和接单期号。
- `backend/internal/guaji/drawsync/worker.go` — 新开奖写库后唤醒策略处理器。
- `backend/internal/guaji/accountsvc/payout_sync.go` — 按账户批量拉取第三方注单、写资金、做策略/第三方对账但不二次推进轮次。
- `backend/internal/handler/admin_play_rules.go`、`admin_runtime_diagnostics.go`、`backend/internal/server/server.go` — 规则发布 API、灰度开关和开发诊断路由。
- `admin/src/api/playRules.ts`、`admin/src/views/PlayRulesView.vue` — 管理员查看、发布、停用规则及查看验证状态。

### Task 1: 建立数据库规则、快照与持久化策略判定模型

**Files:**
- Create: `backend/migrations/00149_play_rule_specs.sql`
- Modify: `backend/sqlc.yaml`
- Create: `backend/internal/db/queries/play_rule_specs.sql`
- Create: `backend/internal/db/queries/strategy_evaluations.sql`
- Modify: `backend/internal/db/queries/cloud_bet_records.sql`
- Modify: `backend/internal/db/sqlcdb/*.go` via `make sqlc`
- Test: `backend/internal/db/sqlcdb/play_rule_specs_integration_test.go`

**Interfaces:**
- Produces `PublishedPlayRuleSpec`, `ClaimStrategyEvaluation`, `CompleteStrategyEvaluation`, `ListRecoverableStrategyEvaluations` sqlc APIs.
- Persists one immutable `rule_snapshot` JSONB and `rule_version` on every third-party accepted `cloud_bet_records` row.

- [ ] **Step 1: Write failing migration/query integration tests**

```go
func TestPublishedRuleUsesLotteryOverrideAndDefault(t *testing.T) {
    // Insert one default and one tron_ffc_3s override for the same catalog key.
    // Assert the override is returned for tron_ffc_3s and default for another lottery.
}

func TestStrategyEvaluationClaimIsUniquePerInstancePeriod(t *testing.T) {
    // Claim the same (instance_id, period_no) twice and assert only the first succeeds.
}
```

- [ ] **Step 2: Run the targeted tests and verify the expected missing-table/query failure**

Run: `go test ./internal/db/sqlcdb -run "TestPublishedRuleUsesLotteryOverrideAndDefault|TestStrategyEvaluationClaimIsUniquePerInstancePeriod" -count=1`

Expected: FAIL because `play_rule_specs` and strategy evaluation query APIs do not exist.

- [ ] **Step 3: Add migration `00149_play_rule_specs.sql`**

Create:

```sql
CREATE TABLE play_rule_specs (
  id BIGSERIAL PRIMARY KEY,
  template_code VARCHAR(32) NOT NULL,
  type_id VARCHAR(32) NOT NULL,
  sub_id VARCHAR(64) NOT NULL,
  lottery_code VARCHAR(64),
  rule_version INT NOT NULL,
  evaluator_key VARCHAR(64) NOT NULL,
  evaluator_version INT NOT NULL,
  evaluation_spec JSONB NOT NULL,
  sample_cases JSONB NOT NULL DEFAULT '[]'::jsonb,
  source_meta JSONB NOT NULL DEFAULT '{}'::jsonb,
  strategy_enabled BOOLEAN NOT NULL DEFAULT false,
  published_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE NULLS NOT DISTINCT (template_code, type_id, sub_id, lottery_code)
);
```

Add `play_rule_spec_revisions` with revision payload, status check (`draft`, `verified`, `published`, `disabled`), actor/reason and timestamps. Add `scheme_strategy_evaluations` with unique `(instance_id, period_no)`, status check (`pending`, `processing`, `completed`, `skipped`, `mismatch`), record/order linkage, local hit/unit fields, diagnostics JSONB and indexed recovery lookup. Add nullable rule snapshot/version/hash columns to `cloud_bet_records`, plus indexes for enabled-rule lookup and pending strategy recovery.

- [ ] **Step 4: Add sqlc queries and generate types**

Implement explicit default/lottery override lookup with `ORDER BY (lottery_code IS NOT NULL) DESC`; implement revision insertion/publish transaction helpers, strategy `INSERT ... ON CONFLICT DO NOTHING`, `FOR UPDATE SKIP LOCKED` recovery selection, completion/skip/mismatch updates, and cloud-bet snapshot read/write queries. Append migration `00149` to `backend/sqlc.yaml`, then run `make sqlc`.

- [ ] **Step 5: Run migration/query tests and commit**

Run: `go test ./internal/db/sqlcdb -run "TestPublishedRuleUsesLotteryOverrideAndDefault|TestStrategyEvaluationClaimIsUniquePerInstancePeriod" -count=1`

Expected: PASS.

Commit: `git commit -m "feat: persist play rules and strategy evaluations"`

### Task 2: 实现数据库规则缓存与受控规则结构

**Files:**
- Create: `backend/internal/playrules/spec.go`
- Create: `backend/internal/playrules/registry.go`
- Create: `backend/internal/playrules/registry_test.go`
- Modify: `backend/internal/server/server.go`

**Interfaces:**
- `type Locator struct { TemplateCode, TypeID, SubID, LotteryCode string }`
- `type Snapshot struct { RuleVersion, EvaluatorVersion int; EvaluatorKey, ContentHash string; Spec json.RawMessage }`
- `func (r *Registry) Resolve(ctx context.Context, loc Locator) (Snapshot, error)`
- `func (r *Registry) Reload(ctx context.Context) error`

- [ ] **Step 1: Write failing registry tests**

```go
func TestRegistryReturnsLotteryOverrideFromOneLoad(t *testing.T) { /* cache hit uses override */ }
func TestRegistryRejectsUnknownOrDisabledStrategyRule(t *testing.T) { /* ErrRuleUnavailable */ }
func TestSnapshotHashChangesWhenRuleVersionChanges(t *testing.T) { /* immutable hash */ }
```

- [ ] **Step 2: Run tests and verify RED**

Run: `go test ./internal/playrules -run "TestRegistry" -count=1`

Expected: FAIL because package and types do not exist.

- [ ] **Step 3: Implement fixed JSON schema and cache**

Define per-key validation for `ssc.*`, `lhc.*`, `pk10.*`, `syxw.*`, `k3.*`, and `pc28.*`; reject unknown evaluator keys and unknown JSON fields. Load only `strategy_enabled=true` published rows into an immutable map guarded by `sync.RWMutex`. Hash canonical JSON plus evaluator/rule versions using SHA-256. Register the cache in `server.go`; reload on publish notification and a bounded version poll.

- [ ] **Step 4: Run tests and commit**

Run: `go test ./internal/playrules -run "TestRegistry" -count=1`

Expected: PASS.

Commit: `git commit -m "feat: add cached database play rule registry"`

### Task 3: 导入 Excel 为草稿修订并提供管理员发布控制

**Files:**
- Create: `backend/cmd/import-play-rules/main.go`
- Create: `backend/cmd/import-play-rules/main_test.go`
- Create: `backend/internal/handler/admin_play_rules.go`
- Modify: `backend/internal/server/server.go`
- Test: `backend/internal/handler/admin_play_rules_test.go`

**Interfaces:**
- CLI: `go run ./cmd/import-play-rules -file <xlsx> -dry-run|-apply`.
- API: `GET /api/v1/admin/play-rules`, `POST /api/v1/admin/play-rules/{id}/verify`, `POST /api/v1/admin/play-rules/{id}/publish`, `POST /api/v1/admin/play-rules/{id}/disable`.

- [ ] **Step 1: Write failing importer and handler tests**

```go
func TestImportRejectsAmbiguousExcelRuleName(t *testing.T) { /* no publish row is created */ }
func TestPublishRequiresVerifiedRevision(t *testing.T) { /* draft returns HTTP 409 */ }
func TestPublishInvalidatesOnlyAffectedRuleCache(t *testing.T) { /* new version visible */ }
```

- [ ] **Step 2: Run tests and verify RED**

Run: `go test ./cmd/import-play-rules ./internal/handler -run "TestImportRejectsAmbiguousExcelRuleName|TestPublishRequiresVerifiedRevision|TestPublishInvalidatesOnlyAffectedRuleCache" -count=1`

Expected: FAIL because importer and publish endpoints do not exist.

- [ ] **Step 3: Implement importer and publication transaction**

Use `github.com/xuri/excelize/v2` to read columns A, E, F, G, H-L and P. Match only when catalog natural key, Guaji rule id and full name produce one candidate; otherwise insert a draft revision with an ambiguity diagnostic. `-dry-run` prints counts and never writes. `-apply` creates draft revisions only. Publish validates `evaluation_spec` and all stored sample cases, promotes the selected revision in one transaction, writes `admin_audit_logs`, and emits a cache invalidation notification.

- [ ] **Step 4: Run tests and commit**

Run: `go test ./cmd/import-play-rules ./internal/handler -run "TestImportRejectsAmbiguousExcelRuleName|TestPublishRequiresVerifiedRevision|TestPublishInvalidatesOnlyAffectedRuleCache" -count=1`

Expected: PASS.

Commit: `git commit -m "feat: import and publish database play rules"`

### Task 4: 让模拟与真实方案共用数据库规则驱动的判定入口

**Files:**
- Create: `backend/internal/schemes/rule_evaluator.go`
- Create: `backend/internal/schemes/rule_evaluator_test.go`
- Modify: `backend/internal/schemes/worker_play.go`
- Modify: `backend/internal/schemes/worker_sim_settle.go`
- Modify: `backend/internal/schemes/betting_preview.go`

**Interfaces:**
- `func evaluateFrozenRule(snapshot playrules.Snapshot, base playRule, balls []string, content string, contrary bool, contraryContent string) (betEvaluation, error)`.
- `ErrStrategyRuleUnavailable` means no local strategy state may be advanced.

- [ ] **Step 1: Write failing evaluator tests**

```go
func TestFrozenSSCDirectRuleMatchesKnownSample(t *testing.T) { /* hit and miss */ }
func TestFrozenLHCGuoguanRulePreservesEmptyPositions(t *testing.T) { /* 大,单,,大,,双 */ }
func TestUnknownEvaluatorNeverFallsBackToLabelHeuristics(t *testing.T) { /* ErrStrategyRuleUnavailable */ }
```

- [ ] **Step 2: Run tests and verify RED**

Run: `go test ./internal/schemes -run "TestFrozen(SSC|LHC)|TestUnknownEvaluator" -count=1`

Expected: FAIL because `evaluateFrozenRule` does not exist.

- [ ] **Step 3: Implement the shared adapter**

Compile only validated `evaluation_spec` parameters into `playRule` and dispatch through the existing typed evaluators (`evaluateSSCByBetMode`, `evaluateLHCByBetMode`, and peers). Do not execute database text as code and do not infer behavior from a display label. Route simulation settlement and strategy evaluation through this adapter; retain existing preview behavior until a published rule exists, then use the same adapter for preview.

- [ ] **Step 4: Run tests and commit**

Run: `go test ./internal/schemes -run "TestFrozen(SSC|LHC)|TestUnknownEvaluator|TestLHC" -count=1`

Expected: PASS.

Commit: `git commit -m "feat: evaluate schemes from frozen rule specs"`

### Task 5: 以持久化 WebSocket 策略处理器推进真实方案

**Files:**
- Create: `backend/internal/cloud/schemestate/strategy.go`
- Create: `backend/internal/cloud/schemestate/strategy_test.go`
- Create: `backend/internal/schemes/strategy_processor.go`
- Create: `backend/internal/schemes/strategy_processor_test.go`
- Modify: `backend/internal/schemes/worker.go`
- Modify: `backend/internal/schemes/worker_guaji.go`
- Modify: `backend/internal/schemes/worker_bet_dedup.go`
- Modify: `backend/internal/guaji/drawsync/worker.go`

**Interfaces:**
- `func ProcessStrategyAfterDraw(ctx context.Context, q *sqlcdb.Queries, inst sqlcdb.SchemeInstance, period string, hit bool, config []byte) error`.
- `func (p *StrategyProcessor) NotifyDraw(lotteryCode, period string)`.
- `func (p *StrategyProcessor) Recover(ctx context.Context) error`.

- [ ] **Step 1: Write failing state and processor tests**

```go
func TestProcessStrategyAfterDrawAdvancesRoundWithoutPnl(t *testing.T) { /* round/pick change; turnover/pnl unchanged */ }
func TestStrategyProcessorHandlesOneInstancePeriodOnce(t *testing.T) { /* duplicate NotifyDraw */ }
func TestRecoveryProcessesPersistedDrawAfterRestart(t *testing.T) { /* no websocket required */ }
func TestNearCloseMarksOnlyThatInstancePeriodSkipped(t *testing.T) { /* no cross-period bet */ }
```

- [ ] **Step 2: Run tests and verify RED**

Run: `go test ./internal/cloud/schemestate ./internal/schemes -run "TestProcessStrategyAfterDraw|TestStrategyProcessor|TestRecovery|TestNearClose" -count=1`

Expected: FAIL because strategy-only transition and processor do not exist.

- [ ] **Step 3: Implement strategy-only state transition**

Extract round and pick advancement from `ProcessAfterSettlement` into `ProcessStrategyAfterDraw`. This function must not update `pnl`, wallet, ledger or lookback amount fields. It locks the instance, reads the accepted cloud-bet snapshot, runs `evaluateFrozenRule`, records the strategy result, then applies only `round_index`, `pick_index`, `current_pick` and `last_direction` once.

- [ ] **Step 4: Implement durable processor and worker integration**

Instantiate `StrategyProcessor` in `server.go`, notify it after successful draw persistence, and run bounded recovery scans on startup and at a short interval. Partition by instance ID, claim rows with `FOR UPDATE SKIP LOCKED`, and keep duplicate notifications idempotent. On third-party acceptance, write the rule snapshot before the record becomes eligible. Replace `hasUnsettledGuajiBet` as the next-bet gate with “previous accepted period has completed local strategy evaluation” only for the enabled rule; retain current behavior for disabled rollout keys.

- [ ] **Step 5: Run tests and commit**

Run: `go test ./internal/cloud/schemestate ./internal/schemes -run "TestProcessStrategyAfterDraw|TestStrategyProcessor|TestRecovery|TestNearClose" -count=1`

Expected: PASS.

Commit: `git commit -m "feat: advance enabled schemes from websocket draws"`

### Task 6: 批量第三方结算、对账与开发者诊断

**Files:**
- Modify: `backend/internal/guaji/accountsvc/payout_sync.go`
- Create: `backend/internal/guaji/accountsvc/payout_sync_batch_test.go`
- Modify: `backend/internal/handler/admin_runtime_diagnostics.go`
- Modify: `backend/internal/server/server.go`
- Test: `backend/internal/handler/admin_runtime_diagnostics_test.go`

**Interfaces:**
- `GET /api/v1/admin/diagnostics/schemes/{instanceId}/strategy` returns strategy period, snapshot hash/version, draw time, processor timings, third-party status and mismatch category.
- `syncAccountPending(ctx, accountID)` fetches one third-party list page/range and settles all matched local rows for that account.

- [ ] **Step 1: Write failing batch/reconciliation tests**

```go
func TestPayoutSyncFetchesOneAccountListForMultiplePendingOrders(t *testing.T) { /* one remote list call */ }
func TestPayoutSyncDoesNotAdvanceStrategySecondTime(t *testing.T) { /* financial settlement only */ }
func TestMismatchComparesContentHashAndHitNotRoundedPnl(t *testing.T) { /* mismatch category */ }
func TestStrategyDiagnosticsExposeBlockedReason(t *testing.T) { /* admin 200 */ }
```

- [ ] **Step 2: Run tests and verify RED**

Run: `go test ./internal/guaji/accountsvc ./internal/handler -run "TestPayoutSyncFetchesOneAccountList|TestPayoutSyncDoesNotAdvanceStrategy|TestMismatchCompares|TestStrategyDiagnostics" -count=1`

Expected: FAIL because account batching and strategy diagnostics do not exist.

- [ ] **Step 3: Implement account batch settlement and reconciliation**

Group pending rows by `guaji_account_id`, use one bounded list fetch per account, map `third_party_bet_id` to local orders, and settle only mapped rows. Keep the existing historical-page recovery only for IDs absent from the list. When a strategy evaluation exists, compare accepted period, normalized content hash, hit/miss and winning-unit count; record a classified mismatch and pause only that instance. Do not call `ProcessFormalAfterSettlement` for an already strategy-completed period.

- [ ] **Step 4: Add developer diagnostics and run tests**

Expose ordered timestamps for draw ingestion, strategy claim/completion, first settlement query, provider settlement and financial commit. Restrict the route to admin authentication; do not expose it to the client UI.

Run: `go test ./internal/guaji/accountsvc ./internal/handler -run "TestPayoutSyncFetchesOneAccountList|TestPayoutSyncDoesNotAdvanceStrategy|TestMismatchCompares|TestStrategyDiagnostics" -count=1`

Expected: PASS.

Commit: `git commit -m "feat: batch guaji settlement and reconcile strategy"`

### Task 7: 管理端规则管理、灰度开关与端到端验证

**Files:**
- Create: `admin/src/api/playRules.ts`
- Create: `admin/src/views/PlayRulesView.vue`
- Modify: `admin/src/router/index.ts`
- Modify: `admin/src/layouts/AdminLayout.vue`
- Create: `backend/internal/schemes/strategy_e2e_test.go`
- Modify: `backend/internal/schemes/e2e_lifecycle_test.go`

**Interfaces:**
- 管理页面显示目录定位、当前版本、修订状态、`strategy_enabled`、样例/影子对账结果和发布/停用操作。
- 真实策略开关只允许管理员对一个 `(lottery_code, evaluator_key)` 打开或关闭。

- [ ] **Step 1: Write failing API/client tests**

```ts
it('shows a published rule and disables strategy without deleting its revision', async () => { /* API-backed view */ })
```

```go
func TestEnabledStrategyAdvancesNextBetBeforeThirdPartyFinancialSettlement(t *testing.T) { /* 60s provider delay */ }
func TestDisabledStrategyKeepsLegacySettlementGate(t *testing.T) { /* rollout safety */ }
func TestMultipleInstancesAdvanceIndependentlyWithoutCrossPeriod(t *testing.T) { /* parallel */ }
```

- [ ] **Step 2: Run tests and verify RED**

Run: `npm.cmd test -- --run src/views/PlayRulesView.spec.ts`

Run: `go test ./internal/schemes -run "TestEnabledStrategy|TestDisabledStrategy|TestMultipleInstancesAdvance" -count=1`

Expected: FAIL because the management page and enabled strategy E2E path do not exist.

- [ ] **Step 3: Implement admin page and feature-flag UX**

Add an admin-only route and navigation item. Render current published rule separately from draft revisions, show verification failures without permitting publish, and require a typed change reason for publish/disable. The client must never read Excel or create rule JSON dynamically; it sends only API commands.

- [ ] **Step 4: Implement end-to-end rollout tests, run verification, and commit**

Run: `go test ./internal/schemes -run "TestEnabledStrategy|TestDisabledStrategy|TestMultipleInstancesAdvance" -count=1`

Run: `npm.cmd test -- --run src/views/PlayRulesView.spec.ts`

Run: `go test ./...`

Run: `npm.cmd run build` from `admin`

Expected: targeted tests, full backend tests and admin build pass; any existing unrelated failure is recorded with exact test name and output before commit.

Commit: `git commit -m "feat: manage strategy rule rollout"`

## Final verification and rollout

- [ ] Run `git diff --check` and verify only planned files are staged.
- [ ] Run `go test ./internal/playrules ./internal/cloud/schemestate ./internal/schemes ./internal/guaji/accountsvc ./internal/handler -count=1`.
- [ ] Run `go build ./cmd/server` from `backend`.
- [ ] Run `npm.cmd run build` from `admin`.
- [ ] Apply migration to a non-production database, import the Excel file with `-dry-run`, then `-apply`; confirm every imported row remains `draft` until verification/publish.
- [ ] Keep all `strategy_enabled` flags off in production for the first deploy; enable one verified `(lottery_code, evaluator_key)` only after the strategy diagnostics show no mismatches.
