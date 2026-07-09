# 彩种目录（P0–P5）× 第三方挂机（T0–T6）统一闭环主计划

> **用途**：将 [`lottery-catalog-migration-plan.md`](lottery-catalog-migration-plan.md) 与 [`third-party-guaji-integration-plan.md`](third-party-guaji-integration-plan.md) 合并为**单一可实施权威视图**：阶段矩阵、依赖、关键产品决策、技术硬规则与阻塞、验收。  
> **更新**：2026-06-09  
> **说明**：Guaji 方案阶段为 **T0–T6**（含 **T1b** Client 子阶段）；无 T7–T9 定义。若口头称「T0–T9」，通常指 **P0–P5（6）+ T0–T3（4）** 或 **P+T 全量 14 个工作流**——本文以文档正式编号为准。

---

## 1. 两个方案的关系

| 维度 | P 轨道（彩种目录） | T 轨道（Guaji 对接） |
|------|-------------------|---------------------|
| **目标** | 47 彩种 + 340 子玩法统一目录、玩法树、维护态 | 真实资金/接单/开奖/派奖走 Hash 挂机平台 |
| **是否推翻对方** | — | **不推翻** P 目录；沿用 `outbound_*` 编码 |
| **资金** | 目录不含赔率（B6）；C8 定稿「第三方为准」 | 替换本地 `member_wallets` 扣款（real） |
| **币种** | — | **USDT/TRX/CNY** 三币种；会员中心切主币种；方案按主币种运行；切币种→全部暂停（guaji §4.4） |
| **开奖** | P5「等待开奖」UI（C17） | T3 WS 订阅写入 `lottery_draws` |
| **接单/派奖** | C8 下注成功 = 第三方接单成功；C17 开奖延迟/缺失**一直「等待开奖」、不超时改文案** | T4 接单、T5 派奖**完全以第三方为准** |
| **大厅** | A3：首页**不平铺** 47 彩种；`GET /public/lotteries` 仍供跟单/自创方案/会员筛选 | — |
| **验收** | `test_p0_p5_smoke.py` + schemes coverage | `integration-checklist.md` §7 |

**完全闭环定义**：会员在 **已启用 Guaji 授权** 前提下，任选 **47 彩种之一** → 加载 **play-tree** → **手动/Worker 真实下注** → 第三方 **接单成功** → **WS/同步派奖** → **投注记录/流水/顶栏 CNY** 一致；Admin 可 **维护彩种对接码**；旧彩种/维护态 **路由拦截** 正常。

---

## 2. 统一阶段矩阵

| 阶段 | 名称 | 状态 | 自动化验收 | 阻塞下一项 |
|------|------|------|------------|------------|
| **P0** | CSV + migration 草案 | ✅ 完成 | `generate_p0_seeds.py` | — |
| **P1** | 47+340 seed、purge、公开/Admin API | ✅ 完成 | smoke P1 项 | — |
| **P2** | `ssc_std` 投注链 + `panel_type` + 去粘贴 | ✅ 后端；🟡 Client 手测 | `TestSSCSubPlayCoverage` | T4 映射输入 |
| **P3** | `lhc_std` 面板 + 结算 | ✅ 后端；🟡 Client 手测 | `TestLHCSubPlayCoverage` | T4 |
| **P4** | syxw/pk10/k3/pc28 全模板 | ✅ 后端；🟡 Client 手测 | `TestP4SubPlayCoverage` | T4 |
| **P5** | 维护态、legacy 跳转、等待开奖 | ✅ 后端+Client；Admin 编辑需 env | `TestP5*` + smoke | T2 运营改码 |
| **T0** | Guaji HTTP/WS 适配 + health | ✅ 完成 | `make guaji-smoke` | T1 |
| **T1** | 授权表、绑/解/切换、批量 pause、门禁 | ✅ 后端 | `go test ./internal/guaji/accountsvc/...` | T1b |
| **T1b** | 授权两页、门禁、主币种、账目改造、Admin 只读 Tab、下线充提/团队 | ✅ 前后端 | checklist §7.1–7.5 | T4 前必须 |
| **T2** | `outbound_*` ↔ game_id/rule_id（解析器 + web_bets/lott 契约） | ✅ 后端 | `TestResolveOutbound*` | T4 |
| **T3** | WS 开奖订阅（`/ws`，实测连通）+ 多彩种线解析 + 福彩过滤 + 入库 + WS-5 | ✅ 后端（真实样例已测；`outbound↔logXXX` 待运营配置） | `TestParseDrawEvents*` | T5 |
| **T4** | `web_bets/lott` 真实下单 + bet_orders 扩展 + dock 余额 | ✅ 后端+Client（待第三方测试号验证） | `TestGuajiRealEnabledToggle` | T5 |
| **T5** | 第三方派奖同步 worker + ledger 镜像（B1 已定）+ real 订单不本地结算 | ✅ 后端（`PayoutSyncWorker` 已装配；待真实注单验证）| 全量 `go test` | T6 |
| **T6** | 监控、Runbook、§7 全量验收 | ✅ 后端（`/admin/guaji/health` + Token 巡检 + Runbook §25） | health/admin 端点 | 上线 |

