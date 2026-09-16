## MODIFIED Requirements

### Requirement: 补签触发时机

系统 SHALL 在以下时机检查并执行补签：应用启动（就绪后）、平台提供系统唤醒钩子时从休眠/睡眠唤醒后、跨天。Windows SHALL 通过电源/会话通知触发唤醒补签；macOS SHALL NOT 依赖系统唤醒钩子，补签 SHALL 依赖每小时巡检兜底（唤醒后最迟于下一个整点补签）。

#### Scenario: Windows 休眠唤醒补签
- **WHEN** Windows 系统从休眠唤醒且已过签到点且今日未签到
- **THEN** 系统执行补签

#### Scenario: macOS 唤醒后巡检补签
- **WHEN** macOS 系统休眠唤醒且已过签到点且今日未签到
- **THEN** 系统在最迟下一个整点巡检时执行补签

#### Scenario: 跳过今日已成功账号
- **WHEN** 补签遍历到今日已成功签到的账号
- **THEN** 系统跳过该账号
