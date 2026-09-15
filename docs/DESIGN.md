# workbuddy-checkin 技术设计

> 配套文档：[PRD.md](./PRD.md)。本文只讲「怎么实现」，需求条目用 `F*` 引用 PRD。

## 1. 技术选型

| 维度 | 选择 | 理由 |
|---|---|---|
| 桌面框架 | Wails v2（对齐 health-tool v2.13.0） | 已在 health-tool 验证：单 exe、交叉编译、托盘/自启/更新器都有现成实现 |
| 后端 | Go 1.25 | 上游协议实现从 work2api 直接移植（纯 stdlib） |
| 前端 | Vue 3 + Vite + TypeScript + Pinia + Tailwind v4 | 与 work2api 管理台同一套心智；`wails generate module` 生成绑定 |
| 存储 | JSON 文件（`os.UserConfigDir()/workbuddy-checkin/`） | 账号数个位级、数据 < 1 MB，SQLite 是过度设计；health-tool 已验证 |
| 网络 | 标准库 `net/http` | 上游客户端无第三方依赖 |
| 系统集成（Windows） | `golang.org/x/sys/windows` + `go-toast` | 自启（HKCU Run）、托盘（手写 Win32）、Toast 通知 |
| 系统集成（macOS） | 系统命令 + 手写 cgo/Objective-C（无新依赖） | 自启走 LaunchAgent + `launchctl`、通知走 `osascript`、更新走 `hdiutil`/`open`；菜单栏图标走 cgo `NSStatusItem`（`osascript` 无法创建常驻状态栏项） |
| 无 GUI 平台 | `*_stub.go`（`//go:build !windows && !darwin`） | Linux 上可 `go test ./...` 与 `wails dev` 调试业务逻辑 |

**平台能力矩阵**：

| 能力 | Windows | macOS |
|---|---|---|
| 托盘 | Shell_NotifyIcon（手写 Win32） | 菜单栏图标 `NSStatusItem`（手写 cgo/Objective-C） |
| 关窗语义 | 隐藏到托盘（`beforeClose` 返回 true + `HideWindowOnClose`） | 隐藏窗口到菜单栏（`beforeClose` 返回 true + `wruntime.WindowHide` → `orderOut`） |
| Dock 图标 | 不适用 | 无（`Info.plist` 的 `LSUIElement` + `startup` 兜底 `Accessory` 策略） |
| 自启 | HKCU Run | LaunchAgent plist + `launchctl bootstrap/bootout` |
| 静默启动 | `--hidden` | `--hidden`（LaunchAgent 传参） |
| 静默后唤起 | 双击 exe / 托盘双击 | 再次双击 `.app`（单实例锁 `OnSecondInstanceLaunch`） |
| 通知 | go-toast | `osascript display notification` |
| 应用内更新 | 下载 exe → `.new` → bat 覆盖重启 | 下载 dmg → `open` 挂载 → 用户拖入 Applications |
| 更新资产名 | `workbuddy-checkin.exe` | `WorkBuddy-checkin-<GOARCH>.dmg` |
| 唤醒补签钩子 | `WM_POWERBROADCAST` / `WM_WTSSESSION_CHANGE` | 无（每小时巡检兜底） |
| 打开数据目录 | `explorer` | `open` |
| 分发 | exe + per-user NSIS 安装器 | 未签名 dmg（arm64 / amd64 各一） |

**不引入**：gin、gorm、viper、wire、redis、sqlite、任何 DI 框架。

**直接依赖预算（≤ 5 个业务依赖，不计 Wails 自身）**：

| 依赖 | 用途 |
|---|---|
| `github.com/wailsapp/wails/v2` | 桌面框架（含其传递依赖，不额外计数） |
| `golang.org/x/sys/windows` | HKCU Run 自启、Win32 托盘 |
| `github.com/go-toast/toast` | Windows Toast 通知 |
| `*`（暂无） | 其余一律用标准库 |

（`github.com/google/uuid` **不引入**：`headers.go` 用 `crypto/rand` 自产 UUID；`Credential.ID` 同样自产。）

## 2. 仓库结构

