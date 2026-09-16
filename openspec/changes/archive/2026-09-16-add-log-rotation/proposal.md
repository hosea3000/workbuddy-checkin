## Why

`app.log` 以 `O_APPEND` 打开后只追加、不轮转、不回收，长时间后台运行会无上限增长（数周~数月可达数十 MB）。本应用是常驻托盘/菜单栏的后台工具，单次运行可达数周不重启，启动时截断解决不了这个问题。

## What Changes

- 为 `app.log` 引入**实时大小轮转**：单文件超过 5 MB 即切分。
- 保留最多 **3** 个历史文件、最长 **30** 天，历史文件以 **gzip** 压缩。
- 接入方式为在 `logging.go` 的 `setupLogging` 中用 `gopkg.in/natefinch/lumberjack.v2` 作为 `log` 的输出目标，替换当前的裸 `*os.File`。
- 保留现有 `stderrWorks()` 分支语义：`wails dev`（有控制台）时日志同时写 stderr，windowsgui 构建下仅写文件。
- **BREAKING（文档约束）**：移除 `AGENTS.md` 中「不引入任何新的第三方 Go 依赖」的禁令；`energye/systray` 的 macOS `NSApplication.delegate` 冲突教训迁移到 `docs/DESIGN.md` 的决策记录中保留。

## Capabilities

### New Capabilities
- `logging`: 应用运行日志的落盘位置、格式、大小/天数上限与轮转行为，以及「日志不含令牌」的约束。

### Modified Capabilities
- `credential-store`: 现有 spec 将 `app.log` 列为数据目录存储内容；本次轮转会在同目录产生 `app-<时间戳>.log[.gz]` 历史文件，该文件清单描述需要同步。

## Impact

- **代码**：`logging.go`（主要改动）、`app.go`（`setupLogging` 调用点无签名变化）、`go.mod`/`go.sum`（新增依赖）。
- **依赖**：新增 `gopkg.in/natefinch/lumberjack.v2`（纯标准库实现，无传递依赖）。
- **文档**：`AGENTS.md`（删约束）、`docs/DESIGN.md`（保留 systray 教训）。
- **平台**：Windows / macOS 行为一致；Linux 下 `go test ./...` 不受影响（`logging.go` 无平台分支）。
- **不涉及**：令牌脱敏策略、日志格式、UI、上游协议。
