## 1. 设置项与模型

- [x] 1.1 `model/settings.go` 的 `Settings` 新增 `TelemetryEnabled bool`、`TelemetryID string`、`TelemetryLastAt int64` 三个字段及 JSON tag
- [x] 1.2 `DefaultSettings()` 中 `TelemetryEnabled: true`
- [x] 1.3 确认 `store.go` 的 `readJSON` 缺字段默认值机制对新增字段生效（旧 `settings.json` 读出为 `TelemetryEnabled=true`）

## 2. internal/telemetry 包

- [x] 2.1 新建 `internal/telemetry/telemetry.go`，定义包级变量 `pingURL`（国内地址，变量便于测试注入）
- [x] 2.2 `NewService(store *store.Store, appVersion string) *Service`
- [x] 2.3 `Start(ctx context.Context)`：**无条件**确保 `TelemetryID` 存在（无则生成 UUID v4，经营 store「读→改→整体写回」落盘，不得直接写 `settings.json`）；启动每小时 `time.Ticker`
- [x] 2.4 tick 逻辑：`!TelemetryEnabled` → skip；`version == "dev"` → skip；`now - TelemetryLastAt <= 24h` → skip；否则发起上报
- [x] 2.5 上报：`POST pingURL`，body 为 `{id, v, os, arch, accounts}`，`accounts = len(store.ListCredentials())`；HTTP 200 才写 `TelemetryLastAt = now`
- [x] 2.6 全链路静默：超时（短超时如 10s）、错误均只 `log.Printf` 而不弹窗、不返回错误给 UI
- [x] 2.7 日志：生成 ID 打前 8 位、上报成功打字段、上报失败打错误、节流与关闭不打日志；**完整 `device_id` 绝不写日志**
- [x] 2.8 `Stop()`：停止 ticker，幂等、可重复调用，允许 `Start` 未调用时调用
- [x] 2.9 UUID 生成不引入第三方依赖（`crypto/rand` 手写 v4，约 10 行）

## 3. 应用装配

- [x] 3.1 `app.go` 的 `App` 结构体新增 `telemetry *telemetry.Service` 字段
- [x] 3.2 `NewApp()` 构造该服务（传入 `st` 与 `version`）
- [x] 3.3 `startup(ctx)` 中调用 `a.telemetry.Start(ctx)`
- [x] 3.4 `shutdown(ctx)` 中调用 `a.telemetry.Stop()`（放在 scheduler / proxy 之后）
- [x] 3.5 `app_settings.go` 确认 `GetSettings` / `SaveSettings` 对新增字段零改动即可透传；如有显式字段拷贝需补齐

## 4. 前端设置页

- [x] 4.1 设置页新增「帮助改进 WorkBuddy」区块，含开关与说明文案（仅上传版本号 / 操作系统 / 账号数量，不含账号或个人信息，可随时关闭）
- [x] 4.2 开关读写走现有设置读写绑定方法，保存后立即生效
- [x] 4.3 `cd frontend && npm run build` 通过

## 5. 文档

- [x] 5.1 改写 `README.md` 第 61 行「所有数据仅保存在本机，不上传任何第三方」
- [x] 5.2 改写 `README.md` 第 64 行「网络仅访问 `copilot.tencent.com` 与 GitHub」— 补充上报地址
- [x] 5.3 README 说明上报内容、默认开启、可在设置页关闭
- [x] 5.4 服务端契约（`POST /ping` + 请求体字段 + 200 无响应体）记录在 README 或 `docs/DESIGN.md`，供独立服务端项目对照

## 6. 验证

- [x] 6.1 测试：距上次上报 < 24h 时不发请求
- [x] 6.2 测试：距上次上报 > 24h 且开关开启时发出一次请求，请求体字段与值正确
- [x] 6.3 测试：请求体不含令牌、邮箱、账号 UID、凭证 ID 等敏感字段
- [x] 6.4 测试：上报失败（超时 / 500）不写 `TelemetryLastAt`，下次 tick 仍会尝试
- [x] 6.5 测试：开关关闭时不发请求；关闭再开启后 `TelemetryID` 不变
- [x] 6.6 测试：`dev` 版本不发起请求
- [x] 6.7 测试：旧 `settings.json` 缺字段时读出 `TelemetryEnabled == true`
- [x] 6.8 测试：`Start` 后 `settings.json` 中必存在 `TelemetryID`（与开关状态无关），且为合法 v4 UUID
- [x] 6.9 测试：`TelemetryID` 的落盘不覆盖 `settings.json` 中同期修改的其他字段
- [x] 6.10 测试：日志中不出现完整 `device_id`
- [x] 6.11 `go vet ./... && go test ./...` 通过
- [x] 6.12 `cd frontend && npm run build` 通过
- [x] 6.13 `GOOS=darwin GOARCH=arm64 go build ./...` 自检通过