```
workbuddy-checkin/
├── main.go                  # Wails 入口（单实例锁、StartHidden、OnBeforeClose）
├── app.go                   # App 结构 + 所有绑定方法（唯一前后端契约面）
├── app_*.go                 # 按域拆分的绑定方法（account/checkin/settings）
├── tray_windows.go          # 托盘（Shell_NotifyIcon，手写 Win32）
├── tray_darwin.go / .m / .h # 菜单栏图标（NSStatusItem，手写 cgo/Objective-C）
├── tray_stub.go             # 其余平台空实现（//go:build !windows && !darwin）
├── autostart_windows.go     # HKCU Run
├── autostart_darwin.go      # LaunchAgent（+ autostart_macos.go：跨平台可测的 plist 生成）
├── notification_windows.go  # Toast
├── notification_darwin.go   # osascript
├── updater*.go              # P2：GitHub Release 检查/下载/替换
├── updater_apply_darwin.go  # 下载 dmg → open 挂载
├── model/                   # 数据契约（凭证、设置、视图 DTO）
├── store/                   # JSON 持久化（原子写 + 损坏自愈）
├── upstream/codebuddy/      # 上游协议客户端（移植自 work2api，纯 stdlib）
│   ├── client.go            # 签到 / 额度
│   ├── auth.go              # OAuth 设备授权 + refresh
│   ├── headers.go           # CLI 伪装头集（逐字对齐，禁改）
│   ├── errors.go            # 受控错误分类
│   └── endpoint.go
├── internal/
│   ├── account/             # 账号服务：登录编排、续期、去重、状态机
│   ├── checkin/             # 签到执行 + 调度 + 补签 + 重试 + 余额刷新
│   └── scheduler/           # 定时器（下次触发点计算、跨天/唤醒触发）
├── frontend/                # Vue3（Vite），wailsjs 绑定生成物
├── build/                   # 图标、windows 资源
├── .github/workflows/release.yml
└── openspec/                # spec 驱动（可选，见 §11）
```

依赖方向单向无环：`main → internal/* → store → model`，`internal/* → upstream/codebuddy`。

## 3. 核心模型

```go
// model/credential.go
type Credential struct {
    ID               string  `json:"id"`                // uuid
    UserID           string  `json:"user_id"`           // "uid_<account_uid>"，稳定主键
    AccountUID       string  `json:"account_uid"`
    Nickname         string  `json:"nickname"`
    PreferredUsername string `json:"preferred_username"`
    Email            string  `json:"email"`
    EnterpriseID     string  `json:"enterprise_id"`
    Domain           string  `json:"domain"`            // JWT iss / 上游下发
    AccessToken      string  `json:"access_token"`      // 明文（见 §8）
    RefreshToken     string  `json:"refresh_token"`
    ExpiresAt        int64   `json:"expires_at"`        // Unix 秒
    RefreshExpiresAt int64   `json:"refresh_expires_at"`
    Status           string  `json:"status"`            // active | relogin_required
    // 今日签到状态（无历史记录表，幂等与补签判定都读这里；跨天自动失效）
    TodayDate        string   `json:"today_date"`        // 本地时区 YYYY-MM-DD
    TodaySuccess     bool     `json:"today_success"`
    TodayMessage     string   `json:"today_message"`     // "已签到" / 失败原因
    TodayCredit      *float64 `json:"today_credit"`      // 本次获得积分，可能没有
    TodayAttemptedAt int64    `json:"today_attempted_at"`
    TodayAttempts    int      `json:"today_attempts"`    // 当日尝试次数（跨天清零；用于失败通知节流）
    CreditBalance    *float64 `json:"credit_balance"`    // 积分余额 = FetchQuotaPersonal 的 remaining
    CreditBalanceTotal *float64 `json:"credit_balance_total"` // 周期总额度（UI 可展示 "640 / 1000"）
    CreditBalanceAt  int64    `json:"credit_balance_at"` // 上次成功查询余额的时间，UI 据此显示「—」或旧值
    CreatedAt        int64   `json:"created_at"`
    UpdatedAt        int64   `json:"updated_at"`
}

// model/settings.go
type Settings struct {
    AutoStart   bool
    UpdateProxy string // 更新检查与下载的加速代理前缀，空为直连
}
```

视图 DTO（`model/view.go`）只暴露 `token_suffix`（末 8 位）而非令牌本体，绑定方法一律返回 DTO。

## 4. 存储

```
%APPDATA%\workbuddy-checkin\
├── credentials.json     # []Credential
├── settings.json        # Settings
└── app.log              # 运行日志（不含令牌）
```

- 写入：`写临时文件 → fsync → rename`（原子替换），权限 0600
- 读取：文件不存在 → 返回空数据；JSON 解析失败 → 重命名为 `<name>.corrupt-<时间戳>` 并以空数据继续（F 鲁棒性）
- 无并发写：所有读写经 `store` 包的 `sync.Mutex`（单进程单实例，锁只防 UI/调度并发）
- 启动时一次性加载进内存，写操作后立即落盘（数据量小，无需脏标记）

