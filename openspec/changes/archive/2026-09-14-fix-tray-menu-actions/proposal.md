## Why

托盘右键菜单的三个命令（打开主界面 / 立即签到（全部）/ 退出）点击后全部无效。根因：`showMenu()` 给 `TrackPopupMenu` 传了 `TPM_RETURNCMD`，选中结果通过函数返回值返回、系统不再投递 `WM_COMMAND`，而代码只在 `WM_COMMAND` 分支里处理命令，返回值被丢弃。这是从 health-tool 半截移植导致的——flag 抄了，返回值分发漏了。

同时移植时漏掉了 health-tool 已验证的托盘图标恢复逻辑：explorer.exe 重启、睡眠恢复、会话解锁后不重挂图标，图标会消失且不再出现。

另外「退出」命令即使分发到位也无法退出：`beforeClose` 在 `MinimizeToTray` 开启时无条件返回 `true`，而 Wails Windows 的 `Frontend.Quit()` 会先调 `OnBeforeClose`、返回 `true` 即中止退出。结果是托盘图标被移除、进程仍在运行。health-tool 用 `quitRequested` 旁路解决，本仓库漏了。

## What Changes

- 托盘菜单改为处理 `TrackPopupMenu` 返回值（对齐 health-tool 已验证实现），删除永不触发的 `WM_COMMAND` 死代码。
- 注册 `TaskbarCreated` 广播消息，explorer.exe 重启后重新添加托盘图标。
- 睡眠恢复（`WM_POWERBROADCAST`）、会话解锁（`WM_WTSSESSION_CHANGE`）时重新添加托盘图标（现有钩子只转发 `scheduler.Wake()`，不重挂图标）。
- 托盘菜单「退出」增加主动退出旁路（`quitting` 标志），使 `beforeClose` 不再拦截退出；对齐 health-tool 的 `quitRequested`。
- 托盘图标仍沿用系统通用图标 `IDI_APPLICATION`（PRD F8 未要求自定义图标，不扩范围）。

## Capabilities

### New Capabilities

<!-- 无 -->

### Modified Capabilities

- `tray-integration`: 新增「托盘图标可用性恢复」要求（explorer 重启 / 睡眠恢复 / 会话解锁后图标重新可用）；原有「托盘菜单」要求的行为不变，本次修复使其真正生效。

## Impact

- **代码**：`tray_windows.go`（菜单分发、图标恢复、消息注册）、`app.go`（`quitting` 旁路）。
- **文档**：`docs/PRD.md` F8 澄清退出语义——退出仅从托盘菜单触发，且必须真正结束进程，不得只移除托盘图标或隐藏窗口。
- **平台**：仅 Windows；`tray_stub.go` 不受影响，Linux 下 `go test ./...` 仍可跑。
- **验证**：托盘为 `//go:build windows`，无法在 Linux 自动化回归；修复后需在 Windows 人工验证三个菜单命令与图标恢复。
- **依赖 / 存储 / 协议**：无变化。
