## MODIFIED Requirements

### Requirement: JSON 文件存储位置
系统 SHALL 将数据存储在 `%APPDATA%\workbuddy-checkin\`（即 `os.UserConfigDir()/workbuddy-checkin/`），包含 `credentials.json`（`[]Credential`）、`settings.json`（`Settings`）、`app.log`，以及日志轮转产生的历史文件（`app-<时间戳>.log[.gz]`，见 `logging` 能力）。所有数据 SHALL 仅存本机，不上传任何第三方。

#### Scenario: 首次启动无文件
- **WHEN** 数据目录或文件不存在
- **THEN** 系统以空数据继续，不报错阻塞

#### Scenario: 日志轮转产物同目录存放
- **WHEN** `app.log` 达到 5 MB 触发轮转
- **THEN** 历史文件以 `app-<时间戳>.log[.gz]` 命名存放于同一数据目录
