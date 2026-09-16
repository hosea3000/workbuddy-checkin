## Context

`logging.go` 的 `setupLogging(dir)` 以 `O_CREATE|O_WRONLY|O_APPEND` 打开 `dir/app.log` 并交给 `log.SetOutput`，此后不做任何大小/年龄管理。43 处 `log.Printf` 调用点（含 `client.SetLogger` 注入的每请求诊断日志）全部汇入这一个文件。

本应用是常驻后台工具（Windows 托盘 / macOS 菜单栏），单次运行可达数周不重启。因此「启动时截断」无法覆盖主要场景，必须运行时轮转。

`logging.go` 现有结构：

```
setupLogging(dir)
 ├─ log.SetFlags(log.LstdFlags)
 ├─ f, err := os.OpenFile(dir/app.log, O_CREATE|O_WRONLY|O_APPEND, 0600)
 └─ if stderrWorks() { log.SetOutput(io.MultiWriter(f, os.Stderr)) }
    else            { log.SetOutput(f) }
```

`stderrWorks()` 的存在原因：Wails 的 windowsgui 子系统无控制台，`os.Stderr` 是无效句柄；`io.MultiWriter` 遇首个错误即短路，会把文件写入也带死。故文件是主输出，stderr 仅在可写时附加。这一约束与轮转正交，必须保留。

## Goals / Non-Goals

**Goals:**

- `app.log` 的磁盘占用有上界，且在任何单次运行周期内都收敛（不依赖重启）。
- 保留可回溯的历史日志（供用户报 bug 时查看），而非直接丢弃。
- 改动局限在 `logging.go` 与依赖声明，不触碰任何日志调用点。

**Non-Goals:**

- 不改日志格式、级别、`log.LstdFlags` 标志。
- 不引入分级日志（debug/info/warn）或结构化日志。
- 不新增 UI 上的日志查看/导出入口。
- 不改令牌脱敏策略（现有「日志不含令牌」约束继续由调用点保证）。

## Decisions

### 决策 1：采用 `gopkg.in/natefinch/lumberjack.v2` 做轮转

**理由**：

- 它是 Go 生态事实标准，**纯标准库实现、无传递依赖**，是依赖成本最低的一类。
- 内部已处理 Windows 平台陷阱：文件被打开时 `rename` 会失败，必须先 `close` 再 rename 再 reopen；手写需自行处理该竞态，且要与 `io.MultiWriter` 的 stderr 分支协调。
- 原生支持本次需要的全部旋钮：`MaxSize` / `MaxBackups` / `MaxAge` / `Compress`。

**备选方案**：

- **手写 `rotatingWriter`（约 25 行）**：可行性成立，但需在 Windows 上正确实现 close→rename→reopen，且失去按天保留与压缩。当仓库禁止新增依赖时这是唯一解；该约束现已移除（见决策 3），故不取。
- **仅启动时截断/改名**：约 5 行、无需加锁，但单次超长运行仍会突破上限，恰好漏掉本应用的主要场景。不取。

### 决策 2：lumberjack 只包住文件写入，不包 `io.MultiWriter`

```
log.SetOutput(io.MultiWriter(&lumberjack.Logger{Filename: dir/app.log, ...}, os.Stderr))
                      ▲ 轮转发生在这里，仅统计文件字节
```

若反序（用 lumberjack 包 MultiWriter）则类型不成立；若把 stderr 也纳入同一个 logger 则会把 stderr 当旋转目标。因此 lumberjack 位于 `MultiWriter` 的**第一个参数位**（文件侧），`stderrWorks()` 分支结构不变：

```
if stderrWorks() { log.SetOutput(io.MultiWriter(lj, os.Stderr)) }
else             { log.SetOutput(lj) }
```

### 决策 3：移除 AGENTS.md 的第三方依赖禁令，但保留 systray 教训

原文「不引入任何新的第三方 Go 依赖」的语境是 macOS 系统集成，其括号内的 `energye/systray` 教训（抢占 `NSApplication.delegate`，破坏 Wails 单实例锁与 ⌘Q 链路）是实打实的踩坑记录。因此：

- 删除该禁令本身（本次放开）。
- 将 `energye/systray` 的冲突原因迁入 `docs/DESIGN.md` 决策表，使知识不丢失。

### 决策 4：轮转参数

| 旋钮 | 值 | 含义 |
|------|-----|------|
| `MaxSize` | 5 | MB，当前文件到 5 MB 即切 |
| `MaxBackups` | 3 | 最多保留 3 个历史文件 |
| `MaxAge` | 30 | 历史文件最长 30 天 |
| `Compress` | true | 历史文件 gzip |

最坏磁盘占用约 5 MB（当前）+ 3 × 5 MB（历史，压缩后更小）。

## Risks / Trade-offs

- **[历史文件命名引入新文件，credential-store spec 的文件清单过时]** → 本 change 同步 MODIFIED `credential-store` 的「JSON 文件存储位置」需求，把 `app-<时间戳>.log[.gz]` 纳入清单。
- **[`Compress: true` 使历史日志需先解压才能查看]** → 用户报 bug 场景下可接受；不提供 UI 解压入口（Non-Goal）。
- **[lumberjack 内部加锁与调用点并发]** → lumberjack 自身并发安全，`log` 包也串行化 `Output`，无需额外锁。
- **[`setupLogging` 在 `store.New` 之后调用，若目录不可写则静默降级]** → 现有行为：`OpenFile` 失败即 return，保持「不阻断启动」语义；lumberjack 惰性建文件，行为等价，沿用。

## Migration Plan

1. `go get gopkg.in/natefinch/lumberjack.v2`，`go mod tidy`。
2. 改 `logging.go`：构造 `*lumberjack.Logger`，替换 `*os.File`。
3. 改 `AGENTS.md`（删禁令）与 `docs/DESIGN.md`（补 systray 记录）。
4. `go vet ./... && go test ./...`；`cd frontend && npm run build`。
5. 回滚：还原 `logging.go` 与 `go.mod`/`go.sum` 即可，无数据迁移。

## Open Questions

- 无。（轮转参数已定；spec 归属已定为新开 `logging` capability。）
