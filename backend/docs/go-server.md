# Go 后端

HTTP + WebSocket 服务，契约以 [`openapi/openapi.yaml`](openapi/openapi.yaml) 为准；接入阶段见 [`docs/integration-plan.md`](docs/integration-plan.md)，联调清单见 [`docs/integration-checklist.md`](docs/integration-checklist.md)。

## 快速开始

```bash
cd backend
cp .env.example .env   # 配置 DB_HOST / DB_PASSWORD 等
make tidy
make migrate-up        # 首次：在远程库执行迁移
make run
```

服务默认监听 **`:8080`**，API 基址 **`http://127.0.0.1:8080/api/v1`**。

数据库说明见 [`docs/database.md`](docs/database.md)。

## 核心端点（摘要）

| 方法 | 路径 | 鉴权 | 说明 |
|------|------|------|------|
| GET | `/api/v1/health` | — | 健康检查（含 `guaji` 连通块，需 `GUAJI_ENABLED=true`） |
| GET | `/api/v1/public/maintenance` | — | 全站维护状态 |
| POST | `/api/v1/client/auth/login` | — | 会员/代理登录（**PostgreSQL `members` bcrypt**） |
| POST | `/api/v1/admin/auth/login` | — | 管理端登录（**PostgreSQL `admin_users` bcrypt**；无 DB 时回退 `.env`） |
| GET | `/api/v1/ws/public` | 可选 | 公共 WS（维护、开奖） |
| GET | `/api/v1/ws/client` | Bearer query | 会员 WS（方案、钱包、聊天） |
| GET | `/api/v1/ws/admin` | Bearer query | 运营 WS（提现队列、方案监控） |

各 Phase REST 路径详见 [`integration-plan.md`](docs/integration-plan.md) Phase 1～5 表格与 OpenAPI。

统一响应：`{ "code": 0, "message": "ok", "data": { ... } }`（见 `internal/apix`）。

## 第三方挂机（Guaji · T0）

`.env` 配置 `GUAJI_*`（见 `.env.example`）。本地连通性：

```bash
cd backend
# .env 中 GUAJI_ENABLED=true，可选 GUAJI_TEST_USERNAME/PASSWORD
make guaji-smoke
```

适配层代码：`internal/guaji/`（登录、余额、WS 匿名探测）。详见 [`third-party-guaji-integration-plan.md`](docs/third-party-guaji-integration-plan.md)。

## 演示账号

| 端 | account | password | 说明 |
|----|---------|----------|------|
| Client | `vs8888` | `vs8888` | 查 `members` 表；种子见 `00009_seed_demo_members.sql`（`member_no` = `M00001`） |
| Admin | `admin` | `admin123` | 查 `admin_users` 表；种子见 `00057`/`00058`；**DB 不可用时**回退 `ADMIN_DEMO_*` |

Client 在 DB 可用时不再使用 `CLIENT_DEMO_*`；Admin 同理。

## 后台 Worker

`.env` 中 `SCHEME_WORKER_ENABLED=true` 时，定时：

- 方案实例结算（real/sim 分路）
- `pending` 投注订单 ↔ `lottery_draws` 开奖结算
- 新开奖写入后 WS 推送 `public.draw.result`

## 目录

```
backend/
  cmd/server/          # 入口
  cmd/migrate/         # goose 迁移
  cmd/guaji-smoke/     # 第三方连通性 smoke
  internal/
    apix/              # 统一响应包
    auth/              # JWT 登录
    config/            # 环境变量
    guaji/             # 第三方 Hash 适配层（T0+）
    db/                # pgx + sqlc
    handler/           # HTTP handlers
    ws/                # WebSocket Hub
    schemes/           # 方案域 + Worker
    ...
  migrations/          # SQL 迁移
  openapi/             # OpenAPI 契约
  contracts/           # TS DTO
  docs/                # 接入与 WS 文档
```

## 联调

Client / Admin `.env.local` 示例见各端 `.env.example` 与 [`integration-checklist.md`](docs/integration-checklist.md) §0。

CORS 已允许 `localhost:5173`（client）与 `5174`（admin）。
