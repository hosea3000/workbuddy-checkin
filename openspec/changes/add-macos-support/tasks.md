## 1. 前置验证（Spike）

- [ ] 1.1 在 macOS 跑通基线构建：`wails build -platform darwin/amd64`（先 `cd frontend && npm run build`），确认可产出 `.app` 并运行
- [ ] 1.2 实测 D3 唤起：以 `--hidden` 启动后再次双击 `.app`，确认 `SingleInstanceLock` → `OnSecondInstanceLaunch` → `showWindow()` 能真正唤出窗口；记录结果，失败则确定备选显示路径
- [ ] 1.3 确认 macOS runner 架构与 `darwin/amd64` 交叉编译（含图标资源）可用

## 2. 平台分支与关窗语义

- [x] 2.1 `platform.go` 增加 `isMac()`（`runtime.GOOS == "darwin"`）
- [x] 2.2 `app.go` 的 `beforeClose` 按平台分支：Windows 保持隐藏到托盘；macOS 置 `quitting=true` 并返回 `false`
- [x] 2.3 `main.go` 的 `HideWindowOnClose` 按平台取值（macOS `false`）
- [ ] 2.4 按 1.2 结果实现/修正 macOS 唤起路径（`OnSecondInstanceLaunch` → `showWindow`）
- [x] 2.5 `app_settings.go` 的 `dirOpenCmd()` 增加 darwin → `open`
- [x] 2.6 用 `isWindows()` 收敛 `app.go:130` 的「已最小化到托盘」提示，macOS 不触发

## 3. macOS 通知

- [x] 3.1 新增 `notification_darwin.go`（`//go:build darwin`），`newNotifier()` 用 `osascript -e 'display notification ...'`
- [x] 3.2 将 `notification_stub.go` build tag 收紧为 `!windows && !darwin`
- [x] 3.3 校验非 Windows/darwin（Linux stub）、Windows、darwin 三平台均无符号冲突

## 4. macOS 开机自启（LaunchAgent）

- [x] 4.1 新增 `autostart_darwin.go`（`//go:build darwin`）：构建/写入 `~/Library/LaunchAgents/com.hosea3000.workbuddy-checkin.plist`（`ProgramArguments=[app, --hidden]`，绝对路径）
- [x] 4.2 `setAutoStart(enable)`：开启用 `launchctl bootstrap gui/$UID <plist>`，关闭用 `launchctl bootout gui/$UID` 并删除 plist，幂等
- [x] 4.3 `autoStartEnabled()` 以 plist 实际存在为准
- [x] 4.4 将 `autostart_stub.go` build tag 收紧为 `!windows && !darwin`
- [x] 4.5 为 plist 内容生成写单元测试（可脱离 macOS 校验路径与参数）

## 5. macOS 自动更新（dmg）

- [x] 5.1 `updater.go` 资产名按 `GOOS`+`GOARCH` 分支：Windows `workbuddy-checkin.exe`；darwin `WorkBuddy-checkin-amd64.dmg` / `WorkBuddy-checkin-arm64.dmg`
- [x] 5.2 为资产名映射写单元测试，锁死三种组合，避免与 CI 命名漂移
- [x] 5.3 `updater_apply.go` 的路径计算在 darwin 重定向到 `~/Downloads/`
- [x] 5.4 新增 `updater_apply_darwin.go`（`//go:build darwin`）：下载 dmg 到 `~/Downloads/` → `open <dmg>` → `quit()`
- [x] 5.5 将 `updater_apply_stub.go` build tag 收紧为 `!windows && !darwin`
- [x] 5.6 `app_update.go` 放开 darwin：`DownloadAndApplyUpdate`/`ApplyUpdateAndRestart` 允许 macOS；`PendingUpdateInfo` 仅 Windows 生效
- [x] 5.7 确认 `downloadUpdate` 与进度回调在 macOS 路径复用无回归

## 6. 构建与图标资源

- [x] 6.1 新增 `build/darwin/icon.icns`（由现有图源生成）
- [x] 6.2 在 `wails.json` 补充 macOS 所需 info（产品名/版本等）

## 7. CI 发布

- [x] 7.1 `.github/workflows/release.yml` 增加 macOS 矩阵 job（`macos-latest`，amd64/arm64）
- [x] 7.2 macOS job：注入版本号、`wails build -platform darwin/<arch>`、用 `hdiutil create` 打包 `.app` 为 `WorkBuddy-checkin-<arch>.dmg`
- [x] 7.3 `gh release create` 上传两个 dmg（与 Windows 资产并列）
- [x] 7.4 确认发布门禁（`go vet` / `go test`）对 macOS job 同样生效

## 8. 文档

- [x] 8.1 README 增加 macOS 章节：构建步骤、按芯片选 dmg、首次打开 `xattr -cr` 去隔离
- [x] 8.2 更新 `docs/PRD.md` 与 `docs/DESIGN.md` 的平台能力矩阵（托盘/自启/通知/更新/补签钩子）
- [x] 8.3 更新根 `AGENTS.md` 的命令说明（macOS 构建、dmg 发布）

## 9. 验证

- [x] 9.1 `go vet ./... && go test ./...`（Linux，含 stub 分支）通过
- [ ] 9.2 Windows 行为回归：关窗隐藏到托盘、托盘菜单、自启、一键更新均不变
- [ ] 9.3 macOS 端到端：关窗退出、自启静默、双击唤起、通知、设置开关自启、dmg 更新流程
