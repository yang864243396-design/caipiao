# 第三方挂机平台对接计划（Hash / iyes.dev）

> **依据**：《挂机协助接口短信邮件版.doc》、[`lottery-catalog-migration-plan.md`](lottery-catalog-migration-plan.md)（P0–P5）、[`unified-closure-masterplan.md`](unified-closure-masterplan.md)、[`integration-plan.md`](integration-plan.md)、[`client/DESIGN.md`](../client/DESIGN.md)  
> **状态**：**产品定稿 v15**（§15–§24 全部闭合；§16 技术审查供 T 阶段实施）  
> **更新**：2026-06-08

---

## 1. 目标与边界

### 1.1 目标

本平台 = **会员前台 + 方案引擎 + 运营后台**；**Hash 挂机平台** = **真实资金与接单系统**。不推翻 P0–P5 彩种目录。

**术语**：

| 用语 | 含义 |
|------|------|
| **平台账号** | 本平台登录身份（`members`） |
| **授权账号** | 第三方 Hash 挂机账号 |
| **启用中的授权账号** | 同一平台账号下 **同一时刻仅 1 个** `is_active=true` |

| 能力 | 第三方 | 本平台 |
|------|--------|--------|
| 授权鉴权 | 登录/MFA/Token | 自助绑定、切换、服务端续期、自动 MFA |
| 实际余额 | `users/i/info` | 会员中心顶栏展示（§4.6） |
| 平台记账 | — | 投注记录、方案盈亏、钱包流水（§4.5） |
| 赔率/开奖/接单/派奖 | 是 | 适配、镜像、同步（C8/C17） |
| 方案 | — | 绑平台账号；Worker 用启用授权 Token（§6） |
| sim | — | 保留；不调第三方；**亦须启用授权**（§3.4） |

### 1.2 不做

- 运营对照表 Admin（C7）；本地赔率（B6）；旧 9 彩种；大厅不平铺 47 彩种。
- 前台撤单；**本平台充值/提现**。
- Client **帐变记录**、**追号记录**页面（§22.3）。
- **Admin / Client 充提** 菜单与 API（§23.1 **甲**）；Dashboard 充值 KPI 等一并调整。
- **团队 / 银行卡 / 开户**（含推广链接、scope=team）前后台 **同步下线**（§24）。
- WS 忽略：福彩 3D、福彩排列 3D、排列 2/3。
- **独立「退出授权」入口**（不单独提供；靠切换/解绑，§2.3）。

### 1.3 环境

| 环境 | HTTP | WebSocket |
|------|------|-----------|
| 测试 | `https://hash.iyes.dev/` | `wss://hash.iyes.dev/?token={token}` |
| 正式 | `https://s9-xia.5rf9q.com/` | `wss://s9-ws.5rf9q.com/?token={token}` |

匿名开奖：`token=Anonymous`。登录：`is_ai: true`。

**环境切换（§20.4 甲）**：由 **部署配置** 决定对接测试或正式域名；**会员不可选**；测试站 ↔ `hash.iyes.dev`，正式站 ↔ `s9-xia.5rf9q.com`。

---

## 2. 已确认产品决策（全量汇总）

### 2.1 环境与基础

1:N 绑定 · 单启用 · 授权账号全局独占 · **部署决定第三方环境**（测试 `hash.iyes.dev` / 正式 `s9-xia.5rf9q.com`；会员不可选，§20.4 **甲**）· 自动 MFA · WS 过滤福彩/排列系列 · 前台不撤单。

### 2.2 资金

| 项 | 结论 |
|----|------|
| 实账 | 第三方 `users/i/info` |
| 平台账 | 仅记录本平台触发的业务 |
| 充提 | **Client + Admin 均删除** 充提入口与业务 API（§17.5 · §23.1 **甲**） |
| 会员中心账目 | **投注记录** + **方案盈亏** + **钱包流水** + **盈亏报表**（四入口，§22.2 **丙**） |
| 投注记录路由 | **沿用** `/member/bet-records`（§22.1 **甲**） |
| 方案盈亏 | **新建** `/member/scheme-pnl`（云端方案近 N 日，对齐 `/bet-records` 逻辑） |
| 盈亏报表 | **保留** `/member/pnl-report`（**仅个人** scope，§24） |
| 下线页面 | **`/member/ledger`** 帐变、**`/member/chase-records`** 追号（§22.3） |
| 余额大卡 | CNY 实账 + **「充值/提现请前往第三方平台」** + 客服（§22.4 **丙**） |
| 团队 | **下线** 团队 Tab、scope=team 切换及团队 API（§24） |
| 银行卡 | **下线** 收款账户绑定/管理（Client + Admin，§24） |
| 开户 | **下线** 下级开户、推广链接等代理拉新能力（Client + Admin，§24） |
| 门禁白名单 | **仅** bind/list；FAQ/帮助/聊天/公告 **一律拦截**（§22.6 **甲**） |
| Demo 充提流水 | **前后台均不展示**；库表历史行可保留，无查询入口（§22.7 + §23.1 **甲**） |
| 游戏详情余额 | **real** 手动下注 dock 展示 **第三方 CNY**（与顶栏同源，§23.2 **甲**） |
| 账目筛选 | 投注记录 / 钱包流水 / **方案盈亏** 均 **默认当前启用授权**；可切「全部历史」（§18.4 **乙** + §20.2 **甲**） |
| 手动余额不足 Toast | **「可用余额不足，请前往第三方平台充值」**（§18.6 **甲**） |

### 2.3 授权与导航（v5 强化）

| 项 | 结论 |
|----|------|
| 绑号 | 会员自助；首绑即用 |
| **全局门禁** | **无启用中授权时**，**仅** bind/list（§22.6 **甲**）；FAQ/帮助/聊天/公告 **不可访问** |
| 切换入口 | **仅会员中心顶栏** → 跳转 **授权列表页（合并切换）** |
| 切换/解绑 | 全部 `running` + `pending` 实例 → **`paused`（已暂停）**（§16.1） |
| 退出平台登录 | **不停止** 方案 |
| 退出授权 | **不单独提供**；用 **切换** 或 **解绑** 达到停用效果 |
| 信息架构 | **绑定独立页**；**列表 + 切换合并** 为一页（§15.4 **丙**） |
| 展示名 | **仅第三方用户名**（§17.1 **甲**） |
| 切换/解绑后跳转 | **留在授权列表页** + Toast 提示去云端中心开启（§17.3 **甲**） |
| 追加绑定 | **可以**；绑成功后 **弹窗询问** 是否立即切换启用（§17.4 **乙**） |
| Token 失效 | Toast + **全局门禁**；**全部 running/pending → paused**（§17.2 **乙**） |
| Token 恢复 | 列表页 **「重新授权」** 刷新 Token（§18.2 **乙**） |
| 切换暂停文案 | 卡片统一 **「已暂停」**（不区分原因；§18.1 **甲**） |
| UI 规范 | 授权页 **遵循 [`client/DESIGN.md`](../client/DESIGN.md)**（数字精算主义） |

