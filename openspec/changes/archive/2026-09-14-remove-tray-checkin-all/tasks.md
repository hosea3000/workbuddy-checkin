## 1. 移除托盘菜单项

- [x] 1.1 `tray_windows.go`：删除 `cmdCheckinAll`、`onCheckinAll` 字段/参数、菜单项与 `wndProc` 分支
- [x] 1.2 `tray_stub.go`：删除 `onCheckinAll` 字段/参数
- [x] 1.3 `app.go`：`newTray` 调用去掉 `CheckinAll` 回调参数

## 2. 文档

- [x] 2.1 `docs/PRD.md` F8 菜单项改为「打开主界面 / 退出」

## 3. 验证

- [x] 3.1 `go vet ./... && go test ./...` 通过
- [x] 3.2 `GOOS=windows GOARCH=amd64 go build ./...` 通过
