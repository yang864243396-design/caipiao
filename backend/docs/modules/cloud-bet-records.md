# Client API 模块 — 云端投注记录

> 契约：`backend/openapi/openapi.yaml` → `GET /client/cloud/bet-records`  
> 类型：`backend/contracts/cloud.ts`

## 接入步骤

1. `client/src/api/cloud/betRecords.ts` — **已实现**
2. `BetRecordsView` / `BetRecordsSchemeDetailView` / `BetDetailView` — **已接 API**
3. 开发环境无 token 时会静默调用 `POST /client/auth/login`（演示账号 vs8888）
4. **后端数据源**：`cloud_bet_records` 表（PostgreSQL）；DB 不可用时回退 Go 内存演示数据
5. **统计窗口**：`days` 默认 3，按 **UTC+8（Asia/Shanghai）自然日** 起算，含今天；响应附带 `dateFrom` / `dateTo`（闭区间展示）；查询条件为 `[since, until)` 左闭右开
6. **单笔详情**：`GET /client/cloud/bet-records/item/{recordNo}`；主键为 `record_no`；超出 3 天窗口返回不存在；列表项带 `recordNo` 供跳转
7. **「我的投注」位名**：由后端 `schemes.FormatBetContentLines` 按方案定义还原 `playRule` 位段后随 `betContentLines` 下发（如中三码 → 千位/百位/十位）。玩法无按位语义（组合/和值/任选/龙虎…）或行数与位段对不上时不下发，前端原样分行。位名不可在前端按行序硬编码——同样是三行，前三码是万/千/百，中三码是千/百/十
8. **极速彩历史注单**：迁移 00136（2026-07-28 00:34 CST）之前下的 `hash_jisu` / `tron_jisu` 注单期号属于对方彩种的期号族，详情页开奖号查不到、恒显示 `—`。已决定不回填：盈亏当时已按第三方结算落库，且这批单会随 3 天查询窗口自然过期

## 字段映射（Mock → API）

| Mock 字段 | API 字段 | 说明 |
|-----------|----------|------|
| `amount: '10.00'` | `amount: 10` | 字符串 → number |
| `pnl: '+5.00'` | `pnl: 5` | 符号由正负表达 |
| `pnlPositive` | — | 删除，由 `pnl >= 0` 推导 |
| `recordCount` | — | 列表不展示笔数 |
| 客户端 `groupBetRecordsByScheme` | `groups[]` | 分组改由服务端返回 |

## 示例响应

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "mode": "real",
    "days": 3,
    "dateFrom": "2026-05-26",
    "dateTo": "2026-05-28",
    "summary": { "totalBet": 235, "dayPnl": -27.5, "winRate": 62.5 },
    "groups": [
      {
        "schemeId": "sch-wan",
        "schemeName": "禄螭万位计划",
        "totalBet": 140,
        "dayPnl": -80,
        "winRate": 66.7
      }
    ]
  }
}
```
