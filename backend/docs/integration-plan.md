# 前后端接入方案（Client / Admin）

> **状态**：Go Phase 0 脚手架已落地；接口以 OpenAPI + contracts 为单一事实来源  
> **关联**：[`openapi/openapi.yaml`](../openapi/openapi.yaml)、[`contracts/`](../contracts/)、[`client/src/api/`](../../client/src/api/)

---

## 1. 目标与原则

| 项 | 约定 |
|----|------|
| 基址 | `VITE_API_BASE_URL` → `{origin}/api/v1`（见 `client/.env.example`） |
| 协议 | REST + JSON；**实时推送**见 [`docs/websocket.md`](websocket.md)（WS-1～WS-5 已落地，不可用时轮询降级） |
| 鉴权 | **Bearer Access Token**；无 Refresh；401 统一清 token 并跳登录 |
| 响应包 | `{ code, message, data }`；`code === 0` 为成功 |
| 金额 | 接口传 **number（元，2 位小数）**；展示层 `toLocaleString` |
| 彩种 | 对外中文展示名；对内 **`code` + 展示名**（与 admin 附录 A 一致） |
| 演示数据 | 后端 PostgreSQL seed（`backend/migrations/*_seed_*`）；前端仅走 `*/api/*` |

**双端路径前缀**

- Client（会员/代理）：`/client/*`
- Admin（运营）：`/admin/*`
- 公共（维护、公告只读）：`/public/*`

---

## 2. 仓库内对接点

```
client/src/api/config.ts      # API_BASE、WS 开关
client/src/api/client.ts      # request() 通用 fetch
client/src/api/types.ts       # 逐步迁入 DTO（或与 shared 同步）
shared/mock/*                 # 双端共享类型/常量（非运行时数据源）

backend/openapi/openapi.yaml  # 接口契约（实现与联调依据）
backend/contracts/*.ts        # TS 类型（可与 shared 合并）
```

**推荐 Client 接入层结构（按域拆分）**

```
client/src/api/
  auth.ts
  cloud/betRecords.ts
  cloud/schemes.ts
  schemes/definitions.ts
  member/wallet.ts
  copyHall/rankings.ts
  ...
```

每个模块：`fetchX()` 直连 REST；页面/composable 只调模块函数。

---

## 3. 分阶段接入计划

### Phase 0 — 基础设施（阻塞项）

| 任务 | 后端 | Client | Admin |
|------|------|--------|-------|
| 选型与脚手架 | **Go**（`backend/cmd/server`），`/health` — 见 [go-server.md](go-server.md) | — | — |
| 登录与 Token | `POST /client/auth/login`、`/admin/auth/login` | 登录页 + token 存储 | 已有 Mock 登录对齐 |
| 请求拦截 | — | `request()` 注入 `Authorization` | 同上 |
| 维护开关 | `GET /public/maintenance` | 大厅拦截（API 轮询 / **WS `public.maintenance`**） | `GET/PUT /admin/operations/maintenance` ✅ |
| 统一错误 | 错误码表 | `ApiError` → ElMessage | 同上 |

### Phase 1 — 云端中心 + 投注记录（**已联调**）

| Client 页面 | 接口 | 状态 |
|-------------|------|------|
| `BetRecordsView` | `GET /client/cloud/bet-records` | ✅ |
| `BetRecordsSchemeDetailView` | `GET /client/cloud/bet-records/{schemeId}` | ✅ |
| `CloudCenterView` 运行列表 | `GET /client/cloud/schemes/running` | ✅ |
| 回头设置 | `GET/PUT /client/cloud/lookback` | ✅（PostgreSQL 按会员持久化） |
| 开启方案 | `POST /client/cloud/instances/{id}/start` | ✅ |
| 暂停 / 恢复 | `POST .../pause`、`POST .../resume` | ✅（PostgreSQL） |

> 旧路径 `POST /client/cloud/schemes/{id}/start` 已 **deprecated**，见 OpenAPI。

**列表页数据流**

1. Tab（真实/模拟）→ query `mode=real|sim`
2. 汇总栏：响应 `summary`（总投注、当日盈亏、胜率）；`dateFrom` / `dateTo` 标注 **UTC+8 自然日** 统计区间
3. 方案卡片：`groups[]`（服务端分组，**不含**明细笔数）
4. 详情页：`records[]` 为原表格字段
5. **后台 Worker**：按 `playTypeId` + `subPlayId` + `schemeGroups` 结算；**real 模式**扣钱包 / 派奖 / `bet_orders`；余额不足 **paused** + 审计；**sim 模式**仅 `cloud_bet_records`；**回头复位**写 `admin_audit_logs`（actor=`scheme-worker`）

### Phase 2 — 会员资产与订单（**已完成**）

