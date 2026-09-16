# token-refresh Specification

## Purpose

在令牌临近过期时自动续期，续期失败按错误类型决定是否置位为待重登并去重通知，且待重登账号会被自动跳过。

## Requirements

### Requirement: 临期自动续期
系统 SHALL 在满足 `now >= ExpiresAt - 24h` 且 `refresh_token` 存在时调用 `client.RefreshToken()`；成功时 SHALL 更新 `access_token`、`refresh_token`（上游会轮换）与有效期。

#### Scenario: 成功续期
- **WHEN** 凭证剩余有效期小于 24 小时且存在 refresh_token
- **THEN** 系统刷新令牌并静默保存新的 access_token 与 refresh_token

#### Scenario: 无 refresh_token
- **WHEN** 凭证剩余有效期小于 24 小时但 refresh_token 为空
- **THEN** 系统跳过续期、不调用上游

### Requirement: 续期调用点
系统 SHALL 在以下时机触发续期检查：应用启动、每次签到前、签到完成后、调度唤醒时。

#### Scenario: 签到前续期
- **WHEN** 执行任一账号签到前
- **THEN** 系统先执行续期检查再调用签到

### Requirement: 续期失败置位与通知去重
系统 SHALL 在续期失败且错误名为 `unauthorized`（含上游 401/403）时将账号置为 `relogin_required` 并发通知一次；其他错误（`refresh_failed` / `ip_restricted` / 网络）SHALL 仅记日志、不置位、不通知。通知 SHALL 在状态实际变更时才发出。

#### Scenario: 凭证被上游拒绝
- **WHEN** 续期请求被上游以 401/403 拒绝
- **THEN** 账号状态置为 `relogin_required`、托盘提示、发一次通知

#### Scenario: 网络错误不置位
- **WHEN** 续期因网络异常失败
- **THEN** 账号状态保持 `active`，仅记录日志，不发通知

#### Scenario: 状态未变更不重复通知
- **WHEN** 账号已处于 `relogin_required` 且再次续期失败
- **THEN** 系统不再重复发送通知

### Requirement: 需重新登录账号被自动跳过
标记为 `relogin_required` 的账号 SHALL 被自动签到跳过，且不发起任何无关上游请求。

#### Scenario: 跳过待重登账号
- **WHEN** 定时签到遍历到一个 `relogin_required` 账号
- **THEN** 系统跳过该账号的签到与余额查询

### Requirement: 静默续期入口

系统 SHALL 提供一个无副作用的续期入口供代理调用：在令牌临期（距 `ExpiresAt` 不足 60 秒或已过期）时执行令牌交换并回写存储，成功返回更新后的凭证，失败直接返回错误。该入口 SHALL NOT 将账号置为 `relogin_required`、SHALL NOT 发送通知、SHALL NOT 修改账号状态。

既有签到路径（`refreshIfNeeded`）的行为 SHALL 保持不变：仍按 24 小时窗口判定、失败且错误名为 `unauthorized` 时置位并发通知一次。

#### Scenario: 静默续期成功
- **WHEN** 代理调用该入口且令牌已过期、上游接受 refresh_token
- **THEN** 返回更新后的凭证，新的 access_token 与 refresh_token 被持久化，不产生通知

#### Scenario: 静默续期失败不置位
- **WHEN** 代理调用该入口且上游以 401/403 拒绝
- **THEN** 返回错误，账号状态保持原值、不发送通知

#### Scenario: 令牌未临期不续期
- **WHEN** 代理调用该入口但令牌距过期仍超过 60 秒
- **THEN** 直接返回原凭证，不发起上游请求

#### Scenario: 无 refresh_token
- **WHEN** 代理调用该入口但凭证没有 refresh_token
- **THEN** 返回原凭证，不发起上游请求

#### Scenario: 签到路径行为不变
- **WHEN** 调度器按既有流程执行签到前的续期检查
- **THEN** 仍按 24 小时窗口判定，失败时仍按原规则置位并去重通知
