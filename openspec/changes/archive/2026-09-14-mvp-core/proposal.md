## Why

CodeBuddy 每日签到目前只能靠部署 work2api 网关（要服务器、数据库、开放端口），对只需要签到的个人用户过重。本仓库要做一个免安装的 Windows 桌面小工具：常驻托盘、到点自动签到、开机补签，双击即用。当前仓库只有 PRD/DESIGN/UI 预览三份文档，没有任何代码，需要先落地可发布的最小可用版本（PRD §9 的 M1~M4，即 v0.1.0）。

## What Changes

- 搭建 Wails v2 + Go 1.25 + Vue3 应用骨架，Linux 下用 `*_stub.go` 保证 `go test ./...` 可跑
- 从 work2api `internal/upstream/codebuddy/` 逐字移植上游协议客户端（签到、OAuth 设备授权、刷新、额度），并修正移植中发现的两处上游 bug
- 实现凭证与设置的本地 JSON 存储（原子写 + 损坏自愈 + 0600）
- 实现 OAuth 设备授权登录闭环（三段轮询、去重原位更新、业务错误人话提示）
- 实现手动签到、每日定时签到、启动补签、失败重试、凭证自动续期、积分余额刷新
- 实现系统集成：托盘驻留、开机自启（HKCU Run）、单实例、关闭最小化、Toast 通知
- 实现前端：账号列表、登录弹窗、设置页、首次运行引导
- **不含**（PRD §6 / P2）：更新器、CI 发布、TRAE、多 provider、与 work2api 互通，留待后续 change

## Capabilities

### New Capabilities
- `account-login`: OAuth 设备授权登录编排（三段轮询、去重原位更新、取消/TTL、业务错误人话提示、令牌脱敏）
- `account-management`: 账号卡片列表、删除（二次确认）、视图 DTO（token_suffix）、积分余额展示与手动刷新
- `token-refresh`: 凭证临期自动续期、`relogin_required` 状态机与通知去重
- `manual-checkin`: 单账号「立即签到」执行、结果判定与反馈
- `scheduled-checkin`: 每日定时触发、多账号抖动串行、幂等、失败重试上限、Toast 通知
- `catch-up-checkin`: 启动/唤醒/跨天补签判定与执行
- `quota-refresh`: 积分余额刷新触发点（启动/签到后/每小时/手动）、失败不改状态
- `credential-store`: `%APPDATA%` JSON 持久化、原子写、损坏自愈、跨天状态失效、设置读写
- `tray-integration`: 托盘菜单/tooltip、开机自启、单实例、静默启动、关闭最小化
- `first-run`: 首次运行引导页（无凭证文件时）与开机自启默认勾选
- `settings`: 设置项读写（签到时间、补签/重试/通知开关、自启、最小化、检查更新开关、打开数据目录）

### Modified Capabilities
<!-- 无：openspec/specs/ 为空，全部为新增能力 -->

## Impact

- **新增代码**：`main.go`、`app*.go`、`model/`、`store/`、`upstream/codebuddy/`、`internal/{account,checkin,scheduler}/`、`frontend/`、各 `*_windows.go` / `*_stub.go`
- **移植来源**：work2api `internal/upstream/codebuddy/`（headers/auth/client/errors/endpoint + 单测）与 `internal/service/credential_common.go`（`jwtPayload`/`applyJWTIdentity`/`oauthUserID`/`shortHash`/`ExtractIssuerInfo` 搬入 `internal/account/`）
- **依赖**：Wails v2、`golang.org/x/sys/windows`、`github.com/go-toast/toast`；**不引入** gin/gorm/viper/wire/redis/sqlite/google-uuid
- **数据**：`%APPDATA%\workbuddy-checkin\{credentials.json, settings.json, app.log}`
- **网络**：仅 `copilot.tencent.com`
- **平台**：Windows 10 1809+ x64；其他平台仅 stub 可测试（不可运行托盘/自启）
