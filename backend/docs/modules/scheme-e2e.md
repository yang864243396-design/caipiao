# 方案全链路 E2E

`internal/schemes/e2e_lifecycle_test.go`

## 为什么要有

创建方案、出号、下注、结算、查询每一环都有组件单测（`internal/schemes` 下 60 个测试文件、
258 个测试函数），但**没有一条测试把它们串起来跑过**。「能建、能跑、能下注、查得到」
这件事一直只靠人工点 `docs/integration-checklist.md`。

组件测试抓不到接缝上的问题：配置存进去和 worker 读出来的形状对不上、
实例状态机卡在某一档不往下走、结算写回的字段查询侧读不到——这些每个组件自己都是对的。

## 覆盖范围

一次跑完 7 种运行类型，每种都走完整链路：

创建方案 → 写投注内容 → 加入云端 → 启动 → worker 跳过首期 → 激活 →
出号下注 → 落 `cloud_bet_records` → 结算 → 经真实 SQL 查询详情

| 运行类型 | 判定是否钉死 |
|---|---|
| `fixed_number` 固定号码 | 是（必中） |
| `fixed_rotate` 定码轮换 | 是（必不中） |
| `adv_fixed_rotate` 高级定码轮换 | 是（必中） |
| `adv_trigger_bet` 高级开某投某 | 是（必中） |
| `random_draw` 随机出号 | 否，出号随机 |
| `hot_cold_warm` 冷热出号 | 否，出号依赖历史频次 |
| `builtin_plan` 内置计划 | 是（必中），单独一个用例 |

出号随机的两种只查通用不变量：内容非空、金额为正、内容落在合法投注空间内、
未中亏满本金 / 中奖盈亏为正。

`builtin_plan` 单独成一个用例，因为它多两步前置——先公开分享出一个方案拿到快照，
再收藏该快照，才能物化。这两步是「收藏方案 → 物化 → 运行」里此前没有任何测试碰过的一段。

每笔注单的投注内容都会过一遍 `ValidateSchemeBetContent`，
所以出号越出合法号池会当场失败，而不是等到事后对账。

## 怎么跑

```bash
make scheme-e2e
```

需要 `DATABASE_URL`，没有则整体跳过。

**建议先停掉本地后端。** 它的 scheme worker 会并发推同一批实例。
断言写的是「终态如此」而不是「由我这一 tick 造成」，所以并发不会误判，只是会变慢。

## 怎么做到不依赖第三方和真实时间

三个关键点：

**全程 `sim_bet = true`。** `usesGuajiThirdParty` 对模拟盘直接返回 false，
下注不出网，授权检查也整段跳过。

**期号靠注入而不是拉取。** `lottery.UpdatePeriodsScheduleFullWithDuration` 直接写 periods 缓存，
把待跳过的首期造成「已封盘」、待下注的期造成「开盘中」。
两次 tick 就能走完「跳过首期 → 激活 → 下注」，不用等真实时间流逝。

**只推自己那一个实例。** 用 `TickInstanceForTest`（`export_test.go`）而不是 `w.tick`——
后者会捞出全库所有 running 实例挨个推一遍，在开发库上等于替别人的方案下注。

## 自清理

方案定义、实例、注单、分享快照、收藏、测试造的开奖行，全部在 `t.Cleanup` 里删掉。

期号用 `99` + 时间戳前缀，不会和真实期号撞车。

**当日模拟启动配额会被真实消耗**（上限 5 次/天），所以测试在启动前把计数清零，
跑完再还原成进入测试时的原值。不这么做的话跑一轮就把该会员的配额吃光了。

## 已知没覆盖的

- **正式盘下注**。要真连第三方，不适合放进自动化测试。目前只有模拟盘链路。
- **玩法维度只有一星定位胆**。这里测的是链路而不是玩法，玩法维度由
  `ssc_coverage_test.go` / `fast_ssc_coverage_test.go` 等覆盖测试负责。

## 周边覆盖

E2E 之外补齐的几处，以及补的过程中查出来的问题：

| 测试 | 覆盖 |
|---|---|
| `internal/schemes/fast_ssc_coverage_test.go` | `fast_ssc_std` 全部 132 个子玩法 |
| `internal/schemes/create_scheme_test.go` | `CreateDefinition` 入参校验与建方案的各条 DB 分支 |
| `internal/cloud/betrecords/service_db_test.go` | 分组 / 明细 / 单笔详情跑真实 SQL |
| `internal/handler/bet_records_test.go` | 三个投注记录接口的鉴权、参数校验、错误码映射 |

### fast_ssc_std 的编号体系与 betMode 来源

种子文件 `docs/seeds/sub_plays.csv` 用的是语义 id（`dingwei/dingwei_ge`），
而库里和生产方案配置用的是第三方数字 id（`g006/13`）——两套编号没有交集。
`ssc_coverage_test.go` 那 175 条遍历跑的全是语义 id，覆盖不到线上真正在用的那套。

`sub_plays.bet_mode` 在库里是 NULL，`resolveSSCPlayRule` 只是把调用方给的 betMode 原样收下，
不做推导。所以 `fast_ssc_coverage_test.go` 的 betMode oracle 取自 `scheme_definitions`
里真实存在过的配置（157 个 (typeId, subId) 对，每对恰好一个 betMode）。
不给 betMode 的话，132 个子玩法里有 105 个会退化成按位选号，遍历看着全绿其实什么都没测到。

`resolveSSCPlayRule` 把 `PlayTemplate` 硬编码成了 `"ssc_std"`，
两个模板事实上共用同一段代码；`TestFastSSCMatchesSSCForSharedSubPlays` 把这个前提钉住了。

### 查出来的三个缺口

- **LHC 验奖不查号池。** 喂完全非法的内容（`"99,100,零"`），82 个六合彩子玩法照样
  全部返回正注数——`evaluateLHCByBetMode` 只数 token。号池闭包目前只靠
  `ValidateSchemeBetContent` 那一层兜住，由 `TestLHCContentPoolClosure` 盯着。
- **`mode` 取值不校验。** `GroupsFilter.Validate()` 不看 mode，而分盘是拿
  `mode == "sim"` 判的，所以 `?mode=typo` 会静悄悄按正式盘返回数据，
  还把 `"typo"` 原样回显在响应里。见 `TestBetRecordGroupsAcceptsUnknownMode`。
- **中奖率把待开奖的算进了分母。** `summarize` 用的是 `hits / len(rows)`，
  方案刚跑起来时这个数会被稀释得偏低。按现状钉在
  `TestGroupsAggregatesFromSQL` 里，改口径需要是一次有意识的产品决定。

### 播测试数据的几个坑

- `cloud_bet_records.record_no` 是 `varchar(32)`，构造 id 时要留出后缀位置。
- 分组和明细查询都 join 了 `scheme_instances`，只播注单一条都查不出来；
  实例又有外键指向 `scheme_definitions`，两张表都得播。
- `scheme_instances.status` 只允许 `pending/running/paused/soft_stopped`。
  测试数据用 `paused`——worker 只捞 `running`，不会误推。
- **正式盘分组还要求注单绑定了挂机账号并回填了第三方注单号**
  （藏在 `ListCloudBetRecordsFiltered` 的 `guaji_account_id` 条件里）。
  所以播数据测查询时用模拟盘，正式盘那条规则由
  `TestGroupsSeparatesSimAndFormal` 单独钉。
- `apix.Validation` 是「HTTP 200 + 信封码 42200」，不是 HTTP 400。
  handler 的校验类断言必须看信封码，只看 `rec.Code` 会永远是绿的。