### 2.4 投注与方案

| 项 | 结论 |
|----|------|
| 出钱账号 | 启用中授权 |
| 方案归属 | 仅平台账号 |
| sim / real | **均须启用中授权** 才可进入非授权页；sim 不调第三方 |
| 授权变更后方案 | 变为 **`paused`**；可 **逐个手动启动** 或 **云端中心「一键开启方案」**（§6.5；沿用现有逻辑） |
| 切换后手动下注 | **允许**（§15.5 **甲**） |
| 解绑唯一启用账号 | **立即门禁**；方案保持 `paused`，重新绑号后再启（§15.3 **甲**） |
| 账号被他人占用 | 提示先让对方解绑；无排队（§16.6 **甲**） |
| 余额不足 | **仅 real**：手动 Toast；Worker paused + **「钱包余额不足」**（§17.6 **乙**） |
| sim 余额 | sim **不校验** 第三方余额；**不因余额不足 paused**（§18.5 **甲**） |
| 切换暂停展示 | 卡片 **「已暂停」**（§18.1 **甲**；`status_reason=manual`，不新增枚举） |

### 2.6 绑定、MFA 与 Admin（v9）

| 项 | 结论 |
|----|------|
| 绑定表单 | **第三方用户名 + 密码**；MFA 后续自动引导（§19.2 **甲**） |
| 重新授权 | **全自动 MFA**（服务端已存材料）；失败再提示走绑定页（§19.1 **甲**） |
| 解绑确认 | **标准确认**弹窗（§19.3 **甲**） |
| 顶栏余额 | **固定展示 CNY**（§20.1）；取自 `users/i/info` 的 CNY 项 |
| 重新授权降级 | 全自动 MFA **连续失败 3 次** 后要求走绑定页重填密码（§20.3 **乙**） |
| 绑号/重授权成功 | **跳转会员中心**（§19.6 **甲**） |
| Admin 授权只读 | 用户名、是否启用、最近同步、**绑定时间、最后投注、最近 Token 失效原因**（§19.5 **乙**）；**无代绑** |

### 2.5 余额刷新（v5）

顶栏余额（会员中心）刷新时机：

1. **进入会员中心时** 拉取 `users/i/info`
2. 会员点 **「刷新」** 手动拉取
3. **发生投注、派奖** 后自动刷新（本平台下单成功 / 结算同步触发）

---

## 3. 授权账号与全局门禁

### 3.1 状态机（v5）

```
平台登录成功
  ├─ 0 个授权绑定 ──▶ 落地：授权绑定页（§16.3 甲）
  └─ ≥1 个绑定
        ├─ 无启用中 ──▶ 落地：授权列表页（空态/选启用；§16.3 乙）
        └─ 有启用中 ──▶ 全站可访问

切换授权 ──▶ 全 running+pending → paused ──▶ 新账号启用 ──▶ 全站可访问（可立即手动下注）
解绑（含启用中）──▶ 全 running+pending → paused ──▶ 若仍有绑定则回「无启用」门禁；0 绑定回绑定页

平台退出登录 ──▶ 方案不停止（Worker 继续）
```

### 3.2 绑定规则

| 规则 | 说明 |
|------|------|
| 首绑即用 | 绑定成功 → 自动 `is_active=true` → **跳转会员中心**（§19.6 **甲**） |
| 单启用 | 切换时 A→false、B→true；**切换前确认停方案** |
| 追加绑定 | 已有启用 A 时仍可绑 B；绑 B 成功后 **弹窗询问是否立即切换**（切换则停方案） |
| 解绑 | **标准确认**：「确定解绑该授权账号？解绑后将停止全部挂机方案。」（§19.3 **甲**） |
| 独占 | 同一 `guaji_username` 同时只绑 1 平台账号；解绑后 **任何人可再绑** |
| 无「退出授权」 | 若要停用：切换到另一账号，或 **解绑当前启用账号** |

### 3.3 授权相关 Client 页面（须符合 DESIGN.md）

| 页面 | 路由（规划） | 职责 |
|------|--------------|------|
| **授权绑定页** | `/member/auth/bind` | **用户名 + 密码**；完成首次 MFA；0 绑定时登录落地页；成功 → **会员中心** |
| **授权列表页（含切换）** | `/member/auth/list` | 已绑账号列表；**「设为启用」** / **「重新授权」** / 解绑；顶栏「切换授权账号」跳此页 |

**列表页交互（§16.2 甲）**：单页列表 — 展示 **第三方用户名**、是否当前启用；非启用行点 **「设为启用」** → 确认弹窗「将停止全部挂机方案」→ 执行切换 → **留在列表页** + Toast「方案已暂停，请到云端中心开启」（§17.3 **甲**）。列表页提供 **「添加授权账号」** 入口跳转绑定页。

规范：Element Plus 组件、色阶分层、无 1px 硬分割线、Primary `#0050cb`、容器内边距 ≥ 1rem 等，见 `client/DESIGN.md` §2–§8。

### 3.4 全局路由守卫（v5 硬规则）

```
hasActiveGuajiAuth(member) ?
  YES → 正常路由
  NO  → 仅允许路径：
         /member/auth/bind
         /member/auth/list
        （及登录/登出；精确路径实施时可微调，语义不变）
```

- **sim、real、手动下注、大厅、方案页、云端中心、会员中心子页（含 FAQ/聊天）** — 全部依赖 `hasActiveGuajiAuth`。
- 无启用授权时访问其他 URL → **重定向** 至授权列表页，Toast「请先启用授权账号」。
- **0 绑定** 时重定向至 **绑定页**。

### 3.5 会员中心顶栏

```
授权账号：testcq01  │  CNY ¥1,234.56  │  [刷新]  [切换授权账号]
```

- 展示 **第三方用户名**（§17.1 **甲**）；余额 **固定 CNY**（§20.1）；切换 **仅** 此入口；余额刷新见 §2.5。

### 3.6 自动化 MFA（§19.1 **甲** · §19.2 **甲**）

- **首次绑定**：会员填用户名 + 密码；MFA 在绑定流程中 **自动引导/处理**；材料 **加密存储**。
- **续期 / 重新授权**：服务端 **全自动** 用已存材料向第三方重登；会员 **无感**。
- **自动失败**：服务端可间隔重试；**连续失败 3 次** 后 Toast 提示 → 引导 **绑定页** 重填密码（§20.3 **乙**）。

### 3.7 Token 失效（§17.2 **乙**）

服务端续期失败或 Worker 无法以当前授权 Token 下单时：

