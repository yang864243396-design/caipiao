# V8 挂机软件逆向总结（方案/运行类型/出号算法）

> 目的：固化对第三方 **V8 挂机软件**（`G:\V8挂机软件`）的逆向结论，供后续开发直接引用，**无需重复反编译**。
> 结论已落地到本平台代码（见文末「代码落点」）。若与代码不一致，以代码为准并回补本文。

---

## 0. 逆向对象与产物

| 项 | 说明 |
|---|---|
| 软件根目录 | `G:\V8挂机软件`（.NET/DevExpress 桌面挂机客户端） |
| 配置文件 | `G:\V8挂机软件\Configuration.txt`（**加密**，非明文，需软件自身解密逻辑） |
| 核心程序集 | `plan.dll`、`V8Main.exe`（含出号/计划逻辑） |
| 关键命名空间/类型 | `IntelligentPlanning.CommFunc`（通用函数）、`FNGDQM*`（固定取码算法） |
| 反编译方式 | 反射枚举类型 + IL 反汇编（`ildasm`/自写反射脚本）。中文路径在 PowerShell 用 `Get-ChildItem -LiteralPath` 处理 |

> 注：反编译中间产物（如临时 dump 文件 `v8_runtypes.txt` / `v8_il_gdqm.txt` / `v8_types_out.txt`）为**一次性临时文件**，可能已清理；本文即其提炼结论。

---

## 1. 运行类型体系（最重要结论）

**V8 的运行类型与玩法类型「正交、无门禁」——任意运行类型可配任意玩法类型（全玩法支持）。** 出号引擎按玩法自适应产号，不做运行类型×玩法的白名单限制。

本平台支持并对齐的 **7 个运行类型**（1:1 映射 V8）：

| # | 运行类型 (`runTypeId`) | 中文名 | 语义 |
|---|---|---|---|
| 1 | `fixed_rotate` | 定码轮换 | 多组固定号码，逐期轮换（本平台：每期换组，`pick_index=(i+1)%n`） |
| 2 | `adv_fixed_rotate` | 高级定码轮换 | 局数列表，按「中后跳转局 / 挂后跳转局」跳转 |
| 3 | `adv_trigger_bet` | 高级开某投某 | 上期开奖某位数字触发档 → 投该档正投/反投内容 |
| 4 | `hot_cold_warm` | 冷热温出号 | 按最近 N 期频次分热/温/冷档选号 |
| 5 | `random_draw` | 随机出号 | 选项宇宙 + 抽样产号 |
| 6 | `builtin_plan` | 内置计画 | V8 抓外部计划窗口；本平台改为「复制收藏方案物化」 |
| 7 | `fixed_number` | **固定取码**（原名「固定号码」） | 条件规则命中上期开奖→投固定号；无规则回退静态复投 |

**本平台已去除 V8 没有独立化的「自定义开某投某」**（V8 将开某投某合并为一种，无「自定义」子类）。旧的 `custom_trigger_bet` 运行类型已从前后端全部移除。

---

## 2. 各运行类型语义与出号算法

### 2.1 固定取码 `fixed_number`（V8 GDQM 动态机制）

V8 的固定取码**不是**静态号码，而是**条件规则**（`FNGDQM` 系列方法）：

- 每条规则：`PosStart / PosEnd`（上期开奖位区间，0-based ball 下标）、`CodeMin / CodeMax`（号码值区间，含端点）、`Numbers`（命中后投注的固定号码）。
- 判定：上期开奖在 `[PosStart, PosEnd]` 位区间内**任一位**号码落在 `[CodeMin, CodeMax]` → 命中（甲：命中→投），投该条 `Numbers`。
- **按序匹配、首条命中即投**；无上期开奖 / 无命中 → 本期不投。
- 无规则时回退静态固定号（每期原样复投），兼容存量。

### 2.2 随机出号 `random_draw`（GetCombinaList 宇宙 + 抽样）

V8 用 `GetCombinaList` 构造该玩法的**合法组合/选项宇宙**，再从中**随机抽样**。本平台对齐为三类：

- **单式/组选单式（整注随机）**：随机抽 N 个完整组合（每位随机取号拼注，去重；混合组选单式排除豹子）。
- **组合家族（组三/组六/组选N/组选复式）**：随机选 K 个号组成号码池，按组选口径展开。
- **属性/聚合家族（大小单双/龙虎/庄闲/特殊号/和值/跨度/不定位/包胆）**：从该玩法选项宇宙随机抽 K 个。

策略 `strategy`：`every` 每期换 / `keep` 不换号 / `after_hit` 中后换 / `after_miss` 挂后换。

> 注：早期另一竞品（富联）用 MD5(seed) 的**确定性伪随机**；V8 是 `GetCombinaList` 宇宙 + 抽样，二者不同。

### 2.3 冷热温出号 `hot_cold_warm`

按最近 N 期**频次降序三等分**为热/温/冷。

- **按位型**（直选复式/组合/定位胆/任选直选复式）：每位一档，按位频次。
- **号码池型**（组三/组六/组选N/组选复式/不定位/包胆）：单档「号码整体频次」。
- **属性家族**（大小单双/龙虎/庄闲/特殊号/和值/跨度）：单档「选项命中频次」——每期对宇宙内每个选项判定命中即计数（大小单双一期可同时命中一个大小档+一个单双档；龙虎命中龙/虎/和之一；和值/跨度命中唯一值）。
- `winRotate` 中奖轮换：命中后池内号码 +1 循环。

### 2.4 高级开某投某 `adv_trigger_bet`

