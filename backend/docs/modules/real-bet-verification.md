# 正式盘真实下单验证

`cmd/real-bet-matrix`（下单矩阵）、`cmd/guaji-smoke`（连通性）、`cmd/bet-probe`（单条规则实测）

## 为什么要有

投注内容的线格式（wire）、`solo` 标志、注数口径全部由第三方规则决定，
本地没有任何权威可比对：单测只能锁住"我们以为的格式"。
这些字段错了不会报错，只会被第三方拒单，或者更糟——按错误注数扣款。

所以正式盘的判定只有一个口径：**用真实账号下真实的单，看第三方收不收**。
2026-07-28 一轮全量下单发现 8 类缺陷，全部是单测覆盖不到的类型（见下）。

## 覆盖是按模板算的，不是按彩种

投注内容/注数/solo 的逻辑全部挂在 `play_template` 上，与具体彩种无关。
33 个在售彩种只有 6 个模板，因此**每个模板跑通一个彩种即可覆盖规则逻辑**：

| 模板 | 子玩法 | 已验证彩种 | 结果 |
|---|---|---|---|
| `ssc_std` 时时彩 | 171 | `tron_ffc_1m` | 170/171 |
| `lhc_std` 六合彩 | 115 | `tron_lhc_1m` | 115/115 |
| `pk10_std` 赛车 | 32 | `tron_pk10_jisu` | 32/32 |
| `syxw_std` 11选5 | 26 | `tron_syxw` | 26/26 |
| `k3_std` 快三 | 9 | `tron_k3_1m` | 9/9 |
| `fast_ssc_std` 秒彩 | 132 | `tron_ffc_15s` | 131/132 |

按彩种独立的只有三件事，与玩法规则无关：

- `outbound_lottery_code`（第三方 game_id）——错了会下到别的彩种
- `guaji_ws_key`（开奖线键）——错了永远查不到开奖号
- 上游是否在售

这三件事每个彩种一注就能验证，不需要重跑整模板。

## 怎么跑

```bash
# 单模板全量（会真实扣款：条数 × 2 元）
go run ./cmd/real-bet-matrix -lottery tron_lhc_1m -out data/real-bet-lhc.jsonl -truncate

# 单条规则复验
go run ./cmd/real-bet-matrix -lottery tron_ffc_1m -type g011 -sub 144 -out data/verify.jsonl -truncate

# 只统计矩阵不下单
go run ./cmd/real-bet-matrix -lottery tron_ffc_15s -dry-run
```

`-verify` 默认开启：下单成功后与第三方 `web_bets` 列表对账，确认注单真的落在对方库里。

## 2026-07-28 修复的缺陷

八类，全部由真实下单暴露，单测都测不出来：

1. **`solo` 参数双向错**——该错误码同时表示"该传 true 却传了 false"和反向，
   只能用 `bet-probe` 逐条实测。涉及 kuadu / zuxuan_ds / zuhe / hunhe / danshi。
2. **`FormatBetContentForRule` 不幂等**——下注链路会格式化两次，2 位号码彩种
   （syxw/pk10）被切坏：`010203` → `000100`。
3. **PK10 冠亚和值大小单双走错分支**——被当成 `hezhi`，wire 出 `03` 而非 `和大`。
4. **六合彩最少选号数硬编码整表错位**——改为从玩法名解析（`5不中` → 5）。
5. **六合彩 `renzhong` 默认选 1 个号**——应按玩法名取 2/3/4/5。
6. **玩法名含替换字符导致模式推断失效**——`lhc_std g003/299`「三全中复式」的
   「式」在库里存成两个 U+FFFD，`Contains(label,"复式")` 恒假，取样退化成单个号码，
   该玩法在正式盘长期无法下单。见 `migrations/00137`，并在 `rulessync.BuildPlan`
   加了替换字符校验（整模板 DELETE+INSERT，报错会回滚，保留上一次的正确名称）。
7. **六合彩子玩法名判定取到组名前缀**——玩法名是「组名+子玩法名」，
   「五行家野家野」里「五行」先命中、「一肖尾数一肖」里「尾数」先命中。
   改为先用去掉组名前缀的子玩法名判定，取不到再退回整串。
8. **任选四组选12 把双区号池当选号列表**——`12,34`（二重号池,单号池）
   被号池补码补成 `12,34,3,4`。

## 期号不一致的告警不是缺陷

`real-bet-matrix` 会打 `guaji place bet period mismatch, trust upstream periods`。
第三方下单接口**不接受期号**，期号由对方分配；本地 `IssueNo` 只是预测值，
系统取上游返回值为准。该告警只说明在期号边界上下单，注单本身是对的。

## 已知上游缺口：秒彩的任选四直选复式（sub_id 141）

上游对该玩法回 `40000 该玩法对应的游戏没有配置`，`tron_ffc_15s` 与 `tron_ffc_6s`
都一样。但同一玩法在 `tron_ffc_1m`（`ssc_std`，game_id=19）实测下单成功，
所以**不是本端内容格式问题，也不是模板级缺失，而是上游没给秒彩这几个游戏配这个玩法**。

它随 rules/v2 同步进来（上游把它标成 `active`），本地改 `sub_plays.enabled`
会被下次同步的整表 DELETE+INSERT 覆盖，因此不做数据修补，按已知缺口记录：
秒彩下单矩阵里它是预期失败项，不要当成本端 bug 反复查。

## 已知待处理

**波场 3 秒彩（`tron_ffc_3s`）开奖未入库。** 2026-07-28 实测：
落后 19 小时，而同一条 `lottery_v2_broadcast` 消息里的 6 秒/15 秒彩正常。
用当前代码跑 `SubscribeDraws` + `drawsync.Worker.Ingest` 40 秒立刻补进 13 条，
说明当前代码是对的、在跑的进程用的是旧构建，需按当前代码重启。

重启前必须带上 WS 尾斜杠的修复（`wsPathOrDefault` 默认 `/ws/`）：
上游把 `/ws` 改成 301 → `/ws/`，而 WebSocket 握手不跟随重定向，
否则重启后 drawsync 会整体连不上，比只丢一个彩种严重得多。
