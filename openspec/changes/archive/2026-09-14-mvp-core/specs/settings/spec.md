## ADDED Requirements

### Requirement: 设置项读写
系统 SHALL 通过 `GetSettings()` / `SaveSettings(s)` 读写以下设置项及默认值：「签到时间」09:30（本地时区）、「启动时补签」开、「失败自动重试」开、「成功通知」开、「失败通知」开、「开机自启」（首启向导询问，默认勾选）、「关闭窗口时最小化到托盘」开、「启动时检查更新」开。

#### Scenario: 修改签到时间
- **WHEN** 用户将签到时间改为其他值并保存
- **THEN** 调度器按新时间触发签到

#### Scenario: 关闭补签
- **WHEN** 用户关闭「启动时补签」并保存
- **THEN** 应用启动时不再执行补签

### Requirement: 打开数据目录
系统 SHALL 提供 `OpenDataDir()`，用系统资源管理器打开 `%APPDATA%\workbuddy-checkin`。

#### Scenario: 打开数据目录
- **WHEN** 用户点击「打开数据目录」
- **THEN** 资源管理器打开该目录

### Requirement: 首次提示最小化到托盘
系统 SHALL 在首次关闭窗口隐藏到托盘时提示一次该行为。

#### Scenario: 首次关闭窗口
- **WHEN** 用户首次关闭主窗口且最小化开关开启
- **THEN** 系统提示一次「应用仍在托盘运行」，此后不再提示
