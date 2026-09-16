# credential-store Specification

## Purpose

以本机 JSON 文件原子、自愈地持久化凭证与设置，串行化并发读写、处理跨天状态失效，并保证令牌仅以 0600 权限明文存储。

## Requirements

### Requirement: JSON 文件存储位置
系统 SHALL 将数据存储在 `%APPDATA%\workbuddy-checkin\`（即 `os.UserConfigDir()/workbuddy-checkin/`），包含 `credentials.json`（`[]Credential`）、`settings.json`（`Settings`）、`app.log`。所有数据 SHALL 仅存本机，不上传任何第三方。

#### Scenario: 首次启动无文件
- **WHEN** 数据目录或文件不存在
- **THEN** 系统以空数据继续，不报错阻塞

### Requirement: 原子写
系统 SHALL 通过「写临时文件 → fsync → rename」原子替换方式写入，文件权限 SHALL 为 0600。

#### Scenario: 写入凭证
- **WHEN** 系统保存凭证
- **THEN** 目标文件被原子替换且权限为 0600，不出现半写文件

### Requirement: 损坏文件自愈
系统 SHALL 在 JSON 解析失败时将原文件重命名为 `<name>.corrupt-<时间戳>` 并以空数据继续启动。

#### Scenario: 凭证文件损坏
- **WHEN** `credentials.json` 内容非法 JSON
- **THEN** 系统将其备份为 `.corrupt-<时间戳>` 并正常进入引导页

### Requirement: 并发读写保护
系统 SHALL 通过 `store` 包的 `sync.Mutex` 串行化所有读写，防止 UI 与调度器并发写冲突。

#### Scenario: 并发保存
- **WHEN** 调度器与 UI 同时请求保存
- **THEN** 写操作被串行化，文件不损坏

### Requirement: 启动加载与落盘策略
系统 SHALL 在启动时一次性将数据加载进内存，并在每次写操作后立即整体落盘（数据小，不做脏标记）。

#### Scenario: 写后立即持久化
- **WHEN** 任一写操作完成
- **THEN** 对应 JSON 文件立即反映最新数据

### Requirement: 跨天状态失效
系统 SHALL 通过 `Credential.TodayDate != 本地今日` 判定今日签到状态失效，SHALL NOT 维护历史记录表。

#### Scenario: 隔天读取今日状态
- **WHEN** 新的一天读取前一天写入的凭证
- **THEN** 该凭证的今日状态被判定为未签到

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
