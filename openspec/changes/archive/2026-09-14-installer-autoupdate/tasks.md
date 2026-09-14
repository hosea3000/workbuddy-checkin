## 1. 文档与定位更新

- [x] 1.1 更新 `docs/PRD.md`：形态改为「安装器 + 绿色版双产物」，从「明确不做」中移除「图形化安装器」，补充安装/升级相关风险与验收项
- [x] 1.2 更新 `docs/DESIGN.md`：能力清单加入 `ci-autorelease` 与 `app-update`，补记安装范围与更新机制的耦合

## 2. 更新器移植

- [x] 2.1 移植 `updater.go`（`checkForUpdates` / `compareVersions` / 三态结果），改仓库 owner/name 与 `updateAssetName = "workbuddy-checkin.exe"`
- [x] 2.2 移植 `updater_apply.go`（`.part`/`.new`/`.new.version` 路径、`downloadUpdate`、`buildUpdateBat`、清理函数），改 bat 名
- [x] 2.3 移植 `updater_apply_windows.go` 与 `updater_apply_stub.go`（非 Windows 下 `go test` 可跑）
- [x] 2.4 移植 `model` 中的 `UpdateCheckResult` / `UpdateDownloadEvent` 等类型
- [x] 2.5 补单测：版本比较、检查三态（含 dev 短路与 HTTP 错误）、下载落位与残渣清理、bat 内容生成

## 3. 应用绑定与设置 UI

- [x] 3.1 `App` 增加绑定方法 `GetVersion` / `CheckUpdate` / `DownloadAndApplyUpdate` / `ApplyUpdateAndRestart` / `PendingUpdateInfo`
- [x] 3.2 启动流程按设置项「启动时检查更新」决定是否自动检查（`dev` 版本短路）
- [x] 3.3 设置页展示当前版本号，并提供「检查更新」按钮与结果反馈（已最新 / 发现新版本 / 失败）
- [x] 3.4 前端接入下载进度事件与「重启更新」入口，发现新版本时展示提示
- [x] 3.5 改绑定方法后重新生成前端绑定（`wails generate module`）

## 4. 安装器工程与中文定制

- [x] 4.1 本地执行 `wails build -nsis -installscope user` 生成 `build/windows/installer/` 工程文件
- [x] 4.2 将 `build/windows/installer/project.nsi` 的 `MUI_LANGUAGE` 改为 `SimpChinese` 并提交
- [x] 4.3 在 `.gitignore` 排除 `build/windows/installer/wails_tools.nsh` 与 `build/windows/installer/tmp/`

## 5. CI 发布工作流

- [x] 5.1 `release.yml` 增加 NSIS 安装步骤（`apt-get install -y nsis`）
- [x] 5.2 `release.yml` 在构建前增加 `go vet ./...` 与 `go test ./...` 门禁
- [x] 5.3 构建命令改为 `wails build -clean -nsis -installscope user -platform windows/amd64 -ldflags "-X main.version=<tag>"`
- [x] 5.4 `gh release create` 同时上传 `workbuddy-checkin.exe` 与 `workbuddy-checkin-amd64-installer.exe`

## 6. 验证

- [x] 6.1 Linux 下 `go vet ./... && go test ./...` 通过（走 stub 分支）
- [x] 6.2 Windows 下本地构建安装器：普通用户安装成功、无 UAC、中文界面、可改安装目录、快捷方式与卸载项正常
- [x] 6.3 推送 `v0.2.0` tag，验证 Release 含双资产且产物版本号与 tag 一致
- [x] 6.4 安装版内端到端更新：检查→下载→重启→版本号变化；并验证取消后「重启更新」与目录不可写时的兜底提示
