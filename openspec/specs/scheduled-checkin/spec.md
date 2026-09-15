# scheduled-checkin Specification

## Purpose

每小时对全部账号执行一次签到巡检，账号之间抖动串行，支持幂等跳过、失败账号下次巡检重试、结果通知与并发保护。

## Requirements

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

### Requirement: 并发保护
系统 SHALL 保证 `runAll` 全程串行，手动签到 SHALL 排队等待而非并发打上游。

#### Scenario: 自动与手动冲突
- **WHEN** 自动签到执行期间用户点击「立即签到」
- **THEN** 手动签到等待自动签到释放后再执行
