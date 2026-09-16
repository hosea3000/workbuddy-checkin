## Why

macOS 用户关闭主窗口时应用直接退出，后台签到随之停止——用户必须一直开着窗口才能保证每日签到，这与「开机自启 + 后台静默签到」的产品初衷矛盾。Windows 已通过系统托盘解决，macOS 却因早期「不做托盘」的范围决定而缺失同等能力。

## What Changes

- macOS 新增菜单栏（menu bar / NSStatusItem）常驻图标，菜单含「打开」与「退出」；左键单击直接弹出菜单。
- macOS 关闭主窗口 SHALL 隐藏窗口到菜单栏而非退出应用，调度器保持运行（与 Windows 行为对齐）。
- macOS 应用不再显示 Dock 图标（`LSUIElement` + 运行时 `Accessory` 激活策略），形态为纯菜单栏常驻应用。
- 首次关窗提示改为两平台通用文案（不再只说「托盘」）。
- **BREAKING（对既有 macOS 行为）**：macOS 关闭窗口不再退出应用；退出仅能经菜单栏菜单、⌘Q 或 Dock/系统菜单触发。
- 文档同步：PRD F8/§6/§8、DESIGN 平台能力矩阵、AGENTS.md 硬约束（解除「macOS 只用系统命令、不引入第三方依赖」中关于菜单栏图标的限制，明确允许手写 cgo/Objective-C）。

## Capabilities

### New Capabilities
<!-- 无新增能力；本变更修改已有 tray-integration 的需求。 -->

### Modified Capabilities
- `tray-integration`: 「托盘驻留」需求从「macOS 不提供托盘、关窗即退出」改为「macOS 提供菜单栏常驻、关窗隐藏」；「托盘菜单与 tooltip」「非 Windows 平台可测试」需覆盖 macOS 实现；新增菜单栏图标行为与退出语义。
- `settings`: 「首次提示最小化到托盘」需求从「仅 Windows 提示、macOS 不提示」改为「两平台首次隐藏窗口时均提示一次」。

## Impact

- 代码：`main.go`（`HideWindowOnClose` 保持 Windows-only 的评估）、`app.go`（`beforeClose` / `startup` / `notifyCloseTipOnce`）、新增 `tray_darwin.go` + `tray_darwin.m`、`tray_stub.go`（build tag 收窄为 `!windows && !darwin`）、`build/darwin/Info.plist`（`LSUIElement`）。
- 资源：新增 `build/darwin/trayTemplate@1x.png`（16×16）、`build/darwin/trayTemplate@2x.png`（32×32）模板图标（已落地）。
- 依赖：macOS 手写 cgo + Objective-C，**不引入任何新的第三方 Go 依赖**（沿用现有 `Cocoa` framework）。
- 平台：仅影响 darwin；Windows 行为不变；Linux 仍走 stub，`go test ./...` 不受影响。
- 文档：`docs/PRD.md`、`docs/DESIGN.md`、`AGENTS.md`。
