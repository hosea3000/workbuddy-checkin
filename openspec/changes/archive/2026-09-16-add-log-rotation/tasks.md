## 1. 依赖接入

- [x] 1.1 `go get gopkg.in/natefinch/lumberjack.v2` 并 `go mod tidy`，确认无传递依赖
- [x] 1.2 确认 `go.mod` 中该依赖为直接 require（非 indirect）

## 2. 改造 logging.go

- [x] 2.1 在 `setupLogging` 中用 `&lumberjack.Logger{Filename: filepath.Join(dir,"app.log"), MaxSize: 5, MaxBackups: 3, MaxAge: 30, Compress: true}` 替换 `os.OpenFile` 返回的 `*os.File`
- [x] 2.2 保持 `stderrWorks()` 分支：可写时 `log.SetOutput(io.MultiWriter(lj, os.Stderr))`，否则 `log.SetOutput(lj)`；确认 lumberjack 位于 MultiWriter 的文件侧
- [x] 2.3 保留 `log.SetFlags(log.LstdFlags)`，不改日志格式与任何调用点
- [x] 2.4 确认构造失败（如目录不可写）时不阻断启动（惰性建文件，行为与原 `OpenFile` 失败即 return 等价）

## 3. 文档更新

- [x] 3.1 删除 `AGENTS.md` 中「不引入任何新的第三方 Go 依赖」的约束
- [x] 3.2 将 `energye/systray` 抢占 `NSApplication.delegate`（破坏 Wails 单实例锁与 ⌘Q）的教训迁入 `docs/DESIGN.md` 决策记录
- [x] 3.3 确认 `docs/DESIGN.md` 系统集成行不再宣称「无新依赖」导致与本次改动矛盾

## 4. 验证

- [x] 4.1 新增测试：以极小 `MaxSize`（如 1 KB）构造 logger，写入超过上限的日志，断言产生压缩历史文件、当前文件重置、不丢日志
- [x] 4.2 新增测试：断言历史文件数量不超过 `MaxBackups`
- [x] 4.3 `go vet ./... && go test ./...` 通过
- [x] 4.4 `cd frontend && npm run build` 通过
- [x] 4.5 `GOOS=darwin GOARCH=arm64 go build ./...` 自检通过
