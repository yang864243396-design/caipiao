# 会员与资产域（Phase 2 · 数据表）

> **状态**：表结构已定案并落地 `migrations/00003`–`00009`  
> **关联**：[`integration-plan.md`](../integration-plan.md) Phase 2 · [`db-schema-conventions.md`](../db-schema-conventions.md)

---

## 1. 表清单

| 表 | 用途 |
|----|------|
| `members` | 会员账号、资料、代理归属、状态 |
| `member_wallets` | 会员钱包余额（可用 + 冻结），与 `members` 1:1 |
| `wallet_ledger` | 帐变流水（只增不改） |
| `recharge_orders` | 充值订单 |
| `withdraw_orders` | 提现订单 |
| `bet_orders` | 全站投注订单（会员投注记录） |
| `member_payout_accounts` | 出款账户（银行卡 / USDT / 第三方钱包等） |
| `chase_orders` | 追号订单 |

---

## 2. ER 关系（简图）

```
members 1──1 member_wallets
   │
   ├──< wallet_ledger
   ├──< recharge_orders
   ├──< withdraw_orders
   └──< member_payout_accounts
```

---

## 3. 索引与约束原则（本域）

| 原则 | 说明 |
|------|------|
| 主键 | 业务表统一 `BIGSERIAL`；对外另设 `member_no` / `order_no` 等业务号 |
| 非空 | 业务必填字段一律 `NOT NULL`；可选字段才允许 NULL |
| 金额 | `NUMERIC(18,2)` + `CHECK (amount > 0)`（订单）或 `CHECK (balance >= 0)`（余额） |
| 唯一 | 登录账号、业务单号、帐变流水号全局唯一 |
| 外键 | `ON DELETE RESTRICT`（会员不可物理删而留孤儿流水） |
| 列表查询 | 复合索引 `(member_id, created_at DESC)` |
| 乐观锁 | `member_wallets.version` 防并发扣款 |

---

## 4. 枚举（存库英文码，接口可映射中文）

| 字段 | 合法值 |
|------|--------|
| `members.status` | `active` 正常 · `frozen` 冻结 |
| `wallet_ledger.txn_type` | `deposit` 入款 · `withdraw` 出款 · `bet_debit` 投注扣款 · `payout` 派奖 · `withdraw_freeze` 提现冻结 · `adjust` 调账 |
| `recharge_orders.status` | `pending` · `paid` · `cancelled` · `failed` |
| `withdraw_orders.status` | `pending_review` 待审核 · `pending_payout` 待打款 · `paid` 已打款 · `rejected` 已驳回 |
| `member_payout_accounts.account_type` | `bank_card` · `usdt_trc20` · `usdt_bsc` · `usdt_ton` · `usdt_sol` · `alipay` · `mpay` · `goubao` |
| `member_payout_accounts.status` | `pending_review` · `active` · `rejected` · `disabled` |

---

## 5. 演示数据

`00009_seed_demo_members.sql` 写入附录 A 对齐账号（`M00001` / `vs8888` 等），供联调 `GET /client/member/profile` 后续接入。

---

## 6. 后续接口（待实现）

| 接口 | 主要表 | 状态 |
|------|--------|------|
| `GET /client/member/profile` | `members` + `member_wallets` | ✅ |
| `GET /client/member/wallet` | `member_wallets` | ✅ |
| `GET /client/orders/ledger` | `wallet_ledger` | ✅ |
| `GET /client/orders/bets` | `bet_orders` | ✅ |
| `POST /client/auth/login` | `members.password_hash`（bcrypt） | ✅ |
| `POST /client/funds/recharge` | `recharge_orders` + `wallet_ledger` | ✅ Demo 即时到账 |
| `GET /client/funds/records` | `recharge_orders` + `withdraw_orders` | ✅ |
| `GET /client/funds/withdraw/context` | `members` + `member_wallets` + `member_payout_accounts` | ✅ |
| `POST /client/funds/withdraw` | `withdraw_orders` + `wallet_ledger` + 冻结余额 | ✅ |
| `GET /client/team/overview` | `members` + `member_wallets` + 代理树 | ✅ |
| `GET /client/team/stats` | `members` + `bet_orders` | ✅ |
| `GET /client/team/members` | `members` + `member_wallets` | ✅ |
| `GET /client/orders/chases` | `chase_orders` | ✅ |
