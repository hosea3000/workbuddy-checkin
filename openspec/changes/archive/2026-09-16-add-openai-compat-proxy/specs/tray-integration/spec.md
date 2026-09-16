## MODIFIED Requirements

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
