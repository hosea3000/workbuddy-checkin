# AGENTS.md

workbuddy-checkin：桌面小工具（Windows / macOS，Wails v2 + Go + Vue3），管理 CodeBuddy（腾讯）账号并自动签到。规划文档是唯一事实来源：`docs/PRD.md`（做什么，F1~F10）、`docs/DESIGN.md`（怎么做）。

## 命令

- 全量检查：`go vet ./... && go test ./...`（Linux 跑 `*_stub.go` 分支）
- 单包 / 单测：`go test ./internal/scheduler/`、`go test ./internal/scheduler/ -run TestRunAll`
- 前端类型检查 + 构建：在 `frontend/` 跑 `npm run build`（= `vue-tsc --noEmit && vite build`）
- Windows 构建：`wails build -platform windows/amd64`；开发：`wails dev`
- macOS 构建（须在 macOS 上）：`wails build -platform darwin/arm64`（或 `darwin/amd64`），再用 `hdiutil create` 打包为 dmg
- 平台编译自检：`GOOS=darwin GOARCH=arm64 go build ./...`（跨平台新增/改文件后至少跑一次）。**注意**：`tray_darwin.go` 用 cgo，Linux 上此命令会跳过它（报 `undefined: Tray`），darwin 侧真正编译只能在 macOS 上验证。
- 改过 `app*.go` 的绑定方法后：`wails generate module` 重生成 `frontend/wailsjs/`

**顺序陷阱**：`main.go` 用 `//go:embed all:frontend/dist`，而 `frontend/dist` 被 gitignore。**全新 clone 必须先 `cd frontend && npm run build`，否则 `go build/test ./...` 直接失败**（CI 也是先建前端再 vet/test）。

**`wails generate module` 副作用**：会把 `frontend/wailsjs/runtime/{package.json,runtime.d.ts,runtime.js}` 的权限位改成 755（内容不变）。提交前 `git checkout -- frontend/wailsjs/runtime/` 还原这些纯 mode 变更。

## OpenSpec 工作流

规划产物在 `openspec/`：`specs/<capability>/spec.md`（已定行为）、`changes/<name>/`（进行中的 proposal/design/specs/tasks）、`changes/archive/`。无 store，命令直接作用于仓库根。

- `openspec list` · `openspec new change "<kebab-name>"` · `openspec status --change "<name>"` · `openspec instructions <artifact> --change "<name>"`
- 校验：`openspec validate --changes "<name>"`（是 `--changes`，不是 `--change`）
- 归档：`openspec archive "<name>"`

实现某 change 前先读它的四类产物；完成后勾 `tasks.md` 再归档。delta spec 用 `## ADDED/MODIFIED/REMOVED Requirements` + `### Requirement:` + `#### Scenario:`（Scenario 标题是四个井号）。

## 硬约束（来自 DESIGN.md，勿偏离）

- 技术栈锁定：Wails v2 + Go 1.26 + Vue3/Vite/TS/Pinia/Tailwind v4；上游客户端纯 `net/http`。
- **不引入** gin、gorm、viper、wire、redis、sqlite 及任何 DI 框架。
- 上游协议从 `/root/code/github/work2api/internal/upstream/codebuddy/` 逐字移植；本仓库与 work2api 不互通，勿复用其模块或依赖。
- `upstream/codebuddy/headers.go` 的 CLI 伪装头与常量（`cli_version = 2.107.0` 等）冻结，禁改。
- 系统集成（托盘/自启/通知）必须同时提供 `*_stub.go`（`//go:build !windows && !darwin`），保证 Linux 下 `go test ./...` 可跑。托盘为 Windows（`tray_windows.go`）与 macOS（`tray_darwin.go`）各一份实现，stub 仅覆盖其余平台。
- macOS 系统集成优先用系统命令（`launchctl`/`osascript`/`hdiutil`/`open`），**不引入任何新的第三方 Go 依赖**；菜单栏图标是唯一例外，用手写 cgo/Objective-C（`tray_darwin.go`/`.m`/`.h`，`NSStatusItem`）——因为 `osascript` 无法创建常驻状态栏项。**不得为此引入 `energye/systray` 等第三方托盘库**（其 macOS 实现会抢占 `NSApplication.delegate`，破坏 Wails 的单实例锁与 ⌘Q 链路）。不签名、不公证。
- 存储是 JSON 文件（`os.UserConfigDir()/workbuddy-checkin/`），非 SQLite；原子写 + 损坏自愈（`.corrupt-<时间戳>`）。
- 令牌明文 0600 存储；**日志 / UI / 通知永不出现 token**，UI 只暴露 `token_suffix`（末 8 位）。
- 依赖方向单向无环：`main → internal/* → store → model`，`internal/* → upstream/codebuddy`。

## 注意

- 无 Makefile；全局 CLAUDE.md 的 `make wire && make sqlc && make swag` / `make buildx env=...` 在本仓库不存在，勿照抄。
- 前端不使用 Wails 事件传业务状态，一律轮询绑定方法（`LoginStatus` 1.5s / `ListAccounts` 30s）。
- 版本号：`main.version` 由发布构建经 `-ldflags` 注入，本地开发为 `dev`；`wails.json` 的 `productVersion` 静态为 0.1.0，CI 构建时按 tag 覆盖。发布 = 推送 `v*` tag，`.github/workflows/release.yml` 自动建 Windows exe + NSIS 安装器 + macOS arm64/amd64 两枚 dmg 的 Release（tag 尾号 +1，如 v0.1.4）。
- `build/bin/`、安装器临时文件、`frontend/dist/` 均被 gitignore。
