## 1. 平台分支与资源

- [x] 1.1 将 `tray_stub.go` 的 build tag 从 `//go:build !windows` 收窄为 `//go:build !windows && !darwin`
- [x] 1.2 在 `build/darwin/Info.plist` 增加 `LSUIElement` = `<true/>`
- [x] 1.3 确认 `build/darwin/trayTemplate@1x.png`（16×16）与 `@2x.png`（32×32）模板图已入库

## 2. macOS 菜单栏实现（cgo + Objective-C）

- [x] 2.1 新建 `tray_darwin.go`（`//go:build darwin`）：定义与 `tray_windows.go` 同签名的 `Tray`、`newTray`、`Start`、`SetTip`
- [x] 2.2 用 `//go:embed build/darwin/trayTemplate@2x.png` 内嵌模板图，经 cgo 传为模板 `NSImage`（`setSize(16,16)`、`image.template = YES`）
- [x] 2.3 新建 `tray_darwin.m`（`//go:build darwin`）：创建 `NSStatusItem` + `NSMenu`（「打开」「退出」），全部 Cocoa 操作用 `dispatch_async` 投递主队列
- [x] 2.4 给 `NSStatusItem.button` 设 `target`/`action`，左键单击回调里 `popUpStatusItemMenu:` 弹菜单
- [x] 2.5 导出 `//export` 的 Go 回调 `woTrayOpen` / `woTrayQuit`，桥接到 `Tray.onOpen` / `Tray.onQuit`；实现 `SetTip` 更新 tooltip
- [x] 2.6 暴露 `setActivationPolicyAccessory()`（cgo），供 `startup` 兜底隐藏 Dock 图标
- [ ] 2.7 `GOOS=darwin GOARCH=arm64 go build ./...` 与 `GOOS=darwin GOARCH=amd64 go build ./...` 通过（cgo 无法从 Linux 交叉编译，须 macOS 上验证；见 4.2）

## 3. 关窗语义与退出

- [x] 3.1 改写 `app.go:beforeClose`：保留 `quitting` 放行，删除 `if !isWindows()` 死代码分支，统一 `notifyCloseTipOnce()` 后返回 `true`
- [x] 3.2 macOS 隐藏窗口调用 `wruntime.WindowHide(ctx)`（`orderOut`，只藏窗口）；确认 `main.go` 保持 `HideWindowOnClose: isWindows()`
- [x] 3.3 `app.go:startup` 在 macOS 上调用 `setActivationPolicyAccessory()`
- [x] 3.4 改写 `notifyCloseTipOnce` 为两平台通用文案（「应用仍在后台运行」），移除其中的 `isWindows()` 前置判断

## 4. 校验与文档

- [x] 4.1 `go vet ./... && go test ./...`（Linux，stub 分支）通过
- [ ] 4.2 macOS 真机验证：关窗不退出、菜单栏图标存在、左键弹菜单、「打开」显窗口并获得焦点、「退出」进程结束
- [ ] 4.3 macOS 真机验证：Dock 无图标、深浅色外观下图标可辨识
- [ ] 4.4 macOS 真机验证：⌘Q 行为符合预期（若被拦截不符合预期，改为直接退出）并复核 `Accessory` 模式下的窗口焦点
- [x] 4.5 同步文档：`docs/PRD.md`（F8 关窗语义与菜单栏、§6 移除「macOS 托盘不做」、§8 验收清单）、`docs/DESIGN.md`（平台能力矩阵、`macOS 只用系统命令` 约束的例外说明）、`AGENTS.md`（硬约束更新）
