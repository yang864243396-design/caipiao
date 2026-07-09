# 全彩种全玩法真实下单矩阵测试报告

> 自动生成于 2026-06-30 00:20:19
> 全量批跑完成 + fail 补跑

## 1. 测试概览

| 指标 | 数值 |
|------|------|
| 矩阵总行数（在测） | 3941 / 4059 |
| 全量设计规模 | 4384 |
| 排除不对接彩种 | 325 行（4 彩种） |
| 成功 (ok) | 3941 |
| 跳过 (skip) | 0 |
| 失败 (fail) | 0 |
| 通过率 (ok/在测总数) | 100.00% |
| 测试账号 | vs8888 |
| 单注金额 (unit) | 2 元 |
| 第三方对账 ok | 3941 |
| 第三方对账 mismatch | 0 |
| 第三方对账 not_found | 0 |

## 2. 不对接彩种（矩阵排除）

以下彩种第三方 `game_id` 已确认**不再对接**，矩阵跑批与报告统计均排除：

| 彩种 code | game_id | 名称 | 说明 |
|-----------|---------|------|------|
| taiwan_ssc_5m | 69 | 台湾5分彩 | 第三方不提供 periods，且产品侧不再对接 |
| taiwan_pk10 | 70 | 台湾PK10 | 第三方不提供 periods，且产品侧不再对接 |
| taiwan_pc28 | 71 | 台湾28 | 第三方不提供 periods，且产品侧不再对接 |
| tron_lhc | 81 | 波场六合彩 | 第三方不提供 periods，且产品侧不再对接 |

## 3. 按彩种汇总

| 彩种 | 总数 | ok | skip | fail | 通过率 | 状态 |
|------|------|----|------|------|--------|------|
| bnb_ffc_1m | 169 | 169 | 0 | 0 | 100.0% | done |
| bnb_ffc_3m | 170 | 170 | 0 | 0 | 100.0% | done |
| bnb_ffc_5m | 170 | 170 | 0 | 0 | 100.0% | done |
| bnb_k3_1m | 9 | 9 | 0 | 0 | 100.0% | done |
| bnb_k3_3m | 9 | 9 | 0 | 0 | 100.0% | done |
| bnb_k3_5m | 9 | 9 | 0 | 0 | 100.0% | done |
| bnb_pk10_5m | 32 | 32 | 0 | 0 | 100.0% | done |
| bnb_pk10_jisu | 32 | 32 | 0 | 0 | 100.0% | done |
| bnb_syxw | 26 | 26 | 0 | 0 | 100.0% | done |
| bnb_syxw_3m | 26 | 26 | 0 | 0 | 100.0% | done |
| bnb_syxw_5m | 26 | 26 | 0 | 0 | 100.0% | done |
| eth_ffc_1m | 170 | 170 | 0 | 0 | 100.0% | done |
| eth_ffc_3m | 170 | 170 | 0 | 0 | 100.0% | done |
| eth_ffc_5m | 170 | 170 | 0 | 0 | 100.0% | done |
| eth_ffc_new | 170 | 170 | 0 | 0 | 100.0% | done |
| eth_jisu | 170 | 170 | 0 | 0 | 100.0% | done |
| eth_k3 | 9 | 9 | 0 | 0 | 100.0% | done |
| eth_k3_3m | 9 | 9 | 0 | 0 | 100.0% | done |
| eth_k3_5m | 9 | 9 | 0 | 0 | 100.0% | done |
| eth_pk10_5m | 32 | 32 | 0 | 0 | 100.0% | done |
| eth_pk10_jisu | 32 | 32 | 0 | 0 | 100.0% | done |
| eth_syxw | 26 | 26 | 0 | 0 | 100.0% | done |
| eth_syxw_3m | 26 | 26 | 0 | 0 | 100.0% | done |
| eth_syxw_5m | 26 | 26 | 0 | 0 | 100.0% | done |
| hash_ffc_1m | 170 | 170 | 0 | 0 | 100.0% | done |
| hash_ffc_3m | 170 | 170 | 0 | 0 | 100.0% | done |
| hash_ffc_5m | 170 | 170 | 0 | 0 | 100.0% | done |
| hash_jisu | 170 | 170 | 0 | 0 | 100.0% | done |
| tron_ffc_15s | 131 | 131 | 0 | 0 | 100.0% | done |
| tron_ffc_1m | 170 | 170 | 0 | 0 | 100.0% | done |
| tron_ffc_3m | 170 | 170 | 0 | 0 | 100.0% | done |
| tron_ffc_3s | 131 | 131 | 0 | 0 | 100.0% | done |
| tron_ffc_5m | 170 | 170 | 0 | 0 | 100.0% | done |
| tron_ffc_6s | 131 | 131 | 0 | 0 | 100.0% | done |
| tron_jisu | 170 | 170 | 0 | 0 | 100.0% | done |
| tron_k3_1m | 9 | 9 | 0 | 0 | 100.0% | done |
| tron_k3_3m | 9 | 9 | 0 | 0 | 100.0% | done |
| tron_k3_5m | 9 | 9 | 0 | 0 | 100.0% | done |
| tron_k3_jisu | 9 | 9 | 0 | 0 | 100.0% | done |
| tron_lhc_1m | 115 | 115 | 0 | 0 | 100.0% | done |
| tron_lhc_3m | 115 | 115 | 0 | 0 | 100.0% | done |
| tron_lhc_5m | 115 | 115 | 0 | 0 | 100.0% | done |
| tron_pk10_jisu | 32 | 32 | 0 | 0 | 100.0% | done |
| tron_syxw | 26 | 26 | 0 | 0 | 100.0% | done |
| tron_syxw_3m | 26 | 26 | 0 | 0 | 100.0% | done |
| tron_syxw_5m | 26 | 26 | 0 | 0 | 100.0% | done |

## 4. Skip 原因分布

无。

## 5. 失败原因分布

无。

## 6. 第三方投注记录对账

| 状态 | 次数 | 说明 |
|------|------|------|
| ok | 3941 | game_id / periods / bet_amount / rule_id 与 web_bets 一致 |
| mismatch | 0 | 字段不一致 |
| not_found | 0 | 第三方列表未找到注单 |
| skipped | 0 | 未启用对账或无 thirdPartyBetId |

## 7. 失败明细（最多 200 条）

无。

## 8. 结论

- **进度**：在测矩阵尚未跑完（3941 / 4059），以下为当前快照。
- **结论**：在测项全部下单成功，第三方对账通过。

## 9. 原始数据

- 分彩种：`backend/data/real-bet-matrix/by-lottery/*.jsonl`
- 合并：`backend/data/real-bet-matrix/all-results.jsonl`
- 批次汇总：`backend/data/real-bet-matrix/batch-summary.jsonl`
- 跑批日志：`backend/data/real-bet-matrix/batch-run.log`
