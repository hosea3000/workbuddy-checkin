## 1. 项目骨架（M1）

- [x] 1.1 初始化 Go module（`github.com/hosea3000/workbuddy-checkin`）与目录结构（`main.go`/`app*.go`/`model/`/`store/`/`upstream/codebuddy/`/`internal/`/`frontend/`）
- [x] 1.2 接入 Wails v2：`main.go` 单实例锁、`StartHidden`、`OnBeforeClose` 隐藏到托盘；`wails dev` 可启动
- [x] 1.3 实现 `tray_windows.go`（Shell_NotifyIcon：菜单/双击/tooltip）+ `tray_stub.go`（`//go:build !windows`）
- [x] 1.4 实现 `autostart_windows.go`（HKCU Run 读写）+ `autostart_stub.go`
- [x] 1.5 实现 `notification_windows.go`（go-toast）+ `notification_stub.go`
- [x] 1.6 初始化前端 Vue3/Vite/TS/Pinia/Tailwind v4，App 布局 + 路由（`/accounts`、`/settings`、`/welcome`）
- [x] 1.7 `wails generate module` 生成绑定，`api/bindings.ts` 封装 + 统一错误 toast
- [x] 1.8 验证：`go vet ./... && go test ./...` 在 Linux 通过；`wails build -platform windows/amd64` 产出 exe

## 2. 上游客户端移植（M2）

- [x] 2.1 拷贝 `headers.go`/`endpoint.go`/`errors.go`，`google/uuid` 换 `crypto/rand` 自产 UUID；拷贝 `headers_test.go` 并跑通
- [x] 2.2 拷贝 `auth.go`（StartAuth/PollToken/PollAccount/PollAccounts/RefreshToken/parseTokenData/parseAccount/epochOrPtr）+ `auth_test.go`
- [x] 2.3 修正 `auth.go` 的 `RefreshToken` 401/403 错误分类，归一到 `unauthorized`
- [x] 2.4 精简 `client.go` 只保留 `doJSON`/`handleNon200`/`Checkin`/`FetchQuotaPersonal`；删除 `doJSON` 的死代码并实现连接超时
- [x] 2.5 新增 `internal/account/identity.go`，搬入 `jwtPayload`/`applyJWTIdentity`/`oauthUserID`/`shortHash`/`ExtractIssuerInfo` 及对应单测
- [x] 2.6 引入 `golang.org/x/sys/windows`、`github.com/go-toast/toast`，确认依赖清单符合 DESIGN §1

## 3. 数据模型与存储（M2）

- [x] 3.1 定义 `model/credential.go`（含 `TodayDate`/`TodaySuccess`/`TodayAttempts`/`CreditBalance`/`CreditBalanceTotal`/`CreditBalanceAt`/`Status`）
- [x] 3.2 定义 `model/settings.go` 与默认值，及 `model/view.go`（`AccountView`/`CheckinResult`/`CheckinSummary`，仅暴露 `token_suffix`）
- [x] 3.3 实现 `store` 包：原子写（temp→fsync→rename）、0600、`sync.Mutex`、损坏自愈（`.corrupt-<ts>`）、启动加载/写后落盘
- [x] 3.4 实现跨天状态失效判定与设置缺失字段默认值填充
- [x] 3.5 `store` 单测：原子写、损坏自愈、跨天失效、设置默认值

## 4. 登录编排（M2）

- [x] 4.1 实现 `internal/account` 状态机（`idle→awaiting_login→awaiting_account→done|failed|canceled`，10min TTL）与三段轮询编排
- [x] 4.2 实现去重双查（`uid_<account_uid>` → `account_uid` 兜底）与原位更新
- [x] 4.3 实现业务错误人话映射（12005/11212/11216/10081）
- [x] 4.4 绑定方法：`StartLogin()`/`LoginStatus()`/`CancelLogin()`；`account` 单测（假 client：状态机、去重、取消、TTL）
- [x] 4.5 前端 `LoginDialog.vue`：授权链接/复制/重开浏览器/倒计时/取消，1.5s 轮询、成功后自动关闭并刷新卡片

## 5. 手动签到与账号管理（M3）

- [x] 5.1 实现 `internal/checkin` 单账号 `performCheckin`（`CredentialSnapshot` 正确映射：`BearerToken`/`AccountUID`/`UserID`/`Domain`/`EnterpriseID`）
- [x] 5.2 绑定方法 `CheckinNow(id)`/`ListAccounts()`/`DeleteAccount(id)`/`CheckinSummary()`
- [x] 5.3 前端 `AccountCard.vue`/`StatusBadge.vue`：卡片字段、立即签到 loading/结果、删除二次确认、待重登红色徽章
- [x] 5.4 前端账号列表 30s 轮询同步后台状态
- [x] 5.5 `checkin` 单测：成功/已签/失败判定、Snapshot 映射

## 6. 调度、补签与重试（M4）

- [x] 6.1 实现 `internal/scheduler` 下次触发点计算（今天 HH:MM，已过则明天；触发后重算）+ 假时钟单测
- [x] 6.2 实现 `runAll`（逐个、5~20s 抖动、`sync.Mutex` 串行、跳过已成功与 `relogin_required`）
- [x] 6.3 实现失败重试（30min、`TodayAttempts` 持久化上限 3、跨天归零）+ 单测
- [x] 6.4 实现 `CatchUp()`（启动/唤醒/跨天）+ 单测
- [x] 6.5 转发 `WM_POWERBROADCAST`/`WM_WTSSESSION_CHANGE` 到 `scheduler.Wake()`
- [x] 6.6 绑定方法 `CheckinAll()`；`scheduler` 单测（触发点、补签条件、重试上限）

## 7. 续期与余额刷新（M4）

- [x] 7.1 实现 `RefreshIfNeeded`（临期 24h 判定 + 调用点）与失败置 `relogin_required` + 通知去重
- [x] 7.2 实现 `refreshQuota`（写 `CreditBalance`/`CreditBalanceTotal`/`CreditBalanceAt`，失败仅日志）
- [x] 7.3 余额刷新触发点：启动/签到后/每 1h/手动；`RefreshQuota(id)` 绑定方法返回 `{balance, total, at}`
- [x] 7.4 前端卡片余额展示（`—` 或旧值、可点刷新）+ `total` 展示
- [x] 7.5 单测：续期成功/失败置位、余额失败不改状态、`relogin_required` 跳过

## 8. 设置、引导与通知（M1/M5 交界，本 change 含）

- [x] 8.1 绑定方法 `GetSettings()`/`SaveSettings(s)`/`OpenDataDir()` + 前端 `SettingsView.vue`
- [x] 8.2 前端 `WelcomeView.vue`（无凭证时默认路由，添加账号按钮 + 自启默认勾选）
- [x] 8.3 Toast 通知接入签到成功/失败与待重登，受设置开关控制
- [x] 8.4 首次关闭窗口最小化到托盘的一次性提示

## 9. 集成验证

- [x] 9.1 全量 `go vet ./... && go test ./...`（Linux stub 分支）通过
- [x] 9.2 `wails build -platform windows/amd64` 产出单文件 exe ≤ 15MB，冷启动 ≤ 2s
- [x] 9.3 按 PRD §8 验收清单逐条人工验证（1~9 项；第 10 项余额相关）
- [x] 9.4 核对日志/界面/通知中不出现任何 token（仅 `token_suffix`）
