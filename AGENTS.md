# AGENTS.md

workbuddy-checkin：Windows 桌面小工具（Wails v2 + Go + Vue3），管理 CodeBuddy（腾讯）账号并自动每日签到。**当前仓库只有设计文档，尚无代码**。动手前先读 `docs/PRD.md`（做什么，F1~F10）与 `docs/DESIGN.md`（怎么做）——两者是唯一事实来源。

## 硬约束（来自 DESIGN.md，实现时勿偏离）

- 技术栈锁定：Wails v2 + Go 1.25 + Vue3/Vite/TS/Pinia/Tailwind v4；上游客户端纯 `net/http`。
- **不引入** gin、gorm、viper、wire、redis、sqlite 及任何 DI 框架；直接依赖 ≤ 8 个。
- 上游协议从 `/root/code/github/work2api/internal/upstream/codebuddy/` **逐字移植**（`auth.go` / `headers.go` / `errors.go` / `endpoint.go` 直接拷贝，连单测一起；只去掉 `google/uuid` 等内部依赖）。本仓库与 work2api 不互通，不要复用其模块或依赖。
- `upstream/codebuddy/headers.go` 的 CLI 伪装头与常量（`cli_version = 2.107.0` 等）冻结，禁改。
- 系统集成（托盘/自启/通知）必须提供 `*_stub.go`（`//go:build !windows`），保证 Linux 下 `go test ./...` 可跑。
- 存储是 JSON 文件（`os.UserConfigDir()/workbuddy-checkin/`），非 SQLite；原子写 + 损坏自愈（`.corrupt-<时间戳>`）。
- 令牌明文 0600 存储；**日志 / UI / 通知永不出现 token**，UI 只暴露 `token_suffix`（末 8 位）。
- 依赖方向单向无环：`main → internal/* → store → model`，`internal/* → upstream/codebuddy`。

## 命令（DESIGN.md §13，代码落地后生效）

- 测试 / 静态检查：`go vet ./... && go test ./...`（Linux 跑 stub 分支）
- 构建：`wails build -platform windows/amd64`；开发：`wails dev`
- 前端绑定：`wails generate module`（改 `app*.go` 绑定方法后重跑）

## 注意

- 全局 CLAUDE.md 的 `make wire && make sqlc && make swag` / `make buildx env=...` 在本仓库不存在（无 Makefile），勿照抄。
- 仓库尚未 `git init`，无 CI；`build/`、`.github/workflows/release.yml` 均为计划产物。
- 前端不使用 Wails 事件传业务状态，一律轮询绑定方法（`LoginStatus` 1.5s / `ListAccounts` 30s）。
- 系统托盘、自启、更新器照抄 health-tool 已验证实现（health-tool 仓库不在本机，按 DESIGN.md §1 描述找对应实现）。
