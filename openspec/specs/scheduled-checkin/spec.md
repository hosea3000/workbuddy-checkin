# scheduled-checkin Specification

## Purpose

每日按设定时间对全部账号执行签到，账号之间抖动串行，支持幂等跳过、失败重试上限、结果通知与并发保护。

## Requirements

### Requirement: 每日定时签到
系统 SHALL 在每日 `签到时间`（默认 09:30，本地时区）触发全部账号签到。下次触发点 = 今天 `HH:MM`，已过则明天 `HH:MM`；每次触发后 SHALL 重新计算触发点以容忍系统时间/时区变更。

#### Scenario: 到点触发
- **WHEN** 本地时间到达签到时间
- **THEN** 系统执行全部账号签到

#### Scenario: 已过签到点启动
- **WHEN** 应用启动时已过今日签到时间
- **THEN** 下次触发点计算为明天的签到时间（当日补签由补签能力处理）

### Requirement: 多账号抖动串行
系统 SHALL 逐个签到全部账号，账号之间随机间隔 5~20 秒；SHALL NOT 并发打上游。

#### Scenario: 两个账号顺序签到
- **WHEN** 定时签到遍历到多账号
- **THEN** 各账号按 5~20s 随机间隔先后执行，不并发

### Requirement: 签到幂等
系统 SHALL 在凭证已记录「今天成功签到」时跳过该账号；上游返回「已签到」SHALL 视为成功。

#### Scenario: 当日已成功则跳过
- **WHEN** 账号的 `today_date` 为今天且 `today_success` 为真
- **THEN** 系统跳过该账号签到

### Requirement: 失败重试上限
系统 SHALL 在签到失败且「失败自动重试」开启时于 30 分钟后重试；当日每账号最多 3 次，计数 SHALL 持久化到 `Credential.TodayAttempts` 并在跨天归零，重启后仍有效；`relogin_required` 账号 SHALL NOT 重试。

#### Scenario: 失败后重试
- **WHEN** 签到失败且重试开关开启且当日尝试次数小于 3
- **THEN** 系统在 30 分钟后重试该账号

#### Scenario: 达到上限停止重试
- **WHEN** 当日尝试次数已达 3 次
- **THEN** 系统当天不再重试该账号

#### Scenario: 重启后计数保留
- **WHEN** 应用在当日重试 1 次后重启
- **THEN** `TodayAttempts` 仍为 1，剩余可重试次数正确

### Requirement: 签到通知
系统 SHALL 在成功且「成功通知」开启时发送「签到成功」+ 账号名 + 积分；失败且「失败通知」开启时发送「签到失败」+ 原因。通知 SHALL NOT 含任何令牌。

#### Scenario: 成功通知
- **WHEN** 自动签到成功且成功通知开启
- **THEN** 系统发送含账号名与积分的 Toast

#### Scenario: 关闭通知
- **WHEN** 对应通知开关关闭
- **THEN** 系统不发送该 Toast

### Requirement: 并发保护
系统 SHALL 保证 `runAll` 全程串行，手动签到 SHALL 排队等待而非并发打上游。

#### Scenario: 自动与手动冲突
- **WHEN** 自动签到执行期间用户点击「立即签到」
- **THEN** 手动签到等待自动签到释放后再执行
