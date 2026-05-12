# 目录说明

| 路径 | 说明 |
|------|------|
| `client/` | **用户端**（C 端）—— Vue3 + Vite + TS，精密终端/游戏大厅等，见 [client/README.md](client/README.md) |
| `admin/` | **管理后台**（B 端）—— Vue3 占位，端口 5174，见 [admin/README.md](admin/README.md) |
| `backend/` | **后端** —— 仅占位与说明，见 [backend/README.md](backend/README.md) |
| `skills/` | **技能资产索引**：`skills-lock.json`、[skills/README.md](skills/README.md)；**Cursor 实际加载路径** 仍为根目录 `.cursor/skills`、`.agents/skills`（勿改路径名） |
| `.cursor/` | Cursor 规则与项目级 skills（IDE） |
| `.agents/` | Agent 设计类 skills，与 [`.cursor/skills/SKILLS_INDEX.md`](.cursor/skills/SKILLS_INDEX.md) 配套 |

**根目录** `package.json` 提供：`dev:client` / `build:client` / `dev:admin` / `build:admin`（分别转发到 `client/`、`admin/`）。
