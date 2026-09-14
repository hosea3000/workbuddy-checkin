## ADDED Requirements

### Requirement: 账号卡片列表
系统 SHALL 通过 `ListAccounts()` 返回 `[]AccountView`，每个卡片展示：昵称（无昵称显示 user_id 短码）、邮箱、状态徽章、今日签到状态、积分余额、凭证有效期、令牌末 8 位。

#### Scenario: 展示账号列表
- **WHEN** 用户打开主界面
- **THEN** 系统按账号逐条展示卡片，含昵称/邮箱/状态徽章/今日签到/积分余额/有效期

#### Scenario: 无昵称回退短码
- **WHEN** 账号缺少昵称
- **THEN** 卡片显示 user_id 短码作为标题

### Requirement: 删除账号
系统 SHALL 提供 `DeleteAccount(id)` 删除本地凭证，删除前 SHALL 二次确认；删除 SHALL NOT 调用上游任何接口。

#### Scenario: 确认后删除
- **WHEN** 用户点击删除并在确认弹窗确认
- **THEN** 系统移除该本地凭证、卡片消失、不产生任何上游请求

#### Scenario: 取消删除
- **WHEN** 用户在确认弹窗点击取消
- **THEN** 系统保留该凭证不变

### Requirement: 账号数量上限提示
系统 SHALL 在账号数超过 20 时仅提示不建议继续添加，SHALL NOT 阻止用户操作。

#### Scenario: 超过建议上限
- **WHEN** 已有 20 个账号且用户再添加
- **THEN** 系统提示数量较多、不建议继续，但仍允许添加

### Requirement: 视图 DTO 脱敏
所有绑定方法 SHALL 返回 DTO（`model/view.go`），令牌字段 SHALL 仅以 `token_suffix`（末 8 位）形式暴露。

#### Scenario: DTO 不含完整令牌
- **WHEN** 前端调用任意账号相关绑定方法
- **THEN** 返回结构中不含 `access_token` / `refresh_token` 完整值
