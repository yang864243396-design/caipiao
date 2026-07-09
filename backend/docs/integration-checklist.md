# 三端联调验收清单

> **用途**：本地/预发环境 E2E 验收；配合 [`integration-plan.md`](integration-plan.md)、[`third-party-guaji-integration-plan.md`](third-party-guaji-integration-plan.md)、[`websocket.md`](websocket.md)、[`go-server.md`](go-server.md)  
> **前提**：PostgreSQL 已迁移；`backend/.env` 中 `DB_REQUIRED=true`、`WS_ENABLED=true`；第三方对接须配置 `GUAJI_*`（见 guaji 方案 §1.3、§20.4）

---

## 0. 环境准备

### 0.1 启动服务

```bash
# 终端 1 — 后端
cd backend
cp .env.example .env   # 填 DB_PASSWORD 等
make migrate-up
make run               # :8080

# 终端 2 — Client
cd client
cp .env.example .env.local
# 编辑：VITE_API_BASE_URL=http://127.0.0.1:8080/api/v1
#       （演示数据：后端 migrate + seed，无需前端 Mock 开关）
#       VITE_WS_ENABLED=true
npm run dev            # :5173

# 终端 3 — Admin
cd admin
cp .env.example .env.local
# 编辑：同上 + VITE_WS_ENABLED=true
npm run dev            # :5174
```

### 0.2 演示账号

| 端 | account | password | 备注 |
|----|---------|----------|------|
| Client | `vs8888` | `vs8888` | `member_no` = `M00001` |
| Admin | `admin` | `admin123` | 查 `admin_users` 表；无 DB 时回退 `.env` |

### 0.3 健康检查

- [ ] `GET http://127.0.0.1:8080/api/v1/health` → `code: 0` 且 `data.db: up`（若 8080 被其他服务占用，改用实际端口，如 `:8099`）
- [ ] Client 登录页可打开，无 CORS 报错
- [ ] Admin 登录页可打开，无 CORS 报错

> **REST 自动化**：`backend/scripts/e2e-smoke.ps1 -BaseUrl http://127.0.0.1:<PORT>/api/v1`（覆盖 §1–§6 大部分 REST 项；**§7 Guaji 待接口落地后扩展**；WS / UI 项需人工）

---

## 1. 鉴权与维护（Phase 0）

| # | 步骤 | 预期 |
|---|------|------|
| 1.1 | Client 用 `vs8888/vs8888` 登录 | 跳转大厅；`localStorage` 有 token |
| 1.2 | 错误密码登录 | 提示凭据无效 |
| 1.3 | Admin 用 `admin/admin123` 登录 | 进入仪表盘；侧栏按角色显示 |
| 1.4 | Admin 维护页开启全站维护 | `GET /public/maintenance` → `enabled: true` |
| 1.5 | Client 刷新大厅 | 维护拦截页展示 |
| 1.6 | Admin 关闭维护 | Client 可正常进入 |
| **WS-1** | 维护开关时 Client 已连接 WS | 无需刷新，维护页/恢复自动切换 |

---

## 2. 云端中心（Phase 1）

| # | 步骤 | 预期 |
|---|------|------|
| 2.1 | 云端中心 → 运行中方案列表 | 有数据或空态；非 Mock seed |
| 2.2 | 投注记录 Tab 切换真实/模拟 | 列表随 `mode` 变化 |
| 2.3 | 进入某方案投注明细 | 明细与 REST 一致 |
| 2.4 | 暂停 / 恢复某运行实例 | 状态变更；审计可查 |
| **WS-2** | 暂停后云端列表 | `client.scheme.instance` 事件触发列表刷新 |

---

## 3. 会员资产与订单（Phase 2）

| # | 步骤 | 预期 |
|---|------|------|
| 3.1 | 会员中心 → 资料 / 钱包 | 余额与 DB `member_wallets` 一致 |
| 3.2 | 帐变流水 | 分页/筛选正常 |
| 3.3 | 投注记录 / 追号（self） | 与 `bet_orders` / `chase_orders` 一致 |
| 3.4 | 团队 Tab（scope=team） | 下级数据可见（演示账号有团队） |
| 3.5 | 提现：填写金额提交 | 订单 pending；余额冻结 |
| 3.6 | 充提记录 Tab | `GET /client/funds/records` 有充值/提现行 |
| 3.7 | 充值页渠道列表 | 与 Admin 上架渠道一致（非本地 Mock 渠道） |
| 3.8 | 充值页输入金额提交 | 即时到账；充提记录可见；钱包余额增加 |
| 3.9 | 银行卡绑定 / 列表 | CRUD 正常 |
| **WS-2** | Worker 派奖或扣款后 | `client.wallet` 事件；钱包数字更新 |