---

## 3. 依赖关系（实施顺序）

```
P0 → P1 ─┬→ P2 ─┐
         ├→ P3 ─┼→ [本地投注链验收] ─→ T2 ─→ T4 ─→ T5 → T6
         ├→ P4 ─┘         ↑              ↑
         └→ P5 ────────────┘              │
                                          │
T0 → T1 → T1b ────────────────────────────┘
              T0 ─→ T3 ───────────────────→ T5
```

| 依赖 | 说明 |
|------|------|
| P1 → 全部 | 玩法树、`sale_status`、`outbound_*` 字段 |
| P2–P4 → T4 | `BetPayload` + eval 规则 → 第三方下单体 |
| P5 → T2 | 维护态改 `outbound_lottery_code`（C46） |
| T1 → T1b/T4 | 无启用授权 Token 无法调第三方 |
| T4 → T5 | 接单成功后才有派奖同步 |
| **P 不等待 T** | 目录/本地 sim 链可先行；**real 上线前必须 T4+T5** |

---

## 4. 分域交付清单

### 4.1 数据层

| 表/字段 | P 阶段 | T 阶段 |
|---------|--------|--------|
| `lottery_catalog` + `play_types` + `sub_plays` | P1 seed | 只读 |
| `bet_orders.outbound_*` | P1/P5 C41/C43 | T4 扩展 `guaji_account_id`、`third_party_bet_id`、`currency`(主币种快照) |
| `member_guaji_accounts` | — | T1 |
| `members.primary_currency` | — | **T1b**（USDT/TRX/CNY，默认 CNY） |
| `wallet_ledger.guaji_account_id` | — | T5 |

### 4.2 API

| 路径 | P | T |
|------|---|---|
| `GET /public/lotteries` | P1 | — |
| `GET /public/lotteries/{code}/play-tree` | P1 | — |
| `PATCH /admin/games/lottery-catalog/{code}` | P5 | T2 改对接码 |
| `POST /client/games/{code}/bets` | P2–P4 本地 | **T4** real 走 Guaji（runMode=real）；sim/降级本地 |
| `POST /client/guaji/accounts/bind` 等 | — | T1 |
| `GET /client/guaji/balance` | — | T1（按主币种） |
| `GET/PUT /client/guaji/primary-currency` | — | **T1b**（切换→批量 pause） |

### 4.3 Client

| 能力 | P | T |
|------|---|---|
| `useLotteryRouteGuard` | P5 ✅ | — |
| `fetchPlayTree` + `panel_type` 面板 | P2–P4 ✅ | — |
| 「等待开奖」 | P5 ✅ | T3 换数据源 |
| `/member/auth/bind`、`/member/auth/list` | — | T1b |
| `useGuajiAuthGuard` 全局门禁 | — | T1b |
| 顶栏/游戏详情 **主币种实账** | — | T1b |
| **切换主币种弹窗 + 全部暂停** | — | T1b |
| Admin 会员详情 **授权只读 Tab** | — | T1（API 已建）/ T1b（UI） |

### 4.4 授权、门禁与第三方环境（T 轨道硬规则）

