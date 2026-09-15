## MODIFIED Requirements

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

## ADDED Requirements

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
