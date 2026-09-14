## Context

托盘菜单当前含三项：打开主界面 / 立即签到（全部）/ 退出。批量签到在主界面已有入口（`App.CheckinAll`，前端 `bindings.ts` 调用），托盘项冗余。

## Goals / Non-Goals

**Goals**
- 托盘菜单精简为「打开主界面 / 退出」。

**Non-Goals**
- 不删除 `App.CheckinAll()` 及其 Wails 绑定，不改前端批量签到入口。
- 不改托盘菜单其他项、tooltip、双击打开、退出逻辑。

## Decisions

**决策 1：只删托盘入口，保留 `CheckinAll` 方法**
- `CheckinAll` 仍被前端使用，删除会破坏主界面批量签到。
- 仅移除 `cmdCheckinAll` 常量、`onCheckinAll` 回调字段/参数、菜单项与 `wndProc` 分支。

## Risks / Trade-offs

- [菜单索引/命令 ID 变化] → 命令 ID 用 `iota` 生成，删除 `cmdCheckinAll` 后 `cmdQuit` 值从 3 变 2，但菜单项与分发同源，无外部依赖。安全。

## Migration Plan

无。

## Open Questions

无。
