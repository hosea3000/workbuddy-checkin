## Context

`main.go` 恒设 `StartHidden: true`，`startup` 用 `!isAutoStartEnabled()` 决定是否 `showWindow()`。health-tool 的已验证实现是把 `StartHidden` 交给 `hiddenFromArgs(os.Args[1:])`，并在 HKCU Run 命令里追加 `--hidden`（`autostart.go:14-27`）。

## Goals / Non-Goals

**Goals**
- 手动启动始终显示主窗口；自启启动静默进托盘。
- 启动方式判定确定、可测（不依赖 Wails OnStartup 时序）。

**Non-Goals**
- 不改 `GetSettings` 的 `AutoStart` 数据来源（仍读存储值，不实时读注册表）。
- 不改托盘菜单/退出逻辑（属 `fix-tray-menu-actions`）。

## Decisions

**决策 1：用 `--hidden` 命令行标记，而非 `isAutoStartEnabled()`**
- `isAutoStartEnabled()` 描述的是「配置状态」，不是「本次启动来源」，二者不等价；手动双击时会被误判。
- 自启命令写入 `--hidden` 后，启动来源由进程参数直接决定，无歧义。
- 选 health-tool 方案。

**决策 2：`StartHidden` 在构造 `options.App` 时确定，不在 `OnStartup` 里补 show**
- Wails `OnStartup` 在 goroutine 中执行且 `start()` 会在 `StartHidden` 时跳过显示（`frontend.go:980`），事后 show 有时序竞争。
- 从根上让 `StartHidden` 反映真实意图即可，无需事后补救。

**决策 3：新增无构建标签的 `autostart.go` 承载纯逻辑**
- `hiddenFromArgs` / `buildAutoStartCommand` 与平台无关，放无标签文件可在 Linux 单测覆盖。
- `setAutoStart`（windows）复用 `buildAutoStartCommand`；stub 不变。

## Risks / Trade-offs

- [已安装旧版写入的 Run 命令不含 `--hidden`] → 旧命令仍能启动，只是不会静默（首次升级后开机自启会弹窗一次）；用户下次开关自启即被重写为带 `--hidden`。可接受，不写迁移逻辑。
- [删除 `isAutoStartEnabled` 影响面] → 全仓库仅 `startup` 一处引用，删除安全。

## Migration Plan

无数据迁移。自启命令在用户下次切换「开机自启」开关时重写为带 `--hidden` 形式。

## Open Questions

无。
