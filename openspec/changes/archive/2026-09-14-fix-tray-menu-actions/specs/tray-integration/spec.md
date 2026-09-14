## MODIFIED Requirements

### Requirement: 托盘驻留
应用 SHALL 常驻系统托盘；关闭主窗口 SHALL 隐藏到托盘而非退出；退出 SHALL 仅能从托盘菜单触发，且 SHALL 真正结束进程——不得只移除托盘图标或隐藏窗口。

#### Scenario: 关闭窗口不退出
- **WHEN** 用户关闭主窗口
- **THEN** 应用隐藏到托盘并继续调度签到

#### Scenario: 托盘退出
- **WHEN** 用户在托盘菜单选择「退出」
- **THEN** 进程退出、托盘图标消失

#### Scenario: 双击托盘图标
- **WHEN** 用户双击托盘图标
- **THEN** 主界面显示

## ADDED Requirements

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
