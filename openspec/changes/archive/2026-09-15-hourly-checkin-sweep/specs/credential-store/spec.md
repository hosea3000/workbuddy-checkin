## MODIFIED Requirements

### Requirement: 设置读写
系统 SHALL 提供 `GetSettings()` / `SaveSettings(s)`，持久化开机自启与 GitHub 更新代理前缀，并对缺失字段应用默认值。

#### Scenario: 保存设置
- **WHEN** 用户修改并保存设置
- **THEN** `settings.json` 持久化新值，重启后生效

#### Scenario: 缺失字段用默认值
- **WHEN** `settings.json` 缺少某字段
- **THEN** 系统对该字段应用默认值
