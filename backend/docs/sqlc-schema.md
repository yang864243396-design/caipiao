# sqlc Schema 维护说明

sqlc **不连接** PostgreSQL，而是用 `sqlc.yaml` 里列出的 migration / patch SQL 拼出「逻辑 schema」，再与 `internal/db/queries/*.sql` 做类型检查。

## 生成命令

```bash
cd backend
make sqlc
# 或
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.29.0 generate
```

## 新增 migration 时

1. 在 `migrations/` 写好 goose 迁移并 `make migrate-up` 应用到远程/本地库。
2. 若 migration 含 **DDL**（`CREATE` / `ALTER` / `DROP COLUMN` 等），按版本号追加到 `sqlc.yaml` 的 `schema:` 列表。
3. **不要**把纯 seed（`INSERT`/`UPDATE`/`DELETE` 数据）或一次性数据清理 migration 列入 schema，除非其中包含 queries 依赖的新列。
4. 运行 `make sqlc`，修复 queries 与生成代码。

## 特例

| 文件 | 说明 |
|------|------|
| `00062_drop_team_agents.sql` | UP 会 `DROP cms_promo_channel`，但 `content.sql` 仍查询该表 |
| `internal/db/sqlc/patches/99_cms_promo_for_sqlc.sql` | **仅 sqlc 使用**，在 00062 之后重建 `cms_promo_channel` |
| `00098_guaji_outbound_iyesdev_fix.sql`（已重编号为 `00107_guaji_outbound_iyesdev_fix.sql`） | 纯 `UPDATE` 数据，无需列入 schema |
| `00097_cloud_bet_orphan_cleanup.sql` | 纯 `DELETE`，无需列入 schema |
| `00070_lottery_play_catalog_seed.sql` | 纯 seed；表结构已在 `00069` |

## 与 goose 重复版本号

存在两个 `00098_*.sql` 时，**只把含 DDL 的文件**（`00098_scheme_instance_start_skip.sql`）列入 sqlc；数据修正文件走 goose 即可。

## 排查失败

```text
column "xxx" does not exist  →  缺 ALTER migration，补进 sqlc.yaml
relation "yyy" does not exist →  缺 CREATE migration，或需 sqlc patch
```
