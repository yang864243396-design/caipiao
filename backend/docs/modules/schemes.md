# 方案域 — 产品与后端逻辑

> **状态**：**产品定案已全部闭合**；**OpenAPI / contracts 已落稿**（§8）。  
> **关联**：[`openapi/openapi.yaml`](../../openapi/openapi.yaml)、[`contracts/schemes.ts`](../../contracts/schemes.ts)

---

## 1. 范围

**自创 / 反买 / 跟单**；会员 **私池** + **分享池**；倍投/期次/回头；云端全局规则；Admin **全站方案监控**（用户 Tab + 分享池 Tab）。

---

## 2. 核心概念：方案 vs 实例

| 概念 | 说明 | 在哪 |
|------|------|------|
| **方案（方案定义）** | 配置档案 | 私池 |
| **实例（云端实例）** | 执行单元：状态、按期下注、注单 | 云端中心、后台用户 Tab |

```text
方案定义 ──1:1──▶ 云端实例 ──每期──▶ 注单
```

- 实例在 **添加至云端 / 投注** 时 **显式创建**，非跑起来才生成。
- **多路并行** = 多份 **独立方案**（各 1 实例），互无关联（D76）。
- **分享池快照** = frozen 配置；与会员私池 **永久脱钩**；**仅运营** 删改。

---

## 3. 术语

| 动作 | 结果 |
|------|------|
| **添加至云端（配置页 · 无实例）** | 1 实例 pending；自创可选公开 → 或写分享池 |
| **添加至云端（配置页 · 已有实例）** | **复制新方案**（整包配置）+ 新 pending（D77）；新方案 **私密** |
| **添加至云端（跟单链路/下载）** | 新 **跟单** 方案 + pending（源快照为 **自创** 配置） |
| **投注（跟单/反买）** | 新方案 + running |

---

## 4. 产品定案

### 4.1 类型与入口

| 类型 | 入口 | 可进分享池（会员） |
|------|------|-------------------|
| **自创** | 新增方案 → 配置 | **可**（首次添加 + 选公开） |
| **跟单** | 模板；分享池/下载；投注 Tab | **否** |
| **反买** | 计划反集 · 投注 | **否** |

### 4.2 方案 · 实例

- **1 方案定义 : 1 实例**。
- 第二路 = 新建/复制 **独立方案** + 新实例（D76）。
- **已有实例再点添加至云端**（D77）：自动复制；名称 `原名-2`，占用则 `-3`、`-4`…（**D81**）；**整包复制** 倍投/期次/内容/锁定字段（**D82**）；新方案 **kind 与源相同**（自创→自创）；**默认私密**。
- 防连点 **1 秒** → 报错（D67）。

### 4.3 删除（D75、D79）

- **仅实例非 running** 可删方案 → 级联删实例；**running 禁止删**。
- 删后可 **同名** 新建并再添加至云端。

### 4.4 分享池（D78、D80）

| 规则 | 说明 |
|------|------|
| 会员写入 | **仅自创** + **首次** 添加 + 选 **公开** |
| 快照 type | **恒为自创**（D80）；**不可能** 出现跟单/反买型快照（含运营录入） |
| 运营 | 分享池 Tab：**PATCH 完整 config** / DELETE（**不可改 type**）；下载 **立即** 更新（D72） |
| 脱钩 | 会员改私池不同步；删会员方案不影响快照；删/改快照不影响已复制私池（D70） |
| 展示 | **不展示** 发布者昵称（D71） |
| 复制出 | 会员从池复制 → 私池方案 type=**跟单**（配置来自自创快照） |

### 4.5 配置与修改

- **永久锁定**：名称、彩种、运行/玩法/子玩法。
- **方案名**：**同会员内** 唯一。
- **分享状态**：仅自创、首次添加前可选；之后不可改。
- **运行中**：可改非倍投/期次，下一 **官方期号** 生效；不可改倍投/期次。
- **回头**：不暂停/封停；仅重置倍投轮次。

### 4.6 权限

| 角色 | 允许 | 禁止 |
|------|------|------|
| 会员 | 改非锁定；恢复 paused（§6.4）；删非 running | 解封；改分享池 |
| 运营 | 强停；解封→paused；分享池 PATCH/DELETE（**自创快照 only**） | 改会员私池；造跟单/反买快照 |

---

## 5. 数据流