| 规则 | 口径 | 出处 |
|------|------|------|
| **`hasActiveGuajiAuth` 判定** | `is_active = true` **AND** `token_expires_at > now()` **AND** `last_token_error IS NULL` | guaji §16.6 |
| **全局门禁白名单** | 无启用授权时**仅** `/member/auth/bind`、`/member/auth/list` 放行；**FAQ/帮助/聊天/公告/sim/大厅/方案页 一律拦截** | guaji §3.4 · §22.6 |
| **单启用** | 每 `member_id` 至多 1 条 `is_active`（partial unique index）；切换 A→false、B→true | guaji §3.2 |
| **全局独占** | 同一 `guaji_username` 同时只绑 1 平台账号；解绑后任何人可再绑 | guaji §3.2 |
| **切换/解绑** | 全部 `running`+`pending` → `paused`（单事务）；平台 logout **不停**方案 | guaji §16.1 · §16.6 |
| **主币种** | **USDT/TRX/CNY**，`members.primary_currency` 默认 CNY；会员中心切换；方案按主币种运行；**切换主币种 → 弹窗 + 全部 `running`+`pending` → `paused`**（同切换授权） | guaji §4.4 · §6.2 |
| **重新授权** | 全自动 MFA；连续失败 **3 次** → 引导绑定页重填密码 | guaji §19.1 · §20.3 |
| **首绑/重授成功跳转** | → **会员中心** | guaji §19.6 |
| **第三方环境** | **部署配置**决定：测试 `hash.iyes.dev` / 正式 `s9-xia.5rf9q.com`；**会员不可选** | guaji §20.4 |
| **Admin 授权只读 Tab 字段** | 用户名、是否启用、绑定时间、最近同步、最后投注、最近 Token 失效原因；**无代绑/解绑** | guaji §19.5 · §11.1 |

### 4.5 下线与账目页改造（T1b 清理）

**前后台同步下线**（Guaji 上线后本平台不再承载资金出入与代理体系）：

| 域 | Client | Admin |
|----|--------|-------|
| **充提** | 充值/提现入口与 API（`funds/recharge` 等） | 提现审批、出款、充值渠道菜单与 API、Dashboard 充值 KPI、提现队列 WS |
| **团队** | 团队 Tab、`scope=team` 切换、`team/*` API | 代理/团队相关运营面 |
| **银行卡** | 收款账户绑定/管理（`payout-accounts`） | 银行卡审核 |
| **开户/推广** | 下级开户、推广链接 | 代理拉新、推广渠道 |
| **帐变/追号** | `/member/ledger`、`/member/chase-records`（404/重定向，菜单移除） | — |

**会员中心账目四入口（保留/改造）**：

| 入口 | 路由 | 口径 |
|------|------|------|
| 投注记录 | `/member/bet-records`（沿用） | 默认当前启用授权；可切「全部历史」 |
| 方案盈亏 | `/member/scheme-pnl`（**新建**） | 复用 cloud bet-records summary + `guaji_account_id` 筛选 |
| 盈亏报表 | `/member/pnl-report`（保留） | **仅个人** scope，无 team 切换 |
| 钱包流水 | `/member/fund-records` | 仅 `bet_debit`/`payout` 镜像；不展示 Demo 充提 |

> 余额大卡：**CNY 实账** +「充值/提现请前往第三方平台」+ 客服（guaji §22.4）。

---

## 5. 技术阻塞与硬约束（实施前必须拍板）

| # | 阻塞 | 决策选项 | 出处 |
|---|------|----------|------|
| **B1** | **`wallet_ledger` 余额约束失效** | **已定（方案②）**：保留 `balance_after >= 0 NOT NULL`；real 镜像行 `balance_after` 存**第三方主币种余额快照**，新增 `guaji_account_id`/`currency` 列（00081）；本地/sim 路径不变 | guaji §16.3（已决） |
| **B2** | **real 扣款路径切换** | 现网 `worker_wallet.go` 在事务内扣 `member_wallets`；T4 改为第三方 `web_bets/lott` 接单成功后再镜像 ledger，**逐步废弃 real 路径对本地 `member_wallets` 的写** | guaji §16.3 · C8 |
| **B3** | **余额不足判定** | real 下单前查 `users/i/info` CNY；sim **不校验**第三方余额、不因余额 paused | guaji §18.5 |
| **B4** | **`bet_orders` 扩展** | T4 前加 `guaji_account_id`、`third_party_bet_id`、`currency`，与既有 `outbound_*` 快照并存 | guaji §16.2 |
| **B5** | **凭证加密** | 密码/MFA/Token 用 `GUAJI_CREDENTIALS_KEY`（32B）AES-GCM；轮换 Runbook 入 T6 | guaji §16.6-T5 |
| **B6** | **术语对齐** | 方案写 `bet_payout`，库表实为 `payout`（`00005_wallet_ledger.sql`）；**统一用 `payout`** | guaji §16.2 |

---

## 6. 闭环验收（分阶段）

### 6.1 P 轨道（可独立验收）

