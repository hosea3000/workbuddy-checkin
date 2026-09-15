## ADDED Requirements

### Requirement: 每小时签到巡检
系统 SHALL 以 1 小时为固定间隔执行一次签到巡检：遍历全部账号，对「今日未成功签到」且非 `relogin_required` 的账号执行签到。巡检 SHALL NOT 依赖任何可配置的签到时间。应用启动后 SHALL 立即执行首次巡检，其后每小时一次。

#### Scenario: 巡检签到未签账号
- **WHEN** 一次巡检遍历到某账号今日未成功签到
- **THEN** 系统对该账号执行签到

#### Scenario: 跳过已签账号
- **WHEN** 一次巡检遍历到某账号今日已成功签到
- **THEN** 系统跳过该账号，不产生上游签到请求

#### Scenario: 跳过待重登账号
- **WHEN** 一次巡检遍历到 `relogin_required` 账号
- **THEN** 系统跳过该账号

#### Scenario: 失败账号下次巡检重试
- **WHEN** 某账号本次巡检签到失败且当日仍未成功签到
- **THEN** 系统在下一个小时的巡检中再次尝试，SHALL NOT 受当日尝试次数上限限制

### Requirement: 当日尝试计数跨天归零
系统 SHALL 在 `Credential.TodayDate` 不等于本地今日时，将 `Credential.TodayAttempts` 视为 0；写入新的当日签到状态前 SHALL 归零该计数，SHALL NOT 跨天累积。

#### Scenario: 跨天重新计数
- **WHEN** 新的一天首次对某账号执行签到
- **THEN** `TodayAttempts` 从 0 开始计数，而非沿用前一天的值

## MODIFIED Requirements

### Requirement: 签到通知
系统 SHALL 在成功且「成功通知」开启时发送「签到成功」+ 账号名 + 积分；失败且「失败通知」开启时 SHALL 仅在当日首次失败时发送「签到失败」+ 原因，同一账号当日的后续失败 SHALL NOT 重复通知。通知 SHALL NOT 含任何令牌。

#### Scenario: 成功通知
- **WHEN** 自动签到成功且成功通知开启
- **THEN** 系统发送含账号名与积分的 Toast

#### Scenario: 当日首次失败通知
- **WHEN** 某账号当日第一次签到失败且失败通知开启
- **THEN** 系统发送一次失败 Toast

#### Scenario: 后续失败不重复通知
- **WHEN** 某账号当日已失败过，随后一次巡检再次失败
- **THEN** 系统不重复发送失败 Toast

#### Scenario: 关闭通知
- **WHEN** 对应通知开关关闭
- **THEN** 系统不发送该 Toast

## REMOVED Requirements

### Requirement: 每日定时签到
**Reason**: 移除可配置的「签到时间」，签到改为每小时巡检，不再依赖单一时点。
**Migration**: 由「每小时签到巡检」要求替代；启动/唤醒时的即时巡检由 `catch-up-checkin` 要求覆盖。

### Requirement: 失败重试上限
**Reason**: 每小时巡检本身即重试机制；固定 30 分钟重试与每日 3 次上限在当天较晚恢复网络时会导致漏签。
**Migration**: 失败账号在下一次每小时巡检时自动重试，不设当日尝试次数上限；`relogin_required` 账号仍被跳过。
