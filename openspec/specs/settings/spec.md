# settings Specification

## Purpose

读写应用设置项及其默认值，提供打开数据目录的入口，并在首次将窗口隐藏到后台（系统托盘 / 菜单栏）时向用户给出一次行为提示。

## Requirements

### Requirement: 设置项读写
系统 SHALL 通过 `GetSettings()` / `SaveSettings(s)` 读写以下设置项及默认值：「开机自启」（首启向导询问，默认勾选）。签到、补签、失败重试、成功/失败通知、关闭窗口最小化到托盘、启动时检查更新均为默认行为，不提供设置项。

#### Scenario: 开机自启以实际状态为准
- **WHEN** 用户通过系统外部方式改动（或删除）了开机自启项后打开设置页
- **THEN** 「开机自启」显示实际状态，保存设置时按当前选择重新同步

#### Scenario: 不再提供签到时间设置
- **WHEN** 用户打开设置页
- **THEN** 页面不出现「签到时间」或任何与签到时点相关的配置项

### Requirement: 打开数据目录

系统 SHALL 提供 `OpenDataDir()`，用系统资源管理器打开数据目录：Windows 用 `explorer` 打开 `%APPDATA%\workbuddy-checkin`，macOS 用 `open` 打开 `~/Library/Application Support/workbuddy-checkin`。

#### Scenario: Windows 打开数据目录
- **WHEN** Windows 用户点击「打开数据目录」
- **THEN** 资源管理器打开该目录

#### Scenario: macOS 打开数据目录
- **WHEN** macOS 用户点击「打开数据目录」
- **THEN** Finder 打开数据目录

### Requirement: 首次提示隐藏到后台

系统 SHALL 在用户首次关闭主窗口、窗口被隐藏到后台（Windows 系统托盘 / macOS 菜单栏）时提示一次该行为，此后不再提示；提示文案 SHALL 平台无关地表达「应用仍在后台运行」。

#### Scenario: Windows 首次关闭窗口
- **WHEN** Windows 用户首次关闭主窗口
- **THEN** 系统提示一次「应用仍在后台运行」，此后不再提示

#### Scenario: macOS 首次关闭窗口
- **WHEN** macOS 用户首次关闭主窗口
- **THEN** 系统提示一次「应用仍在后台运行」，此后不再提示，且应用不退出

#### Scenario: 仅提示一次
- **WHEN** 用户第二次及以后关闭主窗口
- **THEN** 不再出现该提示

### Requirement: 手动检查更新入口
设置页 SHALL 提供「检查更新」按钮，点击后执行一次更新检查，并向用户展示结果：已是最新、发现新版本（可进入更新流程）或检查失败（展示中文错误提示）。

#### Scenario: 手动检查发现新版本
- **WHEN** 用户在设置页点击「检查更新」且存在新版本
- **THEN** 展示新版本号并提供「立即更新」入口

#### Scenario: 手动检查已是最新
- **WHEN** 用户在设置页点击「检查更新」且当前已是最新
- **THEN** 展示「已是最新版本」

#### Scenario: 手动检查失败
- **WHEN** 用户在设置页点击「检查更新」且请求失败
- **THEN** 展示面向用户的中文失败提示

### Requirement: 当前版本展示
设置页 SHALL 展示当前应用版本号。

#### Scenario: 查看当前版本
- **WHEN** 用户打开设置页
- **THEN** 显示当前应用版本号