1. Toast「授权已失效，请重新绑定」
2. **全局门禁**（同无启用授权）
3. **全部 `running` + `pending` → `paused`**（与切换停方案一致）
4. 会员在列表页点 **「重新授权」** → **全自动** 刷新 Token（§19.1 **甲**）；成功后 **跳转会员中心**（§19.6 **甲**）；方案恢复仍须 **手动/一键开启**

---

## 4. 余额与资金

### 4.1 双轨

- **实账**：第三方，顶栏展示。
- **行为账**：本平台镜像，供三页查询。

### 4.2 删除充提与页面下线（§17.5 · §22）

- 移除 Client **充值、提现** 入口；**Admin 同步下线** 充提审批、充值渠道、相关 KPI/WS（§23.1 **甲**）。
- **`/member/fund-records`** → **仅钱包流水**（`bet_debit` / `payout` 镜像）。
- **下线**：`/member/ledger`（帐变）、`/member/chase-records`（追号）；会员中心菜单移除对应入口。
- **余额大卡**（§22.4 **丙**）：展示 **CNY 实账** + 提示「充值/提现请前往第三方平台」+ **客服**；去掉充值/提现按钮。
- **下线团队 / 银行卡 / 开户**（§24）：会员中心无团队 Tab；无收款账户入口；无下级开户/推广；彩种统计、盈亏报表等 **仅个人** scope。

- 余额不足（**real**）：手动 Toast **「可用余额不足，请前往第三方平台充值」**；Worker 卡片 **「钱包余额不足」**（§17.6 **乙** + §18.6 **甲**）。

### 4.3 会员中心账目页（§22）

| 页 | 路由 | 数据 / 职责 |
|----|------|-------------|
| **钱包流水** | `/member/fund-records` | `wallet_ledger` 镜像（`bet_debit` / `payout`） |
| **投注记录** | **`/member/bet-records`**（沿用现网，§22.1 **甲**） | `bet_orders` + guaji 筛选 |
| **方案盈亏** | **`/member/scheme-pnl`**（新建，§22.2 **丙**） | 按 **方案实例** 云端盈亏（近 N 日；逻辑对齐 **`/bet-records`**） |
| **盈亏报表** | **`/member/pnl-report`**（保留，§22.2 **丙**） | **个人** 汇总报表（**无 team scope**，§24） |

**云端中心** **`/bet-records`**（近三日投注明细）**保留**，与会员中心「投注记录」职责分开（§22.1 **甲** 丙义）。

**筛选（§18.4 · §20.2）**：投注记录、钱包流水、**方案盈亏** 默认 **当前启用授权**；可切「全部历史」。盈亏报表、彩种统计 **仅个人** 维度（§24）。

**下线**：`/member/ledger`、`/member/chase-records`（§22.3）；团队 / 银行卡 / 开户相关入口与 API（§24）。

### 4.4 币种与主币种（v16 · 多币种，2026-06-09 补充）

> **变更**：原「顶栏固定 CNY」升级为 **三币种 + 会员可切主币种**。

- **支持币种**：**USDT / TRX / CNY** 三种。
- **主币种**：会员在 **会员中心** 切换；持久化 `members.primary_currency`（默认 **CNY**）。
- **顶栏余额**：展示 **主币种** 实账，取自 `users/i/info` 对应币种字段；格式 `{符号} {金额}`（CNY=`¥`、USDT=`USDT`、TRX=`TRX`）。
- **方案运行**：real 下注扣款、余额校验、`bet_orders.currency` 快照、`wallet_ledger` 镜像 **均按主币种**。
- **切换主币种**（与切换授权同逻辑）：
  - **弹窗确认**：「切换主币种将停止全部挂机方案，确定继续？」
  - 执行后 **全部 `running` + `pending` → `paused`**（`status_reason=manual`；卡片「已暂停」）。
  - 切换后须 **会员手动 / 一键开启** 恢复（同 §6.5）。
- **字段映射**（实施时按第三方文档核对，`UserAccount`）：CNY→`balance_cny`、TRX→`balance_trx`、USDT→`balance`/`balance_fixed`。
- **无对应币种余额**：显示 `0.00`（沿用 §21.1 容错）。

### 4.5 钱包流水（已定：甲）

| 类型 | 时机 |
|------|------|
| `bet_debit` | 第三方接单成功、写入 `bet_orders` |
| `bet_payout` | 结算同步、注单派奖（库表 `txn_type` = **`payout`**） |

**不包含**：第三方充值/提现/人工调账。前后台 **均无** Demo 充提查询入口（§23.1 **甲**）。

### 4.6 余额刷新

进入会员中心 · 手动刷新 · 投注成功/派奖同步后自动刷新。

---

## 5. API / WS 摘要

- 核心：`auth/login`、`users/i/info`、`agents/i/real/rate`、`web_bets/lott`、`web_bets/`。
- 开奖：匿名 WS + REST；忽略福彩 3D、福彩排列 3D、排列 2/3。
- 最后 3 秒禁投；不对前台开放 cancel。

---

## 6. 挂机方案

### 6.1 模型

- 实例只存 `member_id`；运行取 `is_active` 授权 + 服务端 Token。
- `bet_orders` 快照 `guaji_account_id`。

### 6.2 停止规则

| 操作 | 方案 |
|------|------|
| 切换授权 | 全部 `running` + `pending` → **`paused`**（`status_reason=manual`；卡片 **「已暂停」**） |
| 解绑授权 | 同上 |
| **切换主币种** | **同切换授权**：弹窗确认 → 全部 `running` + `pending` → **`paused`**（§4.4） |
| 退出平台 | **不停止** |
| 运营强停 | **`soft_stopped`**（不变；一键开启 **不** 覆盖） |
| 余额不足（real） | Worker → **`paused`** + `insufficient_funds`；卡片 **「钱包余额不足」** |
| 余额不足（sim） | **不暂停**（§18.5 **甲**） |
| Token 失效 | 全部 running/pending → **`paused`**（§3.7） |

### 6.3 平台退出 ≠ 解绑/切换

退出平台仅失效 Client JWT；**启用授权与服务端 Token 保持**；Worker 继续。

### 6.4 sim 模式

| | real | sim |
|---|------|-----|
| 启用授权 | 必须 | **必须**（方可进非授权页） |
| 第三方 API | 调用 | **不调用** |
| 余额校验 | 第三方实账；不足 → paused + Toast | **不校验**；不因余额不足 paused（§18.5 **甲**） |
| 流水 | `bet_orders` + ledger | `cloud_bet_records` |

### 6.5 授权变更后的重启（v6）

- 状态：切换/解绑后实例为 **`paused`**，`status_reason=manual`；卡片文案 **「已暂停」**（§18.1 **甲**；与手动暂停相同，不区分原因）。
- 恢复方式（均须 **会员手动** 操作，切换/绑号后 **不自动** 重启）：
  - **手动**：云端中心卡片上「开启方案」/「继续运行」
  - **一键开启方案**：沿用 **云端中心现有按钮**（§15.1–15.2）；批量处理当前列表中全部 **`pending` + `paused`** 实例（与现网 `enableAllSchemes()` 一致）
