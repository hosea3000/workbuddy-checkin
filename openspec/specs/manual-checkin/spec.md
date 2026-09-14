# manual-checkin Specification

## Purpose

提供单个账号的手动签到入口，定义以业务码与消息文本为准的签到成功判定规则，并让结果在原地反馈给用户。

## Requirements

### Requirement: 手动签到执行
系统 SHALL 提供 `CheckinNow(id)`，点击「立即签到」后按钮进入 loading，结果原地更新，并 SHALL 更新该凭证上的今日状态。

#### Scenario: 签到成功
- **WHEN** 用户点击「立即签到」且上游返回 `code=0`
- **THEN** 卡片显示「已签到 HH:MM，+N 积分」（N 缺失时不显示积分）

#### Scenario: 今日已签
- **WHEN** 上游消息包含「已签到」
- **THEN** 卡片显示「今日已签到」，视为成功而非失败

#### Scenario: 签到失败
- **WHEN** 请求网络异常 / 上游 5xx / 限流 / 需重新登录
- **THEN** 卡片显示对应人话失败原因

### Requirement: 签到成功判定
系统 SHALL 以 `code == 0` 或响应消息包含「已签到」作为成功判定，SHALL 仅在 `code == 0` 时提取本次积分。

#### Scenario: 幂等已签
- **WHEN** 上游返回消息「今日已签到」
- **THEN** 系统判定为成功且不记录积分
