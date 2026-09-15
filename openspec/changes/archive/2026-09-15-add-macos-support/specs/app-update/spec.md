## MODIFIED Requirements

### Requirement: 下载并自替换升级

系统 SHALL 将新版本产物下载到应用同目录，以 `.part` 作为下载中临时文件、完成后落位为 `.new`；用户确认重启后 SHALL 在进程退出后覆盖当前可执行文件并启动新版本。覆盖失败时 SHALL 重试，重试耗尽则启动旧版本并保留已下载文件。macOS SHALL 改为下载对应架构的 `.dmg` 到 `~/Downloads/`，随后 `open` 挂载引导用户拖入 Applications 完成安装，SHALL NOT 原地替换 `.app`。下载资产名 SHALL 按 `GOOS` 选择：Windows 为 `workbuddy-checkin.exe`；macOS 按 `GOARCH` 选择 `WorkBuddy-checkin-amd64.dmg` 或 `WorkBuddy-checkin-arm64.dmg`。

#### Scenario: Windows 下载新版本
- **WHEN** Windows 用户在有可用更新时点击「立即更新」
- **THEN** 新版本 exe 被下载到同目录，下载完成后落位为 `.new`，界面展示下载进度

#### Scenario: Windows 重启应用更新
- **WHEN** 下载完成且用户确认立即重启
- **THEN** 应用退出、`.new` 覆盖当前 exe、新版本启动

#### Scenario: Windows 稍后应用
- **WHEN** 下载完成但用户选择稍后
- **THEN** 保留 `.new`，界面提供「重启更新」入口，重启后仍可应用

#### Scenario: Windows 覆盖失败回退
- **WHEN** 覆盖 exe 连续失败达到重试上限
- **THEN** 启动旧版本并保留 `.new`，不损坏现有程序

#### Scenario: macOS 下载 dmg
- **WHEN** macOS 用户点击「立即更新」
- **THEN** 下载与当前架构匹配的 `.dmg` 到 `~/Downloads/`，界面展示下载进度

#### Scenario: macOS 挂载安装
- **WHEN** macOS 下 dmg 下载完成且用户确认
- **THEN** 系统挂载该 dmg 并弹出安装窗口，应用退出，用户拖入 Applications 完成替换

#### Scenario: macOS 按架构选择资产
- **WHEN** macOS 应用检查更新且存在多个 dmg 资产
- **THEN** 选择与运行时 `GOARCH` 一致的 dmg 下载地址

### Requirement: 待应用更新恢复与清理

系统 SHALL 在 Windows 启动时识别同目录中已下载待应用的 `.new` 及其版本标记，并 SHALL 清理下载残渣（`.part`）与孤儿版本标记。macOS SHALL NOT 使用 `.new`/待应用机制。

#### Scenario: Windows 启动时发现待应用更新
- **WHEN** Windows 应用启动且同目录存在 `.new` 及其版本标记
- **THEN** 向用户展示待应用的版本并可一键重启更新

#### Scenario: Windows 清理下载残渣
- **WHEN** Windows 应用启动且同目录存在 `.part` 残渣或孤儿的版本标记
- **THEN** 删除这些残渣文件

#### Scenario: macOS 无待应用更新
- **WHEN** macOS 应用启动
- **THEN** 不展示待应用更新入口

### Requirement: 目录不可写兜底

当可执行文件所在目录对当前用户不可写时，Windows SHALL NOT 尝试自替换升级，而 SHALL 提示用户前往 Release 页面手动更新。macOS 因下载到 `~/Downloads/` 不涉及应用目录写权限，SHALL NOT 触发该兜底。

#### Scenario: Windows 安装目录不可写
- **WHEN** exe 所在目录不可写且用户触发更新
- **THEN** 提示「程序目录不可写，请手动更新」，不产生 `.part`/`.new` 文件

#### Scenario: macOS 不触发目录不可写兜底
- **WHEN** macOS 用户触发更新
- **THEN** 正常下载到 `~/Downloads/`，不出现目录不可写提示
