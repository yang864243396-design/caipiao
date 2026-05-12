# Skills 资产（本目录）

- **`skills-lock.json`**：与 `.agents/skills` 中各包对应的版本/哈希锁（自 `short` 仓库同步逻辑）。
- **Cursor 可发现的技能本体**（勿删、勿改仓库根下目录名，否则 IDE 可能无法加载）：
  - `../.cursor/skills/` — 项目级，含 `SKILLS_INDEX.md`、`ui-ux-pro-max` 等。
  - `../.agents/skills/` — 设计类 Impeccable / Taste 等，由 `SKILLS_INDEX` 做路由，避免一次读太多。

**入口**：从 `../.cursor/skills/SKILLS_INDEX.md` 的决策树选 skill，再点进对应子目录的 `SKILL.md`。
