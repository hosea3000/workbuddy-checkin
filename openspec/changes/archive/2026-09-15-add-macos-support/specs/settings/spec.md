## MODIFIED Requirements

### Requirement: 打开数据目录

系统 SHALL 提供 `OpenDataDir()`，用系统资源管理器打开数据目录：Windows 用 `explorer` 打开 `%APPDATA%\workbuddy-checkin`，macOS 用 `open` 打开 `~/Library/Application Support/workbuddy-checkin`。

#### Scenario: Windows 打开数据目录
- **WHEN** Windows 用户点击「打开数据目录」
- **THEN** 资源管理器打开该目录

#### Scenario: macOS 打开数据目录
- **WHEN** macOS 用户点击「打开数据目录」
- **THEN** Finder 打开数据目录

### Requirement: 首次提示最小化到托盘

系统 SHALL 在 Windows 上首次关闭窗口隐藏到托盘时提示一次该行为；macOS 上因关闭窗口即退出，SHALL NOT 出现「最小化到托盘」提示。

#### Scenario: Windows 首次关闭窗口
- **WHEN** Windows 用户首次关闭主窗口
- **THEN** 系统提示一次「应用仍在托盘运行」，此后不再提示

#### Scenario: macOS 关闭窗口
- **WHEN** macOS 用户关闭主窗口
- **THEN** 应用退出，不出现「最小化到托盘」提示
