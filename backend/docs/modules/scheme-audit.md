# 合法投注空间与只读对账

## 为什么要有这一层

号池、选项宇宙、注数这三件事此前分散在四处各推一份：`guajibet`（豹子零注、直选复式上限）、
`attributeUniverse`（和值/跨度宇宙）、`ruleNumberPool`（号池）、前端 `playConfig.ts`。
任何一处漏掉就是一个静默 bug——**能正常下注、能正常结算，只是选出来的号在业务上是错的**，
不报错、不告警，只能靠断言体检翻出来。

极速彩 `game_id` 对调那个 bug 就是这个形状，潜伏了很久。

## 底座：`internal/schemes/play_universe.go`

单一权威定义，纯函数、不碰 DB。保存校验、下注前校验、对账命令、单测都应查这里。

| 入口 | 作用 |
|---|---|
| `UniverseForScheme(kind, config)` | 该玩法的合法投注空间：内容形态、位名、号池、全选注数 |
| `ValidateSchemeBetContent(kind, config, content, maxUnits)` | 号池闭包 + 注数上下限，返回违规清单 |
| `CountBetUnitsForScheme(kind, config, content)` | 按第三方 `bets_nums` 口径推算注数 |
| `MaxBetUnitsForScheme(kind, config)` | 该玩法单组注数上限 |
| `UnreachableHotColdOptions(kind, config)` | 冷热候选宇宙里理论上永远开不出的选项 |
| `HotColdRouting(kind, config)` | 冷热计频实际走的分支 vs 应走的分支 |

### 内容形态（`UniverseKind`）

- `perPosition` 按位分行：定位胆、直选复式
- `tokenList` 单行号池：组选复式、不定位、包胆、任选
- `combos` 单式：逗号分隔的定长组合
- `attribute` 属性选项：大小单双、龙虎、和值、跨度、尾数
- `""` **形态未知**，一律不做断言

### 可达值域用穷举，不用笛卡尔边界

`attributeUniverse` 用「单号池上下界 × 位数」推和值宇宙，对同期号码不能重复的彩种是错的：

- PK10 冠亚和实际只有 3..19，笛卡尔给出 2..20，多出的 **2 / 20 永远开不出**
- 组选和值实际 1..26，0 / 27 只有豹子能组成，而组选不可下豹子
- PK10 冠亚跨度最小为 1（两名次必不相同），笛卡尔给出 0

这些选项频次恒为 0，在「最热→最冷」排序上永远垫底——**方案一取冷号就稳定押中它们**。
`reachableAttributeUniverse` 改为穷举该位段所有合法开奖组合再收集取值，
按 `templateDrawsDistinctBalls`（PK10 / 11 选 5 / 六合彩号码不重复）与
`ruleExcludesAllSame`（组选排除全同号）施加约束。组合数超过 200 万时放弃穷举并在 `Note` 标注。

### 已知缺口

- **时时彩五星特殊号形态未知**：`betMode=teshu` 下，「豹子/顺子」选文字、「一帆风顺」选一个 0-9 数字，
  只看 `betMode` 分不出来，必须查子玩法表。接上之前一律判未知、不做断言，
  在对账报告的「未覆盖」里可见
- **注数上限只有直选复式有权威定义**（`zhixuanFushiMaxBetUnits`）。
  其余玩法 `MaxBetUnitsForScheme` 返回 0，这不是「无限制」而是「本端还没定义」

### 防分叉

`isFushiBaoziContent` 与下注链路的 `guajibet.IsFushiBaoziZeroBet` 是同一个判断的两处实现。
`TestFushiBaoziMatchesGuajibet` 用同一组输入交叉断言两者结论一致——
两处各判一遍正是「我的投注位名错位」那次的病根。

## 只读对账：`cmd/scheme-audit`

不写库、不调第三方。

```
make scheme-audit                             # 近 30 天，报告写到 backend/reports/
make scheme-audit-ci                          # 近 2 天，有 P0/P1 则 exit 1
go run ./cmd/scheme-audit -days 7 -limit 5000
```

输出一份 JSONL 明细和一份 Markdown 汇总。

### 检查项

**注单级**

| 检查 | 级别 | 断言 |
|---|---|---|
| `fund_conservation` | P0 | pnl 与 status 自洽；返奖 = 本金 + 盈亏 |
| `adjudication_mismatch` | P0 | 用开奖号本地重算，与落库 status 一致 |
| `draw_missing` | P1 | 注单期号在 `lottery_draws` 查得到（期号归属） |
| `content_*` | P1 | 投注内容落在合法投注空间内（号池闭包 / 注数 / 零注） |
| `payout_source` | P1/P2 | 模拟盘中奖必有返奖；正式盘缺失说明走了本地兜底 |
| `status_stale` | P2 | 开奖号已入库则不应长期 pending |
| `bet_units_mismatch` | P2 | 落库注数与按内容重算一致 |
| `display_fields` | P3 | 详情页字段不缺 |