- **`soft_stopped`（已封停）** 不受一键开启影响；须运营解封后再手动恢复。

---

## 7. 数据模型（规划）

- `member_guaji_accounts`：`guaji_username` UNIQUE、`is_active` 每 member 至多一条 true。
- `members.primary_currency`：主币种 **USDT / TRX / CNY**（默认 CNY；会员中心可切，切换停方案）。
- `bet_orders`：`guaji_account_id`、`third_party_bet_id`、`currency`（= 下单时主币种快照）。
- `wallet_ledger`：`bet_debit` / `bet_payout` + `guaji_account_id` 快照。
- 方案表 **不存** `guaji_account_id`。

---

## 8. 架构

```
Client 路由守卫(hasActiveGuajiAuth)
  ├─ 授权两页：绑定 + 列表（含切换/重新授权）
  └─ 业务页 → API → guaji 适配层 → Hash
       Worker（7×24，不依赖 Client 在线）
```

---

## 9. 实施阶段 T0–T6

| 阶段 | 交付 | 状态 |
|------|------|------|
| T0 | 环境、WS、测试号 | **已完成** |
| T1 | 授权表、绑/解/切换、首绑即用、**全局门禁**、切换停方案 | **已落地**（后端 API + 批量 pause + Client 授权页/门禁） |
| T1b | 授权两页 UI；顶栏主币种实账（USDT/TRX/CNY）；删充提；账目页改造（§4.3）；下线 ledger/chase；Admin 授权只读 Tab | **已落地**（dock 余额随 T4） |
| T2 | game_id + rule_id 映射 | **已落地**（`games.ResolveOutbound` + `guaji.PlaceLottBet` 契约） |
| T3 | WS 开奖 + 过滤 | **已落地**（`SubscribeDraws` + `drawsync.Worker` 入库/WS-5/过滤；真实协议待测试号核对） |
| T4 | 真实投注；Worker 服务端 Token | **手动下注已落地**（`PlaceRealBet` + bet_orders 扩展 + dock 余额）；**Worker real 路径 + 第三方测试号验证待续** |
| T5 | 结算同步 + ledger 镜像 + 余额刷新钩子 | **已落地骨架**（B1 方案② + `MirrorRealLedger` + 结算 worker 跳过 real + `QuerySettlement` 契约；派奖同步 worker 待测试号） |
| T6 | 监控、Runbook | **已落地**（`/admin/guaji/health` + Token 巡检告警 + Runbook §25 + checklist §7.8） |

### 9.1 验收要点

- [ ] 无启用授权时访问 `/games/*` → 重定向授权页。
- [ ] sim 方案在无启用授权时不可进入编辑/运行页。
- [ ] 切换/解绑 → `running`/`pending` 变 `paused`；平台 logout → running 保持。
- [ ] 云端中心「一键开启」可恢复 `pending`/`paused` 实例（含切换后暂停的）。
- [ ] 投注/派奖后会员中心余额更新。
- [ ] 授权两页视觉符合 DESIGN.md。
- [ ] Token 失效后列表「重新授权」可恢复门禁。
- [ ] 账目页默认当前授权筛选；可切全部历史。
- [ ] 首绑/重新授权成功 → 会员中心；重新授权全自动 MFA。
- [ ] Admin 授权 Tab 只读字段符合 §19.5。

---

## 10. 数据流（简图）

**真实投注**：启用授权 → `web_bets/lott` 成功 → `bet_orders` + `ledger(bet_debit)` → 结算 → `ledger(bet_payout)` → 触发余额刷新。

**切换**：确认 → 全 `running`/`pending` → `paused` → 改 `is_active` → `users/i/info` → 门禁恢复（可立即手动下注；挂机须另启）。

---

## 11. 改造摘要

| 域 | 内容 |
|----|------|
| 后端 | `guaji/*`；`hasActiveGuajiAuth` 中间件；授权 API；门禁；ledger 类型 |
| Client | 授权两页 + 门禁；顶栏/游戏详情 **CNY**；删充提；账目页改造；**下线** 团队/银行卡/开户；**沿用**一键开启 |
| Admin | 会员详情 **授权只读 Tab**；**下线** 充提、银行卡审核、代理/开户/推广等（§23.1 **甲** · §24） |

### 11.1 Admin 授权只读（§19.5 **乙**）

会员详情 Tab，**只读、无代绑/解绑**：

| 字段 | 说明 |
|------|------|
| 第三方用户名 | `guaji_username` |
| 是否启用 | `is_active` |
| 绑定时间 | 首次绑定成功时间 |
| 最近同步时间 | 最近一次 Token 续期或 `users/i/info` 成功 |
| 最后投注时间 | 该平台账号下最近一笔 `bet_orders` |
| 最近 Token 失效原因 | 续期/下单失败摘要（不含密钥） |

**不展示**：Token、密码、MFA 材料。

---

## 12. 确认记录归档

### 第三轮（§13）

首绑即用 · 切换&解绑停方案、平台退出不停 · 三账目页 · 任何人可再绑 · 切换仅会员中心 · 保留 sim。

### 第四轮（§14）

| 编号 | 结论 |
|------|------|
| 14.1 | **甲** 流水仅本平台镜像 bet_debit/bet_payout |
| 14.2 | **须启用授权**；无启用时 **仅授权三页**；三页按 **DESIGN.md** |
| 14.3 | 现有 **`stopped`**；手动启动 + **一键开启方案** |
| 14.4 | 进入会员中心 + 手动刷新 + 投注/派奖后刷新 |
| 14.5 | **乙** 不单独「退出授权」 |

### 第五轮（§15）

| 编号 | 结论 |
|------|------|
| 15.1 | **沿用现网** 云端中心「一键开启方案」逻辑（`pending` + `paused`）；**须手动点击** |
| 15.2 | **同 15.1**；按钮 **仅** 在云端中心 `/cloud` |
| 15.3 | **甲** 解绑唯一启用账号 → 立即门禁；方案保持暂停，重新绑号后再启 |
| 15.4 | **丙** 绑定独立页；列表 + 切换 **合并** 为一页 |
| 15.5 | **甲** 切换后 **允许** 立即手动下注 |

### 第六轮（§16）

| 编号 | 结论 |
|------|------|
| 16.1 | **甲** 切换/解绑：`running` + `pending` → **`paused`**（与现有一键开启兼容） |
| 16.2 | **甲** 列表页单页交互：行内「设为启用」+ 切换前确认停方案 |
| 16.3 | **甲+乙 分场景**：0 绑定 → 绑定页；≥1 绑定无启用 → 列表页（空态/选启用） |
| 16.4 | *见 §17.1* |
| 16.5 | *见 §17.2* |
| 16.6 | **甲** 账号被占用：提示让对方解绑；无排队/客服代绑 |

