## Context

设置页（`frontend/src/views/SettingsView.vue`）目前有两张卡片：「开机自启」与「软件更新」。后者把「当前版本 vX.Y.Z」与更新操作按钮、GitHub 加速代理设置混在一起。应用是开源的，但界面无任何仓库入口；仓库根也没有 `LICENSE`。

约束（来自 DESIGN.md）：前端不使用 Wails 事件传业务状态，一律轮询绑定方法。打开外部链接已有先例：`app.go:67`、`app_account.go:42` 用 `wruntime.BrowserOpenURL`。前端侧 Wails runtime 包（`frontend/wailsjs/runtime`）已暴露 `BrowserOpenURL`，不需要新增 Go 绑定方法或重跑 `wails generate module`。

## Goals / Non-Goals

**Goals:**
- 补上仓库 `LICENSE`（MIT）。
- 设置页新增「关于」区块：开源地址（可点击跳转）、许可证、版本号。
- 把版本号从「软件更新」区块移出，消除重复与归属混乱。

**Non-Goals:**
- 不做「复制仓库地址」按钮、不加 issue/star 徽章、不加遥测或反馈表单。
- 不改 `app*.go`、不改托盘菜单、不加新的 Go 绑定方法。
- 不在关于区块展示构建时间、提交哈希等额外元信息。

## Decisions

**1. 用已有的前端 `BrowserOpenURL`，而不是新增 Go 方法。**

`@wailsjs/runtime` 已含 `BrowserOpenURL`，前端一行调用即可。新增 Go 方法要走 `wails generate module`（还有 git 权限位副作用），对一个静态外链没有收益。备选：加 `OpenGitHub()` 包装以便未来集中管理外链——当前只有一条链接，YAGNI，需要时再加。

**2. 地址写死为常量，不做配置。**

仓库地址不会变；写成组件内的 `const` 常量。备选：放进 `Settings` 模型——会引入无意义的持久化字段与迁移噪音。

**3. 关于区块作为设置页第三张卡片，位于操作按钮下方。**

设置页用固定的卡片堆叠布局（`max-w-2xl`），保持一致：`开机自启` → `软件更新` → `关于`。不放进「软件更新」卡片内部，否则又回到混装。

**4. `LICENSE` 用标准 MIT 全文，年份 2026，版权人 hosea3000。**

未签名、未在 UI 之外额外声明。

## Risks / Trade-offs

- [地址写死，写错只能发新版本修] → 已核对 `go.mod` 的 module path 与 `git remote -v` 均为 `hosea3000/workbuddy-checkin`。
- [关于区块版本号与更新区块版本号重复的问题被"移动"而非"共享"] → 采纳移动方案：版本只出现在关于区块，更新区块只留状态文案与按钮。
- [老版本用户看不到关于区块] → 随下次发布自然覆盖，无需迁移。

## Migration Plan

纯前端 + 仓库文件变更，无数据迁移。回滚即 revert 提交。发布走既有 `v*` tag 流水线。

## Open Questions

（无）