```bash
cd backend
python docs/seeds/generate_p0_seeds.py
go test ./internal/games/... ./internal/schemes/... -count=1
# 后端运行于 :8081
PORT=8081 python scripts/test_p0_p5_smoke.py
```

### 6.2 T 轨道（依赖第三方测试号）

```bash
cd backend
make guaji-smoke          # T0
go test ./internal/guaji/... -count=1   # T0 + T1
# 人工：integration-checklist.md §7
```

### 6.3 完全闭环（P + T 联合）

1. Admin：`VITE_LOTTERY_CATALOG_P5=true`，维护态改某彩种 `outbound_lottery_code` 为真实第三方码  
2. Client：绑定 Guaji → 启用授权 → 进入 `tron_ffc_1m` 游戏详情  
3. 手动下注 real → 第三方接单 → `bet_orders` 有快照  
4. 开奖 WS/同步 → 派奖 → 顶栏 CNY 刷新  
5. 切换授权 → 全部 `running`/`pending` → `paused`  

---

## 7. 当前最大断层（2026-06-09 复核）

> T0–T6 后端 + Client 代码均已落地并通过 `go build` / `go test` / 三端 `npm run build`。
> 剩余项均为**依赖第三方测试号的联调适配**或**非阻塞收尾**，无未落地的本平台核心代码。

| # | 断层 | 影响 | 状态 / 下一动作 |
|---|------|------|----------|
| 1 | **真实下单 `40060`** | real 下单被第三方拦（双账号复现，未扣款） | ⛔ **第三方侧**：需其开通挂机账号下单权限/确认 40060 机制（见 §26.5b） |
| 2 | **T4 Worker real 接单** | 挂机方案 Worker 仍本地模拟结算 | 范式重构（事前接单），依赖 #1 放行；手动下注已通、T5 已能处理真实注单 |
| 3 | **真实协议字段适配** | `web_bets/lott` 的 `content`/`rule_id` 数值、`QuerySettlement` 字段 | 拿可下单账号后按 Runbook §25 + checklist §7 核对 |
| 4 | `outbound_lottery_code` ↔ `lottery_logXXX` | 开奖入库需运营配置彩种线键 | 运营维护态配置（参考 guaji §26.4） |
| 5 | P2–P4 无 E2E 下注脚本 | 340 子玩法 UI 覆盖靠手测 | 可补 `scripts/test_catalog_bet_smoke.py` |
| — | ~~T5 派奖同步 worker~~ | — | ✅ `PayoutSyncWorker` 已实现装配 |
| — | ~~OpenAPI guaji 路径~~ | — | ✅ 已补全 11 条，YAML 校验通过 |

---

## 8. 文档索引

| 文档 | 内容 |
|------|------|
| [`lottery-catalog-migration-plan.md`](lottery-catalog-migration-plan.md) | P0–P5 产品决策 C1–C48 |
| [`third-party-guaji-integration-plan.md`](third-party-guaji-integration-plan.md) | T0–T6 + §16 技术审查 |
| [`integration-checklist.md`](integration-checklist.md) | Phase 0–5 + §7 Guaji |
| [`integration-plan.md`](integration-plan.md) | 三端 Phase 0–5 历史 |
| **本文** | 阶段矩阵 + 依赖 + 授权/门禁/环境硬规则 + 技术阻塞 + 闭环验收 |

---

## 9. 修订记录

