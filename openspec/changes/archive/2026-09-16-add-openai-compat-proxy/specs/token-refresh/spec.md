## ADDED Requirements

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