### 第七轮（§17）

| 编号 | 结论 |
|------|------|
| 17.1 | **甲** 界面 **仅显示第三方用户名** |
| 17.2 | **乙** Token 失效 → Toast + 门禁 + **全部 running/pending → paused** |
| 17.3 | **甲** 切换/解绑成功 → **留在授权列表页** + Toast |
| 17.4 | **乙** 可追加绑定；绑成功后 **弹窗询问是否立即切换** |
| 17.5 | **甲** 删充提；资金记录页 **改为仅钱包流水** |
| 17.6 | **乙** 手动 Toast 拒绝；Worker paused + 卡片 **「钱包余额不足」**（沿用现网） |

### 第八轮（§18）

| 编号 | 结论 |
|------|------|
| 18.1 | **甲** 切换/解绑暂停 → 卡片统一 **「已暂停」** |
| 18.2 | **乙** Token 失效 → 列表 **「重新授权」** 刷新 Token |
| 18.3 | **丙** 钱包流水 `/member/fund-records`；投注记录、方案盈亏 **独立路由** |
| 18.4 | **乙** 账目 **默认当前启用授权**；可切 **「全部历史」** |
| 18.5 | **甲** sim **不校验** 第三方余额、不因余额不足 paused |
| 18.6 | **甲** 手动余额不足 Toast：**「可用余额不足，请前往第三方平台充值」** |

### 第九轮（§19）

| 编号 | 结论 |
|------|------|
| 19.1 | **甲** 「重新授权」**全自动 MFA**；失败再走绑定页 |
| 19.2 | **甲** 绑定页：**用户名 + 密码** |
| 19.3 | **甲** 解绑 **标准确认**弹窗 |
| 19.4 | **甲** 顶栏 **仅主币种** |
| 19.5 | **乙** Admin 只读：用户名、启用、同步、绑定时间、最后投注、Token 失效原因 |
| 19.6 | **甲** 首绑/重授权成功 → **会员中心** |

### 第十轮（§20）

| 编号 | 结论 |
|------|------|
| 20.1 | **CNY** 顶栏 **固定展示 CNY** 余额（非多币种择优规则） |
| 20.2 | **甲** 方案盈亏 **同 18.4**：默认当前授权，可切全部历史 |
| 20.3 | **乙** 重新授权自动失败 **连续 3 次** 后走绑定页重填密码 |
| 20.4 | **甲** **部署配置** 决定测试/正式域名；会员不可选 |

### 第十一轮（§21）

| 编号 | 结论 |
|------|------|
| 21.1 | **甲** `users/i/info` 无 CNY → 顶栏显示 **`CNY ¥0.00`** |

---

## 13. 产品决策状态

**§15–§24 已全部确认**，产品向决策 **闭合**（v15）。

实施入口：§9 T0–T6 · §9.1 · §16 · [`integration-checklist.md`](integration-checklist.md) §7。

---

### 第十二轮归档（§22）

| 编号 | 结论 |
|------|------|
| 22.1 | **甲** 投注记录 **沿用** `/member/bet-records` |
| 22.2 | **丙** **`pnl-report`**（个人汇总）+ **`scheme-pnl`**（方案实例云端盈亏）并存 |
| 22.3 | **删除** `/member/ledger` 帐变 + `/member/chase-records` 追号 |
| 22.4 | **丙** 余额大卡：CNY + 第三方充提提示 + 客服 |
| 22.5 | **甲→§24 覆盖** 团队 **下线**（原「首版隐藏」） |
| 22.6 | **甲** 无授权时 **仅** bind/list；FAQ/聊天等 **一律拦截** |
| 22.7 | **乙→§23.1 甲覆盖** Client 不展示 Demo 充提；Admin **同步下线**（不再保留后台查询） |

---

### 第十三轮归档（§23）

| 编号 | 结论 |
|------|------|
| 23.1 | **甲** Admin **同步下线** 充提相关菜单与 API（与 Client 一致） |
| 23.2 | **甲** 游戏详情手动下注区展示 **第三方 CNY 实账**（与顶栏同源） |

---

### 第十四轮归档（§24 · 产品补充）

| 编号 | 结论 |
|------|------|
| 24.1 | **银行卡**：Client + Admin **同步下线** 收款账户绑定/审核（`payout-accounts` 等） |
| 24.2 | **团队**：Client + Admin **同步下线** 团队 Tab、scope=team 切换及团队查询 API |
| 24.3 | **开户**：Client + Admin **同步下线** 下级开户、推广链接、代理管理等拉新/代理能力 |

**范围说明（实施 T1b 对照）**

| 端 | 下线内容 |
|----|----------|
| **Client** | 团队 Tab/入口；`scope=team`（盈亏报表、彩种统计等）；银行卡 CRUD；开户中心；推广设定 |
| **Client API** | `GET/POST /client/member/payout-accounts` · `GET /client/team/*` · `POST /client/team/members` · `GET/POST /client/team/promo-links` |
| **Admin** | 银行卡审核；代理 L1/L2、佣金上限、推广渠道等菜单 |
| **Admin API** | `GET/POST /admin/funds/payout-accounts/*` · `GET/PUT /admin/agents/*` |

库表历史数据可保留；**无前后台查询/操作入口**（与充提下线策略一致）。

---

## 23. ~~待您确认~~（已归档 → 见 §13 第十三轮）

## 16. 技术审查（对照现网代码 · 2026-06-08）

> 产品项已闭合。本节记录 **实现前** 须消化的技术差距、冲突与建议顺序；**不改动代码**，仅供 T 阶段排期。

### 16.1 现网与方案差距总览

| 域 | 现网 | 方案要求 | 差距 |
|----|------|----------|------|
| 第三方授权 | **无** `guaji/*`、无 `member_guaji_accounts` | T1 授权表 + 绑/解/切换 API | **从零建设** |
| Client 门禁 | 仅平台 JWT `beforeEach` | `hasActiveGuajiAuth` + 授权两页路由 | **未实现** |
| 顶栏余额 | `GET /client/member/wallet`（**本地** `member_wallets`） | `users/i/info` **CNY** 实账 | **数据源切换** |
| real 投注扣款 | `worker_wallet.go` 锁 **本地钱包** + `wallet_ledger` | 第三方 `web_bets/lott` 接单成功后再镜像 ledger（C8） | **核心改造（T4/T5）** |
| 手动下注 | `GameDetailView` → 本地 API + **本地/Mock 余额** | 须走第三方 + 启用授权 Token；dock **CNY 实账**（§23.2 **甲**） | **T4 + T1b** |
| 充提 | Phase 2 已接 `funds/recharge` 等 | **Client + Admin 同步删除**（§23.1 **甲**） | **T1b 清理** |
| 团队/银行卡/开户 | Phase 2 已接 `team/*`、`payout-accounts` | **Client + Admin 同步下线**（§24） | **T1b 清理** |
| `bet_orders` | 已有 `outbound_*` 快照（C41/C43） | 尚需 `guaji_account_id`、`third_party_bet_id`、`currency` | **迁移** |
| `wallet_ledger` | 有 `bet_debit`/`payout`；绑 **本地余额** `balance_after` | 镜像流水 + `guaji_account_id`；**不驱动本地余额** | **语义重构（见 16.3）** |
| 切换停方案 | 仅单实例 `pause` API | 切换/解绑 **批量** `running`+`pending`→`paused` | **需批量接口或事务** |
| OpenAPI / checklist | 无 guaji 路径 | 须先契约后实现 | **T1 前置** |