| 日期 | 变更 |
|------|------|
| 2026-06-08 | 初版：合并 P/T 方案；P0/P1/T0 标完成；T1 启动实施 |
| 2026-06-09 | 升级为内容级整合：补 §4.4 授权/门禁/环境硬规则、§4.5 下线与账目页清单、§5 技术阻塞（wallet_ledger 约束等）、§1 补 C8/C17/A3 |
| 2026-06-09 | **多币种补充**：USDT/TRX/CNY 主币种（`members.primary_currency`）；会员中心切主币种；方案按主币种运行；切换主币种 → 全部暂停（同切换授权） |
| 2026-06-09 | **T1b 收尾**：会员中心顶栏主币种实账 + 切换；下线 ledger/chase 路由；盈亏报表/彩种统计移除 team scope；Admin 授权只读 Tab；新建方案盈亏页 |
| 2026-06-09 | **T2 落地**：`games.ResolveOutbound`（彩种+玩法 → game_id/rule_id）；`guaji.PlaceLottBet` web_bets/lott 请求契约（T4 调用）；集成测试通过 |
| 2026-06-09 | **T4 落地**：`bet_orders` 扩展 `guaji_account_id`/`third_party_bet_id`/`currency`（00080）；`games.GuajiBetPlacer` 网关 + `accountsvc.PlaceRealBet`（取授权 Token + 主币种余额校验 + web_bets/lott）；real 不扣本地钱包，sim/降级走本地；Client 手动下注 runMode=real + dock 主币种实账。**待第三方测试号端到端验证** |
| 2026-06-09 | **T3 落地**：`guaji.SubscribeDraws` 匿名开奖 WS + `ParseDrawEvent` 容错解析 + `IsIgnoredDrawGame` 福彩/排列过滤；`drawsync.Worker` 按 `outbound_lottery_code` 反查入库 `lottery_draws` + WS-5 广播 + 退避重连。**真实消息协议待测试号核对** |
| 2026-06-09 | **T5 落地**：B1 定为方案②（00081 `wallet_ledger` 加 `guaji_account_id`/`currency`，real 镜像存第三方余额快照）；`member.MirrorRealLedger` 镜像 bet_debit/payout（不动本地钱包）；结算 worker **跳过 real 第三方订单**（C8，由派奖同步处理）；`guaji.QuerySettlement` 注单查询契约。**派奖同步 worker + 真实字段待测试号** |
| 2026-06-09 | **T6 落地**：`accountsvc.Health` + `GET /admin/guaji/health` 授权健康指标；`RunTokenMonitor` 每 5 分钟巡检临期/失效 Token 告警；Runbook §25（凭证轮换/故障处置/环境切换）+ checklist §7.8 |
| 2026-06-09 | **真实协议对齐**（接口文档 `_tmp_guaji_parsed.md`）：测试账号 testcq01~10 入 `.env.example`；**币种数字编码 0=usdt/1=trx/3=cny**（`CurrencyCode`）；`web_bets/lott` 改真实体 `bet_contents[]`+`game_id`(数字)+`currency`(数字)+`bet_multiple[]` |
| 2026-06-09 | **实测抓包对齐**（`cmd/guaji-capture` + testcq01）：登录 `success:true`、users/i/info 裸对象、agents/rate `code=0` → `parseEnvelope` 兼容 success/0/200/201 + `UserInfo` 裸对象解析（**登录/余额实测打通**，CNY=100000）；**WS 端点 `/ws`**（`GUAJI_WS_PATH`）实测 `wsReachable:true`；开奖 `ParseDrawEvents` 重写为「一条消息→多彩种线」+ `DrawBalls.BallsFor(template)`（实测样例测试通过，含 `lhc_num`）；`drawsync` 按 `lottery_logXXX` 键反查彩种+模板选号。详见 guaji 方案 §26 |
| 2026-06-09 | **彩种线实测汇总**：抓 40s 得真实键 `lottery_log033/05/101/103/115/125`（波场，5玩法共享）、`eth_lottery_log`、`tw_lottery_log`；**一键多彩种** → `drawsync.resolveLotteries` 改返回全部匹配彩种各按模板入库；运营配置参考表入 guaji §26.4 |
| 2026-06-09 | **真实下单联调**：`web_bets/lott` 请求体格式被第三方接受；卡 `40060「用户没有设置密保」`（testcq01/02 双账号复现，已设密保仍报，`wp_password`/`security_code` 均无效，**未扣款**）→ 第三方接口/账号开通统一问题；登记 `CodeSecurityRequired=40060` + guaji §26.5b。**下单代码就绪，待第三方放行** |
| 2026-06-09 | **整体回归（除真实下单）全绿**：`go test ./internal/...` 全 PASS；client/admin `npm run build` PASS；guaji 只读 smoke（http/ws/login/balance）全 true；`test_p0_p5_smoke.py`（P0–P5 + T1 guaji 路由）**all checks passed**（8081 最新二进制） |
| 2026-06-09 | **T5 派奖同步 worker**：`accountsvc.PayoutSyncWorker` 扫 real pending 注单（`guaji_account_id` 非空）→ `QuerySettlement` → `SettleBetOrder` + `MirrorRealLedger`（第三方余额快照）+ `PublishWallet` 余额刷新；server 装配（GUAJI_ENABLED 时）。**T4 Worker**：标注为本地模拟，真实接单需「事前接单」范式重构 + 待 40060 放行（worker.go 注记）。**OpenAPI** 补全 guaji 11 条路径（auth-status/accounts/bind/activate/reauth/delete/balance/primary-currency + admin guaji-accounts/health），YAML 校验通过 |
