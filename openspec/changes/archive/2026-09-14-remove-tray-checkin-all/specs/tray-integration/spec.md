## MODIFIED Requirements

### Requirement: 托盘菜单与 tooltip
托盘菜单 SHALL 包含「打开主界面 / 退出」；tooltip SHALL 显示 `workbuddy-checkin — 今日已签到 X/Y` 或 `— 有账号需重新登录`。系统 SHALL 提供 `CheckinSummary()` 返回 `{checkedIn, total, reloginRequired}` 供 tooltip 与顶栏计数使用。

#### Scenario: 打卡计数 tooltip
- **WHEN** 2 个账号中 2 个今日已签到
- **THEN** tooltip 显示「今日已签到 2/2」

#### Scenario: 有待重登提示
- **WHEN** 存在 `relogin_required` 账号
- **THEN** tooltip 显示「有账号需重新登录」