### 16.2 数据模型（建议迁移顺序）

1. **`member_guaji_accounts`**（T1）  
   - 字段建议：`id`、`member_id`、`guaji_username`（**GLOBAL UNIQUE**）、`password_enc`、`mfa_material_enc`、`access_token_enc`、`token_expires_at`、`is_active`、`bound_at`、`last_sync_at`、`last_token_error`（Admin 用）、`created_at`、`updated_at`  
   - 约束：每 `member_id` 至多一条 `is_active=true`（partial unique index）

2. **`bet_orders` 扩展**（T4 前）  
   - `guaji_account_id`、`third_party_bet_id`、`currency`  
   - 与现有 `outbound_lottery_code` / `outbound_play_code` **并存**

3. **`wallet_ledger` 扩展**（T5）  
   - `guaji_account_id`（nullable，历史 Demo 行为 NULL）  
   - 索引：`(member_id, guaji_account_id, created_at DESC)`

4. **术语对齐**  
   - 方案 §4.5 写 `bet_payout` → 库表现网为 **`payout`**（`00005_wallet_ledger.sql`）；实施时 **统一用 `payout`**，更新方案表述即可。

### 16.3 本地钱包 vs 第三方实账（关键架构点）

现网 **real** Worker（`applyRealModeSettlement`）在同一事务内：扣 `member_wallets` → 写 `wallet_ledger` → 写 `bet_orders`。

方案要求：**实账在第三方**；本平台 ledger 仅为 **行为镜像**（§4.1、C8）。

**建议实施路径（技术，非新产品决策）：**

| 模式 | 本地 `member_wallets` | `wallet_ledger.balance_after` | 余额不足判断 |
|------|----------------------|-------------------------------|--------------|
| **real**（对接后） | **不再扣减**（或只读废弃） | 存第三方余额快照或 **nullable** | 下单前查 `users/i/info` CNY |
| **sim** | 可保留现状或冻结 | 维持现网 | 不校验第三方（§18.5） |

⚠️ `chk_wallet_ledger_balance_after >= 0` 在「镜像、不维护本地余额」下可能 **失效**；T5 须二选一：**放宽约束** / **改存第三方余额快照** / **镜像行不写 balance_after**。

### 16.4 API 与 Client 待补清单

**后端（OpenAPI 先行）**

| 方法 | 路径（建议） | 用途 |
|------|--------------|------|
| POST | `/client/guaji/accounts/bind` | 绑号 + MFA |
| GET | `/client/guaji/accounts` | 列表 |
| POST | `/client/guaji/accounts/{id}/activate` | 设为启用（内含批量 pause） |
| POST | `/client/guaji/accounts/{id}/reauth` | 重新授权 |
| DELETE | `/client/guaji/accounts/{id}` | 解绑（内含批量 pause） |
| GET | `/client/guaji/balance` | 代理 `users/i/info` → CNY |
| GET | `/admin/members/{memberNo}/guaji-accounts` | Admin 只读 Tab |

**Client 路由（规划已有，未注册）**

- `/member/auth/bind`、`/member/auth/list`  
- `/member/scheme-pnl`（新建）；沿用 `/member/bet-records`  
- 下线路由：`/member/ledger`、`/member/chase-records`  
- `router.beforeEach`：`hasActiveGuajiAuth` + 白名单

**须下线/改造的现网调用**

- `fetchMemberWallet` → 顶栏 + **`GameDetailView` dock** 改调 guaji balance（§23.2 **甲**）
- `MemberCenterView` 充值/提现入口
- `POST /client/funds/recharge` 等（方案删除；**Admin 同步下线** §23.1 **甲**）
- Admin：`DashboardView` 充值 KPI、`useAdminQueueSync` 提现 WS、提现审批/充值渠道菜单与 API
- Client/Admin **团队、银行卡、开户**：`team/*`、`payout-accounts`、`agents/*` 等（§24）
- `MemberPnlReportView` / `MemberLotteryStatView` 移除 **scope=team** 切换

### 16.5 行为与一致性核对

| 项 | 结论 |
|----|------|
| 实例状态 | 现网 `pending/running/paused/soft_stopped`；方案已对齐 **`paused`**，无 `stopped` ✅ |
| 一键开启 | `CloudCenterView.enableAllSchemes()` 仅 `pending`+`paused`；切换后 `paused` **可覆盖** ✅ |
| 余额不足文案 | `instance_status.go` 已输出 **「钱包余额不足」**；与 §17.6 一致 ✅ |
| Token 失效门禁 | `hasActiveGuajiAuth` 须 = `is_active` **且** Token 有效；失效时建议 **`is_active` 保持**、另设 `token_valid=false` 触发门禁（避免丢绑定关系） |
| 平台 logout 不停方案 | Worker 不依赖 Client JWT；**仍依赖** 服务端 guaji Token ✅ |
| 结算 C8/C17 | 现网 Worker 按 **本地开奖** 结算 real 订单；对接后 real 须改为 **第三方接单/派奖同步**（T5），与 [`lottery-catalog-migration-plan.md`](lottery-catalog-migration-plan.md) C8/C17 一致 |
| game_id / rule_id | T2：`web_bets/lott` 映射；seed 已有 `outbound_*`，须运营维护真实第三方码（C46） |
| WS 福彩过滤 | T3；匿名 WS + 文档已列忽略彩种 |
| 方案盈亏页 | ✅ **`/member/scheme-pnl`** 新建 + **`/member/pnl-report`** 保留（§22.2 **丙**） |
| 投注记录路由 | ✅ **`/member/bet-records`**（§22.1 **甲**） |
| 帐变/追号 | ✅ **下线** ledger + chase（§22.3） |
| 团队 scope | ✅ **下线** team Tab + scope=team（§24） |
| 银行卡/开户 | ✅ **前后台同步下线**（§24） |
| 门禁白名单 | ✅ **仅** bind/list（§22.6 **甲**） |
| Admin 充提 | ✅ **同步下线**（§23.1 **甲**） |
| 游戏详情余额 | ✅ dock **CNY 实账**（§23.2 **甲**） |

### 16.6 技术待确认项（实施前由研发拍板，默认建议已给出）

