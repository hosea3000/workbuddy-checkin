## Context

仓库目前只有 `docs/PRD.md`、`docs/DESIGN.md`、`ui-preview.html`，无任何代码。本 change 落地 PRD §9 的 M1~M4（v0.1.0），即除更新器/CI/引导页之外的完整可用版本。技术约束来自 `docs/DESIGN.md`（唯一事实来源）：

- 栈锁定：Wails v2 + Go 1.25 + Vue3/Vite/TS/Pinia/Tailwind v4，上游客户端纯 `net/http`
- 上游协议从 work2api `internal/upstream/codebuddy/` 逐字移植
- 不引入 gin/gorm/viper/wire/redis/sqlite，依赖 ≤ 5 个业务依赖
- 非 Windows 平台必须能编译运行测试（stub 分支）

## Goals / Non-Goals

**Goals:**
- M1：Wails 骨架 + 托盘/自启/单实例 + 空壳 Vue3 界面，双击可运行、可托盘驻留
- M2：上游客户端移植 + OAuth 登录闭环 + JSON 凭证存储，能添加账号并看到卡片
- M3：手动签到 + 今日状态 + 卡片状态 + 通知
- M4：调度 + 补签 + 重试 + 凭证续期 + 积分余额刷新
- 全程 `go vet ./... && go test ./...` 在 Linux 通过（stub 分支）

**Non-Goals:**
- 更新器（P2 / M5）、CI 发布、TRAE/其他 provider、与 work2api 互通
- 图形安装器、代码签名、macOS/Linux 发行版
- DPAPI 加密（一期明文 0600，见 DESIGN §12）

## Decisions

### D1：上游移植策略（含两处必须修正的上游 bug）

从 work2api 逐字拷贝 `headers.go`/`auth.go`/`client.go`/`errors.go`/`endpoint.go` 与其单测。额外：`jwtPayload`/`applyJWTIdentity`/`oauthUserID`/`shortHash`/`ExtractIssuerInfo` 位于 work2api 的 service 层（`credential_common.go`），不在 upstream 包内，需一并搬入本仓库 `internal/account/identity.go`。

移植时必须修正（已写入 DESIGN §5）：
- `doJSON` 的 `connectCtx` 建后即弃（死代码）→ 删除，改用 `Transport.DialContext` 的 `net.Dialer{Timeout:10s}` 或仅保留请求级 ctx 超时
- `RefreshToken` 的 401/403 判定不可达（`doAuthJSON` 已提前返回错误）→ 统一归一到 `unauthorized` 错误名，否则续期失败永远不会置 `relogin_required`

**备选**：自己重写上游客户端 → 否，逆向协议的正确性风险高，移植已验证实现更稳。

### D2：存储用 JSON 文件而非 SQLite

账号数个位级、数据 < 1MB，SQLite 是过度设计（DESIGN §1/§4）。方案：`%APPDATA%\workbuddy-checkin\` 下 `credentials.json` + `settings.json`，写时「临时文件 → fsync → rename」原子替换、0600，解析失败重命名 `.corrupt-<ts>` 后以空数据继续。全局 `sync.Mutex` 串行化读写；启动一次性加载内存、写后整体落盘。

**备选**：DPAPI 加密 → 仅 Windows 生效、复杂度高，威胁模型（本机其他用户）用 NTFS 用户目录 + 0600 已覆盖；接口留在 store 层以便二期替换。

### D3：前后端契约只走绑定方法 + 轮询，不用 Wails 事件

所有业务状态由 `app*.go` 的绑定方法返回 DTO，前端轮询（`LoginStatus` 1.5s / `ListAccounts` 30s）。避免 Wails 事件通道的生命周期管理坑（DESIGN §9）。新增 `CheckinSummary()` 供托盘 tooltip 与顶栏计数，避免前端拉全量再算。

### D4：登录三段瀑布 + 去重双查

照抄 work2api `oauth_service.go` 的 `Poll`：`PollToken` → `PollAccount` → `PollAccounts`（第三段失败按 pending）。一期虽只用单账号，但必须保留第三段，否则账号信息未就绪就被消费。去重：先按 `uid_<account_uid>` 查，再按 `account_uid` 兜底——防止首次 `account_uid` 缺失时以 `oauth_<token_hash>` 落库、二次登录产生重复卡片。

### D5：调度单实体内串行

`internal/scheduler` 用 `time.Timer` 计算下次触发点（今天 HH:MM，已过则明天），触发后重算。`runAll` 全程持 `sync.Mutex`，手动签到排队。账号间 5~20s 抖动防风控（PRD §7）。重试计数持久化到 `Credential.TodayAttempts`（跨天归零），保证重启后重试上限仍生效。

**备选**：用 cron 库 → 无必要，单触发点用标准库 `time.Timer` 足够。

### D6：token 状态建模与跨天失效

`Credential` 内联今日状态字段（`TodayDate`/`TodaySuccess`/`TodayAttempts`/...），跨天靠 `TodayDate != today` 失效，不建历史表。`CreditBalance`/`CreditBalanceTotal`/`CreditBalanceAt` 记录余额与查询时间，UI 据 `CreditBalanceAt` 判断显示「—」或旧值。

### D7：系统集成照抄 health-tool 并强制 stub

托盘（手写 Win32 `Shell_NotifyIcon`）、自启（HKCU Run）、Toast（`go-toast`）全部提供 `*_windows.go` + `*_stub.go`（`//go:build !windows`），保证 Linux `go test ./...` 可编译运行。依赖仅 Wails v2 / `golang.org/x/sys/windows` / `go-toast`。

## Risks / Trade-offs

- [上游接口为逆向所得，可能变更] → 逐字移植已验证实现 + 契约单测回归；错误分类明确；失败通知用户而非静默
- [上游风控（同 IP 多账号）] → 账号间 5~20s 抖动、不并发、余额查询是唯一额外轮询接口且间隔集中常量
- [refresh_token 过期无法自动续期] → 临期 24h 主动刷新 + 到期置 `relogin_required` 提醒重登
- [杀软误报未签名 exe] → 仅写 HKCU Run、开源可查、README 说明（PRD §7）
- [托盘/通知在非 Windows 不可验证] → stub 保证可测试，真机行为列入 PRD §8 人工验收
- [数据规模小但 SaveCredential 重写整文件] → 账号数个位级可接受，`ponytail:` 注明天花板与升级路径

## Migration Plan

新仓库、无历史数据，无迁移。落地顺序按里程碑（见 tasks.md）：M1 骨架 → M2 移植+登录+存储 → M3 手动签到 → M4 调度/补签/重试/续期/余额。每阶段保持 `go test ./...` 绿。回滚 = 丢弃该 change 分支，仓库无线上影响。

## Open Questions

- 托盘/Toast 的 health-tool 参考实现本机不在（AGENTS.md 已注明），需按 DESIGN §1 描述自行实现或从作者处获取；若届时不可得，D7 的 Win32 托盘代码需以最小可用为准
- `CheckinSummary()` 聚合是否包含余额刷新时机，留待实现时按需决定（当前倾向不包含）
