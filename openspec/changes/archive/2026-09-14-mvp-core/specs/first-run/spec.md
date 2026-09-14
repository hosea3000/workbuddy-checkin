## ADDED Requirements

### Requirement: 首次运行引导
系统 SHALL 在无凭证文件时显示欢迎页，包含：一句话说明工具用途、「添加账号」按钮（直接进入登录流程）、开机自启勾选（默认勾选）。引导页 SHALL NOT 显示任何需要用户理解的配置项。

#### Scenario: 首次启动
- **WHEN** 应用首次启动且无 `credentials.json`
- **THEN** 显示欢迎页而非账号列表

#### Scenario: 从引导页添加账号
- **WHEN** 用户在欢迎页点击「添加账号」
- **THEN** 系统直接进入设备授权登录流程

#### Scenario: 自启默认勾选
- **WHEN** 欢迎页展示
- **THEN** 开机自启默认勾选
