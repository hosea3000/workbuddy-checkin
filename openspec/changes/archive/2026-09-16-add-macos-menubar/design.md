## Context

现状：`tray_stub.go`（`//go:build !windows`）为空实现，macOS 复用 stub；`main.go` 设 `HideWindowOnClose: isWindows()`；`app.go:beforeClose` 在非 Windows 直接放行退出。因此 macOS 关窗即退出，调度器随之停止。

已核实的事实（读 Wails v2.13.0 源码，非猜测）：

| 事实 | 位置 |
|---|---|
| 红叉（`hideOnClose=false`）→ `processMessage("Q")` → `Frontend.Quit()` → **经过 `OnBeforeClose`** | `WindowDelegate.m:16`、`dispatcher.go:69` |
| ⌘Q / Dock 退出 → `applicationShouldTerminate` → `"Q"` → **同样经过 `OnBeforeClose`** | `AppDelegate.m:38` |
| `wruntime.WindowHide` → `[mainWindow orderOut]`，**只藏窗口不藏 App** | `WailsContext.m:419` |
| `wruntime.Quit` → `Quit()` → **经过 `OnBeforeClose`** | `frontend.go:362` |
| Wails 在 `applicationWillFinishLaunching` 硬写 `setActivationPolicy:Regular`，会盖掉 `LSUIElement` | `AppDelegate.m:44` |
| Wails v2.13.0 **无内置 systray**，`mac.Options.ActivationPolicy` 是注释掉的 | `pkg/options/mac/mac.go` |
| `energye/systray` 的 `nativeStart()` 会 `[NSApplication setDelegate:owner]`，抢占 Wails 的 AppDelegate | `systray_darwin.m:281` |

约束：macOS 不引入新的第三方 Go 依赖；只影响 darwin；Linux 下 `go test ./...` 必须继续可跑。

## Goals / Non-Goals

**Goals:**
- macOS 菜单栏常驻图标，左键单击弹出「打开 / 退出」菜单。
- macOS 关闭主窗口 = 隐藏窗口，进程与调度器继续运行。
- macOS 无 Dock 图标，形态为纯菜单栏常驻应用。
- 与 `tray_windows.go` 保持同一套 `Tray` 接口，`app.go` 无平台分支。
- 不引入任何新的第三方 Go 依赖。

**Non-Goals:**
- 不为 macOS 增加电源/会话唤醒钩子（`onWake` 在 macOS 不触发，仍靠每小时巡检兜底）。
- 不做 macOS 代码签名/公证、universal 包。
- 不改 Windows 任何行为。
- 不做菜单栏图标的动态 Dock 图标切换（打开窗口时不临时恢复 Dock 图标）。

## Decisions

### D1：手写 cgo + Objective-C，而非引入 `energye/systray`

`energye/systray` 的 macOS 实现会在 `nativeStart()` 里 `[NSApplication setDelegate:owner]`，顶掉 Wails 的 `AppDelegate`——而后者承担单实例锁、⌘Q 终止链路与 open-file/URL 回调。两种用法（`Run` 自持 App / `Register`+`nativeStart`）都会抢占 delegate，只是时机不同。

手写 cgo 只创建一个 `NSStatusItem` + `NSMenu`，**完全不碰 `NSApplication.delegate`**，与 Wails 零冲突，且不需要 energye 那套「自己拥有 App」的完整逻辑（那正是它抢 delegate 的原因）。

代价：约 150 行 Objective-C，需自行处理主线程调度。可接受，且与本仓库手写 Win32 托盘的风格一致。

### D2：窗口隐藏用 `wruntime.WindowHide`（`orderOut`），不用 `HideWindowOnClose`

`HideWindowOnClose: true` 在 macOS 的分支是 `[NSApp hide:nil]`——隐藏**整个应用**（等同 ⌘H），语义错误：会顶掉其他 App 的菜单栏，且红叉路径上完全绕过 `OnBeforeClose`。

因此 macOS 保持 `HideWindowOnClose: false`，红叉仍走 `"Q"` → `OnBeforeClose`，由 `beforeClose` 拦截并调用 `wruntime.WindowHide`（`orderOut`，只藏窗口）。

### D3：拦截统一走 `beforeClose` 返回 `true`

经核实，macOS 的红叉与 ⌘Q 最终都汇入 `Frontend.Quit()` → `OnBeforeClose`。因此 `beforeClose` 可两平台统一：