---

## 4. 玩法与方案（Phase 3）

| # | 步骤 | 预期 |
|---|------|------|
| 4.1 | 跟单大厅 rankings | 榜单与 Admin 运营配置一致 |
| 4.2 | 分享池下载 → 私池 | 方案出现在云端可添加列表 |
| 4.3 | 方案编辑：倍投 / 期次保存 | `PUT` 成功；刷新仍保留 |
| 4.4 | 游戏详情页 | 期号、倒计时、走势正常 |
| 4.5 | 手动下注（选号） | 扣钱包；`bet_orders` 新增 pending |
| 4.6 | 历史开奖列表 | 分页正常 |
| **WS-5** | Worker 写入新开奖 | 游戏详情期号/开奖球更新（订阅 `public.draw:{code}`） |

---

## 5. 内容与聊天（Phase 4）

| # | 步骤 | 预期 |
|---|------|------|
| 5.1 | 平台公告列表 / 详情 | 已读状态更新 |
| 5.2 | FAQ / 帮助中心 | 内容与 Admin CMS 一致 |
| 5.3 | 意见回馈提交 | 成功提示 |
| 5.4 | 聊天 Hub | 未读数 / 会话列表 |
| 5.5 | 进入客服会话发消息 | 消息落库；可能收到 auto-reply |
| 5.6 | 系统讯息页 | 历史列表可读 |
| **WS-3** | 会话页已打开时对方发消息 | 新消息即时出现 |
| **WS-3** | Admin 下发系统讯息给 `M00001` | Client 系统讯息页即时出现新条 |

### 5.7 系统讯息下发（Admin → Client）

1. Admin → 内容 → 系统讯息模板 → 某行点「下发」
2. 会员号填 `M00001` 或 `vs8888`，选模板（可选覆盖正文）
3. 提交成功
4. Client（同账号在线）→ 系统讯息页无需刷新可见新消息
5. `GET /client/chat/system-messages` 含该条

---

## 6. Admin 运营面（Phase 5）

| # | 模块 | 步骤 | 预期 |
|---|------|------|------|
| 6.1 | 仪表盘 KPI | 打开 Dashboard | 充值/提现/投注等指标有数 |
| **WS-4** | Client 提交提现或 Admin 审批 | Dashboard「待审核提现」等 KPI 自动刷新 |
| **WS-4** | Client Demo 充值成功 | Dashboard「今日成功充值」自动刷新 |
| 6.2 | 会员 | 搜索 `M00001` → 详情 | 资料、帐变、运营动作 |
| 6.3 | 提现审批 | 通过 pending 提现 | 状态变更 |
| 6.4 | 提现出款 | 确认已打款 | 完成闭环 |
| **WS-4** | 6.3 操作时 Admin 已开 WS | 提现队列页自动刷新 |
| 6.5 | 方案监控 | 强停 / 解封分享池方案 | Client 侧实例状态受影响 |
| **WS-4** | 6.5 操作 | 方案监控列表自动刷新 |
| 6.6 | 跟单大厅运营 | 改榜 / 重置 | Client 大厅反映 |
| 6.7 | 彩种目录 | 上下架某彩种 | Client 大厅可见性变化 |
| 6.8 | 充值渠道 | 下架某渠道 | Client 充值页不再展示 |
| 6.9 | Admin 账号 | 新建/禁用账号 | 仅 active 可登录 |
| 6.10 | 角色 RBAC | 限制 menuPaths | 侧栏菜单过滤 |
| 6.11 | 审计日志 | 上述写操作后 | `GET /admin/system/audit-logs` 有记录 |
| 6.12 | 报表 | 彩种统计 / 盈亏 | 图表或表格有数据 |

---

## 7. 第三方授权（Guaji · T1–T5）

> **依据**：[`third-party-guaji-integration-plan.md`](third-party-guaji-integration-plan.md) 定稿 v15 · §9.1 / §16 / §22–§24  
> **前提**：后端已部署 guaji 适配层；测试环境对接 `hash.iyes.dev`（或 `.env` 配置的第三方基址）；准备可用 **Hash 测试账号**（用户名 + 密码）。

