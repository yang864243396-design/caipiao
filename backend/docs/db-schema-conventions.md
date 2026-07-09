# 数据库 Schema 规范（建表备注）

> **强制**：Phase 2 起所有**业务表**迁移必须包含 **表备注** 与 **全部字段备注**。  
> PostgreSQL 使用 `COMMENT ON TABLE` / `COMMENT ON COLUMN`（**不用**列定义里的 `COMMENT '...'` 语法，PG 不支持）。

---

## 1. 基本要求

| 项 | 要求 |
|----|------|
| 表备注 | 每张业务表一条 `COMMENT ON TABLE`，说明业务用途、生命周期 |
| 字段备注 | 每个列一条 `COMMENT ON COLUMN`，不可遗漏 |
| 语言 | **简体中文**（枚举/状态字段写清取值含义） |
| 金额 | 注明单位：**元，2 位小数**，对应 `NUMERIC(18,2)` |
| 时间 | 注明 **UTC** 或业务时区；`TIMESTAMPTZ` 统一写 UTC |
| 外键 | 备注中写 `关联 xxx.id` |
| 索引 | 重要索引建议 `COMMENT ON INDEX`（可选但推荐） |
| 非空 | 业务必填列 **必须** `NOT NULL`；可空列在 COMMENT 中说明何时为 NULL |
| 约束 | 金额/状态用 `CHECK`；业务号/登录名用 `UNIQUE`；关联用 `FOREIGN KEY` |

---

## 1.1 索引规范

| 场景 | 做法 |
|------|------|
| 主键 | 默认 `BIGSERIAL PRIMARY KEY` |
| 唯一业务号 | `UNIQUE` 约束或 `CREATE UNIQUE INDEX`（二选一，推荐 CONSTRAINT） |
| 会员列表/流水 | 复合索引 `(member_id, created_at DESC)` |
| 状态筛选 | 管理端高频筛选时加 `(status, created_at DESC)` |
| 部分索引 | 仅热点子集，如 `WHERE status = 'pending_review'` |
| 命名 | `uq_` 唯一 · `idx_` 普通 · `pk_` 主键（PG 自动） |

**避免**：无查询用途的冗余索引；过宽索引（把大 JSONB 放进索引键）。

## 1.2 非空与默认值

| 类型 | 约定 |
|------|------|
| 创建/更新时间 | `created_at TIMESTAMPTZ NOT NULL DEFAULT now()` |
| 状态字段 | `NOT NULL DEFAULT 'pending'`，配合 `CHECK` |
| 金额 | `NOT NULL DEFAULT 0` + `CHECK` 范围 |
| 可空外键 | 仅当业务确实可选（如 `order_ref` 无关联单号时为 NULL） |

## 1.3 CHECK / FK 规范

```sql
CONSTRAINT chk_orders_amount_positive CHECK (amount > 0),
CONSTRAINT fk_orders_member FOREIGN KEY (member_id) REFERENCES members (id) ON DELETE RESTRICT
```

- 状态字段：`CHECK (status IN ('pending', 'paid', ...))`
- 外键默认 **`ON DELETE RESTRICT`**；级联仅用于明确的从属表（如实例删定义）
- 软删除用 `deleted_at TIMESTAMPTZ NULL`，**不用**物理删会员+流水

---

## 2. 迁移文件结构

```sql
-- +goose Up
-- +goose StatementBegin
CREATE TABLE members (
    id         BIGSERIAL PRIMARY KEY,
    account    VARCHAR(32) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE members IS '会员账号主表';
COMMENT ON COLUMN members.id IS '主键';
COMMENT ON COLUMN members.account IS '登录账号，全局唯一';
COMMENT ON COLUMN members.created_at IS '注册时间（UTC）';

CREATE UNIQUE INDEX uq_members_account ON members (account);
COMMENT ON INDEX uq_members_account IS '登录账号唯一';
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS members;
```

模板文件：[`migrations/_template_business_table.sql.example`](../migrations/_template_business_table.sql.example)

---

## 3. 字段备注写法

### 3.1 枚举 / 状态

```sql
COMMENT ON COLUMN orders.status IS '订单状态：pending=待支付，paid=已支付，cancelled=已取消';
```

若使用 PostgreSQL ENUM：

```sql
CREATE TYPE order_status AS ENUM ('pending', 'paid', 'cancelled');
COMMENT ON TYPE order_status IS '订单状态枚举';
```

### 3.2 JSONB

```sql
COMMENT ON COLUMN scheme_definitions.config IS '方案配置 JSON（结构见 schemes.md §8）';
```

### 3.3 软删除 / 审计

```sql
COMMENT ON COLUMN members.deleted_at IS '软删除时间（UTC）；NULL 表示未删除';
COMMENT ON COLUMN wallet_ledger.operator IS '操作人：system=系统，admin:{id}=管理员，member:{id}=会员';
```

---

## 4. Review 检查清单

提交 migration 前确认：

- [ ] `COMMENT ON TABLE` 已写
- [ ] 表中**每一列**均有 `COMMENT ON COLUMN`
- [ ] 状态/类型字段已列出合法取值
- [ ] 金额、时间单位已标明
- [ ] 必填列均已 `NOT NULL`，且有合理 `DEFAULT`
- [ ] 金额/状态/`CHECK`、唯一键、`FOREIGN KEY` 已声明
- [ ] 列表查询字段已建索引（含复合索引顺序）
- [ ] 每个索引（重要）有 `COMMENT ON INDEX`
- [ ] `goose` Up/Down 成对，Down 可回滚
- [ ] 已在本地或测试库 `make migrate-up` 通过

---

## 5. 查看备注

```sql
-- 表备注
SELECT obj_description('members'::regclass);

-- 所有列备注
SELECT
    a.attname AS column_name,
    col_description(a.attrelid, a.attnum) AS comment
FROM pg_attribute a
JOIN pg_class c ON c.oid = a.attrelid
WHERE c.relname = 'members'
  AND a.attnum > 0
  AND NOT a.attisdropped
ORDER BY a.attnum;
```

---

## 6. 与 sqlc / 代码的关系

- COMMENT **仅存在于数据库**，不自动生成 Go struct 注释。
- 表结构以 `migrations/*.sql` 为单一事实来源；产品字段含义可同时引用 `docs/modules/*.md`。
- 后续可在 sqlc 生成后，对关键 struct 补 Go 文档注释，内容与 DB COMMENT 保持一致。

---

## 7. 非业务表

`schema_bootstrap`、`goose_db_version` 等脚手架/工具表也应尽量补备注；业务表标准同上。
