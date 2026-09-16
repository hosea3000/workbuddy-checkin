# tray-integration Specification

## Purpose

应用常驻系统托盘，提供菜单与 tooltip、单实例、开机自启与静默启动，并保证非 Windows 平台可测试。

## Requirements

### Requirement: 托盘驻留

应用 SHALL 在 Windows 常驻系统托盘、在 macOS 常驻菜单栏；两平台上关闭主窗口 SHALL 隐藏窗口而非退出应用，调度器 SHALL 继续运行；退出 SHALL 仅能从托盘/菜单栏菜单或系统退出入口（⌘Q）触发，且 SHALL 真正结束进程——不得只移除图标或隐藏窗口。

#### Scenario: Windows 关闭窗口不退出
- **WHEN** 用户在 Windows 关闭主窗口
- **THEN** 应用隐藏到托盘并继续调度签到

#### Scenario: Windows 托盘退出
- **WHEN** 用户在托盘菜单选择「退出」
- **THEN** 进程退出、托盘图标消失

#### Scenario: Windows 双击托盘图标
- **WHEN** 用户双击托盘图标
- **THEN** 主界面显示

#### Scenario: macOS 关闭窗口不退出
- **WHEN** 用户在 macOS 关闭主窗口
- **THEN** 主窗口隐藏，应用继续在菜单栏常驻并调度签到，进程不结束

#### Scenario: macOS 菜单栏退出
- **WHEN** 用户在 macOS 菜单栏菜单选择「退出」
- **THEN** 进程退出、菜单栏图标消失

#### Scenario: macOS ⌘Q 退出
- **WHEN** 用户在 macOS 按 ⌘Q
- **THEN** 进程退出，不残留后台进程

#### Scenario: macOS 关闭窗口后仍能补签
- **WHEN** macOS 用户关闭主窗口，此后到达每小时巡检时刻
- **THEN** 调度器照常执行，未签到账号被补签

### Requirement: 托盘菜单与 tooltip

菜单 SHALL 包含「打开 / 退出」；Windows 为系统托盘菜单，macOS 为菜单栏菜单且左键单击图标即弹出。菜单 SHALL 额外提供「模型代理」开关项，其状态与设置页开关一致。tooltip SHALL 显示 `workbuddy-checkin — 今日已签到 X/Y` 或 `— 有账号需重新登录`。系统 SHALL 提供 `CheckinSummary()` 返回 `{checkedIn, total, reloginRequired}` 供 tooltip 与顶栏计数使用。

#### Scenario: 打卡计数 tooltip
- **WHEN** 2 个账号中 2 个今日已签到
- **THEN** tooltip 显示「今日已签到 2/2」

#### Scenario: 有待重登提示
- **WHEN** 存在 `relogin_required` 账号
- **THEN** tooltip 显示「有账号需重新登录」

#### Scenario: macOS 左键单击弹菜单
- **WHEN** 用户在 macOS 左键单击菜单栏图标
- **THEN** 弹出含「打开」「退出」的菜单

#### Scenario: macOS 菜单打开主界面
- **WHEN** 用户在 macOS 菜单栏菜单选择「打开」
- **THEN** 主窗口显示并获得焦点

#### Scenario: 托盘切换模型代理
- **WHEN** 用户在托盘/菜单栏菜单点击「模型代理」开关
- **THEN** 代理服务按新状态启动或停止，设置页开关同步反映该状态

#### Scenario: 托盘菜单反映代理状态
- **WHEN** 用户打开托盘/菜单栏菜单
- **THEN** 「模型代理」项显示当前是否启用

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

系统 SHALL 为托盘、自启、通知等系统集成提供平台构建分支，保证 Linux 下 `go test ./...` 可运行：无对应实现的平台 SHALL 提供 stub（`//go:build` 排除已实现平台）。托盘 SHALL 为 Windows 与 macOS 分别提供实现，stub SHALL 仅用于其余平台。

#### Scenario: Linux 运行测试
- **WHEN** 在 Linux 执行 `go test ./...`
- **THEN** 编译并运行 stub 分支，测试通过

#### Scenario: macOS 编译
- **WHEN** 为 darwin 构建
- **THEN** 编译 macOS 菜单栏实现（cgo + Objective-C）而非 stub

#### Scenario: Windows 编译
- **WHEN** 为 windows 构建
- **THEN** 编译 Windows 托盘实现而非 stub

### Requirement: macOS 菜单栏图标与激活策略

macOS SHALL 在菜单栏常驻一个模板（Template）图标以适配深浅色外观，SHALL NOT 显示 Dock 图标；应用 SHALL 以 `Accessory` 激活策略运行。图标在应用整个生命周期内 SHALL 保持可见，直至用户退出。

#### Scenario: 无 Dock 图标
- **WHEN** macOS 应用启动后观察 Dock
- **THEN** Dock 中不出现应用图标，仅菜单栏出现图标

#### Scenario: 深浅色适配
- **WHEN** 用户切换系统为深色外观
- **THEN** 菜单栏图标自动适配，保持清晰可见

#### Scenario: 主窗口隐藏后图标仍在
- **WHEN** 用户关闭主窗口
- **THEN** 菜单栏图标仍可见且菜单可用
