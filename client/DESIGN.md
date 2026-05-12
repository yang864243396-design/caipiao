# 设计系统文档：数字精算主义 (Digital Actuarialism)

> **本仓库默认设计规范。** `client/` 下前端界面的新增与改版应以本文为准；完整细节见以下章节。

## 1. 核心理念与北极星指标 (Creative North Star)

本设计系统旨在为交易与游戏管理平台构建一个**「通透、精准、具有呼吸感」**的数字生态。我们拒绝平庸的「模板化」界面，转而追求一种**「高级社论感 (High-End Editorial)」**的视觉表达。

**北极星指标：透明度与层级意识**

我们将 UI 视为一系列物理层叠的精细载体，而非扁平的像素集合。通过有意的**不对称布局 (Intentional Asymmetry)**、**重叠元素**和**大比例字重对比**，打破传统网格的沉闷，营造出一种类似「App Store」的高端零售质感。每一处留白都是为了引导注意力的流动，每一道阴影都是为了定义空间的深浅。

---

## 2. 色彩体系 (Colors)

本系统的色彩逻辑基于**「无线化 (No-Line Rule)」**原则。我们严禁使用 1px 的实线进行区域分割。

### 核心色彩应用

- **Primary (#0050cb / #0066ff)：** 科技蓝。不仅是行动点，更是专业主义的象征。建议在主要 CTA 或关键数据展示中使用 `primary` 到 `primary_container` 的微弱渐变，以增加视觉的「灵魂」深度。
- **Surface 层级 (Tonal Layering)：**
  - **背景 (Surface - #f7f9fb)：** 整体画板的底色。
  - **容器 (Surface Container Lowest - #ffffff)：** 用于最核心的内容卡片，模拟纸张的洁净感。
  - **嵌套 (Surface Container Low / High)：** 用于区分卡片内部的次级区域，通过色调偏移而非线条来定义边界。

### 设计准则

- **「无线化」规则：** 绝对禁止使用 1px 边框切分区块。必须通过背景色阶的变化（例如在 `surface` 背景上放置 `surface-container-low` 区域）来产生自然的边界感。
- **磨砂玻璃效应 (Glassmorphism)：** 对于浮动面板（如交易即时反馈或悬浮菜单），必须使用半透明的 `surface` 色彩并配合 `backdrop-blur` (20px–40px)，使背景色隐约透出，增强界面的整合感。

---

## 3. 字体系统 (Typography)

我们采用混合字体策略，确保在处理复杂交易数据时具备极高的可读性与美感。

- **中文首选：** Noto Sans SC (思源黑体)。
- **数字与西文：** Inter（用于正文/标签）或 Plus Jakarta Sans（用于大标题）。

### 层级逻辑

- **Display / Headline：** 追求极端的字重对比。大标题使用 **Plus Jakarta Sans** 的 Bold / ExtraBold 字重，配合紧凑的字间距，展现权威感。
- **Body：** 保持在 14px (body-md) 到 16px (body-lg)，行高设定为 1.6–1.8，确保在阅读长篇游戏日志或交易条款时具备出色的呼吸感。
- **Label：** 针对交易界面的微小数据，使用 `label-sm` (11px)，但必须配合全大写（西文）或字间距微调，确保精致度。

---

## 4. 深度与高度 (Elevation & Depth)

深度不是通过「投影」堆砌的，而是通过**「影之韵律」**实现的。

- **环境阴影 (Ambient Shadows)：** 严禁使用深灰色或高不透明度的投影。所有投影必须是极其弥散的（Blur > 20px），不透明度控制在 4%–8% 之间。阴影颜色应混入少量 `on-surface` 的色调，使其看起来像是自然光照下的环境遮挡。
- **分层原则 (The Layering Principle)：**
  - 底部：`surface`（基础背景）
  - 中部：`surface-container-low`（内容分组）
  - 顶部：`surface-container-lowest` (#ffffff)（交互卡片）
- **幽灵边框 (Ghost Border)：** 若因无障碍需求必须使用边框，请使用 `outline-variant` 并将不透明度降至 10%–20%。它应该是一种「若有若无」的存在，而非视觉焦点。

---

## 5. 组件设计规范 (Components)

### 按钮 (Buttons)

- **Primary：** 使用 `primary` 到 `primary_container` 的垂直渐变。圆角固定为 `md (0.75rem)`，营造亲和力与现代感的平衡。
- **Secondary：** 仅使用 `primary_fixed_dim` 背景，文本使用 `on_primary_fixed`。

### 输入字段 (Input Fields)

- **状态表达：** 默认状态不显示全闭合边框，仅显示底部的微弱 `outline_variant` 或浅色背景填充。焦点状态下，通过 `primary` 色的 2px 底部光晕或整体容器的微弱阴影提升来表达。

### 卡片与列表 (Cards & Lists)

- **禁止使用分割线：** 列表项之间严禁使用传统的分割线。应使用垂直间距 (Spacing Scale) 或交替的背景色块 (Zebra striping at 2% opacity) 来区分。
- **交易管理特供：** 引入「数据磁贴 (Data Tiles)」，利用 `surface-container-high` 作为背景，将核心数值（如胜率、交易额）以大字号 `display-sm` 呈现，建立信息层级。

### 纸片/标签 (Chips)

- 用于过滤游戏类型或交易状态。必须使用 `full` 圆角，背景色与文字色保持低对比度，仅在激活状态下使用 `primary`。

---

## 6. 执行准则 (Do's and Don'ts)

### 鼓励 (Do)

- **拥抱非对称：** 在仪表盘布局中，允许左右侧边栏宽度不等，创造动态平衡。
- **留白即功能：** 将留白视为一种「组件」，用于引导用户从复杂的交易数据中解脱。
- **色彩暗示：** 仅在关键状态（错误/警告/成功）使用 `error` 或 `tertiary` 颜色，保持界面的纯净度。

### 避免 (Don't)

- **禁止使用 100% 黑色：** 所有的文字和边框应使用 `on_surface` 或 `outline` 系列色调，避免视觉割裂。
- **禁止过度装饰：** 拒绝任何与功能无关的装饰性线条、图标投影。
- **禁止拥挤：** 任何容器边缘与内容的边距不得小于 `lg (1rem)`。

---

通过遵循此系统，我们将不仅仅是在构建一个管理工具，而是在雕琢一个高效、优雅且充满未来感的数字资产中心。

---

## 7. 在本仓库中的落地

| 项目 | 说明 |
|------|------|
| **默认依据** | `client/` 下 Vue 应用的新增与改版 UI **默认遵循本文**「数字精算主义」。 |
| **代码位置** | `client/src/`（组件、视图、全局样式）。 |
| **字体加载** | `client/index.html` 已引入 Inter、Plus Jakarta Sans、Noto Sans SC；与 §3 对齐。 |
| **渐进迁移** | 既有页面可能仍含历史样式；改版时优先按 §2–§6 收敛，而非一次性全量重写。 |

## 8. Element Plus（客户端组件库）

| 项目 | 说明 |
|------|------|
| **优先策略** | `client/` 内**优先使用 [Element Plus](https://element-plus.org/)** 提供的表格、表单、按钮、反馈、导航等组件，减少自建 markup 与样式维护量。 |
| **工程集成** | 已配置 `unplugin-vue-components` + `unplugin-auto-import`（见 `vite.config.ts`），按需引入样式；全局中文与主题变量见 `App.vue`（`el-config-provider`）、`src/styles/element-plus-theme.css`。 |
| **与设计系统对齐** | 主色与 §2 一致（`--el-color-primary` 等）；列表/卡片场景优先 `el-table` 的 `stripe`、幽灵边框语义，避免多余硬分割线与 §2「无线化」冲突。 |
| **el-table 布局** | **禁止**表体/表头出现横向滚动条；列宽用 **`min-width`** 配合 `fit`（默认）在容器内分配，内容过长在单元格内**换行**。全局样式见 `src/styles/el-table-layout.css`。若业务极少数场景必须横滑，在 `<el-table>` 上增加 **`el-table--scroll-x`**。 |
| **例外** | 强定制视觉块（如 Bento、走势图 SVG、开奖球）可保留原生结构 + 局部样式；新建列表/表单/对话框**默认**走 Element Plus。 |

*如需在 Cursor 中自动提示本规范，见仓库 `.cursor/rules/digital-actuarialism-design.mdc`。*