**方案级**

| 检查 | 级别 | 断言 |
|---|---|---|
| `pool_range` | P1 | 号池与彩种模板相符（防 `ruleNumberPool` 默认 0-9 静默降级） |
| `config_*` | P1 | 保存的投注内容合法 |
| `hotcold_unreachable` | P2 | 冷热候选宇宙无恒 0 频次选项 |
| `hotcold_routing` | P2 | 属性玩法走属性计频，按位玩法走按位计频 |

**彩种级**

| 检查 | 级别 | 断言 |
|---|---|---|
| `period_family_mismatch` | P0 | 某彩种注单期号零命中开奖库 → 下注链路与开奖链路期号族不一致 |
| `period_family_partial` | P1 | 命中率 < 50%，疑似部分错配或开奖同步缺口 |

彩种级断言**只看近窗口**（`-recent`，默认 24h）。整窗命中率会被迁移前的历史坏数据长期压低，
让「昨天刚修好」和「现在还坏着」长得一模一样——极速彩修复当天就踩过这个坑，
30 天口径显示 7/706，按小时看其实修复后是 100%。整窗数字只作为背景写进 detail。

修完映射想立刻验证就把窗口调小：`go run ./cmd/scheme-audit -recent 2h`。

### 读报告的两条纪律

**「近窗口」列区分历史遗留与当下仍在发生。** 判定逻辑改过之后，历史注单必然与今天的重算结果
不一致——那是漂移不是活 bug。只有近窗口内仍在增长的项才需要立刻处理。

**「未覆盖」不等于「无问题」。** 因缺方案定义、缺开奖号、形态未知而跳过的检查都在那里列出，
带次数。判不准就不报——审计工具的误报会直接摧毁可信度，宁可漏报。

### 后加列的处理

`bet_units` / `payout_amount` 是迁移 00135 才加的列，之前的行天然为空。
命令启动时用「首条非空记录的投注时间」自动探测列启用时刻，更早的行不参与这两项检查，
跳过次数计入「未覆盖」。

## 保存时拦截：`ValidateSchemeConfig`

对账是事后体检，拦截才能止血。`UpdateDefinition` 在写库前调 `ValidateSchemeConfig`，
与对账的 `config_*` 检查**同一入口**——保证「保存时拦下的」和「对账时报出的」是同一套判据，
否则两边各自漂移，最后谁也不信谁。命中时返回 `ErrInvalidSchemeContent`，
handler 原样透出越界原因（只说「内容无效」用户无从下手改）。

两条克制：

- **只在 patch 真动了投注内容时校验**（`patchTouchesBetContent`）。否则改个方案名就会被
  历史遗留的非法配置挡住，用户既看不懂也修不了。
- **判不准就不拦。** 玩法内容形态未知（如时时彩五星特殊号）时直接放行。

开这道闸之前先用对账清了两类校验器自身的误报，否则会大面积误伤正常方案：

- **任选的位名前缀**：内容形如 `千,十|12,34`，前半段选的是位不是号。
  `stripPositionLabelPrefix` 复用下注链路的 `parseRenxuanPosPicksContent` 剥掉。
- **数值属性的补零写法**：和值号池是 `0..27`，内容常写成 `04`、`07`，数值相同但字符串不等。
  `canonAttrToken` 按数值归一后再比。

## 期号归属自检：`internal/guaji/periodaudit`

对账要攒够注单才看得出来，自检不用等。持续比对 `lottery.PeriodsScheduleFor`（下注链路，
按 `outbound_lottery_code` 取的当期）与 `lottery_draws` 最新一期（开奖链路，按 `guaji_ws_key`），
不同族就 `slog.Error`，同一彩种一小时内只报一次，恢复时记 `recovered`。

**是周期性而非只在启动跑一次**：进程刚起来时 periods 缓存是空的，那一刻什么都比不出来。

判据只有位数（`lottery.ComparePeriodFamily`）。试过用数值相对差补充「同体系内错配」，
但长期号里有意义的位埋得太深：日期族差半年，相对差也只有 5.7e-8，跟相邻期没法区分。
定不出有依据的阈值就不定，假装精确的规则只会制造误报。

**已知盲区**：两个期号体系相同的彩种互相错配（如哈希一分彩 ↔ 哈希五分彩），位数与数值都接近，
本函数判不出。那种情况只能靠 `audit-ws-keys` 比对 WS 键，或靠本命令统计注单期号命中率发现。
三者互补，缺一不可。