```
if a.quitting.Load() { return false }  // 真正退出时放行
a.notifyCloseTipOnce()
return true                            // 隐藏窗口（Windows 由 HideWindowOnClose 生效，macOS 显式调 WindowHide）
```

删除现有 `if !isWindows() { quitting=true; return false }` 死代码（macOS 上它从未被触发过——因为红叉走 `"Q"` 也回到同一函数，但原先的 `false` 让关窗变成退出）。

### D4：Dock 图标 = `LSUIElement` + 运行时兜底 `Accessory`

`build/darwin/Info.plist` 是 Go template 原样写盘（`packager.go:117`），加 `LSUIElement=true` 会被保留；但 Wails 在 `applicationWillFinishLaunching` 立即改回 `Regular`，存在时序竞争。

故两处都做：Info.plist 设 `LSUIElement`（启动瞬间生效），`app.startup` 里再调一次 cgo 暴露的 `setActivationPolicyAccessory()`（兜底覆盖 Wails 的写死值）。

### D5：菜单栏图标用模板图，`//go:embed` 内嵌

菜单栏需适配深浅色，用模板图（纯黑 + alpha）。已落地 `build/darwin/trayTemplate@1x.png`（16×16）与 `@2x.png`（32×32），由用户提供的彩色透明原图自动转制。

选择「提交缩放后的小图 + embed」而非「提交原图 + 运行时缩放」：零依赖、零运行时开销、结果确定（Go 标准库无高质量缩放，引入 `x/image` 会新增依赖）。两文件共约 1.2 KB。

### D6：主线程调度

`NSStatusItem` 必须在主线程创建/更新。Wails 的 `OnStartup`（Go 侧 `startup`）不在主线程，故 cgo 侧把所有 Cocoa 操作经 `dispatch_async(dispatch_get_main_queue(), ^{...})` 投递到主队列。`SetTip` 同理（可能从任意 goroutine 调用）。

### D7：左键单击弹菜单，不做双击打开

macOS 菜单栏惯例是左键单击弹出菜单。实现方式：给 `NSStatusItem.button` 设 `target`/`action`，回调里 `[statusItem popUpStatusItemMenu:menu]`——不依赖任何 delegate，完全自包含。Windows 保持双击打开托盘图标不变。

## Risks / Trade-offs

- [Accessory 模式下窗口可能抢不到焦点] → `showWindow` 已有 `wruntime.WindowShow`（内部 `makeKeyAndOrderFront` + `activateIgnoringOtherApps`）；**须 macOS 真机验证**。不足则退化为打开窗口时临时切 `Regular`（Dock 图标会闪现，体验较差，仅作后备）。
- [`LSUIElement` 与 Wails 的 `setActivationPolicy:Regular` 时序竞争] → D4 双写兜底；真机验证 Dock 无图标。
- [cgo 在跨平台构建中的处理] → 已实测：`.m` 带 `//go:build darwin` 时在 Linux 上被完全忽略，`go build/vet ./...` 与 `GOOS=darwin go build` 均正常。`tray_stub.go` 收窄为 `!windows && !darwin`，避免与 darwin 实现重复定义。
- [⌘Q 的退出语义变化] → 现在 macOS 的 ⌘Q 会先经 `beforeClose`；因 `quitting` 未置位会被拦截并隐藏窗口，用户可能觉得「⌘Q 没反应」。**须真机验证 ⌘Q 行为**，必要时改为「⌘Q 视为退出意图，直接置 `quitting`」。
- [菜单栏模板图在小尺寸下的可辨识度] → 已由用户确认预览效果；若真机观感不佳，替换 embed 的 PNG 字节即可，逻辑不动。
- [进程崩溃/强杀时残留菜单栏图标] → 系统在进程结束时自动回收 `NSStatusItem`，无需处理。

## Migration Plan

无数据迁移；纯行为变更。旧版本 macOS 用户升级后首次关窗会由「退出」变为「隐藏到菜单栏」，配一次性的行为提示（见 `settings` delta）。

回滚：还原 `tray_stub.go` 的 build tag、`app.go` 的 `beforeClose`、移除 `Info.plist` 的 `LSUIElement` 与 `tray_darwin.*` 即可。

## Open Questions

- macOS 上 `Accessory` 模式是否影响 Wails 窗口首次显示与焦点获取——**必须 macOS 真机验证**（见 Risks）。
- ⌘Q 是否应直接退出而不拦截——**须真机确定预期交互**后定稿。
