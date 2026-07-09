# 后端（`backend/`）



Go HTTP 服务 + OpenAPI 契约。Client / Admin 通过 `/api/v1` 对接。



## 文档与契约



| 路径 | 说明 |

|------|------|

| [docs/integration-plan.md](docs/integration-plan.md) | **接入方案计划**（分阶段、Client/Admin 对照） |

| [docs/go-server.md](docs/go-server.md) | **Go 服务** 启动、目录、Phase 0 路由 |

| [docs/modules/schemes.md](docs/modules/schemes.md) | 方案域产品定案 + OpenAPI 对照 |

| [docs/modules/cloud-bet-records.md](docs/modules/cloud-bet-records.md) | 云端投注记录模块 |

| [openapi/openapi.yaml](openapi/openapi.yaml) | **OpenAPI 3.0** 接口定义 |

| [contracts/](contracts/) | TypeScript DTO（前端对照） |



## 快速启动（Go）



```bash

cd backend

make tidy

make run

```



默认 `http://127.0.0.1:8080/api/v1`。演示账号见 [docs/go-server.md](docs/go-server.md)。



## 环境变量（Client / Admin）



```env

VITE_API_BASE_URL=http://127.0.0.1:8080/api/v1

# 演示数据由 PostgreSQL seed 提供（migrate 后）

```



## 实现顺序



1. **Phase 0**：`/health`、登录、维护开关 — **Go 脚手架已就绪**
2. **Phase 1**：云端中心 + 投注记录 — **Go 后端 + Client 已联调**

3. Phase 2：会员资产与订单

4. Phase 3：方案域 / 跟单

5. Phase 5：Admin 运营面



## 技术选型

- **Go 1.22+** · `cmd/server` · 标准库 `net/http` · JWT（`golang-jwt/jwt`）
- **PostgreSQL 18** · **pgx** · **sqlc** · **goose** — 见 [docs/database.md](docs/database.md)
- 契约单一事实来源：`openapi/openapi.yaml`



详见 [docs/integration-plan.md](docs/integration-plan.md)。

