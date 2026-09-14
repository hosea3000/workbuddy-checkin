## ADDED Requirements

### Requirement: 积分余额查询
系统 SHALL 通过 `client.FetchQuotaPersonal(ctx, snapshot)` 查询积分余额，返回 `(total, remaining, err)`，并写入 `CreditBalance = remaining`、`CreditBalanceTotal = total`、`CreditBalanceAt`。

#### Scenario: 成功查询余额
- **WHEN** 余额查询成功
- **THEN** 系统更新该账号的积分余额、总额度与查询时间

#### Scenario: 从未成功过显示占位
- **WHEN** 账号从未成功查询过余额
- **THEN** 卡片显示「—」

### Requirement: 余额刷新触发点
系统 SHALL 在以下时机刷新余额：应用启动时、每次签到成功后、定时每 1 小时、用户手动点击卡片刷新（`RefreshQuota(id)`）。

#### Scenario: 签到成功后刷新
- **WHEN** 某账号签到成功
- **THEN** 系统随后刷新该账号积分余额

#### Scenario: 每小时刷新
- **WHEN** 距离上次余额刷新达到 1 小时
- **THEN** 系统刷新余额，账号间沿用 5~20s 抖动串行

#### Scenario: 手动刷新
- **WHEN** 用户点击卡片上的刷新按钮
- **THEN** 系统立即调用 `RefreshQuota(id)` 并返回最新 `{balance, total, at}`

### Requirement: 余额失败不影响签到
余额查询失败 SHALL 仅记录日志，SHALL NOT 修改账号 `Status`、SHALL NOT 发通知，且 SHALL 保留上次的余额值。

#### Scenario: 查询失败保留旧值
- **WHEN** 余额查询网络失败
- **THEN** 账号状态不变、卡片保留上次余额值、不产生通知

#### Scenario: 断网不影响签到结果
- **WHEN** 断网环境下签到结果正常但余额查询失败
- **THEN** 签到状态正确更新，余额保持旧值

### Requirement: 待重登账号跳过余额
`relogin_required` 账号 SHALL 跳过余额查询，不空打上游。

#### Scenario: 跳过待重登账号余额
- **WHEN** 余额刷新遍历到 `relogin_required` 账号
- **THEN** 系统跳过该账号
