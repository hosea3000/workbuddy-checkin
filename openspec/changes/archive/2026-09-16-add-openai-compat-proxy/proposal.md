## Why

本机需要把已登录的 CodeBuddy 账号变成 OpenAI 兼容接口，供 Cline / Continue 等支持 OpenAI 协议的客户端使用，以替代当前另跑一个 `codebuddy2api`（Python/FastAPI）进程的方案——目标是单 exe、零额外运行时。

`upstream/codebuddy/` 已预埋聊天地基（`ChatURL()` 无调用者、`GenerateHeaders` 已导出），且参考实现 work2api 已有验证过的纯 Go 聊天链路（`chat_executor.go` 等，零 Web 框架依赖）。本次是把这条链路回迁并装配成桌面内的常驻本地服务。

## What Changes

- 新增本地 HTTP 服务，监听 `127.0.0.1:<可配置端口>`，**无鉴权**（仅本机可访问），可在设置页开关与改端口
- 新增 `POST /v1/chat/completions`：接受 OpenAI Chat Completions 请求，流式 SSE 透传、非流式聚合返回
- 新增 `GET /v1/models`：上游 `/v3/config` 模型 ∪ 本地配置模型（有序去重，TTL 缓存）
- 新增「当前凭证」概念：账号卡片提供「设为当前凭证」按钮，代理只使用该凭证；**不做凭证池、不做轮换、不做自动降级**
- 首个账号登录成功时自动成为当前凭证；删除当前凭证时顺位到列表首个 `active` 账号，无则置空
- 新增静默续期入口：代理请求前若当前凭证临期则续期，失败仅返回错误给客户端，**不置 `relogin_required`、不发系统通知**（该职责仍归签到巡检）
- 回迁上游 SSE 解析与事件语义（`sse.go` / `events.go`）及请求预处理、聊天执行器
- 托盘菜单新增代理服务开关项

明确不做：Anthropic 协议、API Key 鉴权、多用户/会话、用量统计、凭证池与轮换、代理侧额度查询。

## Capabilities

### New Capabilities

- `openai-compat-proxy`: OpenAI 兼容聊天补全与模型列表端点、上游 SSE 到 OpenAI chunk 的转换与聚合、代理服务生命周期（监听/端口/开关）、代理侧当前凭证选取与错误映射

### Modified Capabilities

- `settings`: 新增「模型代理」开关与监听端口两个设置项及其默认值
- `account-management`: 新增「设为当前凭证」操作、当前凭证标记，以及删除当前凭证时的顺位维护
- `account-login`: 登录成功且尚无当前凭证时，自动将新账号设为当前凭证
- `token-refresh`: 新增静默续期入口，续期失败不置位、不通知，仅向调用方返回错误
- `tray-integration`: 托盘/菜单栏菜单新增「模型代理」开关项

## Impact

- **新增代码**：`internal/proxy/`（HTTP 服务、协议适配、流式/聚合）、`upstream/codebuddy/{sse.go,events.go}`，约 900-1100 行，其中约 85% 从 `../work2api` 回迁
- **修改代码**：`model/settings.go`（+3 字段）、`model/view.go`（+1 字段）、`app_account.go`（设为当前凭证、删除顺位）、`app_settings.go`（端口/开关读写与校验）、`internal/account`（登录时自动设当前）、`internal/checkin`（静默续期入口）、`tray_*.go`（菜单项）、前端账号卡片与设置页
- **依赖**：不引入任何新第三方依赖（回迁代码仅用到 `google/uuid`，替换为已有的 `crypto/rand` 自产 UUID）；严格保持 `net/http` + 标准库
- **存储**：`settings.json` 新增字段（向后兼容，缺失即取默认值）
- **安全面**：新增本机监听端口，无鉴权；日志 / UI / 通知仍不得出现完整令牌
- **参考实现**：`../work2api`（Go，回迁来源）、`../codebuddy2api`（Python，只读参考，不修改）
