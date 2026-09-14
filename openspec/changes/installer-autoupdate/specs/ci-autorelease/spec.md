## ADDED Requirements

### Requirement: Tag 触发发布

系统 SHALL 在推送形如 `v*` 的 tag 时触发发布工作流，构建 Windows amd64 产物并创建对应的 GitHub Release。

#### Scenario: 推送版本 tag
- **WHEN** 仓库收到匹配 `v*` 的 tag 推送
- **THEN** 工作流构建 `windows/amd64` 产物并创建以该 tag 命名的 GitHub Release

#### Scenario: 非 tag 推送不触发
- **WHEN** 普通分支提交被推送且未创建 `v*` tag
- **THEN** 不触发发布工作流

### Requirement: 版本号注入

发布构建 SHALL 从 tag 派生版本号（去掉前导 `v`），并同时注入到可执行文件的运行时版本变量（`-ldflags "-X main.version=<版本>"`）与 `wails.json` 的 `info.productVersion`。

#### Scenario: 版本号来自 tag
- **WHEN** 以 tag `v0.2.0` 触发发布
- **THEN** 产物运行时版本为 `0.2.0`，且 Windows 文件版本资源显示 `0.2.0`

### Requirement: 双资产发布

发布 SHALL 同时上传两个 Release 资产：原始可执行文件 `workbuddy-checkin.exe`（供应用内更新与绿色版使用）和 per-user 安装器 `workbuddy-checkin-amd64-installer.exe`（供首次安装使用）。

#### Scenario: 一次构建产出两个资产
- **WHEN** 发布工作流完成
- **THEN** Release 中同时存在 `workbuddy-checkin.exe` 与 `workbuddy-checkin-amd64-installer.exe` 两个资产

#### Scenario: 资产命名稳定
- **WHEN** 应用内更新按文件名查找资产
- **THEN** 能找到名为 `workbuddy-checkin.exe` 的资产

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