## 5. 上游客户端（移植策略）

从 work2api `internal/upstream/codebuddy/` **逐字拷贝**并只做「去内部依赖」改造：

| 源文件 | 动作 |
|---|---|
| `headers.go` | 直接拷贝（`github.com/google/uuid` 换成 `crypto/rand` 自产 UUID，去掉该依赖） |
| `auth.go` | 直接拷贝（`StartAuth` / `PollToken` / `PollAccount` / `PollAccounts` / `RefreshToken` / `parseTokenData` / `parseAccount` / `epochOrPtr`） |
| `client.go` | 只保留 `Checkin` + `doJSON` + `handleNon200` + `FetchQuotaPersonal`（积分余额） |
| `errors.go` / `endpoint.go` | 直接拷贝 |
| `identity.go` | **新增**：从 work2api `internal/service/credential_common.go` 搬 `jwtPayload` / `applyJWTIdentity` / `oauthUserID` / `shortHash` / `ExtractIssuerInfo` 到 `internal/account/`（这些在 work2api 属 service 层，不在 `upstream/codebuddy/`，移植清单勿漏） |
| 丢弃 | `chat/sse/events/quota(FetchModels)/headers(IDE 变体)` 等网关专属部分 |

同时拷贝 work2api 的对应单测（`auth_test.go` / `headers_test.go`）——这些测试是上游协议契约的回归网。

**移植时必须修正的上游 bug**（勿逐字照抄）：

1. `client.go` 的 `doJSON` 里 `connectCtx` 建了又丢弃（`_ = connectCtx`），是死代码。移植时删除该段，改用 `Transport.DialContext` 的 `net.Dialer{Timeout: 10s}` 实现连接超时，或仅保留请求级 ctx 超时。
2. `auth.go` 的 `RefreshToken` 依赖 `doAuthJSON` 返回 401/403 分支，但 `doAuthJSON` 在 401/403 时已直接返回错误，导致 `parseRefreshResponse` 永远走不到 401/403 判定，凭证被拒会误报为 `refresh_failed`。移植时改为在 `RefreshToken` 中判断 `errors.As(err, &AuthError{Name: "unauthorized"})`，或将 401/403 归一到 `unauthorized` 错误名，保证 §6.2 的 `relogin_required` 置位能触发。

**常量冻结**：`cli_version = 2.107.0`、`OpenAIJSPackageVersion`、`NodeRuntimeVersion`、端点路径全部集中在一处，升级只需改一个文件。

## 6. 账号服务（`internal/account`）

### 6.1 登录编排（F1）

```
UI: Login()
 ├─ client.StartAuth() ────────────▶ POST /v2/plugin/auth/state?platform=CLI
 │                                    ← { state, authUrl }
 ├─ runtime.BrowserOpenURL(authUrl)      // 系统浏览器
 ├─ 每 5s client.PollToken(state) ─▶ GET /v2/plugin/auth/token?state=
 │     code=11217 → 继续等待
 │     code≠0    → 业务错误 → 弹窗人话提示（errors.go 映射表）
 │     code=0    → TokenData
 ├─ client.PollAccount(state, td) ─▶ GET /v2/plugin/login/account?state=
 │     code=12151 → 继续等待（账号信息准备中）
 ├─ client.PollAccounts(td) ────────▶ GET /v2/plugin/accounts
 │     失败 → 按 pending 处理（等待账号列表准备完成）
 ├─ 组装 Credential（applyJWTIdentity 补昵称/邮箱，见 §5 identity.go）
 ├─ 去重：按 `uid_<account_uid>` 查已存在 → 原位更新（状态置 active）
 │     再按 account_uid 兜底查一次（防 account.UID 缺失时首次用 oauth_<hash> 落库、二次登录产生重复卡片）
 └─ store.Save + 刷新 UI
```

- 三段瀑布（token → account → accounts）照抄 work2api `oauth_service.go` 的 `Poll`；一期单账号虽只用 `PollAccount` 的当前账号，但**必须保留 `PollAccounts` 这一跳**，否则账号信息未就绪就被消费。

