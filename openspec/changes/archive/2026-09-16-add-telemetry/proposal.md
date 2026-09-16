## Why

作者无法知道有多少人在使用本软件。GitHub Releases 的 `download_count` 只能反映安装包下载次数，无法回答「有多少台机器在真正运行」「当前有多少活跃用户」「用户平均托管几个账号」。

本软件的用户在国内，服务端也部署在国内，网络可达性有保障；上报内容仅含版本号、操作系统、架构与账号数量这类非敏感信息，不涉及令牌、邮箱、账号 ID 或任何可识别个人身份的数据。因此可以引入一个最小化的匿名遥测能力，换取对产品实际使用规模的可见性。

## What Changes

- 新增 `internal/telemetry` 包，按 `internal/scheduler` / `internal/proxy` 的生命周期模式提供 `Start(ctx)` / `Stop()`。
- 应用启动后启动一个**每小时**的 ticker；每次 tick 检查「距上次成功上报是否超过 24 小时」，超过则发起一次上报。
- 上报为 fire-and-forget 的 `POST`，请求体仅含 `id`（随机 UUID）、`v`（版本号）、`os`、`arch`、`accounts`（账号数量）。**成功后**记录时间戳；失败静默处理、不记录时间戳，下一小时自然重试。
- 服务端契约：`POST <硬编码地址> /ping`，收到即返回 HTTP 200，无响应体；客户端不读取响应体。
- `Settings` 新增三个字段：`TelemetryEnabled`（默认 **true**）、`TelemetryID`（首次生成随机 UUID）、`TelemetryLastAt`。
- 设置页新增「帮助改进 WorkBuddy」栏：开关 + 说明文案（仅上传版本号 / 系统 / 账号数量，可随时关闭）。
- **文档**：README 的「隐私与安全」段落必须改写——现有「所有数据仅保存在本机，不上传任何第三方」与「网络仅访问 `copilot.tencent.com` 与 GitHub」两句将不再成立。
- **本次不含服务端实现**：服务端为独立项目/仓库，本 change 只冻结客户端行为与 HTTP 契约。

## Capabilities

### New Capabilities
- `telemetry`: 匿名使用数据上报的触发时机、上报内容边界、失败语义、开关行为与隐私约束。

### Modified Capabilities
- `settings`: 设置项清单新增遥测开关与相关字段，并声明其默认值与关闭后的行为。

## Impact

- **代码**：新增 `internal/telemetry/`（含 `telemetry.go` 与平台无关，若需 stub 按 `!windows && !darwin` 约定）；`model/settings.go`（新增字段与默认值）；`app.go`（`startup` 启动、`shutdown` 停止）；`app_settings.go`（读写新设置项）；前端设置页。
- **网络**：新增一条出站请求到硬编码的国内上报地址。与更新检查不同，**不使用** `Settings.UpdateProxy`（用户与服务端均在国内）。
- **文档**：`README.md` 隐私段落；`openspec/specs/settings/spec.md` 经 delta 更新。
- **隐私边界**：上报内容 SHALL NOT 包含令牌、`token_suffix`、邮箱、`UserID`、`AccountUID`、`Credential.ID`、昵称、企业 ID、域名、签到结果或积分。账号数量 SHALL 只报数量本身。
- **不涉及**：签到逻辑、上游协议、存储原子写策略、更新检查与下载。