| 编号 | 问题 | 默认建议 |
|------|------|----------|
| **T1** | 切换/解绑批量 pause 实现 | 单事务 SQL：`UPDATE scheme_instances SET status='paused' WHERE member_id=? AND status IN ('running','pending')` + 发 WS |
| **T2** | `hasActiveGuajiAuth` 判定 | `is_active=true` AND `token_expires_at > now()` AND `last_token_error IS NULL`（或等价健康标记） |
| **T3** | 镜像 `wallet_ledger` 的 `balance_after` | real 镜像行写 **第三方 CNY 余额快照**；逐步废弃本地 `member_wallets` 在 real 路径的使用 |
| **T4** | 方案盈亏 `/member/scheme-pnl` 数据源 | **复用** `GET /client/cloud/bet-records` summary + `guaji_account_id` 筛选（§22.2 **丙**） |
| **T5** | 密码/MFA 加密 | `backend/.env` 专用 `GUAJI_CREDENTIALS_KEY`（32-byte）；rotation Runbook 写 T6 |
| **T6** | `integration-checklist.md` | 新增 **§7 第三方授权（T1–T5）** 验收块（与 §9.1 对齐）✅ |

### 16.7 建议实施顺序（修订）

```
T0 环境 + 测试号 + guaji 适配层骨架
  → T1 表 + API + 批量 pause + 门禁中间件（可先 mock 第三方）
  → T1b Client 授权两页 + 顶栏/游戏详情 CNY + 删充提/团队/银行卡/开户 + Admin 同步下线 + fund-records 改流水
  → T2 outbound 码 ↔ game_id/rule_id 映射表 + Admin 维护 outbound_lottery_code
  → T3 WS 开奖接入 + 过滤
  → T4 web_bets/lott 真实下单 + Worker 改走 guaji Token（停本地扣款）
  → T5 第三方结算同步 + ledger 镜像 + 余额刷新钩子
  → T6 监控、Token 续期告警、Runbook、checklist §7
```

### 16.8 双重审查结论（v15）

| 维度 | 状态 |
|------|------|
| **产品主链路**（授权/门禁/切换/资金/sim/real） | ✅ §15–§24 闭合 |
| **现网页面对齐** | ✅ §22–§24 已确认 |
| **资金架构** | ⚠️ real 改第三方；§16.3 研发默认 |
| **验收** | ✅ checklist §7（含 Admin 下线、游戏详情 CNY、§24） |
| **契约** | ⚠️ OpenAPI guaji 待 T1；下线 API 须在 OpenAPI 标注 deprecated 或移除 |
| **追号能力** | ⚠️ Client 页删除；后端 `chase_orders` API **暂保留**，首版无入口（§22.3） |
| **团队/代理/银行卡** | ✅ §24 前后台同步下线；`MemberPnlReportView`/`MemberLotteryStatView` 去 team scope |
| **Admin 遗留** | ⚠️ Dashboard KPI / 提现 WS / 会员详情 ledger「入款/出款」类型须 T1b 一并清理 |

---

## 14. 文档索引

| 文档 | 关系 |
|------|------|
| [`lottery-catalog-migration-plan.md`](lottery-catalog-migration-plan.md) | P0–P5 已完成 |
| [`integration-plan.md`](integration-plan.md) | Phase 2 充提等待下线；授权门禁待同步 |
| [`integration-checklist.md`](integration-checklist.md) | **§7** 第三方授权 E2E 验收 |
| [`client/DESIGN.md`](../client/DESIGN.md) | 授权两页 UI 规范 |
| `挂机协助接口短信邮件版.doc` | 第三方契约 |

---

## 26. 真实协议实测（2026-06-09，测试号 testcq01）

> 用 `cmd/guaji-capture` 实测抓包；代码已据此对齐。原文档部分字段过期，以此节为准。

### 26.1 成功码与响应格式（不统一）

| 接口 | 格式 | 成功标志 |
|------|------|----------|
| `POST /auth/login`（admin 域名） | `{"success":true,"data":{token,refresh_token,token_type,username,is_temp_pwd_user}}` | **`success:true`**（无 code） |
| `GET /api/users/i/info` | **裸对象**（无 data 包裹） | HTTP 200 |
| `GET /api/agents/i/real/rate` | `{"code":0,"data":{real_rate,lott_odds,...}}` | **code=0** |
| `.../web_bets/lott/periods` | `{"code":201,"data":[...]}` | code=201 |

→ 代码 `parseEnvelope` 兼容 `success` 优先、`code∈{0,200,201}`；`UserInfo` 支持裸对象。

### 26.2 余额（users/i/info.account）

```
balance / available_balance        → USDT
balance_trx / available_balance_trx → TRX
balance_cny                         → CNY
```
实测 testcq01：USDT 26033.93 / TRX 105640.8 / CNY 100000。币种下单编码 **0=usdt 1=trx 3=cny**。

### 26.3 开奖 WS（实测端点 **`wss://hash.iyes.dev/ws`**）

> 文档原 `wss://…/?token=` 握手返回 HTTP 200（非升级）；真实端点是 **`/ws`**，匿名 `?token=Anonymous`，需浏览器 UA。

一条 `lottery_v2_broadcast` = 一个区块 + **多彩种线**（`lottery_logXXX` 各自 `periods`）+ 区块衍生的**多玩法号码字段（共享）**：

```json
{"send":true,"message":{
  "type":"lottery_v2_broadcast","block_num":83441446,"created":"...",
  "last5_num":"94819",                  // ssc_std（极速 5 位连写）
  "last11_5_num":"06,04,08,01,09",      // syxw_std（11选5）
  "last_pk10_num":"03,06,...",          // pk10_std
  "last_k3_num":"2,3,5",                // k3_std
  "lhc_num":"49,21,18,04,39,36,34",     // lhc_std（6 正+1 特；文档解析缺，实测有）
  "lottery_log101":{"periods":"...","next_periods":"..."},
  "lottery_log033":{"periods":"105202606091971",...}
}}
```

- **号码字段 = 玩法维度**；`lottery_logXXX` = 彩种线维度（各自期号）。
- 心跳/忽略类型：`block` / `block-new` / `long_dragon_update` / `fc3d_*` / `pl35_*` / `fc_pl3d_*`。
- 以太坊：`eth_lottery_v2_broadcast` + `last10_6_num` + `eth_lottery_logXX`；台湾：`tw_lottery_v2_broadcast` + `last_tw5_num`/`last_tw28_num` + `tw_lottery_log`。

→ 代码：`ParseDrawEvents` 一条消息拆多 `DrawEvent`（每 `lottery_logXXX` 一个，携全玩法号码）；`drawsync.Worker` 按 `outbound_lottery_code = lottery_logXXX 键` 反查**所有匹配彩种**（一键多彩种），各按 `play_template` 经 `DrawBalls.BallsFor(template)` 选号入库。实测彩种线见 §26.4。