- 前端用**同一个绑定方法轮询**：`LoginStatus()` 返回 `{stage, done, error, credential}`，前端每 1.5s 调一次（避免 Wails 事件通道的生命周期管理）
- 取消：`CancelLogin()` 停止轮询并作废 state（对应 work2api 的 `Consume` 墓碑，本应用只需内存标记）
- 状态机：`idle → awaiting_login → awaiting_account → done | failed | canceled`，10 分钟 TTL

### 6.2 续期（F6）

- `RefreshIfNeeded(cred)`：`now >= ExpiresAt - 24h` 且 `RefreshToken != ""` → `client.RefreshToken()`
- 成功：更新 access_token / refresh_token（上游会轮换）/ 有效期
- 失败：错误名 `unauthorized`（含上游 401/403）→ `Status = relogin_required` + 通知一次（去重：状态变更时才通知）；其余错误（`refresh_failed` / `ip_restricted` / 网络）只记日志、不置位
- 调用点：启动、每次签到前、签到后、调度唤醒时

## 7. 签到与调度（`internal/checkin`、`internal/scheduler`）

### 7.1 单账号签到

```
performCheckin(cred):
  ensureTokenFresh(cred)                       # §6.2
  success, code, msg, credit, err = client.Checkin(snapshot)
  success = (code == 0) || strings.Contains(msg, "已签到")   # 与上游语义一致
  cred.TodayDate/TodaySuccess/TodayMessage/TodayCredit/TodayAttemptedAt = ...
  refreshQuota(cred)                          # 签到成功后查一次余额（失败不影响签到结果）
  store.SaveCredential(cred)
```

`CredentialSnapshot`（headers.go 定义）由 `Credential` 映射——**注意字段语义与 `GenerateHeaders` 的取用顺序**：

| Snapshot 字段 | 取自 `Credential` | 说明 |
|---|---|---|
| `BearerToken` | `AccessToken` | 必填，缺则 `GenerateHeaders` 直接报错 |
| `AccountUID` | `AccountUID` | **`X-User-Id` 实际取此值**（headers.go 优先 AccountUID） |
| `UserID` | `UserID` | 仅当 `AccountUID` 为空时兜底。**注意 `Credential.UserID` 是 `uid_<account_uid>` 带前缀，勿在正常路径用它填头** |
| `Domain` | `Domain` | 经 `safeDomainRe` 校验，非法则回退 Host |
| `EnterpriseID` | `EnterpriseID` | 有值才注入 `X-Enterprise-Id` / `X-Tenant-Id` |

### 7.1.1 积分余额刷新

`refreshQuota(cred)` = `client.FetchQuotaPersonal(ctx, snapshot)` → 返回 `(total, remaining, err)`，写 `CreditBalance = remaining` / `CreditBalanceTotal = total` / `CreditBalanceAt`（失败只记日志，**不**改 `Status`、**不**通知）。

触发点：① 应用启动时 ② 每次签到成功后 ③ 定时每 1 小时 ④ 手动 `RefreshQuota(id)`（卡片上的刷新按钮）。

- 与签到共用同一个每小时 tick（先巡检签到，再刷余额），单账号串行、账号间沿用 5~20s 抖动
- `relogin_required` 账号跳过，不空打上游
- 上游风控约束（PRD §7）：余额查询是唯一的额外轮询接口，不因它增加请求频率；间隔常量集中一处便于调整

### 7.2 调度器

```
        ┌──────────────────────────────────────────────────────┐
        │ scheduler.loop(ctx)                                  │
        │                                                      │
        │   time.NewTicker(time.Hour)                          │
        │        │                                             │
        │        ▼                                             │
        │   RunAll()            ← 签到巡检（跳过已签/待重登）   │
        │   RefreshAllQuotas()  ← 余额刷新                      │
        └──────────────────────────────────────────────────────┘

  触发源：① 每小时 tick  ② 启动（托盘就绪后）立即 RunAll
          ③ 休眠唤醒 / 会话解锁 立即 RunAll
```

- 无固定签到时间：去掉 `NextTrigger` 与长 timer，改为单个每小时 ticker（`time.NewTicker(time.Hour)`）
- 巡检即幂等：`RunAll` 跳过 `TodaySuccess && TodayDate == 今天` 与 `relogin_required` 的账号
- 重试：不设固定 30 分钟间隔与每日次数上限；失败账号在下一次每小时巡检自动重试（1h 间隔 + 账号间 5~20s 抖动节流）
- 唤醒事件：托盘消息循环的 `WM_POWERBROADCAST` / `WM_WTSSESSION_CHANGE` 钩子直接调用 `scheduler.RunAll()`
- 并发保护：`RunAll` 全程持 `runMu`，手动签到排队等待（不并发打上游）