### 7.1 环境与绑定（T1 / T1b）

| # | 步骤 | 预期 |
|---|------|------|
| 7.1.1 | 新会员（或清空 guaji 绑定后）Client 登录 | **0 绑定** → 落地 **`/member/auth/bind`**，无法进大厅 |
| 7.1.2 | 绑定页填写 **第三方用户名 + 密码**，完成 MFA | 成功 → **`is_active=true`** → 跳转 **会员中心** |
| 7.1.3 | 会员中心顶栏 | 展示 **第三方用户名** + **`CNY ¥…`** + **[刷新]** + **[切换授权账号]** |
| 7.1.4 | 第三方仅返回非 CNY 余额 | 顶栏仍显示 **`CNY ¥0.00`**（§21.1） |
| 7.1.5 | 点顶栏 **[刷新]** | 余额重新拉取 `users/i/info` CNY |
| 7.1.6 | 已有启用 A，列表页 **添加授权账号** 绑 B | 绑成功后 **弹窗询问是否切换**；选否则 B 为「已绑未启用」 |
| 7.1.7 | 绑定已被他人占用的 `guaji_username` | 提示 **须先让对方解绑**；绑定失败 |
| 7.1.8 | Admin → 会员详情 → **授权 Tab** | **只读**：用户名、启用、绑定时间、最近同步、最后投注、最近 Token 失效原因；**无代绑** |

### 7.2 全局门禁（T1）

| # | 步骤 | 预期 |
|---|------|------|
| 7.2.1 | 有绑定但 **无启用中** 授权 | 仅可访问 **`/member/auth/bind`**、**`/member/auth/list`** |
| 7.2.2 | 无启用授权时直接访问 `/`、`/cloud`、`/play/detail` | **重定向** 授权页；Toast「请先启用授权账号」 |
| 7.2.3 | **sim** 方案编辑/运行页 | 无启用授权时 **同样被拦截** |
| 7.2.4 | 列表页对未启用行点 **「设为启用」** | 确认「将停止全部挂机方案」→ 切换成功 → **留在列表页** + Toast 去云端中心开启 |
| 7.2.5 | 无启用授权时访问 `/member/faq`、`/member/chat` | **同样拦截**（§22.6 **甲**） |

| # | 步骤 | 预期 |
|---|------|------|
| 7.3.1 | 切换授权（A→B）前云端有 **running / pending** 实例 | 切换后 **全部变 `paused`**；卡片文案 **「已暂停」** |
| 7.3.2 | 切换完成后 | **新授权已启用**；可 **立即手动下注**；挂机方案仍停 |
| 7.3.3 | 云端中心 → **一键开启方案** | 批量恢复 **`pending` + `paused`**（含切换后暂停的）；**须手动点击** |
| 7.3.4 | 解绑当前 **唯一启用** 账号 | **立即门禁**；方案保持 `paused`；仅绑定页可访问 |
| 7.3.5 | 解绑确认弹窗 | 标准文案含「解绑后将停止全部挂机方案」 |
| 7.3.6 | Client **退出平台登录**（非解绑） | 方案 **不停止**；Worker 仍运行（服务端 Token 保持） |
| **WS-2** | 7.3.1 批量 pause 后 | 云端中心列表通过 `client.scheme.instance` 刷新 |

### 7.4 Token 失效与重新授权（T1）

| # | 步骤 | 预期 |
|---|------|------|
| 7.4.1 | 模拟 Token 失效（过期 / 服务端标记失效） | Toast「授权已失效…」→ **全局门禁** |
| 7.4.2 | Token 失效时 | 全部 **running / pending → `paused`** |
| 7.4.3 | 列表页点 **「重新授权」** | **全自动 MFA** 刷新 Token（会员无感） |
| 7.4.4 | 重新授权成功 | 跳转 **会员中心**；全站可访问；方案 **仍须手动/一键开启** |
| 7.4.5 | 连续自动重试 **3 次** 仍失败 | 引导 **绑定页** 重填密码 |

### 7.5 资金与账目（T1b / T5）