| 模块 | 代表接口 | 状态 |
|------|----------|------|
| 登录 | `POST /client/auth/login`（PostgreSQL bcrypt） | ✅ |
| 会员中心 | `GET /client/member/profile`、`GET /client/member/wallet` | ✅ |
| 帐变 | `GET /client/orders/ledger`（`scope=self\|team`） | ✅ |
| 投注记录 | `GET /client/orders/bets`（`scope=self\|team`） | ✅ |
| 投注结算 Worker | `pending` 订单 ↔ `lottery_draws` 开奖 + 派奖 | ✅ |
| 充值/提现 | `GET /client/funds/records` · `GET /client/funds/recharge-channels` · `POST /client/funds/recharge`（内测演示即时到账） · `GET/POST /client/funds/withdraw` | 充提记录 ✅ · 充值渠道 ✅ · **充值下单 ✅（Demo）** · 提现 ✅ |
| 追号 | `GET /client/orders/chases`（`scope=self\|team`） | ✅ |
| 银行卡 | `GET/POST /client/member/payout-accounts` | ✅ |
| 团队总览 | `GET /client/team/overview`、`/stats`、`/members` | ✅ |
| 开户中心 | `POST /client/team/members` | ✅ |
| 推广设定 | `GET/POST /client/team/promo-links` | ✅ |

> **Phase 2 收尾**：团队 Tab（帐变/投注/追号 `scope=team`）、开户、推广已纳入 Phase 2；**充值下单**已接 `POST /client/funds/recharge`（内测演示即时到账，非真实第三方支付）；**充提记录**已接 `GET /client/funds/records`。  
> **Guaji 对接后（§23.1 甲 · §24）**：Client/Admin **同步下线** 充提、团队、银行卡、开户模块；real 余额改第三方 CNY；详见 [`third-party-guaji-integration-plan.md`](third-party-guaji-integration-plan.md)。

### Phase 3 — 玩法与方案域（**已完成**）

| 模块 | 代表接口 | 状态 |
|------|----------|------|
| 跟单大厅 | `GET /client/copy-hall/rankings?lotteryCode=&board=`（slots 含 `playTypeId`/`subPlayId` → 游戏详情 query） | ✅ |
| 方案下载 / 分享池 | `GET /client/schemes/share-catalog` | ✅ |
| 私池 CRUD | `GET/POST/PATCH/DELETE /client/schemes` | ✅ |
| 分享池 → 私池 | `POST /client/schemes/share/{id}/add-to-cloud` | ✅ |
| 自创添加 / fork 上云 | `POST /client/schemes/{id}/add-to-cloud`、`fork-and-add-to-cloud` | ✅ |
| 倍投 / 期次 | `GET /client/schemes/{id}`、`PUT .../bet-multiplier`、`PUT .../rounds` | ✅ |
| 跟单 / 反买投注 | `POST .../share/{id}/follow-bet`、`POST .../contrary/bet` | ✅ |
| 玩法选项种子 | `GET /client/games/{code}/scheme-options` | ✅ |
| 云端全局规则 | `GET/PUT /client/cloud/global-settings` | ✅ |
| 游戏详情 | `GET /client/games/{code}/detail`、下注 `POST .../bets`（真实选号 `betPayload.groupContent` + 扣钱包） | ✅ |
| 历史开奖 | `GET /client/games/{code}/draws` | ✅ |

> **方案域** 产品定案 + OpenAPI 对照见 [`modules/schemes.md`](modules/schemes.md) §8；类型见 [`contracts/schemes.ts`](../contracts/schemes.ts)。Phase 3 建议顺序：跟单大厅 → 分享池/私池 → 倍投期次 → 游戏详情。

### Phase 4 — 内容与互动（**已完成**）

| 模块 | 代表接口 | 状态 |
|------|----------|------|
| 平台公告 | `GET /client/content/announcements`、`GET .../{id}` | ✅ |
| 常见问题 | `GET /client/content/faq`、`GET .../{id}` | ✅ |
| 帮助中心 | `GET /client/content/help` | ✅ |
| 意见回馈 | `POST /client/content/feedback` | ✅ |
| 聊天室 Hub | `GET /client/chat/hub` | ✅ |
| 系统讯息 | `GET /client/chat/system-messages` | ✅ |
| 会话消息 | `GET/POST /client/chat/threads/{peerId}/messages` | ✅ |

> WebSocket 实时推送见 [`websocket.md`](websocket.md)（WS-1～WS-5 已落地；充值 Demo 即时到账，生产替换第三方支付）。

### Phase 5 — Admin 运营面（**已完成**）

