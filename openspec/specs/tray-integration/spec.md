# tray-integration Specification

## Purpose

应用常驻系统托盘，提供菜单与 tooltip、单实例、开机自启与静默启动，并保证非 Windows 平台可测试。

## Requirements

### Requirement: 托盘驻留

应用 SHALL 在 Windows 常驻系统托盘；Windows 上关闭主窗口 SHALL 隐藏到托盘而非退出，退出 SHALL 仅能从托盘菜单触发，且 SHALL 真正结束进程——不得只移除托盘图标或隐藏窗口。macOS SHALL NOT 提供托盘：关闭主窗口 SHALL 直接退出应用。

#### Scenario: Windows 关闭窗口不退出
- **WHEN** 用户在 Windows 关闭主窗口
- **THEN** 应用隐藏到托盘并继续调度签到

#### Scenario: Windows 托盘退出
- **WHEN** 用户在托盘菜单选择「退出」
- **THEN** 进程退出、托盘图标消失

#### Scenario: Windows 双击托盘图标
- **WHEN** 用户双击托盘图标
- **THEN** 主界面显示

#### Scenario: macOS 关闭窗口即退出
- **WHEN** 用户在 macOS 关闭主窗口
- **THEN** 应用退出，进程结束，不残留后台进程

### Requirement: 托盘菜单与 tooltip
托盘菜单 SHALL 包含「打开主界面 / 退出」；tooltip SHALL 显示 `workbuddy-checkin — 今日已签到 X/Y` 或 `— 有账号需重新登录`。系统 SHALL 提供 `CheckinSummary()` 返回 `{checkedIn, total, reloginRequired}` 供 tooltip 与顶栏计数使用。

#### Scenario: 打卡计数 tooltip
- **WHEN** 2 个账号中 2 个今日已签到
- **THEN** tooltip 显示「今日已签到 2/2」

#### Scenario: 有待重登提示
- **WHEN** 存在 `relogin_required` 账号
- **THEN** tooltip 显示「有账号需重新登录」

### Requirement: 托盘图标可用性恢复
系统 SHALL 在托盘图标失效或消失后自动重新添加，无需用户重启应用。至少覆盖：explorer.exe 重启、系统从睡眠/休眠恢复、用户会话解锁。

#### Scenario: explorer 重启后恢复
- **WHEN** Windows explorer.exe 重启并广播 `TaskbarCreated`
- **THEN** 应用重新添加托盘图标，tooltip 与菜单保持可用

#### Scenario: 睡眠恢复后恢复
- **WHEN** 系统从睡眠/休眠恢复（`WM_POWERBROADCAST` 的 `PBT_APMRESUMEAUTOMATIC` / `PBT_APMRESUMESUSPEND`）
- **THEN** 应用重新添加托盘图标

#### Scenario: 会话解锁后恢复
- **WHEN** 用户会话解锁（`WM_WTSSESSION_CHANGE` 的 `WTS_SESSION_UNLOCK`）
- **THEN** 应用重新添加托盘图标

### Requirement: 单实例
应用 SHALL 保证单实例：重复双击 exe 只唤起已有窗口，SHALL NOT 启动第二个调度器。

#### Scenario: 重复启动
- **WHEN** 应用已在运行且用户再次双击 exe
- **THEN** 已有窗口被唤起、不产生新进程调度器

### Requirement: 开机自启

系统 SHALL 提供开机自启，支持设置页开关与首次运行向导勾选；自启状态 SHALL 以系统实际状态为准。Windows SHALL 通过 HKCU Run 实现（无需管理员）；macOS SHALL 通过写入 `~/Library/LaunchAgents/` 下的 LaunchAgent plist 并经 `launchctl` 注册/注销实现。卸载 = 删程序 + 关闭自启开关。

#### Scenario: Windows 开启自启
- **WHEN** 用户在 Windows 开启「开机自启」
- **THEN** HKCU Run 写入应用启动命令

#### Scenario: Windows 关闭自启
- **WHEN** 用户在 Windows 关闭「开机自启」
- **THEN** HKCU Run 项被移除

#### Scenario: macOS 开启自启
- **WHEN** 用户在 macOS 开启「开机自启」
- **THEN** LaunchAgent plist 被写入并经 `launchctl` 注册

#### Scenario: macOS 关闭自启
- **WHEN** 用户在 macOS 关闭「开机自启」
- **THEN** LaunchAgent 被注销、plist 被移除

#### Scenario: 自启状态以实际为准
- **WHEN** 用户通过系统外部方式（如系统设置的登录项）改动自启后打开设置页
- **THEN** 「开机自启」显示系统实际状态

### Requirement: 静默启动

应用 SHALL 仅在由开机自启拉起时静默启动（不弹窗），手动启动 SHALL 显示主窗口，不得因已启用自启而静默。启动方式 SHALL 通过命令行标记 `--hidden` 区分。macOS 静默启动后 SHALL 能由用户再次启动应用经单实例锁唤出主窗口。

#### Scenario: 自启静默
- **WHEN** 应用由开机自启启动（启动命令带 `--hidden`）
- **THEN** 不显示主窗口

#### Scenario: 手动启动显示窗口
- **WHEN** 用户手动启动应用（无 `--hidden`），即使已启用开机自启
- **THEN** 主窗口正常显示

#### Scenario: macOS 静默启动后唤起
- **WHEN** macOS 应用以 `--hidden` 运行中，用户再次启动应用
- **THEN** 单实例锁命中，已有窗口被唤出显示

### Requirement: 非 Windows 平台可测试

系统 SHALL 为托盘、自启、通知等系统集成提供平台构建分支，保证 Linux 下 `go test ./...` 可运行：无对应实现的平台 SHALL 提供 stub（`//go:build` 排除已实现平台）。

#### Scenario: Linux 运行测试
- **WHEN** 在 Linux 执行 `go test ./...`
- **THEN** 编译并运行 stub 分支，测试通过

#### Scenario: macOS 编译
- **WHEN** 为 darwin 构建
- **THEN** 编译 macOS 专用实现而非 stub
