## Context

工具通过 `//go:build windows` / `!windows` 分文件实现系统集成。macOS 当前落到 `*_stub.go`，编译可通过但功能全空；同时现有"关闭窗口隐藏到托盘"语义在无托盘的 macOS 上会让用户关不掉窗口。

Wails v2.13 的 darwin 实现（已查证源码）带来两个关键事实：

- `AppDelegate.m`：`applicationShouldTerminateAfterLastWindowClosed` 返回 `NO`，关窗不自动退出；`windowShouldClose` 在 `HideWindowOnClose=true` 时 `[NSApp hide]`，`false` 时发 `"Q"` 消息。
- `dispatcher`：`"Q"` → `sender.Quit()` → darwin `Frontend.Quit()` → 若 `OnBeforeClose` 存在，其返回 `true` 会阻止退出。
- `AppDelegate.m` 未实现 `applicationShouldHandleReopen`，点 Dock 图标不会自动恢复窗口。
- `SingleInstanceLock` 在 darwin 用 `NSDistributedNotificationCenter`，第二次启动能触发 `OnSecondInstanceLaunch`。

结论：macOS 的"关窗即退出"必须同时改 Go 侧 `beforeClose` 与 C 侧 `HideWindowOnClose`，只改一处无效。

## Goals / Non-Goals

**Goals:**

- macOS（amd64 / arm64）可构建、可运行、可分发的完整支持。
- 无需托盘即可闭环：关窗即退出、自启静默、双击唤起。
- 自启（LaunchAgent）、通知（osascript）、自动更新（dmg）在 macOS 可用。
- Windows 行为零回归。
- 不引入新的第三方 Go 依赖。

**Non-Goals:**

- macOS 托盘 / 菜单栏图标。
- 代码签名与公证（Apple Developer 账号）。
- macOS 上"原地替换 `.app`"式自更新。
- macOS 睡眠唤醒的即时补签钩子。
- universal 二进制（改为分别发布两个 dmg）。

## Decisions

### D1. macOS 无托盘，关窗即退出

`beforeClose` 按平台分支：Windows 保持"隐藏到托盘并阻止关闭"；macOS 置 `quitting=true` 并返回 `false`。同时 `main.go` 的 `HideWindowOnClose` 在 macOS 设为 `false`（C 层会发 `"Q"` → 走 `beforeClose` 放行退出）。

- 备选：Dock 菜单（`NSApplicationActivationPolicyAccessory`）——macOS 原生但需要额外 cgo，且与"关窗退出"语义重叠，放弃。
- 备选：`fyne.io/systray`——业界标准但需引入依赖且争抢主线程，明确排除。

### D2. 自启用 LaunchAgent

新增 `autostart_darwin.go`：写 `~/Library/LaunchAgents/com.hosea3000.workbuddy-checkin.plist`，`ProgramArguments` 为 `[/path/to/app, --hidden]`，用 `launchctl bootstrap gui/$UID <plist>` 注册、`launchctl bootout gui/$UID` 注销。`autoStartEnabled()` 以 plist 是否存在为准。

- 备选：`osascript` 操作系统登录项——更脆、不易幂等，放弃。

### D3. 静默启动后靠单实例锁唤起

自启以 `--hidden` 启动（`StartHidden=true`）。用户再次双击 `.app` → 第二个实例命中 `SingleInstanceLock` → `OnSecondInstanceLaunch` → `showWindow()`（`WindowShow` + `WindowUnminimise`）。

- 风险：darwin 无 `applicationShouldHandleReopen`，若 `StartHidden` 下窗口根本未创建，`WindowShow` 可能无效。设为 Open Question，实现前实测；备选是用 `HideApplication`/`ShowApplication` 或用 `alwaysOnTop`+`makeKeyAndOrderFront` 的等价调用。

### D4. 通知用 osascript

新增 `notification_darwin.go`，`Notify` 执行 `osascript -e 'display notification "body" with title "title"'`。零依赖、无授权门槛。托盘专属文案（"已最小化到托盘"）在 macOS 不触发。

### D5. 自动更新 = 下载 dmg + open 挂载

- `updater.go`：资产名按 `GOOS`（+`GOARCH`）选择——Windows `workbuddy-checkin.exe`；macOS `WorkBuddy-checkin-<arch>.dmg`。
- 新增 `updater_apply_darwin.go`：下载 dmg 到 `~/Downloads/`（而非 `.app` 同目录，规避 `Contents/MacOS` 不可写与签名问题），`open <dmg>` 后 `quit()`。
- `app_update.go`：放开 darwin 的 `"不支持自动更新"` 短路；`PendingUpdateInfo` 等 `.new` 机制仅 Windows 生效。
- 复用现有 `downloadUpdate` 与进度回调，`exeUpdatePaths` 在 darwin 重定向到 `~/Downloads/`。

- 备选：原地替换 `.app`——涉及 bundle 结构、`/Applications` 权限、签名失效，成本高且脆弱，放弃。

### D6. 分两个 dmg，两条 CI job

CI 增加 `macos-latest` 矩阵（amd64/arm64）：`wails build -platform darwin/<arch>` 产出 `.app`，`hdiutil create` 打包为 `WorkBuddy-checkin-<arch>.dmg`，上传为 Release 资产。Windows job 不变。不签名、不公证。

### D7. stub build tag 收紧

`autostart_stub.go`、`notification_stub.go`、`updater_apply_stub.go` 的 tag 从 `!windows` 收紧为 `!windows && !darwin`，避免与新增 darwin 实现重复定义。`tray_stub.go` 保持 `!windows` 不变（macOS 复用空托盘）。

## Risks / Trade-offs

- [macOS 静默启动后无法唤起窗口] → 实现前用真机实测；失败则改用 `HideApplication`/`ShowApplication` 或在 `OnSecondInstanceLaunch` 中调用更底层的显示路径。
- [dmg 资产名与 `updateAssetName` 不一致导致更新静默失效] → 用单元测试锁死三种 `GOOS`+`GOARCH` 组合的资产名。
- [`~/Library/Application Support` 与 launchd 工作目录差异] → 数据路径基于 `os.UserConfigDir()` 绝对路径，不依赖 cwd；LaunchAgent plist 用绝对路径。
- [未签名 app 首次打开被 Gatekeeper 拦截] → README 说明 `xattr -cr`，并在 Release 说明中提示。
- [macOS 无唤醒钩子，补签最迟延迟 1 小时] → 接受，每小时巡检兜底；必要时后续在 `showWindow` 顺带 `RunAll`。
- [两个 dmg 让用户选择困惑] → README 说明如何查芯片（关于本机）。

## Migration Plan

1. 先跑通 `wails build -platform darwin/amd64`（基础构建验证）。
2. 改关窗语义 + `open` 目录 + osascript 通知（最小可用集）。
3. 真机实测 D3 唤起（Open Question），据结果定实现。
4. LaunchAgent 自启。
5. dmg 自动更新。
6. CI 矩阵 + 文档。

回滚：各 darwin 文件与分支独立，撤销对应文件与分支即可；Windows 路径不受影响。

## Open Questions

- `StartHidden=true` 下 `OnSecondInstanceLaunch` → `WindowShow` 能否真正唤出窗口？（需真机实测，影响 D3）
- macOS Runner 是否为 arm64，交叉编译 amd64 时 cgo/图标资源是否都正常？（影响 D6）
