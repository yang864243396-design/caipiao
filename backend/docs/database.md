# PostgreSQL 接入说明

> **定案**：PostgreSQL **18.x** · **pgx** + **sqlc** + **goose** · HTTP 仍用标准库 `net/http`

## 连接配置

复制 `.env.example` 为 `.env`（已在 `.gitignore`，勿提交密码）：

```env
DB_HOST=192.168.100.239
DB_PORT=5432
DB_NAME=caipiao
DB_USER=caipiaoapp
DB_PASSWORD=***
DB_SSLMODE=disable
DB_REQUIRED=true
```

或使用完整 DSN：

```env
DATABASE_URL=postgres://caipiaoapp:***@192.168.100.239:5432/caipiao?sslmode=disable
```

| 变量 | 说明 |
|------|------|
| `DB_SSLMODE` | 内网常用 `disable`；生产建议 `require` 或 `verify-full` |
| `DB_REQUIRED` | `true`：连不上 DB 则服务不启动；`false`：降级为 Phase 1 内存 Store |

## 迁移（goose）

```bash
cd backend
make migrate-up      # 执行 migrations/
make migrate-status  # 查看版本
make migrate-down    # 回滚一步
```

首次迁移会创建 `schema_bootstrap` 表，用于验证账号具备建表权限。

**建表规范（强制）**：每张业务表、每个字段必须有中文 `COMMENT`；非空、CHECK、唯一、外键与列表索引见 [db-schema-conventions.md](db-schema-conventions.md) 与 `migrations/_template_business_table.sql.example`。

Phase 2 会员资产表结构见 [modules/members.md](modules/members.md)（`00003`–`00009`）。

## 健康检查

```bash
curl http://127.0.0.1:8080/api/v1/health
```

成功示例：

```json
{
  "code": 0,
  "message": "ok",
  "data": {
    "status": "ok",
    "db": "up",
    "postgres": "18.x ..."
  }
}
```

## 远程库网络要求

1. PostgreSQL `listen_addresses` 含内网 IP 或 `*`
2. `pg_hba.conf` 允许 **Go 应用服务器出口 IP** + 用户 `caipiaoapp`
3. 防火墙/安全组放行 `5432`（仅对白名单 IP）

## sqlc（后续 Phase 2+）

- 配置：`sqlc.yaml`
- SQL 文件：`internal/db/queries/*.sql`
- 生成代码：`internal/db/sqlcdb/`

```bash
# 安装后执行
sqlc generate
```

## 目录

```
backend/
  migrations/           # goose SQL
  cmd/migrate/          # 迁移 CLI（读 .env）
  internal/db/          # pgx 连接池
  internal/db/queries/  # sqlc 查询（待增）
  sqlc.yaml
```