| 模块 | 代表接口 | 状态 |
|------|----------|------|
| Admin 登录 | `POST /admin/auth/login` | ✅ |
| 内容 CMS · 公告 | `GET/PUT/DELETE /admin/content/announcements` | ✅ |
| 内容 CMS · FAQ | `GET/PUT/DELETE /admin/content/faq/*` | ✅ |
| 内容 CMS · 帮助 | `GET/PUT/DELETE /admin/content/help/articles` | ✅ |
| 内容批量加载 | `GET /admin/content/bundle` | ✅ |
| 提现审批 / 出款 | `GET /admin/funds/withdraw/orders`、`POST .../approve|reject|confirm-paid` | ✅ |
| 方案监控 / 分享池 | `GET /admin/schemes/instances`、`POST .../force-stop|release-stop`、`PATCH/DELETE .../share/{id}` | ✅ |
| 操作审计 | `GET /admin/system/audit-logs`（写操作服务端留痕） | ✅ |
| 跟单大厅运营 | `GET /admin/copy-hall/rankings`、`PUT .../boards/{lottery}/{board}`、`POST .../reset` | ✅ |
| 彩种目录 | `GET /admin/games/lottery-catalog`、`PATCH .../lottery-catalog/{code}` | ✅ |
| 大厅运营位 | `GET /admin/content/bundle`（含 lobbySlots）、`PUT /admin/content/lobby-slots` | ✅ |
| 站点品牌 | `GET /admin/content/bundle`（含 siteBrand）、`PUT /admin/content/site-brand`、`GET /public/site-brand` | ✅ |
| 系统维护 | `GET/PUT /admin/operations/maintenance`、`GET /public/maintenance` | ✅ |
| 仪表盘 KPI | `GET /admin/dashboard/kpi` | ✅（WS-4：`withdraw`/`scheme`/`dashboard.kpi` 事件 + 降级轮询） |
| 会员查询 | `GET /admin/members`、`GET .../{memberNo}`、`GET .../{memberNo}/ledger` | ✅ |
| 会员运营事件 | `POST /admin/members/{memberNo}/ops` | ✅ |
| 投注与帐变 | `GET /admin/orders/bets`、`GET /admin/orders/chases`、`GET /admin/orders/ledger` | ✅ |
| 银行卡审核 | `GET /admin/funds/payout-accounts`、`POST .../{id}/approve|reject` | ✅ |
| 代理管理 | `GET /admin/agents/l1|l2`、`PUT .../{code}/commission-cap`、`GET .../members` | ✅ |
| 推广与渠道 | `GET/PUT /admin/agents/promo-channel` | ✅ |
| 运营报表 | `GET /admin/reports/lottery-stat`、`GET /admin/reports/pnl` | ✅ |
| 方案模板库 | `GET/PUT/DELETE /admin/games/scheme-templates`、`POST .../reset`、`GET /client/games/scheme-templates` | ✅ |
| 角色 RBAC | `GET/PUT/DELETE /admin/system/roles` + Admin 侧栏/路由 menuPaths 过滤；**账号绑定角色** `admin_users.role_id` | ✅ |
| Admin 账号 CRUD | `GET/POST/PUT/DELETE /admin/system/users` + Admin 账号管理页 | ✅ |
| 充值渠道 | `GET/PUT /admin/funds/recharge-channels`、`GET /client/funds/recharge-channels` | ✅ |
| 系统讯息模板 | `GET/PUT/DELETE /admin/content/system-message-templates` | ✅ |
| 系统讯息下发 | `POST /admin/content/system-messages/send`（单会员 + WS 推送） | ✅ |
| 其他占位 | 客服聊天占位 | 待接 / 跳过 |

> Admin / Client 前端均已接 API；演示数据来自 DB seed；占位页见 [`GoLiveMemoView`](../../admin/src/views/system/GoLiveMemoView.vue)。

与 Client 共享领域模型，路径走 `/admin/*`；**Admin RBAC**：登录响应与 JWT 含 `roleId`，由 `admin_users` 表绑定，侧栏/路由按 `menuPaths` 过滤。重点：

- 仪表盘 KPI、**全站方案监控**（`GET /admin/schemes/instances?scope=user|share`）、跟单大厅运营
- 分享池快照维护（`PATCH/DELETE /admin/schemes/share/{id}`）、强停/解封（`force-stop` / `release-stop`）
- 提现审批/出款、彩种目录、审计日志

---

## 4. 模块 ↔ 路由对照（Client）

