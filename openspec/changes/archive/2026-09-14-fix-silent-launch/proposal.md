## Why

手动启动（双击 exe）时应用不显示主窗口，直接静默进托盘；首次启动同样可能如此。根因是启动可见性判断错误：

- `main.go` 恒设 `StartHidden: true`，靠 `startup` 里 `if !isAutoStartEnabled() { showWindow() }` 补救。Wails 的 `OnStartup` 在 goroutine 中执行，与窗口显示流程并发，`start()` 遇到 `StartHidden` 会跳过 `ShowWindow`（`frontend.go:980`），时序不定 → 窗口可能一直隐藏。
- `isAutoStartEnabled()`（HKCU Run 项是否存在）被当作「本次是否由自启拉起」的信号。一旦用户开启自启，之后手动双击 exe 也会静默进托盘。

health-tool 用命令行标记 `--hidden` 区分两种启动方式，从根上避免上述问题。

## What Changes

- 新增 `--hidden` 命令行标记解析（`hiddenFromArgs`），`main.go` 的 `StartHidden` 改由该标记决定。
- `setAutoStart` 写入 HKCU Run 的启动命令改为 `"exe" --hidden`，确保只有自启拉起才静默。
- 删除 `startup` 里基于 `isAutoStartEnabled()` 的条件显示窗口；手动启动时由 Wails 直接显示。
- 删除因此不再使用的 `isAutoStartEnabled`（windows + stub）。
- 新增 `autostart.go`（无构建标签）承载跨平台纯逻辑 `hiddenFromArgs` / `buildAutoStartCommand`，并补单测。

## Capabilities

### New Capabilities

<!-- 无 -->

### Modified Capabilities

- `tray-integration`: 静默启动要求补充——仅由开机自启（带 `--hidden`）拉起时静默；手动启动 SHALL 显示主窗口。

## Impact

- **代码**：`main.go`、`app.go`、`autostart_windows.go`、`autostart_stub.go`，新增 `autostart.go` 与 `autostart_test.go`。
- **行为**：手动启动始终显示窗口；自启启动静默进托盘。修复「开启自启后双击无反应」。
- **平台**：`hiddenFromArgs` 跨平台可测（Linux `go test` 覆盖）；其余不变。