### 7.3 补签（F5）

补签与每小时巡检是同一段逻辑：`RunAll()` 对「今天未成功签到」且非 `relogin_required` 的账号执行签到，已成功账号跳过。
调用点：应用启动（托盘就绪后）、休眠唤醒、会话解锁、每小时 tick。补签、成功/失败通知均为默认行为，不提供开关。

## 8. 安全与隐私

- 令牌明文存于 `%APPDATA%`（0600）。**为什么不用 DPAPI**：DPAPI 加密只在 Windows 生效，会让 Linux 下的 `go test` 与数据迁移变复杂；本工具的威胁模型是「本机其他用户读取」，NTFS 用户目录 + 0600 已覆盖。`x/sys/windows` 已提供 `CryptProtectData`，若后续要提高安全等级，只改 `store` 一层即可（见 §12 开放问题）
- 日志：只记 `user_id` 前 8 位、HTTP 状态、错误类别；**永不打印 token/refresh_token/完整响应体**
- UI：令牌只显示末 8 位（`token_suffix`），无「查看明文」入口
- 通知：只含账号昵称 + 结果，不含任何标识串
- 网络：仅 `copilot.tencent.com`（HTTPS，证书校验走默认 `http.Transport`）+ `api.github.com`（仅更新检查，可关）
- 不收集遥测、不写注册表（除 HKCU Run 自启）、不装服务

## 9. 前后端契约

所有绑定方法在 `app.go` 系列文件，返回 DTO（`model/view.go`）。前端**不使用** Wails 事件做业务状态，只做轮询（简单、可测、无生命周期坑）。

| 方法 | 入参 | 返回 | 对应 |
|---|---|---|---|
| `ListAccounts()` | — | `[]AccountView` | F2 |
| `CheckinSummary()` | — | `{checkedIn, total, reloginRequired}` | F8 托盘 tooltip / 顶栏计数，避免前端拉全量再算 |
| `StartLogin()` | — | `{authUrl, expiresIn}` | F1 |
| `LoginStatus()` | — | `{stage, done, error, account?}` | F1 |
| `CancelLogin()` | — | `{ok}` | F1 |
| `CheckinNow(id)` | `id` | `CheckinResult` | F3 |
| `RefreshQuota(id)` | `id` | `{balance, total, at}` | F2 积分余额 |
| `CheckinAll()` | — | `[]CheckinResult` | F4 |
| `DeleteAccount(id)` | `id` | `{ok}` | F2 |
| `GetSettings()` / `SaveSettings(s)` | — | `Settings` | F7 |
| `OpenDataDir()` | — | `{ok}` | F7 |
| `CheckUpdate()`（P2） | — | `{hasUpdate, version, url}` | F9 |

`AccountView`：`{id, nickname, email, status, today: {checkedIn, time, credit, message}, creditBalance, creditBalanceTotal, tokenSuffix, expiresAt, todayAttempts}`。

## 10. 前端设计

```
frontend/src/
├── main.ts
├── App.vue                 # 布局：顶栏（标题/设置/状态）+ 路由视图
├── router.ts               # /accounts（默认）、/settings、/welcome（首启）
├── stores/accounts.ts      # Pinia：账号列表 + 轮询（登录中 1.5s / 平时 30s）
├── stores/settings.ts
├── api/bindings.ts         # 封装 wailsjs 生成的方法，统一错误 → toast
├── components/
│   ├── AccountCard.vue     # 昵称/状态徽章/今日签到/积分余额（可点刷新）/操作按钮
│   ├── StatusBadge.vue     # 已签到 / 待签到 / 需重新登录 / 签到中
│   ├── LoginDialog.vue     # 授权链接 + 复制 + 打开浏览器 + 倒计时 + 取消
│   └── ui/                 # Button / Card / Toast / Switch / TimePicker
└── views/{AccountsView,SettingsView,WelcomeView}.vue
```

- 视觉：沿用 work2api 管理台的 Tailwind + `@lucide/vue` 图标；深浅色跟随系统
- 状态轮询：`LoginStatus` 1.5s（仅登录弹窗打开时）；`ListAccounts` 30s（同步托盘/后台签到带来的状态变化）
- Toast：`CToastHost` 风格（自研 60 行，不引第三方）

## 11. OpenSpec 工作流（可选）

health-tool 用 openspec 管了 10 个能力 spec。本项目管理粒度建议：

