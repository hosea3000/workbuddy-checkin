# ci-autorelease Specification

## Purpose

推送 `v*` tag 时自动构建 Windows 产物、注入版本号、发布原始可执行文件与 per-user 中文安装器两个资产，并在发布前做构建门禁、在卸载时拒绝运行中的应用。

## Requirements

### Requirement: Tag 触发发布

系统 SHALL 在推送形如 `v*` 的 tag 时触发发布工作流，构建 Windows amd64 产物与 macOS amd64/arm64 产物，并创建对应的 GitHub Release。

#### Scenario: 推送版本 tag
- **WHEN** 仓库收到匹配 `v*` 的 tag 推送
- **THEN** 工作流构建 Windows 与 macOS 产物并创建以该 tag 命名的 GitHub Release

#### Scenario: 非 tag 推送不触发
- **WHEN** 普通分支提交被推送且未创建 `v*` tag
- **THEN** 不触发发布工作流

### Requirement: 版本号注入

发布构建 SHALL 从 tag 派生版本号（去掉前导 `v`），并同时注入到可执行文件的运行时版本变量（`-ldflags "-X main.version=<版本>"`）与 `wails.json` 的 `info.productVersion`；Windows 与 macOS 构建 SHALL 使用同一版本号。

#### Scenario: 版本号来自 tag
- **WHEN** 以 tag `v0.2.0` 触发发布
- **THEN** Windows 与 macOS 产物运行时版本均为 `0.2.0`

### Requirement: 双资产发布

发布 SHALL 上传 Windows 两个 Release 资产：原始可执行文件 `workbuddy-checkin.exe`（供应用内更新与绿色版使用）和 per-user 安装器 `workbuddy-checkin-amd64-installer.exe`（供首次安装使用）。

#### Scenario: 一次构建产出两个资产
- **WHEN** 发布工作流完成
- **THEN** Release 中同时存在 `workbuddy-checkin.exe` 与 `workbuddy-checkin-amd64-installer.exe` 两个资产

#### Scenario: 资产命名稳定
- **WHEN** 应用内更新按文件名查找资产
- **THEN** 能找到名为 `workbuddy-checkin.exe` 的资产

### Requirement: macOS 双架构 dmg 发布

发布工作流 SHALL 在 macOS runner 上为 `darwin/amd64` 与 `darwin/arm64` 分别构建 `.app` 并打包为 `.dmg`，作为 Release 资产上传，命名分别为 `WorkBuddy-checkin-amd64.dmg` 与 `WorkBuddy-checkin-arm64.dmg`。产物 SHALL NOT 代码签名或公证。

#### Scenario: 产出两个 dmg
- **WHEN** 发布工作流完成
- **THEN** Release 中同时存在 `WorkBuddy-checkin-amd64.dmg` 与 `WorkBuddy-checkin-arm64.dmg` 两个资产

#### Scenario: dmg 命名稳定
- **WHEN** macOS 应用内更新按架构查找资产
- **THEN** 能找到与运行时 `GOARCH` 一致命名的 dmg 资产

#### Scenario: 未签名产物
- **WHEN** 用户从 dmg 安装后首次打开应用
- **THEN** 应用为未签名/未公证状态，README 说明需 `xattr -cr` 去隔离

### Requirement: dmg 打包使用系统工具

CI SHALL 使用 macOS 自带工具（如 `hdiutil`）打包 dmg，SHALL NOT 引入额外的第三方打包依赖。

#### Scenario: 无额外依赖打包
- **WHEN** macOS 构建 job 执行
- **THEN** 仅使用 runner 预装的系统工具完成 `.app` → `.dmg` 打包

### Requirement: Per-user 中文安装器

安装器 SHALL 以 per-user 范围构建（安装到 `%LOCALAPPDATA%\Programs\`，不请求管理员权限），界面语言为简体中文，并保留安装目录选择页。

#### Scenario: 普通用户安装
- **WHEN** 普通用户（非管理员）双击运行安装器
- **THEN** 安装在不触发 UAC 的情况下完成，程序位于当前用户的 `%LOCALAPPDATA%\Programs\` 下

#### Scenario: 中文安装界面
- **WHEN** 用户打开安装向导
- **THEN** 向导界面为简体中文

#### Scenario: 可选择安装目录
- **WHEN** 用户进入安装向导的目录选择页
- **THEN** 用户可更改安装目录，默认目录为 `%LOCALAPPDATA%\Programs\workbuddy-checkin`

### Requirement: 发布前构建门禁

发布工作流 SHALL 在构建产物前执行静态检查与单元测试（`go vet ./...` 与 `go test ./...`），任一失败则中止发布。

#### Scenario: 测试失败中止发布
- **WHEN** 发布流程中 `go vet` 或 `go test` 失败
- **THEN** 不构建、不创建 Release

### Requirement: 卸载时拒绝运行中的应用

卸载程序 SHALL 在检测到应用正在运行时中止卸载并结束卸载器进程，同时提示用户先退出应用；SHALL NOT 在应用运行中继续卸载或删除程序文件。应用未运行时 SHALL 正常卸载。

#### Scenario: 应用运行时卸载
- **WHEN** 用户运行卸载程序且应用正在运行（含仅驻留托盘）
- **THEN** 弹出中文提示「应用正在运行，请先退出后再卸载」，卸载器随即关闭，程序文件保留

#### Scenario: 应用未运行时卸载
- **WHEN** 用户运行卸载程序且应用未运行
- **THEN** 正常执行卸载