| # | 步骤 | 预期 |
|---|------|------|
| 7.5.1 | 会员中心 **无充值/提现**；**无团队 Tab**；**无银行卡/开户入口** | 已移除（§22.4–22.5 · §24） |
| 7.5.2 | 余额大卡 | **CNY 实账** +「充值/提现请前往第三方平台」+ 客服 |
| 7.5.3 | **`/member/fund-records`** | 仅 **钱包流水**（`bet_debit` / `payout`）；Client **不展示** Demo 充提 |
| 7.5.4 | **`/member/bet-records`**、**`/member/scheme-pnl`** | 默认 **当前启用授权**；可切 **「全部历史」** |
| 7.5.5 | **`/member/pnl-report`**、**`/member/lottery-stat`** | **仅个人** scope；**无** team 切换（§24） |
| 7.5.14 | Client 访问团队/银行卡/开户相关 API | **404 或路由不存在**（§24） |
| 7.5.15 | Admin 侧栏 | **无** 银行卡审核、代理管理、推广渠道等（§24） |
| 7.5.6 | 访问 **`/member/ledger`**、**`/member/chase-records`** | **404 或重定向**；菜单无入口（§22.3） |
| 7.5.7 | **real** 手动下注余额不足 | Toast：**「可用余额不足，请前往第三方平台充值」** |
| 7.5.8 | **real** Worker 余额不足 | 实例 **`paused`**；卡片 **「钱包余额不足」** |
| 7.5.9 | **sim** 实例 | **不因** 第三方余额不足而 paused |
| 7.5.10 | real 投注/派奖成功后 | 会员中心顶栏 CNY **自动刷新** |
| 7.5.11 | `wallet_ledger` 镜像 | 仅 **`bet_debit` / `payout`**；含 `guaji_account_id`（T5） |
| 7.5.12 | Admin 侧栏 / Dashboard | **无** 提现审批、充值渠道、今日充值 KPI 等充提入口（§23.1 **甲**） |
| 7.5.13 | 游戏详情 → 手动下注 dock | 展示 **CNY 实账**（与顶栏同源；进入页/下注前刷新，§23.2 **甲**） |

### 7.6 真实投注与结算（T4 / T5 · 对接完成后）

| # | 步骤 | 预期 |
|---|------|------|
| 7.6.1 | real Worker / 手动下注 | 走 **`web_bets/lott`**；**不扣** 本地 `member_wallets` |
| 7.6.2 | 第三方接单成功 | 写入 `bet_orders`（含 `guaji_account_id`、`third_party_bet_id`、`outbound_*` 快照） |
| 7.6.3 | 结算同步 | 以 **第三方派奖** 为准（C8/C17）；镜像 `payout` 流水 |
| 7.6.4 | 开奖 WS | **忽略** 福彩 3D、福彩排列 3D、排列 2/3 |

### 7.7 部署环境（§20.4）

| # | 步骤 | 预期 |
|---|------|------|
| 7.7.1 | 测试站 `.env` | 固定 `hash.iyes.dev`；会员绑定页 **不可选** 环境 |
| 7.7.2 | 正式站 `.env` | 固定 `s9-xia.5rf9q.com` |

### 7.8 监控与运维（T6）

| # | 步骤 | 预期 |
|---|------|------|
| 7.8.1 | `GET /api/v1/health` | `data.guaji` 含 `enabled/httpReachable/wsReachable`（配 `GUAJI_TEST_*` 时含 `loginOk/balanceCny`） |
| 7.8.2 | `GET /admin/guaji/health`（Admin） | 返回 `totalBindings/activeAccounts/erroredTokens/expiringSoon` |
| 7.8.3 | Token 临期/失效（`expiringSoon>0` 或 `erroredTokens>0`） | 后端日志出现 `guaji token health alert`（每 5 分钟巡检） |
| 7.8.4 | `make guaji-smoke`（配 `GUAJI_TEST_*` + `GUAJI_WS_PATH=/ws`） | **实测**：http=true、ws=true、login=true、balanceCny>0 |
| 7.8.4b | `make guaji-capture` | 打印登录/余额/开奖原始报文，核对协议（成功码、字段名、`lottery_logXXX`/号码字段） |
| 7.8.5 | 凭证轮换 | 见 Runbook §A（`GUAJI_CREDENTIALS_KEY` 轮换流程） |

> **Runbook** 见 [`third-party-guaji-integration-plan.md`](third-party-guaji-integration-plan.md) §25。

---

## 8. WebSocket 总表