- **触发**：上期开奖某位开出的**数字**（0-9 档）；龙虎/PC28 特殊按其结果。
- **投注**：命中档对应的正投/反投内容（内容为**投注玩法**格式，可为任意玩法）。
- 触发是按位数字，投注内容由玩法评估——因此天然支持全玩法。
- 投向状态机 `mode`：一直正投 / 一直反投 / 前正后反 / 前反后正（跨期翻转）。
- 无匹配走启用第 1 行（Q4c）。

### 2.5 定码轮换 / 高级定码轮换 / 内置计画

- `fixed_rotate`：多组号码，本平台每期换组。（注：V8 文档另有「挂 N 期换组」变体，本平台默认每期换。）
- `adv_fixed_rotate`：`jushuList: [{ju, content, afterHit, afterMiss}]`，按中/挂结果跳转局。
- `builtin_plan`：V8 抓外部计划窗口的 `计划号` 文本；本平台因云端架构改为「收藏方案快照物化复制」（`builtinPlan.snapshotId` + 物化配置）。

---

## 3. 反集 / 反买（contrary）

- **不是所有玩法都支持反集**：反集是「取补集」语义，仅对**有对立面**的玩法成立（定位胆/号池取补、龙虎对立、大小单双等属性对立）；和值/跨度/形态等无补集不适用。
- 本平台实现：**支持反集的玩法默认开启、不使用默认关闭**；作为独立 `kind=contrary` 方案（`planInverseNumbers` 已是反集内容，按原玩法直接结算）。

---

## 4. 投注/倍投/结算（与 V8 对齐）

| 项 | 结论 |
|---|---|
| 注额 | `注单位(默认2元) × 注数 × 当期倍数` |
| 倍投 | `rounds[]`（4 Tab 统一编译为 rounds）；`round_index` 独立于出号游标 → **双游标并行** |
| 出号 vs 倍投 | `round_index`（倍投）与 `pick_index / current_pick / last_direction`（出号）**相互独立推进** |
| 止盈止损/回看 | 回看(lookback) 止盈止损；触发时倍投+出号一起复位 |
| 正式盘推进 | **派奖后**统一推进游标（`ProcessFormalAfterSettlement` → `AdvancePickAfterFormalSettlement`）；下单时冻结、避免每期重生成 |

---

## 5. 与本平台对齐结论（已落地）

1. **全玩法矩阵放开**：`ValidateRunTypePlay` 不再限制运行类型×玩法（前端 `runTypeMatrix.ts` 同步放开）。
2. **移除自定义开某投某**：前后端、迁移枚举全部清除。
3. **固定号码 → 固定取码**：改名 + 动态规则机制（有规则走规则，无规则回退静态）。
4. **冷热温属性家族**：后端分档接口 `POST /client/schemes/hot-cold-warm/tiers` 复用 `evaluatePlayHit` 按选项命中频次分档。
5. **随机出号全家族**：单式/组选/属性宇宙抽样。
6. **方案内容录入改输入框**：数字玩法单输入框，逗号分隔各位、每位连写（如 `123,34,56,78,56`），系统按号池 token 宽度拆分匹配。

---

## 6. 代码落点（authoritative）

| 关注点 | 文件 |
|---|---|
| 运行类型常量/全玩法放开 | `backend/internal/schemes/run_types.go`（`ValidateRunTypePlay` 全放行） |
| 出号分发/各类型引擎 | `backend/internal/schemes/worker_pick.go`（`resolvePick` / `pickFixedPick` / `decideFixedPick` / `pickRandomDraw` / `pickHotColdWarm` / `pickTriggerBet`） |
| 配置解析 | `backend/internal/schemes/worker_config.go`（`fixedPickRule` / `resolveFixedPick` / `resolveHotColdWarm` / `resolveTriggerBet`） |
| 属性选项宇宙 | `backend/internal/schemes/worker_pick.go`（`attributeUniverse` / `randomAttributeContent`） |
| 冷热温属性分档接口 | `backend/internal/schemes/hot_cold_warm_tiers.go` + `hot_cold_warm_tiers_service.go` + `handler/schemes.go`（`HotColdWarmTiers`） |
| 正式盘游标推进 | `backend/internal/schemes/worker_pick.go`（`AdvancePickAfterFormalSettlement`）+ `cloud/schemestate/settle.go` + `schemestate_wire.go`（DI 破循环依赖） |
| 玩法命中评估 | `backend/internal/schemes/worker_play.go`（`evaluatePlayHit`）、`ssc_special_eval.go`（龙虎 `longhuPositions` 用 `CatalogSubID`） |
| 迁移 | `backend/migrations/00129_run_type_align_v8.sql`（移除 custom 枚举、固定号码→固定取码） |
| 前端矩阵/录入/固定取码面板 | `client/src/utils/runTypeMatrix.ts`、`client/src/utils/pickPanelOptions.ts`、`client/src/components/schemes/SchemeGroupInputPanel.vue`、`client/src/views/play/AdvancedSchemeEditView.vue` |

---

## 7. 复用提示（给后续对话）

- 需要 V8 运行类型/出号/固定取码算法结论时，**直接引用本文**，不必重新反编译 `G:\V8挂机软件`。
- 若需更深入的 IL 细节（如 `FNGDQM` 具体分支、`Configuration.txt` 解密），才需重新反编译：核心命名空间 `IntelligentPlanning.CommFunc`、方法名前缀 `FNGDQM`、`GetCombinaList`。
- 本平台以「无门禁全玩法 + 出号引擎按玩法自适应」为总原则，新增玩法默认即被各运行类型支持。
