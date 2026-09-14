## Why

托盘菜单的「立即签到（全部）」与主界面批量签到入口重复，且托盘菜单应保持精简（打开 / 退出）。移除该菜单项，降低误触风险。

## What Changes

- 从托盘菜单移除「立即签到（全部）」项及其回调链路（`cmdCheckinAll`、`onCheckinAll`）。
- 保留 `App.CheckinAll()` 绑定方法与前端批量签到入口，仅去掉托盘入口。

## Capabilities

### New Capabilities

<!-- 无 -->

### Modified Capabilities

- `tray-integration`: 「托盘菜单与 tooltip」要求的菜单项由「打开主界面 / 立即签到（全部）/ 退出」改为「打开主界面 / 退出」，并移除「托盘触发全部签到」场景。

## Impact

- **代码**：`tray_windows.go`、`tray_stub.go`、`app.go`（`newTray` 少一个参数）。
- **文档**：`docs/PRD.md` F8 菜单项列表。
- **不变**：`App.CheckinAll()` 及其 Wails 绑定、前端批量签到按钮、tooltip、单实例、自启。