```text
[自创 · 首次添加至云端]
    ├─ 私密 → pending
    └─ 公开 → pending + 分享池快照（type=自创）

[已有实例 · 再点添加至云端] → fork 新方案（私密，整包复制）+ pending

[分享池 · 自创快照] ──添加至云端/下载──▶ 会员私池 type=跟单 + pending
[分享池] ──投注 Tab 投注──▶ 跟单 + running
[计划反集 · 投注] ──▶ 反买 + running
```

---

## 6. 状态机

| 状态 | 会员端 |
|------|--------|
| pending / running / paused / soft_stopped | 待开启 / 运行中 / 已暂停 / 已封停 |

**→ paused**：手动；断期停投+维护；总/单方案止损止盈；时间窗外。  
**→ soft_stopped**：**仅** 运营强停。  
**恢复 running**（§6.4）：时间窗内 + 未触总/单方案止损止盈；进窗 **手动**；解封→paused 后仍须手动恢复（D73）。  
**彩种维护**：断期停投关 → 补开奖 + §6.4 满足则自动续投；开 → 手动恢复。

**事件**：`INSTANCE_START` | `INSTANCE_PAUSE` | `INSTANCE_FORCE_STOP` | `INSTANCE_RELEASE_STOP` | `LOOKBACK_RESET` | `LOTTERY_MAINTENANCE`

---

## 7. Admin

| Tab | 内容 | 操作 |
|-----|------|------|
| **用户** | 方案 + 实例 | 强停、解封 |
| **分享池** | **仅自创型** 快照 | PATCH 完整 config、DELETE |

KPI running = 用户 Tab 同源；real/sim 分列。

---

## 8. HTTP 接口（OpenAPI 已定义）

> 契约：`backend/openapi/openapi.yaml`（tag `client-schemes`）· 类型：`backend/contracts/schemes.ts`

### 8.1 Client

| 能力 | 路径 |
|------|------|
| 私池 CRUD | `/client/schemes` |
| 添加至云端 | `POST /client/schemes/{id}/add-to-cloud` |
| 复制并上云 | `POST /client/schemes/{id}/fork-and-add-to-cloud`（有实例；D77–D82） |
| 分享池→私池 | `POST /client/schemes/share/{id}/add-to-cloud` → 跟单 + pending |
| 跟单投注 | `POST /client/schemes/share/{id}/follow-bet` → running |
| 反买 | `POST /client/schemes/contrary/bet` → running |
| 启停 | `POST /client/cloud/instances/{id}/start\|pause\|resume` |
| 分享池列表 | `GET /client/schemes/share-catalog`（**仅自创快照**） |
| 全局规则 | `GET/PUT /client/cloud/global-settings` |

**错误码**：`SCHEME_NAME_DUPLICATE` | `SCHEME_ADD_CLOUD_TOO_FAST` | `SCHEME_DELETE_WHILE_RUNNING` | `SCHEME_SHARE_NOT_ALLOWED` | `SCHEME_SNAPSHOT_KIND_IMMUTABLE`

### 8.2 Admin

| 能力 | 路径 |
|------|------|
| 用户方案 | `GET /admin/schemes/instances?scope=user` |
| 分享池 | `GET /admin/schemes/instances?scope=share` |
| 改/删快照 | `PATCH/DELETE /admin/schemes/share/{id}`（config only；**kind 恒为自创**） |
| 强停/解封 | `POST .../force-stop` | `POST .../release-stop` |

### 8.3 数据表

| 表 | 约束 |
|----|------|
| `scheme_definitions` | `kind`∈{自创,反买,跟单}；`UNIQUE(member_id, scheme_name)` |
| `scheme_instances` | `UNIQUE(definition_id)`；ON DELETE CASCADE |
| `scheme_share_snapshots` | **`kind` 固定自创**；无 member FK |
| `member_cloud_settings` | 每会员一行 |
| `lottery_scheme_option_sets` | 种子只读 |

---

## 9. Mock 改版（已对齐）

| 项 | 状态 |
|----|------|
| `AdvancedSchemeEditView` | ✅ 自创才显示分享；添加/fork；1s 防连点；running 禁止删 |
| `SchemeMonitorView` | ✅ 用户 + 分享池 Tab；分享池仅自创 |
| kind / status | ✅ 自创·反买·跟单；待开启·运行中·已暂停·已封停 |
| Admin 改会员方案 | ✅ 已移除（仅分享池快照可编辑） |

Mock 层：`client/src/mock/schemeDefinitionsMock.ts` · `admin/src/mock/schemeShareSnapshotsSeed.ts`

---

## 10. 定案索引（D75–D82）