| 阶段 | 端点 | Topic / 事件 | 触发源 | 验证方式 |
|------|------|--------------|--------|----------|
| WS-1 | `/ws/public` | `public.maintenance` | Admin 维护开关 | Client 维护拦截无需轮询 |
| WS-2 | `/ws/client` | `client.scheme.instance` | 实例启停 / Worker | 云端中心列表刷新 |
| WS-2 | `/ws/client` | `client.wallet` | 扣款 / 派奖 | 钱包余额刷新 |
| WS-3 | `/ws/client` | `client.chat.thread.message` | REST 发消息 | 会话页即时消息 |
| WS-3 | `/ws/client` | `client.chat.system.message` | Admin 系统讯息下发 | 系统讯息页即时 |
| WS-4 | `/ws/admin` | `admin.withdraw.queue.changed` | 提现审批/出款 | Admin 提现页刷新 |
| WS-4 | `/ws/admin` | `admin.scheme.monitor.changed` | 方案强停等 | Admin 监控页刷新 |
| WS-4 | `/ws/admin` | `admin.dashboard.kpi.changed` | Demo 充值到账 | Dashboard 今日充值刷新 |
| WS-5 | `/ws/public` | `public.draw.result` | 新开奖入库 | 游戏详情开奖区更新 |

**降级验证**：`.env` 设 `VITE_WS_ENABLED=false`（或后端 `WS_ENABLED=false`）→ 各页仍可通过 REST / 轮询正常工作。

---

## 9. 已知跳过项

| 项 | 说明 |
|----|------|
| **§3 充提（Guaji 上线后）** | §3.5–3.8 充值/提现 Demo 流程 **废弃**；§3.4 团队 Tab、§3.9 银行卡 **废弃**；§6.3–6.4 提现闭环 **废弃**；以 **§7** 为准 |
| **§3 团队/开户（Guaji 上线后）** | §3.4 scope=team、§3.9 银行卡 CRUD **废弃**；以 **§7.5.14–7.5.15**（§24）为准 |
| `POST /client/funds/recharge` 真实支付 | Guaji 方案删除本平台充提；Client/Admin 相关 API 待下线 |
| `GET/POST /client/team/*`、`payout-accounts` | Guaji 方案 §24 前后台同步下线 |
| Admin 客服 IM | 占位页，无完整会话管理 |
| Redis 多实例 WS | 单进程 Hub；水平扩展待做 |
| 系统讯息批量/分群广播 | 仅单会员 `POST .../send` |
| OpenAPI CMS/Chat | ✅ 已登记 Client/Admin 内容、聊天、维护、公共品牌/大厅位 |

---

## 10. 构建门禁（提交前）

```bash
cd backend && go build ./...
cd client && npm run build
cd admin && npm run build
```

全部通过后再标记联调完成。

### 10.1 REST 冒烟（可选）

```powershell
cd backend
go build -o bin/server.exe ./cmd/server
# 确保迁移已执行：go run ./cmd/migrate up
$env:PORT='8099'; .\bin\server.exe   # 另开终端

.\scripts\e2e-smoke.ps1 -BaseUrl http://127.0.0.1:8099/api/v1
```

**2026-05-29 本地结果**（`:8099` 最新二进制 + 迁移 00060）：**48 PASS / 0 FAIL**（含 WS 端点 HTTP 400 探活）。

| 类别 | 自动化 | 需人工（浏览器 / WS 实时） |
|------|--------|---------------------------|
| §0 环境/CORS | health | 登录页、CORS |
| §1 鉴权/维护 | 1.1–1.4、1.6 | 1.5 维护拦截 UI；**WS-1** |
| §2 云端 | 2.1–2.2 | 2.3–2.4 明细/暂停；**WS-2** |
| §3 会员资产 | 3.1–3.9（含 Demo 充值） | 3.5 提现提交；**WS-2** 钱包 |
| §4 玩法方案 | 4.1–4.2、4.4、4.6 | 4.3 编辑保存；4.5 手动下注；**WS-5** 开奖 |
| §5 内容聊天 | 5.1–5.7 | **WS-3** 会话/系统讯息 |
| §6 Admin | 6.1–6.3、6.5–6.12 | 6.3–6.4 提现闭环；6.9–6.10 RBAC；**WS-4** KPI/队列 |
| **§7 Guaji** | — | **7.1–7.7 全量人工**（依赖第三方测试号；T4/T5 完成后补 7.6） |