```
openspec/specs/
├── account-login/         # F1 设备授权 + 去重
├── account-management/    # F2 列表/删除
├── manual-checkin/        # F3
├── scheduled-checkin/     # F4 定时 + 抖动 + 重试
├── catch-up-checkin/      # F5 补签
├── token-refresh/         # F6 续期
├── quota-refresh/         # 积分余额刷新（§7.1.1）
├── settings/              # F7
├── tray-integration/      # F8 托盘/自启/单实例/静默启动
├── first-run/             # F10
├── app-update/            # F9 检查更新 + 下载 + 自替换升级
└── ci-autorelease/        # tag → Release exe + per-user 中文 NSIS 安装器（照抄 health-tool）
```

## 12. 开放问题

| # | 问题 | 倾向 |
|---|---|---|
| 1 | 令牌是否用 DPAPI 加密 | 一期明文 + 0600，接口留在 store 层；二期再评估 |
| 2 | ~~是否展示剩余额度（`FetchQuotaPersonal`）~~ | 已定：卡片显示积分余额（= remaining），启动/签到后/每 1h/手动刷新（见 §7.1.1） |
| 3 | ~~签到点是否支持多个（如 09:30 + 21:30）~~ | 已定：改为每小时巡检，无固定签到点，天然容错 |
| 4 | 是否需要「便携模式」（数据放 exe 同目录） | 需要用户级自启路径配合，成本不高；倾向 v1.1 |
| 5 | ~~更新器是否一期就做~~ | 已定：随安装器一起落地（移植 health-tool updater，见 `app-update`） |
| 6 | 前端是否复用 work2api 的 UI 组件库 | 两端无共享 module，倾向各自维护但抄样式与交互 |
| 7 | 是否需要「导出诊断包」 | 用户反馈问题时很有用（脱敏日志）；倾向 P2 |
| 8 | ~~macOS 静默启动后能否经单实例锁唤出窗口~~ | 已实现路径（`OnSecondInstanceLaunch` → `showWindow`），**待 macOS 真机实测**；失败则改用 `ShowApplication` 等底层显示 |
| 9 | macOS 补签无唤醒钩子 | 接受每小时巡检兜底（最迟延迟 1h）；若体验不足，后续在 `showWindow` 顺带 `RunAll` |
| 10 | macOS 菜单栏图标能否与 Wails 共存（cgo `NSStatusItem`） | 已选独立实现，**完全不碰 `NSApplication.delegate`**，避免与 Wails 的 AppDelegate 冲突（`energye/systray` 会抢占 delegate，故弃用）；待 macOS 真机实测 |
| 11 | macOS `Accessory` 激活策略下窗口能否抢到焦点 | 依赖 `wruntime.WindowShow` 内部的 `makeKeyAndOrderFront` + `activateIgnoringOtherApps`；**待 macOS 真机实测**，不足则临时切回 `Regular`（Dock 图标会闪现） |
| 12 | macOS ⌘Q 是否应直接退出而非被 `beforeClose` 拦截 | **待 macOS 真机确定预期交互**；当前统一拦截为「隐藏窗口」，如不符合预期则在 ⌘Q 路径直接置 `quitting` |

## 13. 验证方式

| 层 | 手段 |
|---|---|
| 上游协议 | 从 work2api 移植的 `upstream/codebuddy/*_test.go`（httptest 打桩） |
| 存储 | `store/*_test.go`：原子写、损坏文件自愈、跨天状态失效 |
| 调度 | `scheduler/*_test.go`：注入假时钟，验证下次触发点、补签条件、重试次数上限 |
| 账号服务 | `internal/account/*_test.go`：登录状态机（假 client）、去重、续期失败置位 |
| 余额刷新 | `internal/checkin/*_test.go`：假 client 验证刷新触发点、失败不改状态、`relogin_required` 跳过 |
| 端到端（Windows / macOS） | PRD §8 验收清单人工执行；`wails dev` 下浏览器调 UI。macOS 额外验证：关窗隐藏不退出、菜单栏左键弹菜单、Dock 无图标、`Accessory` 下窗口焦点、⌘Q 语义、自启静默后双击唤起、dmg 更新流程 |
| CI | `go vet ./... && go test ./...`（Linux 跑 stub 分支）+ Windows job（`wails build -nsis ...`）+ macOS 矩阵 job（`darwin/amd64`、`darwin/arm64` 各出 dmg），发布 exe / 安装器 / 两个 dmg 四资产 |
