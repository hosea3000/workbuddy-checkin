# logging Specification

## Purpose
TBD - created by archiving change add-log-rotation. Update Purpose after archive.
## Requirements
### Requirement: 日志输出目标与格式
系统 SHALL 将运行日志写入数据目录下的 `app.log`，时间戳格式为 `log.LstdFlags`。`wails dev`（stderr 可写）时 SHALL 同时写 stderr；windowsgui 构建（stderr 不可写）时 SHALL 仅写文件，且不得因 stderr 写入失败而丢失文件日志。

#### Scenario: 有控制台时双写
- **WHEN** 应用在 stderr 可写的环境（如 `wails dev`）中运行并产生一条日志
- **THEN** 该日志同时出现在 stderr 与 `app.log`

#### Scenario: 无控制台时仅写文件
- **WHEN** 应用以 windowsgui 构建运行（`os.Stderr` 写入返回错误）并产生一条日志
- **THEN** 该日志写入 `app.log`，不因 stderr 不可用而丢失

### Requirement: 日志轮转与保留上限
系统 SHALL 在 `app.log` 达到 5 MB 时将其切分为历史文件并新建当前文件。系统 SHALL 保留最多 3 个历史文件、最长 30 天，SHALL 对历史文件进行 gzip 压缩。轮转 SHALL 在单次运行周期内实时生效，不依赖应用重启。

#### Scenario: 达到大小上限即切分
- **WHEN** `app.log` 写入后累计超过 5 MB
- **THEN** 原文件被轮转为历史文件，新的 `app.log` 从空文件继续写入，日志不丢失

#### Scenario: 历史文件数量受控
- **WHEN** 轮转产生的历史文件超过 3 个
- **THEN** 最旧的历史文件被删除，历史文件总数不超过 3

#### Scenario: 历史文件压缩
- **WHEN** 产生一个历史日志文件
- **THEN** 该文件以 gzip 形式落盘

#### Scenario: 长时间运行不无限增长
- **WHEN** 应用持续运行数周并不断产生日志
- **THEN** `app.log` 及其历史文件的磁盘占用保持在 5 MB + 3 个历史文件的有界范围内

### Requirement: 日志不含令牌
系统 SHALL NOT 在 `app.log` 或 stderr 中出现任何完整访问令牌；需要标识账号时 SHALL 仅使用账号短码或 `token_suffix`。

#### Scenario: 记录涉及凭证的事件
- **WHEN** 系统记录登录、续期或代理凭证相关事件
- **THEN** 日志中出现的是账号短码或令牌尾号，而非完整令牌