| # | 定案 |
|---|------|
| D75 | 删非 running 级联删实例；可同名重建 |
| D76 | 多路 = 新方案 + 新实例 |
| D77 | 有实例 → 自动复制新方案 + pending |
| D78 | 仅自创可选分享；fork 默认私密 |
| D79 | running 禁止删方案 |
| D80 | 分享池快照 **type 恒为自创**；**不可能** 有跟单/反买快照（**Y1**） |
| D81 | 复制命名 `-2`、`-3`… 顺延（**Y2**） |
| D82 | fork **整包复制** 配置（**Y3**） |

---

## 11. 结案：Y1–Y3

| # | 结论 |
|---|------|
| Y1 | → **D80**（不可能有跟单/反买快照） |
| Y2 | → **D81** |
| Y3 | → **D82** |

---

## 12. 方案域闭合检查

| 模块 | 状态 |
|------|------|
| 方案 vs 实例 | ✅ |
| 三类型与入口 | ✅ |
| 1:1 / fork / 删除 | ✅ |
| 分享池（仅自创快照） | ✅ |
| 状态机 / 恢复 / 维护 | ✅ |
| Admin 双 Tab | ✅ |
| HTTP / 表结构草案 | ✅ OpenAPI 已落稿 |
| 前端 Mock | ✅ §9 |

**无未决产品问题。** 研发可按 §8 开工；与 `admin-frontend-plan.md` §25/§30 冲突 **以本文件为准**。

---

## 13. 文档维护

变更顺序：本文件 → `openapi.yaml` → `contracts/schemes.ts`。

---

## 14. 变更记录

| 日期 | 说明 |
|------|------|
| 2026-05-24 | 初稿～X 结案 |
| 2026-05-24 | OpenAPI + `contracts/schemes.ts` 落稿 |
| 2026-05-24 | 前端 Mock 对齐 §9（Client 方案 Mock + Admin 双 Tab） |

---

## 15. 后台 Scheme Worker（`internal/schemes/worker*.go`）

> 环境变量：`SCHEME_WORKER_ENABLED`、`SCHEME_WORKER_TICK_SEC`（见 `backend/.env.example`）

### 15.1 周期

1. 扫描 `status=running` 实例，倒计时归零后取下一期 `lottery_draws`（seed 用完后确定性合成）
2. 读方案 `config` → 解析 `playTypeId` / `subPlayId` / `schemeGroups[roundIndex]`
3. 按玩法引擎判定 hit/miss，写 `cloud_bet_records`，更新实例倍投轮次

### 15.2 玩法段（5 位：万千百十个）

| `playTypeId` | 取号段 |
|--------------|--------|
| `hou4` | 千佰十个（1–4） |
| `qian3` | 万千百（0–2） |
| `zhong3` | 千百十（1–3） |
| `dingwei` | 单胆 |

| `subPlayId` | 规则摘要 |
|-------------|----------|
| `zhixuan_fs` | 直选复式：多行=每位一池；单行=同池笛卡尔积 |
| `zhixuan_ds` | 直选单式：N 位 token 精确匹配 |
| `zuxuan_fs` | 组选复式：数字池或排序 token |

### 15.3 资金（仅 `run_mode=real`）

同一事务：`bet_debit` →（命中）`payout` → `bet_orders`（已结算）→ `cloud_bet_records`。余额不足：实例 **paused**，审计 `余额不足暂停 …`。

### 15.4 回头（lookback）

- **个别**：单实例 `lookback_pnl` 达阈值 → **仅**复位倍投轮次 `round_index→0` 并清 `lookback_pnl`（不暂停）
- **整体**：`member_lookback_runtime` 累计达阈值 → 同会员同通道全部 running 实例同上复位
- **不出号**：不清理 `pick_index` / `current_pick` / `last_direction`（与富联「回到倍投起点」一致；定码轮换/高级定码跳局不受回头打断）
- 复位事件写入 `admin_audit_logs`（actor=`scheme-worker`）

### 15.5 会员手动投注（Client 真实选号 + 结算 Worker）

**下单**：`POST /client/games/{code}/bets` 须传 `betPayload.groupContent`（Client 选号 UI 生成）；`NormalizeBetPayload` 校验玩法段后写入 `bet_orders.bet_payload`。旧 pending 无 payload 时 `EnsureBetPayload` 仍 fallback 确定性选号。

**结算**（`orders/bets/settlement_worker.go`）：与 Scheme Worker 共用 tick 与 `lottery_draws`（缺失期号时确定性合成）。扫描 `bet_orders.status=pending` → `EvaluateBetPayload` + 同期开奖 → 命中写 `payout` 帐变 → `win`/`lose`。
