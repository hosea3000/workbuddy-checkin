# account-login Specification

## Purpose

通过上游 OAuth 设备授权添加 CodeBuddy 账号，轮询完成登录并去重入库，全程令牌脱敏。

## Requirements

### Requirement: 启动 OAuth 设备授权
系统 SHALL 通过上游 `POST /v2/plugin/auth/state?platform=CLI` 启动设备授权，拿到 `state` 与 `authUrl`，并 SHALL 用系统默认浏览器打开该链接。`authUrl` 必须通过安全校验（绝对 http(s)、无 userinfo、无控制字符），否则视为启动失败。

#### Scenario: 成功启动并打开浏览器
- **WHEN** 用户点击「添加账号」
- **THEN** 系统返回 `authUrl`、自动用系统浏览器打开、登录状态进入 `awaiting_login`

#### Scenario: 上游启动失败
- **WHEN** 上游返回非 0 业务码或无效响应
- **THEN** 系统提示「认证启动失败，请稍后重试」，状态进入 `failed`

### Requirement: 三段轮询登录
系统 SHALL 按 5 秒间隔执行三段轮询：`PollToken`（`code=11217` 继续等待）→ `PollAccount`（`code=12151` 继续等待）→ `PollAccounts`（失败按 pending 处理）。三段全部就绪后才组装凭证入库。

#### Scenario: 等待用户完成登录
- **WHEN** `PollToken` 返回 `code=11217`
- **THEN** 系统保持 `awaiting_login` 并继续轮询

#### Scenario: 账号信息准备中
- **WHEN** `PollAccount` 返回 `code=12151`
- **THEN** 系统进入 `awaiting_account` 并继续轮询

#### Scenario: 登录成功
- **WHEN** 三段轮询均成功返回
- **THEN** 系统组装凭证、入库、状态进入 `done`、弹窗自动关闭、卡片列表出现新账号

### Requirement: 登录状态查询与取消
系统 SHALL 提供 `LoginStatus()` 供前端轮询（前端每 1.5s 调用），返回 `{stage, done, error, account?}`；SHALL 提供 `CancelLogin()` 停止轮询并作废当前 `state`。登录状态 SHALL 在 10 分钟后超时。

#### Scenario: 取消登录
- **WHEN** 用户在弹窗中点击「取消」
- **THEN** 系统停止轮询、作废 state、状态进入 `canceled`

#### Scenario: 授权链接过期
- **WHEN** 授权链接超过 10 分钟未完成
- **THEN** 系统提示「已过期，请重试」并停止轮询

### Requirement: 登录去重原位更新
系统 SHALL 以 `uid_<account_uid>` 为稳定主键去重；若已存在同主键凭证则原位更新（视为续命）并将状态置 `active`，不产生重复卡片。当首次登录时 `account_uid` 缺失而以 `oauth_<token_hash>` 落库的情况下，二次登录 SHALL 通过 `account_uid` 兜底查询命中并原位更新。

#### Scenario: 重复登录同一账号
- **WHEN** 用户对已存在账号再次完成登录
- **THEN** 系统原位更新该凭证且提示「账号已存在，凭证已更新」，卡片数量不变

### Requirement: 登录业务错误人话提示
系统 SHALL 将已知上游业务码映射为人话提示：`12005` 企业许可证没有可用席位、`11212` CodeBuddy 许可证已过期、`11216` 试用授权已过期、`10081` 当前 IP 被访问策略限制。

#### Scenario: 席位不足
- **WHEN** 上游返回 `code=12005`
- **THEN** 系统显示「企业许可证没有可用席位」而非原始错误码

### Requirement: 登录令牌脱敏
登录全过程中的 `access_token` / `refresh_token` SHALL NOT 出现在界面、日志或通知中；UI 仅暴露 `token_suffix`（末 8 位）。

#### Scenario: 界面展示凭证
- **WHEN** 账号登录成功并展示卡片
- **THEN** 界面仅显示令牌末 8 位，不出现完整令牌

### Requirement: 登录后自动成为当前凭证

系统 SHALL 在登录成功且当前尚无当前凭证时，自动将新登录的账号设为当前凭证。若已存在当前凭证，SHALL NOT 覆盖用户的选择。

#### Scenario: 首个账号登录成功
- **WHEN** 用户在无任何当前凭证的情况下登录成功
- **THEN** 该账号自动成为当前凭证

#### Scenario: 已有当前凭证时不覆盖
- **WHEN** 用户已设置账号 A 为当前凭证，随后登录账号 B
- **THEN** 当前凭证仍为 A，不因新登录而变更

#### Scenario: 重复登录同一账号
- **WHEN** 当前凭证为该账号且用户再次完成其登录
- **THEN** 当前凭证保持为该账号，卡片原位更新
