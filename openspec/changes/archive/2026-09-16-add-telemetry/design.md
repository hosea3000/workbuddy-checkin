## Context

作者需要「有多少用户」这个数字，但 GitHub Releases 的 `download_count` 只能给出安装包下载次数，无法区分「下载了没装」「装了没跑」「跑了但卸载了」。`app_update.go` 的 `CheckUpdate()` 会打 GitHub API，但 GitHub 不暴露请求来源，拿不到去重后的设备数。

同时本仓库在 `README.md` 中已向用户做出明确承诺：

```
README:61  所有数据仅保存在本机，不上传任何第三方
README:64  网络仅访问 copilot.tencent.com 与 GitHub（更新检查/下载，可关闭）
```

引入上报会与这两句冲突，因此本次改动**必须同时改写文档**，否则将构成对既有用户的欺骗。

本软件托管的是用户本人的 CodeBuddy 账号与令牌（`Credential` 含 `AccessToken` / `RefreshToken` / `Email` / `AccountUID`），用户把令牌交给本软件已属高信任行为。任何上报都必须远离这些字段，否则会摧毁该信任。

现有生命周期模式（`internal/scheduler`、`internal/proxy`）均为「`New` 构造 → `Start(ctx)` → `Stop()`」，`app.go` 的 `startup` / `shutdown` 是唯一装配点。

## Goals / Non-Goals

**Goals:**

- 获得去重后的设备数（约等于日活），以及账号数量分布。
- 上报对用户完全无感：不阻塞启动、不弹窗、不报错、不产生可见日志。
- 上报内容不含任何可识别个人或账号身份的信息。
- 用户可在设置页一键关闭，关闭后立即停止。

**Non-Goals:**

- 不做服务端实现（独立项目，本次只冻结 HTTP 契约）。
- 不做首次启动弹窗告知（作者明确选择不做）。
- 不做上报内容的本地可查界面（不在本次范围）。
- 不做账号数量封顶（作者明确要求不封顶，数据仅作者自己查看）。
- 不追求精确活跃统计（连续开机超过 24 小时的机器会被自然节流，可接受）。

## Decisions

### 决策 1：每小时 tick + 「距上次成功 > 24h」门槛，而非每日定时器

```
启动 → Start(ctx)
  ticker: 每 1h 触发一次
    if !TelemetryEnabled          → skip
    if now - TelemetryLastAt ≤ 24h → skip
    上报 → 成功则 TelemetryLastAt = now
```

**理由**：

- 本应用是常驻后台工具，但 macOS 关窗即退出（`app.go:135` 的 `beforeClose`），Windows 关窗仅隐藏到托盘仍在跑。用固定 24h 定时器会在「每天只开机 8 小时」的 Windows 用户上永不触发；用小时级轮询 + 24h 门槛则每次开机都能自然命中。
- 无需管理一个 24h 周期定时器的首次触发时刻与重算逻辑，实现更简单。

**备选方案**：每日一次定时器——在「每天只开机 8 小时」场景下会漏报，不取。启动时一次性判断——长时间不关机的机器（macOS 若不退出）会漏报，且启动时刻网络未必就绪，不取。

### 决策 2：失败不写时间戳 → 语义是「每小时重试直到成功」

```
失败路径：网络错误 / 非 200 / 超时
  → 不写 TelemetryLastAt
  → 下一小时 tick 因 now-lastAt > 24h 仍成立 → 再次尝试
```

这是作者有意识选择的语义，而非「不重试」。带来的性质：用户长期连不上服务端时，其机器会持续每小时静默发一枪，直到成功为止。好处是服务器抖动、用户临时挂 VPN 等场景下数据最终不会丢。**必须写进 README 或设置页说明之外的行为注释中**，避免后来者误以为是 bug。

**替代语义（不取）**：失败也写时间戳 → 真正的「每天一次」，但服务器抖一次即丢当天数据。

### 决策 3：`device_id` 存 `settings.json`，不复用 `Credential.ID`

```go
// model/settings.go
TelemetryEnabled bool   // 默认 true
TelemetryID      string // 随机 UUID v4，Start() 时无条件生成
TelemetryLastAt  int64  // 上次成功上报的 Unix 秒
```

**生成方式**：`crypto/rand` 取 16 字节，按 RFC 4122 摆位（版本位置 4、变体位 8/9/a/b），格式化为 v4 UUID。122 位随机量，**不含时间、机器码、MAC、主机名**——这是「匿名」的技术依据。不引入第三方 UUID 依赖（约 10 行手写）。

**生成时机：`Start()` 时无条件生成，与上报开关无关。**

