# Client API 模块 — 云端投注记录

> 契约：`backend/openapi/openapi.yaml` → `GET /client/cloud/bet-records`  
> 类型：`backend/contracts/cloud.ts`

## 接入步骤

1. `client/src/api/cloud/betRecords.ts` — **已实现**
2. `BetRecordsView` / `BetRecordsSchemeDetailView` — **已接 API**（走后端 + DB seed）
4. 开发环境无 token 时会静默调用 `POST /client/auth/login`（演示账号 vs8888）
5. **后端数据源**：`cloud_bet_records` 表（PostgreSQL）；DB 不可用时回退 Go 内存演示数据
6. **统计窗口**：`days` 默认 3，按 **UTC+8（Asia/Shanghai）自然日** 起算，含今天；响应附带 `dateFrom` / `dateTo`（闭区间展示）；查询条件为 `[since, until)` 左闭右开

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