### 26.4 实测彩种线键（测试环境，40s 抓样）

> 每个 `lottery_logXXX` 同区块衍生 **5 玩法号码（ssc/syxw/pk10/k3/lhc 共享）**；台湾/以太坊独立。
> **一个键可对应多个本平台彩种**（不同 `play_template`）；`drawsync` 已支持一键多彩种入库。

| 彩种线键 | 含玩法 | 说明（待运营核对周期↔彩种） |
|----------|--------|------------------------------|
| `lottery_log033` | ssc/syxw/pk10/k3/lhc | 波场极速彩线 |
| `lottery_log05`  | ssc/syxw/pk10/k3/lhc | 波场（另一周期线） |
| `lottery_log101` / `log103` / `log115` / `log125` | 同上 | 波场 1/3/5 分等周期线 |
| `eth_lottery_log` | ssc/syxw/pk10/k3 | 以太坊极速线 |
| `tw_lottery_log` | tw28 | 台湾 28 |

### 26.5b 真实下单实测结论（testcq01，2026-06-09）

| 步骤 | 结果 |
|------|------|
| `POST /api/web_bets/lott`（game_id=29, rule_id=13, bet_content=`,,,13579,`, currency=3, 10 CNY） | **请求体格式被接受**（未报字段/参数错误） |
| 卡点 | **`code=40060 "用户没有设置密保"`**（HTTP 400），**未扣款**（接单失败前不扣） |
| 设密保 `POST /auth/login/security`（147258） | `code=40000 "已设置过密保，无法重复设置"` → **账号已设密保** |
| 携带 `wp_password=147258` 重试 | 仍 `40060` |
| `GET /web_bets/`（无 filters） | `{"data":[],"code":"0","count":0}`（结构确认） |
| `POST /web_bets/lott/periods`（`game_id` + `num_periods`） | `{"code":201,"data":[...]}` | code=201 |
| `GET /web_bets/lott/periods` | `405 Method Not Allowed` | **不可用**；代码已改为 POST |

**结论**：下单请求体（`bet_contents[]`/`game_id`/`currency`）与本平台 `PlaceLottBet` 实现一致、被第三方接受；**唯一卡点是 `40060` 密保验证**。

**多账号一致复现（决定性证据）**：

| 账号 | 密保（security_question） | 加 wp_password | 加 security_code | 下单结果 |
|------|---------------------------|----------------|------------------|----------|
| testcq01 | 已设（reminder=147258） | 无效 | 无效 | `40060` |
| testcq02 | 已设（reminder=1） | — | — | `40060` |

- 文档 §11 下单体**无密保/资金密码字段**；我方请求完全符合文档。
- 补 `wp_password`、`security_code` 均无效；**换账号（testcq02）同样 40060**。
- 两账号均已设密保（`security_question` 可查），却都报「没有设置密保」。

→ **判定为第三方下单接口/账号开通的统一问题**（非单账号、非请求字段）。疑似：① 挂机账号需主管后台开通**下单/投注权限**（§18「主管同意后安排运维配置」）；② 下单校验依赖 **Google 2FA**（两账号均 `enabled_google:false`）；③ 服务端 40060 判定逻辑。**需第三方技术确认；非本平台代码问题。**

### 26.5 仍待运营配置 / 下单联调

| 项 | 说明 |
|----|------|
| `outbound_lottery_code` ↔ `lottery_logXXX` | 运营维护态把各彩种对接码配为对应 WS 彩种线键（如波场极速→`lottery_log033`）；周期线↔具体彩种需对照 §8 平台彩种表确认 |
| 下单 `game_id` 数字 / `rule_id` / `bet_content` 位段 | `web_bets/lott` 真实下单需运营填数字 game_id + rule_id；bet_content 位段编码按玩法（§11 示例 `",,,13579,"`），下单联调核对（**真实扣款，需小额验证**） |

### 26.5 测试账号

`testcq01`~`testcq10`（密码同名）；资金密码/密保 `147258`。抓包：`make guaji-capture`（设 `GUAJI_TEST_USERNAME/PASSWORD`）。

---

## 25. 运维 Runbook（T6）

### 25.1 监控点

| 指标 | 来源 | 告警阈值 |
|------|------|----------|
| HTTP/WS 可达 | `GET /health` → `data.guaji` | `httpReachable=false` 持续 |
| 授权健康 | `GET /admin/guaji/health` | `erroredTokens>0`、`expiringSoon` 增长 |
| Token 巡检 | 后端日志 `guaji token health alert`（每 5 分钟） | 出现即关注 |
| 开奖入库 | `drawsync` 日志 `guaji draw ws disconnected` 频繁重连 | 连接不稳 |

### 25.2 `GUAJI_CREDENTIALS_KEY` 轮换

> 凭证（密码/MFA/Token）以 AES-GCM 加密存 `member_guaji_accounts`，密钥来自 `GUAJI_CREDENTIALS_KEY`（32 字节）。

1. 轮换前确认无大量在跑 real 方案（建议低峰）。
2. 直接换 key 会导致旧密文无法解密 → 会员需 **重新授权**（绑定页重填密码）。
3. **推荐**：上线前固定 key；如必须轮换，发布公告并引导会员重新授权（`reauth_fail_count≥3` 已自动引导绑定页）。
4. 轮换后监控 `erroredTokens`，必要时批量提示重新授权。

### 25.3 常见故障处置

| 现象 | 处置 |
|------|------|
| 大量 `erroredTokens` | 检查第三方可用性 / key 是否被改；引导会员「重新授权」 |
| 会员投注报「授权已失效」 | Token 过期或 `last_token_error` 非空；列表页「重新授权」全自动 MFA |
| 会员报「余额不足」但第三方有钱 | 核对主币种（USDT/TRX/CNY）是否与第三方余额币种一致 |
| 开奖不更新 | 查 `drawsync` 重连日志 + WS 基址；确认非忽略彩种（福彩/排列） |
| real 订单长期 pending | 派奖同步未完成；查 `QuerySettlement` 与第三方注单状态 |

### 25.4 环境切换

部署配置决定第三方域名（§20.4）：测试 `hash.iyes.dev` / 正式 `s9-xia.5rf9q.com`；会员不可选。切换需改 `GUAJI_HTTP_BASE`/`GUAJI_WS_BASE`/`GUAJI_AUTH_BASE`/`GUAJI_ORIGIN` 并重启。

---

## 附录 A：接口与环境

```
测试  https://hash.iyes.dev/     wss://hash.iyes.dev/?token={token|Anonymous}
正式  https://s9-xia.5rf9q.com/  wss://s9-ws.5rf9q.com/?token={token|Anonymous}

POST /auth/login
GET  /api/users/i/info
POST /api/web_bets/lott
GET  /api/web_bets/
```
