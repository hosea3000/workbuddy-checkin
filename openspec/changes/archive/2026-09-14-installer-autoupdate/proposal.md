## Why

当前只发布单文件 `workbuddy-checkin.exe`，普通用户下载后不知放哪、找不到、不会卸载、也不知道如何升级；PRD 又把「图形化安装器」列为非目标，无法支撑「推广给普通用户」的定位。本变更引入 per-user 安装器与内置自动更新，把「首次安装」和「后续升级」两件事都变成普通用户熟悉的流程。

## What Changes

- 新增 per-user NSIS 安装器（`-installscope user`，装到 `%LOCALAPPDATA%\Programs\`，无需管理员/UAC），界面改为简体中文，保留目录选择页。
- Release 同时发布两个资产：`workbuddy-checkin.exe`（更新用/绿色版）与 `workbuddy-checkin-amd64-installer.exe`（首次安装用）。
- 应用内新增「检查更新 + 一键自替换升级」：启动时检查（可关）+ 设置页手动检查；下载新版 exe、退出、覆盖自身、重启。
- 设置页展示当前版本号与手动「检查更新」入口。
- **BREAKING**：产品定位由「单文件免安装」调整为「安装器 + 绿色版双产物」；PRD 非目标中移除「图形化安装器」。

## Capabilities

### New Capabilities
- `ci-autorelease`: tag 触发的 CI 构建与发布——产出 Windows exe 与 per-user 中文 NSIS 安装器，并作为 GitHub Release 资产发布。
- `app-update`: 运行时检查 GitHub Release 最新版本、下载新版 exe、自替换并重启；含启动检查与手动检查两种触发。

### Modified Capabilities
- `settings`: 新增「手动检查更新」入口与「当前版本」展示。

## Impact

- CI：`.github/workflows/release.yml`（装 `nsis`、加 `-nsis -installscope user`、上传双资产）
- 构建资产：`build/windows/installer/project.nsi`（中文定制并提交）
- 代码：新增 `updater.go` / `updater_apply.go` / `updater_apply_windows.go` / `updater_apply_stub.go`；`main.go` 的 `version` 注入已就绪；新增绑定 `CheckUpdate` / `DownloadAndApplyUpdate` / `ApplyUpdateAndRestart` / `PendingUpdateInfo`；设置页 UI
- 文档：`docs/PRD.md`（形态、非目标、风险）、`docs/DESIGN.md`（能力清单）
- 依赖：构建期需 `makensis`（NSIS）；运行期无新增依赖（纯 stdlib + Wails 运行时）
