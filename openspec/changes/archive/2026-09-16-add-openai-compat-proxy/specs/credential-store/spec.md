## MODIFIED Requirements

### Requirement: 设置读写
系统 SHALL 提供 `GetSettings()` / `SaveSettings(s)`，持久化开机自启、GitHub 更新代理前缀、模型代理开关、模型代理监听端口与当前凭证 ID，并对缺失字段应用默认值。

#### Scenario: 保存设置
- **WHEN** 用户修改并保存设置
- **THEN** `settings.json` 持久化新值，重启后生效

#### Scenario: 缺失字段用默认值
- **WHEN** `settings.json` 缺少某字段
- **THEN** 系统对该字段应用默认值

#### Scenario: 旧版本文件兼容
- **WHEN** 读取由不含模型代理字段的旧版本写出的 `settings.json`
- **THEN** 系统正常加载，模型代理开关为关闭、端口为默认值、当前凭证为空