```
Start(ctx)
  if Settings.TelemetryID == "" → 生成 UUID v4 → 经 store 落盘
  ...再起 ticker...
```

**理由**：保证 `settings.json` 的 schema 完整性——字段是否存在不随用户开关漂移。将来若需用该 ID 关联其他本地状态，不必处理「字段可能缺失」的分支。

**代价（已知并接受）**：关闭上报的用户，其 `settings.json` 中同样会出现一个随机的 `TelemetryID`。它本身无任何身份含义，且用户打开数据目录可见，属于「可发现的透明」而非隐藏追踪。设置页文案与 README 应说明该字段的用途。

**必须走 store 接口落盘**：`store.SaveSettings` 是**整体覆盖写**（`store.go:193`），`telemetry` 包不得自行写入 `settings.json`，只能「读当前设置 → 改字段 → 整体写回」，否则会覆盖用户同期的其他设置修改。

**其他理由**：

- 复用 `Credential.ID` 会导致「一台机器托管 N 个账号 → 上报 N 个不同 ID → 用户数虚高」，且会把「某台机器有几个账号」这一关联关系暴露在去重键上。**明确排除**。
- 存 `settings.json` 而非新开 `telemetry.json`：作者选择复用现有设置文件，`store.go` 的 `readJSON` / `writeJSON` 原子写与损坏自愈能力直接适用，零新增持久化代码路径。
- 用户重置设置或换机 = 视作新设备。该误差方向与 opt-out 造成的低估相比可忽略，接受。

**备选方案**：
- 新建 `telemetry.json`——语义更干净，但需要 `store.go` 新增读写路径，不取。
- 仅在开关开启时才生成 `TelemetryID`——关闭上报的用户文件更干净，但 schema 随开关漂移，与「结构完整性」目标冲突，不取。

### 决策 4：独立 `internal/telemetry` 包，照抄 scheduler 的生命周期

`app.go:101` 的 `shutdown` 目前停止 scheduler 与 proxy。上报服务需要一个每小时 ticker，因此同样需要 `Stop()`：

```go
// app.go shutdown
if a.scheduler != nil { a.scheduler.Stop() }
if a.proxy != nil     { a.proxy.Stop() }
if a.telemetry != nil { a.telemetry.Stop() }  // 新增
```

把它做成独立包而非塞进 `app.go` 的理由：与 `internal/scheduler`、`internal/proxy` 完全同构（有 `New` / `Start` / `Stop` / `ctx`），放进 `App` 结构体会让本已承载托盘、通知、代理、签到的 `App` 继续膨胀。

### 决策 5：上报地址为包级变量常量，不暴露给用户

```go
// internal/telemetry/telemetry.go
var pingURL = "https://<国内域名>/ping"  // 变量便于测试注入 httptest
```

与 `updater.go:25` 的 `updateAPIBaseURL` 同一模式。用户既不需要也不应该改它，**不新增设置项**。**不使用** `Settings.UpdateProxy`：作者与用户均在国内，直连即可，也避免用户配了 GitHub 加速代理后上报被错误代理。

### 决策 6：请求体字段与边界

```json
{"id":"<uuid-v4>","v":"0.1.4","os":"windows","arch":"amd64","accounts":3}
```

| 字段 | 来源 | 说明 |
|------|------|------|
| `id` | `Settings.TelemetryID` | 随机 UUID，与任何账号无关 |
| `v` | `main.version` | 本地开发为 `dev` |
| `os` / `arch` | `runtime.GOOS` / `runtime.GOARCH` | 平台分布 |
| `accounts` | `len(store.ListCredentials())` | 仅数量，不封顶 |

明确**不上报**：`AccessToken`、`RefreshToken`、`token_suffix`、`Email`、`PreferredUsername`、`Nickname`、`UserID`、`AccountUID`、`EnterpriseID`、`Domain`、`Credential.ID`、签到结果、积分余额、代理开关。

`dev` 版本是否上报：**不上报**（与 `CheckUpdate` 的短路策略一致），避免开发机污染真实数据。

### 决策 6.5：日志可感知，但 `device_id` 永不完整落盘

上报对用户 **UI 层完全无感**：不弹窗、不通知、不展示错误。但「无感」不等于「不可发现」——网络出站连接、`settings.json` 中可见的 `TelemetryID`、以及日志，都是用户可察觉的路径。这是有意保留的「平时无感、想查能查到」的平衡。

日志策略（`app.log` 是 0600 明文文件，但用户报 bug 时可能整份贴到 issue）：

