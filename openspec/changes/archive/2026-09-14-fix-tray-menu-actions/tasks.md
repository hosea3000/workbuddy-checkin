## 1. 修复托盘菜单命令

- [x] 1.1 `showMenu()` 改为接收 `TrackPopupMenu` 返回值并 `switch` 分发 `cmdOpen` / `cmdCheckinAll` / `cmdQuit`（对齐 health-tool）
- [x] 1.2 删除永不触发的 `wmCommand` 分支（`tray_windows.go:266-281`）
- [x] 1.3 菜单构建一次并复用（移到 `create()`，不再每次弹菜单重建/销毁）

## 2. 托盘图标可用性恢复

- [x] 2.1 注册 `TaskbarCreated` 窗口消息，收到后重新 `addIcon()`
- [x] 2.2 `WM_POWERBROADCAST`（睡眠恢复）分支在 `onWake()` 之外补 `addIcon()`
- [x] 2.3 `WM_WTSSESSION_CHANGE`（会话解锁）分支在 `onWake()` 之外补 `addIcon()`

## 3. 修复托盘「退出」无法结束进程

- [x] 3.1 `App` 增加 `quitting atomic.Bool`
- [x] 3.2 `quit()` 置位 `quitting` 后再 `wruntime.Quit`
- [x] 3.3 `beforeClose()` 首行 `if a.quitting.Load() { return false }` 放行主动退出

## 4. 验证

- [x] 4.1 `GOOS=windows GOARCH=amd64 go build ./...` 通过
- [x] 4.2 `go vet ./... && go test ./...`（Linux stub）通过
- [x] 4.3 Windows 人工验证：托盘菜单「打开主界面 / 立即签到（全部）/ 退出」均生效，且「退出」后进程消失
- [x] 4.4 Windows 人工验证：explorer 重启、锁屏解锁后托盘图标仍可用