| 域 | 路由名 | HTTP |
|----|--------|------|
| 云端投注记录 | `bet-records` | `GET .../cloud/bet-records` |
| 方案投注明细 | `bet-records-scheme` | `GET .../cloud/bet-records/{schemeId}` |
| 云端中心 | `cloud` | `GET/PUT .../cloud/*`、实例启停 `POST .../cloud/instances/{id}/*` |
| 方案配置 / 下载 | `scheme-edit`、`scheme-download` 等 | `GET/POST/PATCH /client/schemes/*`、`GET .../share-catalog` |
| 跟单大厅 | `copy-hall` | `GET .../copy-hall/rankings` |
| 会员投注记录 | `member-bet-records` | `GET .../orders/bets`（筛选维度多于云端三日） |

> **说明**：`member-bet-records` 与 `bet-records` 数据源不同——前者全站订单查询，后者云端运行方案近 N 日汇总。

---

## 5. Client 侧改造 checklist（按页）

以 **投注记录** 为例（Phase 1 模板，其它页复制此模式）：

1. 在 `client/src/api/cloud/betRecords.ts` 实现 `fetchBetRecordGroups(mode)`、`fetchBetRecordDetails(mode, schemeId)`
2. `BetRecordsView`：`onMounted` + Tab 切换时调 API；loading/error 态
3. 页面不直接 import 前端 mock seed
4. `BetRecordsSchemeDetailView` 同理
5. 联调：`.env` 设 `VITE_API_BASE_URL=http://localhost:8080/api/v1`，后端 migrate + seed

---

## 6. 环境与联调

完整 E2E 步骤见 [`integration-checklist.md`](integration-checklist.md)。

```env
# client/.env.local
VITE_API_BASE_URL=http://127.0.0.1:8080/api/v1
VITE_WS_ENABLED=true
```

```env
# admin/.env.local
VITE_API_BASE_URL=http://127.0.0.1:8080/api/v1
VITE_WS_ENABLED=true
```

```env
# backend/.env（节选）
WS_ENABLED=true
WS_AUTH_VIA_QUERY=true
CORS_ORIGINS=http://localhost:5173,http://localhost:5174,http://127.0.0.1:5173,http://127.0.0.1:5174
```

后端 CORS 已允许 `5173`（client）、`5174`（admin）。WebSocket：`/api/v1/ws/public`、`/ws/client`、`/ws/admin`。

---

## 7. 待定项（实现前确认）

| # | 问题 | 建议 | 状态 |
|---|------|------|------|
| D1 | 数据库 | **PostgreSQL 18.x** | ✅ 已定 |
| D2 | Go DB 栈 | **pgx + sqlc + goose**；HTTP 标准库 | ✅ 已定 |
| D3 | 连接配置 | `backend/.env` · 见 [database.md](database.md) | ✅ 已配 |
| D4 | 建表备注 | 每表 `COMMENT ON TABLE` + 每字段 `COMMENT ON COLUMN` · [db-schema-conventions.md](db-schema-conventions.md) | ✅ 已定 |
| D5 | 索引/约束 | 非空、CHECK、FK、列表复合索引 · Phase 2 表见 [modules/members.md](modules/members.md) | ✅ 已落地 |
| 1 | 「当日盈亏」时区 | 按会员账号时区或平台 UTC+8 | **平台 UTC+8**（`Asia/Shanghai`） |
| 2 | 「最近三日」起算 | 自然日 0 点 vs 滚动 72h | **已定**：含今天在内 N 个 **UTC+8 自然日**（`Asia/Shanghai`，左闭右开 `[since, until)`） |
| 3 | 模拟记录数据源 | 与真实分路；`mode=sim` 仅查模拟方案 | **纳入**：API `runMode` / `mode` 分路；Client **WS-2** 事件刷新 + REST 降级轮询（[`websocket.md`](websocket.md)） |
| 4 | Token 存储 | Client：`localStorage`；Admin：同左（演示） | **是** |
| 5 | 分页 | 方案分组不分页；明细 `cursor` + `limit` | **是**；统计自然日按 **UTC+8 日界线**，真实/模拟 **分路** |
| 6 | 充值 `POST /client/funds/recharge` | 内测 Demo 即时到账；生产替换第三方支付 | **Demo 已接** |
| 7 | Admin 会员运营动作 | 统一服务端事件 + 审计 | **已接** `POST /admin/members/{memberNo}/ops` |

---

## 8. 文档维护

- 新增/变更接口：**先改** `openapi/openapi.yaml`，再改 `contracts/`，最后 Client `api/*`
- 联调验收：按 [`integration-checklist.md`](integration-checklist.md) 勾选
- **OpenAPI**：Phase 0～5 核心路径已登记（含 CMS/Chat、`/client/funds/recharge`、系统讯息下发）；未列路径以 `server.go` 路由为准
- WebSocket 事件：**先改** [`docs/websocket.md`](websocket.md) 与 `contracts/ws.ts`，再实现 Hub / composable
- 双端 Mock 关键 id 继续对齐 `shared/mock/appendixMock.ts`（附录 A）
