## Context

`tray_windows.go` 从 health-tool 的 `tray_windows.go` 移植而来，但移植不完整：

- health-tool 的 `showMenu()` 处理 `TrackPopupMenu` 的返回值并 `switch` 分发命令（`tray_windows.go:295`），全文件没有 `WM_COMMAND` 分支。
- 本仓库的 `showMenu()` 保留了 `TPM_RETURNCMD`，却丢弃返回值，另写了 `wmCommand` 分支（`tray_windows.go:266-281`）。`TPM_RETURNCMD` 下系统不投递 `WM_COMMAND`，该分支是死代码，三个命令全部无效。
- health-tool 在 `TaskbarCreated`、`WM_POWERBROADCAST`、`WM_WTSSESSION_CHANGE` 时调用 `addIcon()` 重挂图标；本仓库只注册了后两个钩子且只转发 `scheduler.Wake()`，不重挂图标，也没有 `TaskbarCreated`。

## Goals / Non-Goals

**Goals**
- 三个托盘菜单命令真正生效。
- 托盘图标在 explorer 重启 / 睡眠恢复 / 会话解锁后自动恢复。

**Non-Goals**
- 不在设置页新增退出入口；退出仅从托盘菜单触发（与更新后的 PRD F8 一致）。
- 不改用自定义托盘图标（沿用 `IDI_APPLICATION`）。
- 不改用 `WM_CONTEXTMENU`（health-tool 用 `WM_RBUTTONUP` 已验证）。
- 不把 `syscall.NewLazyDLL` 迁移到 `golang.org/x/sys/windows`（与本 bug 无关，避免扩大 diff）。
- 不为托盘写自动化测试（Windows-only，Linux 只跑 stub）。

## Decisions

**决策 1：菜单命令用返回值分发（方案 B），而非去掉 `TPM_RETURNCMD`（方案 A）**
- 方案 A（去掉 flag，靠 `WM_COMMAND`）改动更小，但会保留与 health-tool 不一致的第二套分发路径。
- 方案 B 与 health-tool 已验证实现逐字对齐，且符合 DESIGN.md「系统托盘照抄 health-tool 已验证实现」的约束。选 B。
- 采用 B 后删除 `wmCommand` 分支，避免两套分发逻辑。

**决策 2：图标恢复照抄 health-tool 的钩子集合**
- 注册 `TaskbarCreated` 窗口消息，收到后 `addIcon()`。
- 现有 `wmPowerBroadcast` / `wmWtsSessionChange` 分支在调用 `onWake()` 之外补 `addIcon()`。
- 菜单改为构建一次并复用（health-tool 做法），避免每次弹菜单重建。

**决策 3：退出用 `quitting` 旁路拦截，而非改 `beforeClose` 的默认语义**
- 现象：`MinimizeToTray=true` 时，Wails Windows `Frontend.Quit()` 先调 `OnBeforeClose`，返回 `true` 就中止退出（`frontend.go:459-462`）；托盘图标已 `remove()`，进程却活着。
- health-tool 用 `quitRequested atomic.Bool`：`beforeClose` 首行 `if quitRequested { return false }`，`requestQuit()` 置位后调 `runtime.Quit`。
- 本仓库照抄为 `App.quitting atomic.Bool`：`quit()` 置位再 `wruntime.Quit`，`beforeClose` 首行放行。不改「关闭窗口最小化到托盘」的默认行为。

**决策 4：`notifyIconData` 复用现有结构**
- 现有结构与 health-tool 布局一致（`uTimeoutOrVer` 对应 `uVersion`），`addIcon` 可复用同一份 `nid`，只需在重挂前确保字段正确。

## Risks / Trade-offs

- [重复 `Shell_NotifyIcon(NIM_ADD)` 可能产生重复图标] → Windows 对同一 `(hWnd, uID)` 的 NIM_ADD 是幂等覆盖，且 health-tool 已验证；无需额外去重。
- [Linux 无自动化回归] → 修复后需在 Windows 人工验证：三个菜单命令 + explorer 重启 + 锁屏解锁后图标仍在。
- [`trayInstance` 包级全局] → 本次不改；`addIcon` 通过接收者 `t` 调用，无新增全局依赖。

## Migration Plan

无数据/配置迁移。修复随下次 `wails build -platform windows/amd64` 生效。回滚 = 还原 `tray_windows.go`。

## Open Questions

无。
