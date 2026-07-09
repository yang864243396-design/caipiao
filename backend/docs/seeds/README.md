# P0 彩种与玩法 Seed（草案）

> 对应规划：`backend/docs/lottery-catalog-migration-plan.md` **v1.0（已冻结）**  
> **P1 已接入** `backend/migrations/00069_*`、`00070_*`；`platform_340_play_mapping.csv` 已生成。

## 文件清单

| 文件 | 行数 | 说明 |
|------|------|------|
| `lottery_catalog.csv` | 47 | 含 `outbound_lottery_code` |
| `play_templates.csv` | 6 | — |
| `play_types.csv` | **52** | SSC 15 + LHC 15 + 其余 |
| `sub_plays.csv` | **340** | 含 LHC 82、SSC 175 |
| `ssc_175_play_mapping.csv` | 175 | SSC 玩法对照表（§11-20） |
| `generate_p0_seeds.py` | — | `python backend/docs/seeds/generate_p0_seeds.py` |
| `00069_lottery_play_catalog_draft.sql` | — | P1 迁入 |

## 重新生成 CSV

```bash
python backend/docs/seeds/generate_p0_seeds.py
```

## 已落实（§9，2026-06）

| 项 | 状态 |
|----|------|
| SSC | `qianzhonghou3` + `qianhou3` 已拆类（15 大类 / 175 子玩法） |
| 11选5 | `segment_rule`：`numberPoolMin=1, numberPoolMax=11, pickCount=5` |
| 六合彩 | **82** 子玩法 + **15** 大类 |
| 对外编码 | `outbound_play_code` = `{template}:{type_id}:{sub_id}`；彩种 `outbound_lottery_code` |
| 唯一键 | `(template_code, type_id, sub_id)` |

业务决策见规划 **§8.8 / §14.1**；实施前可选见 **§17**。
