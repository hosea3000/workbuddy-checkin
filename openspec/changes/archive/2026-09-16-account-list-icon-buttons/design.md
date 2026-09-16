## Context

`AccountCard.vue` 行尾目前有四个操作元素：余额刷新（已是图标）、「设为当前凭证」文字按钮、「立即签到」文字按钮（relogin 时替换为红色「重新登录」文字按钮）、删除图标按钮。文字按钮占据横向空间，窄窗口下挤压昵称与状态区域；删除按钮已确立「图标 + `title`」的本仓范式。

同时，`relogin` 分支的「重新登录」按钮与「立即签到」调用的是同一个 action（`checkinNow`），其唯一作用是换文案与配色，不构成独立能力。

约束（来自 DESIGN.md 与 AGENTS.md）：技术栈锁定 Wails v2 + Vue3 + Tailwind v4，图标库为已安装的 `@lucide/vue`，不得引入新依赖；前端不使用 Wails 事件传业务状态。

## Goals / Non-Goals

**Goals:**
- 两个文字按钮改为图标按钮，文案移到 hover 原生 `title`。
- 移除 relogin 卡片上的「重新登录」按钮，同时保留该状态的其余全部视觉与文案信号。
- 图标语义不与既有元素冲突。

**Non-Goals:**
- 不新增绑定方法、不改 `accounts` store、不改 `relogin_required` 状态机。
- 不改 `StatusBadge`、托盘 tooltip、`CheckinSummary()`。
- 不引入自定义 tooltip 组件或任何新依赖。
- 不改余额刷新按钮。

## Decisions

### 图标选型

| 动作 | 图标 | 理由 |
|---|---|---|
| 设为当前凭证 | `Pin` | 行内已有 `[✓ 当前凭证]` 徽章使用 `Check`，若此处也用对勾会造成同形状表达两件事；`Pin`（置顶/固定）与「当前」语义贴合 |
| 立即签到 | `CalendarCheck` | App.vue 侧边导航已用 `CalendarCheck` 代表账号/签到页，语义一致，且与 `Pin` 轮廓差异明显 |

替代方案：签到用 `CheckCheck`（与 StatusBadge 的「已签到」语义混淆）、`CircleCheck`（与徽章对勾仍偏相近），均弃用。

### 签到中状态

沿用 StatusBadge 的既有做法：`checking` 为真时图标区渲染 `Loader2`（`animate-spin`），按钮 `:disabled="checking"`，`title` 切换为「签到中…」。不新增文案。

### 移除「重新登录」按钮

直接删除 `AccountCard.vue` 的 `v-if`/`v-else` 分支，签到按钮不再需要按 `status` 条件渲染，统一为单个图标按钮。relogin 卡片保留：红色 ring、红底头像、`需重新登录` 徽章、「凭证已过期，自动签到已跳过该账号」文案。重新登录入口改由页面顶部既有「添加账号」按钮（`startLogin` → 重走 OAuth）承担。

替代方案：把红色徽章做成可点击入口——被否，徽章是可点击性不明显的展示元素，且会增加事件耦合。

### 复用既有样式范式

两个新图标按钮直接沿用删除按钮的 class（`shrink-0 p-1.5 rounded-lg text-slate-400 hover:text-... hover:bg-... cursor-pointer`），仅 hover 色按语义微调，保证与删除按钮视觉一致。

## Risks / Trade-offs

- [原生 `title` 提示约 500ms 延迟且样式不可控] → 与既有删除按钮一致，接受；不为此引入 tooltip 组件。
- [删除「重新登录」后 relogin 用户的恢复路径变隐蔽] → 红色徽章 + 过期文案 + 顶栏「有账号需重新登录」三重提示仍在，且「添加账号」入口常驻；接受。
- [图标语义不直观，需 hover 才能读懂] → 用 `Pin`/`CalendarCheck` 这两个高度通用的图标降低认知成本；文案差异（签到/重新登录）消失，属本次改动的预期取舍。
- [`docs/PRD.md` 与实现不一致] → 在 tasks 中显式包含文档修订项。