| 事件 | 是否写日志 | 内容 |
|------|-----------|------|
| 生成 `TelemetryID` | 是 | 仅前 8 位，如 `[telemetry] 已生成设备标识 f47ac10b…` |
| 上报成功 | 是 | `[telemetry] 上报成功 v=0.1.4 os=windows arch=amd64 accounts=3` |
| 上报失败 | 是 | `[telemetry] 上报失败: <err>`（仅在确实发起请求后） |
| 被 24h 门槛节流 | 否 | 每小时都写会淹没日志 |
| 开关关闭跳过 | 否 | 无信息量 |

**硬约束：`device_id` 绝不以完整形式出现在日志、UI 或通知中**——与现有「令牌永不出现」的约束同构。理由：完整 ID 一旦随 bug 报告流出，即成为该设备的可关联身份。

### 决策 7：服务端契约冻结

```
POST <pingURL>
Content-Type: application/json
Body: {"id":...,"v":...,"os":...,"arch":...,"accounts":...}
Response: HTTP 200，无响应体

- 无鉴权、无签名、无重试语义、无幂等键（客户端只管发，服务端按 id 去重）
- 客户端不读取响应体，除状态码外不解析任何内容
- 服务端实现不在本仓库
```

**防刷责任全部在服务端，客户端不做任何门槛。** 上报地址随开源代码公开可见，且客户端运行在攻击者手中——任何客户端可计算的凭据（固定 header、编译期常量、甚至 HMAC 密钥）都能被读代码者复现，客户端侧鉴权在原理上不成立。因此：

- 客户端 SHALL NOT 添加签名、固定请求头、编译期密钥等伪鉴权手段（会给后来者错误的安全感）。
- 服务端（独立项目）负责防御：按 IP 限速、新 `id` 突增检测与告警。
- 该数字定位为**量级参考而非精确审计**：少量伪造不影响判断，突增 10 倍视为被刷并忽略。

## Risks / Trade-offs

- **[README 既有隐私承诺被打破]** → 本 change 的 MODIFIED 文档任务必须与代码同批完成，不允许只改代码不改 README。两个承诺句（第 61、64 行）都要改写为「可选、默认开启、仅上传三样非敏感信息、可关闭」。
- **[老用户升级后无任何提示即开始上报]** → 作者明确选择不做首次弹窗。缓解措施：设置页开关的文案必须清晰可读；README 改写。**风险自担，记录在此。**
- **[连续开机 > 24h 的机器漏报，导致日活被低估]** → 可接受，量级判断不受影响。
- **[用户长期离线时每小时重试]** → 单次请求极小、失败静默，实际负担可忽略；语义已在决策 2 说明。
- **[账号数量不封顶，极端值会暴露「批量养号」信号]** → 数据仅作者本人查看，不对外公开；如未来要公开统计页面，必须先重新评估是否封顶。
- **[`device_id` 随 bug 报告流出]** → 日志只打前 8 位、UI 不展示完整 ID；已列为硬约束（决策 6.5）。
- **[上报地址公开，可被刷量伪造]** → 客户端不做伪鉴权（原理上无效），防刷全交服务端（IP 限速 + 新 id 突增告警）；该数字定位为量级参考，且不对外公开，被刷的代价接近于零。
- **[服务端成为单点：服务端挂掉则数据不丢但持续重试]** → 服务端只需「收下、落库、返回 200」，无状态、无鉴权，可用性压力极低。
- **[`store.ListCredentials()` 加锁读取，上报在 goroutine 中调用]** → 该接口已返回深拷贝且自带锁，无并发问题。

## Migration Plan

1. `model/settings.go` 新增三个字段；`DefaultSettings()` 中 `TelemetryEnabled: true`。
2. 新增 `internal/telemetry`：`NewService(store, version)`, `Start(ctx)`, `Stop()`；`Start` 内生成/复用 `TelemetryID` 并起 ticker goroutine。
3. `app.go`：`NewApp` 构造、`startup` 启动、`shutdown` 停止。
4. `app_settings.go`：`GetSettings` / `SaveSettings` 透传新字段（结构体直存，无额外代码）。
5. 前端设置页新增「帮助改进 WorkBuddy」区块。
6. 改写 `README.md` 隐私段落。
7. 测试：httptest 注入 `pingURL`，断言超 24h 才发、失败不写时间戳、关闭后不发、请求体字段与不含敏感字段。
8. `go vet ./... && go test ./...`；`cd frontend && npm run build`；`GOOS=darwin GOARCH=arm64 go build ./...`。
9. 回滚：还原上述文件即可，无数据迁移；已生成的 `TelemetryID` 留在 `settings.json` 中不影响功能。

## Open Questions

- 无。（默认开、每天一次、无首次弹窗、不封顶、地址硬编码、服务端拆开——均已由作者拍板。）
