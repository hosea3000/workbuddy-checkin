# app-update Specification

## Purpose

检查 GitHub Release 最新版本、提示更新并下载自替换升级，覆盖启动时与手动触发、待应用更新的恢复与清理，以及安装目录不可写时的兜底提示。

## Requirements

### Requirement: 检查更新

系统 SHALL 通过 `CheckUpdate()` 请求 GitHub Release 最新版本，将 tag 版本与应用运行时版本做语义化比较，并返回「已是最新 / 有更新 / 检查失败」三态结果。开发版本（版本号为 `dev`）SHALL 短路、不发起网络请求。

#### Scenario: 发现新版本
- **WHEN** Release 最新 tag 高于当前运行时版本
- **THEN** 返回「有更新」，并携带最新版本号与 Release 页面地址

#### Scenario: 已是最新
- **WHEN** 当前运行时版本不低于 Release 最新 tag
- **THEN** 返回「已是最新」

#### Scenario: 开发版本短路
- **WHEN** 运行时版本为 `dev`
- **THEN** 直接返回「已是最新」，不发起网络请求

#### Scenario: 检查失败
- **WHEN** 网络异常、仓库无发布版本或响应异常
- **THEN** 返回「检查失败」并给出面向用户的中文提示，不改变应用其他状态

### Requirement: 更新触发时机

系统 SHALL 在应用启动时自动检查一次更新，并 SHALL 提供用户手动检查更新的入口。系统 SHALL NOT 使用定时轮询。

#### Scenario: 启动自动检查
- **WHEN** 应用启动
- **THEN** 执行一次更新检查

#### Scenario: 手动检查
- **WHEN** 用户在设置页点击「检查更新」
- **THEN** 执行一次更新检查

### Requirement: 发现新版本提示

系统 SHALL 在发现新版本时向用户展示可更新状态（含新版本号），并在设置页提供「立即更新」入口。

#### Scenario: 顶栏提示新版本
- **WHEN** 检查发现新版本
- **THEN** 界面显示「发现新版本 vX.Y.Z」并可进入更新流程

### Requirement: 下载并自替换升级

系统 SHALL 将新版本原始可执行文件下载到应用同目录，以 `.part` 作为下载中临时文件、完成后落位为 `.new`；用户确认重启后 SHALL 在进程退出后覆盖当前可执行文件并启动新版本。覆盖失败时 SHALL 重试，重试耗尽则启动旧版本并保留已下载文件。

#### Scenario: 下载新版本
- **WHEN** 用户在有可用更新时点击「立即更新」
- **THEN** 新版本 exe 被下载到同目录，下载完成后落位为 `.new`，界面展示下载进度

#### Scenario: 重启应用更新
- **WHEN** 下载完成且用户确认立即重启
- **THEN** 应用退出、`.new` 覆盖当前 exe、新版本启动

#### Scenario: 稍后应用
- **WHEN** 下载完成但用户选择稍后
- **THEN** 保留 `.new`，界面提供「重启更新」入口，重启后仍可应用

#### Scenario: 覆盖失败回退
- **WHEN** 覆盖 exe 连续失败达到重试上限
- **THEN** 启动旧版本并保留 `.new`，不损坏现有程序

### Requirement: 待应用更新恢复与清理

系统 SHALL 在启动时识别同目录中已下载待应用的 `.new` 及其版本标记，并 SHALL 清理下载残渣（`.part`）与孤儿版本标记。

#### Scenario: 启动时发现待应用更新
- **WHEN** 应用启动且同目录存在 `.new` 及其版本标记
- **THEN** 向用户展示待应用的版本并可一键重启更新

#### Scenario: 清理下载残渣
- **WHEN** 应用启动且同目录存在 `.part` 残渣或孤儿的版本标记
- **THEN** 删除这些残渣文件

### Requirement: 目录不可写兜底

当可执行文件所在目录对当前用户不可写时，系统 SHALL NOT 尝试自替换升级，而 SHALL 提示用户前往 Release 页面手动更新。

#### Scenario: 安装目录不可写
- **WHEN** exe 所在目录不可写且用户触发更新
- **THEN** 提示「程序目录不可写，请手动更新」，不产生 `.part`/`.new` 文件
