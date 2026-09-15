## MODIFIED Requirements

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

菜单 SHALL 包含「打开 / 退出」；Windows 为系统托盘菜单，macOS 为菜单栏菜单且左键单击图标即弹出。tooltip SHALL 显示 `workbuddy-checkin — 今日已签到 X/Y` 或 `— 有账号需重新登录`。系统 SHALL 提供 `CheckinSummary()` 返回 `{checkedIn, total, reloginRequired}` 供 tooltip 与顶栏计数使用。

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
