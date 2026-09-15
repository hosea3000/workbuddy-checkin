## Why

固定的「签到时间」（默认 09:30）是本产品唯一需要用户理解的配置项，与「零配置、不暴露复杂设置」的产品理念（PRD F10）冲突。更关键的是它只提供一次触发机会：若 09:30 定时器因休眠、进程冻结或未收到唤醒事件而未触发，账号当天即漏签。

改为每小时巡检后，签到不再依赖单一时点，配置项消失，且「当天最终一定会签上」由持续兜底保证。

## What Changes

- 移除「签到时间」设置项（`Settings.CheckinHour/CheckinMinute` 及设置页输入框、账号页时间显示）；`Settings` 仅保留开机自启与更新代理。
- 调度器从「每日 HH:MM 定时器 + 独立的每小时余额 ticker」合并为**单个每小时 ticker**：每次 tick 先执行全部账号的签到巡检（未签则签），再刷新全部账号余额。
- 应用启动（托盘就绪后）与系统休眠唤醒/会话解锁时，立即执行一次相同的签到巡检；不再有「当前时间已过签到点」这一前置条件。
- 删除固定触发点（`NextTrigger`）与 30 分钟独立重试机制：每小时巡检本身就是重试。
- 删除每日尝试次数上限：由「每小时一次 + 账号间 5~20s 抖动」节流防风控。
- 失败通知按「当日首次失败」节流，避免失败账号每小时重复弹 Toast；成功通知照常（每账号每天一次）。
- 修复既有缺陷：`Credential.TodayAttempts` 注释称「跨天清零」但从未归零，导致第 1 天用满上限后重试永久失效（本变更中上限移除，但跨天归零仍按规范修复）。
- **BREAKING**：不再有「每日 09:30 自动签到」的语义。签到发生在应用当天首次运行后的第一个每小时巡检点，或启动/唤醒时立即执行。

## Capabilities

### New Capabilities

<!-- 无 -->

### Modified Capabilities

- `scheduled-checkin`: 「每日定时签到」改为「每小时签到巡检」；移除固定 30 分钟重试与每日 3 次上限；新增失败通知按当日首次失败节流。
- `catch-up-checkin`: 移除「当前时间已过今日签到点」条件；触发时机改为应用启动、休眠唤醒/会话解锁、每小时巡检。
- `settings`: 设置项移除「签到时间」及其默认值 09:30。
- `credential-store`: 「设置读写」要求的持久化字段清单移除「签到时间」，并同步为当前实际字段（开机自启、更新代理）。

## Impact

- **代码**：`model/settings.go`（删两字段）、`internal/scheduler/schedule.go`（删 `NextTrigger`/`ShouldCatchUp`/`ShouldRetry`/`RetryDelay`/`MaxRetriesPerDay`）、`internal/scheduler/scheduler.go`（`loop` 重写、删 `Wake`/`CatchUp`/`scheduleRetry`）、`app.go`（启动/唤醒改调 `RunAll`）、`internal/checkin/service.go`（跨天归零、通知节流）。
- **前端**：`frontend/src/views/SettingsView.vue`、`frontend/src/views/AccountsView.vue`、`frontend/wailsjs/go/models.ts`（`wails generate module` 重生成）。
- **文档**：`docs/PRD.md` F4/F5/F7 与 §2 场景 2；`docs/DESIGN.md` §7.2/§7.3。
- **测试**：`internal/scheduler/schedule_test.go`、`internal/scheduler/scheduler_test.go`。
- **依赖 / 存储 / 协议**：无新增依赖；`settings.json` 旧文件含 `checkinHour/checkinMinute` 字段，Go 反序列化忽略未知字段，向后兼容，无需迁移。
