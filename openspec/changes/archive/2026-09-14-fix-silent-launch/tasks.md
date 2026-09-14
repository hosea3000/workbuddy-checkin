## 1. 启动标记

- [x] 1.1 新增 `autostart.go`（无构建标签）：`autoStartHiddenFlag`、`buildAutoStartCommand`、`hiddenFromArgs`
- [x] 1.2 新增 `autostart_test.go`：`hiddenFromArgs` 与 `buildAutoStartCommand` 用例
- [x] 1.3 `main.go`：`StartHidden: hiddenFromArgs(os.Args[1:])`

## 2. 自启命令与启动逻辑

- [x] 2.1 `autostart_windows.go`：`setAutoStart` 用 `buildAutoStartCommand(exe)` 写入 Run 命令
- [x] 2.2 删除 `isAutoStartEnabled`（`autostart_windows.go` + `autostart_stub.go`）
- [x] 2.3 `app.go`：删除 `startup` 里 `if !isAutoStartEnabled() { showWindow() }` 块

## 3. 验证

- [x] 3.1 `go vet ./... && go test ./...`（含新增 `autostart_test.go`）通过
- [x] 3.2 `GOOS=windows GOARCH=amd64 go build ./...` 通过
- [x] 3.3 Windows 人工验证：手动双击显示窗口；带 `--hidden` 启动静默进托盘
