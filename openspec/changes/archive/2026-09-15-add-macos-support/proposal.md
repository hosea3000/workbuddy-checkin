## Why

工具目前只能构建/分发 Windows 产物，macOS 用户无法使用。代码虽已通过 `*_stub.go` 保证 Linux 可编译，但 macOS 会落到空实现：无自启、无通知、无自动更新，且窗口关闭逻辑（隐藏到托盘）在无托盘的 macOS 上会让用户关不掉窗口。需要让 macOS 成为一等支持平台。

## What Changes

- **macOS 不做托盘**：关闭主窗口即退出应用（Windows 保持隐藏到托盘不变）；`beforeClose` 与 `HideWindowOnClose` 按平台分支。
- **开机自启改为 LaunchAgent**：macOS 写 `~/Library/LaunchAgents/*.plist` 并经 `launchctl` 注册；自启命令带 `--hidden` 静默启动。
- **静默启动后唤起**：macOS 靠单实例锁 `OnSecondInstanceLaunch` 唤起已有窗口（用户双击 `.app`）。
- **通知改为 osascript**：macOS 用 `osascript -e 'display notification'`；托盘专属文案（「已最小化到托盘」）在 macOS 不出现。
- **自动更新改为下载 dmg**：macOS 下载对应架构的 `.dmg` 到 `~/Downloads/` 后 `open` 挂载引导安装，不做原地替换；资产名按 `GOOS`+`GOARCH` 选择。
- **补签钩子不移植**：macOS 无电源/会话唤醒钩子，依赖既有每小时巡检兜底。
- **分两个未签名 dmg 发布**：CI 增加 macOS 矩阵 job（amd64 / arm64），打包 `.app` 为 dmg 并作为 Release 资产；不签名、不公证。
- **文档**：README 补充 macOS 选架构与 `xattr -cr` 去隔离说明；PRD/DESIGN 更新平台能力矩阵。

## Capabilities

### New Capabilities
- （无）

### Modified Capabilities
- `tray-integration`：托盘/自启/静默启动/单实例的要求改为平台分支——macOS 无托盘、关窗即退出、自启用 LaunchAgent、静默启动靠单实例锁唤起。
- `settings`：关闭窗口行为提示与数据目录入口按平台调整（macOS 不提示「最小化到托盘」，目录用 `open` 打开）。
- `app-update`：新增 macOS 的 dmg 下载 + 挂载安装路径，资产名按架构选择；Windows 原地替换路径不变。
- `catch-up-checkin`：明确 macOS 无系统唤醒钩子，补签依赖每小时巡检兜底。
- `ci-autorelease`：新增 macOS amd64/arm64 两个 dmg 的构建与发布。

## Impact

- 新增文件：`notification_darwin.go`、`autostart_darwin.go`、`updater_apply_darwin.go`。
- 修改：`app.go`（`beforeClose` 分支）、`main.go`（`HideWindowOnClose`/可见性）、`updater.go`（资产名按 GOOS+GOARCH）、`updater_apply.go`（macOS 路径重定向 `~/Downloads/`）、`app_update.go`（放开 darwin）、`app_settings.go`（`open`）、stub build tag 收紧为 `!windows && !darwin`。
- 构建：`build/darwin/icon.icns`、`wails.json` macOS info、`.github/workflows/release.yml` 新增 macos job。
- 不引入新依赖（osascript / launchctl / hdiutil 均为系统命令），不签名。
- 未知点：`--hidden` 启动后双击 `.app` 经单实例锁能否唤出窗口，需 macOS 实测。
