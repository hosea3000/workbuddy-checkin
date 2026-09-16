## Context

`App.vue` 的侧栏是 `w-40`（160px）固定宽度，导航项为「图标 + 文字」。窗口 820×640、最小 720×520，折叠后内容区可多出约 104px。前端目前无任何本地存储用法。

## Goals / Non-Goals

**Goals:**
- 一键在「展开（160px）/ 折叠（56px）」间切换，折叠态仍可导航。
- 折叠状态跨启动记住，首次默认展开。

**Non-Goals:**
- 不做窗口变窄自动折叠。
- 不做悬浮临时展开（overlay）。
- 不做图标化 tooltip 的自定义浮层（用原生 `title`）。
- 不改 Go / `model.Settings` / wailsjs 绑定。

## Decisions

### D1 折叠状态存 localStorage，键名 `sidebar-collapsed`

存 `'1'` / `'0'`，读取时任何非 `'1'` 值都视为展开（含 `null`），天然满足「首次默认展开」，无需额外判断首次。

考虑过的替代：写进 `model.Settings`。否决——那会让需要备份与损坏自愈的设置文件多一个纯显示字段，且要动 Go、重生成绑定、补测试。UI 偏好放 UI 层。

### D2 折叠只改宽度与内边距，导航结构不变

`w-40 ↔ w-14`（160px ↔ 56px）加 `transition-[width]`。折叠时隐藏文字、图标居中；`title` 始终挂上（展开态无害，折叠态是唯一提示）。

不用条件渲染两套导航 —— 一套结构、按状态切 class，改动最小且不会出现两份需要同步的点击逻辑。

### D3 开关放在侧栏底部

折叠态仅 56px 宽，按钮需 `w-full` 命中整条。放底部不会随三个导航项的数量变化而位移，语义也清楚（贴着侧栏边界）。

图标用 `ChevronLeft` / `ChevronRight` 二选一，`title` 同步为「折叠」/「展开」。

### D4 选中态与点击在两种状态完全一致

选中态（`bg-indigo-50` 等）与 `router.push` 逻辑不因折叠改变，仅去掉文字节点。折叠态下用户仍需看出当前页。

## Risks / Trade-offs

- **[原生 title 延迟]** Wails WebView 的 `title` 提示有延迟且样式不可控 → 接受；实测若不可用，再考虑自定义浮层（属后续独立变更）。
- **[localStorage 不可用]** WebView 极端情况下写入失败 → 读取已有 `try/catch` 兜底为展开；写入失败只是不记住，不影响本次会话的切换。
- **[动画期重排]** `transition-[width]` 期间内容区会逐帧重排 → 宽度过渡较短（约 150ms），可接受；如觉卡顿可去掉过渡只做瞬切。

## Migration Plan

纯前端、无数据迁移。`localStorage` 无值即展开，老用户升级后行为与现状一致。回滚 = 还原 `App.vue`。

## Open Questions

无。
