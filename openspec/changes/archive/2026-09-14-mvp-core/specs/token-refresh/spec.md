## ADDED Requirements

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
