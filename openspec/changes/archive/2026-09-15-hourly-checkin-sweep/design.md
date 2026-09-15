## Context

当前调度（`internal/scheduler`）分两套：`loop` 用 `NextTrigger` 计算每日 `HH:MM` 触发点、等一个长 timer，另有 `quotaTick` 每 1 小时刷余额。签到触发源为定时器、启动 `CatchUp`、休眠唤醒 `Wake`；`ShouldCatchUp` 要求 `now >= 今日签到点`。时点来自 `Settings.CheckinHour/CheckinMinute`。

已知缺陷：`Credential.TodayAttempts` 注释称「跨天清零」，但 `internal/checkin/service.go:129` 只做 `++`，全仓无重置点。`ShouldRetry` 用 `TodayAttempts < 3` 判定，于是第 1 天用满 3 次后，后续每天的重试永久失效。

本变更去掉时点配置，把「签到」从单次定点改为每小时巡检，并顺手修复该缺陷。

## Goals / Non-Goals

**Goals**
- 去掉「签到时间」配置，签到改为每小时巡检 + 余额刷新。
- 删除固定触发点与独立重试机制，净减代码与状态。
- 修复 `TodayAttempts` 跨天不归零。

**Non-Goals**
- 不改上游协议、签到成功判定、余额查询与令牌续期。
- 不引入「补签开关」「通知开关」等新配置（PRD F5 明确补签为默认行为、无开关）。
- 不改手动签到、账号管理、托盘菜单与更新器。

## Decisions

**决策 1：单个每小时 ticker，巡检与余额同 tick**
- `loop` 改为 `time.NewTicker(time.Hour)`，每 tick 依次执行 `RunAll`（签到巡检）与 `RefreshAllQuotas`（余额）。
- 备选：保留每日定点 + 每小时兜底（原「思路 C」）。否决——仍保留配置项与两套触发逻辑，复杂度收益不划算，且用户已决定去掉时间配置。
- 副作用：签到成功当刻 `PerformCheckin` 已刷过一次余额，同 tick 的 `RefreshAllQuotas` 会再刷一次；每账号每天多一次请求，可接受。

**决策 2：删除每日尝试次数上限**
- 每小时巡检即重试；节流靠 1 小时间隔 + 账号间 5~20s 抖动 + `relogin_required` 跳过。
- 最坏 24 次/天/账号。保留 `TodayAttempts` 计数，若后续观测到上游风控，新增 `MaxAttemptsPerDay` 常量并在 `RunAll` 加一个跳过条件即可加回。
- 备选：保留每日 3 次。否决——当天较晚才恢复网络时会漏签，与「当天必签」目标冲突。

**决策 3：失败通知按当日首次失败节流**
- 判定「当日首次尝试」：写入前若 `TodayDate != 今天 || TodayAttempts == 0`。
- 仅首次失败发通知，否则失败账号每小时弹一次 Toast（最多 24 次/天）。
- 成功通知不受影响：成功后幂等跳过，一天至多一次。

**决策 4：删除 `Wake` / `CatchUp` / `ShouldCatchUp`，调用方直接 `RunAll`**
- 去掉签到点条件后，`ShouldCatchUp` 退化为 `!relogin && !HasCheckedInToday`，与 `RunAll` 的跳过条件完全重合，保留即重复。
- 启动与托盘唤醒语义不变：立即执行一次巡检。

**决策 5：`TodayAttempts` 跨天归零**
- `PerformCheckin` 写入新状态前：`if c.TodayDate != model.Today(now) { c.TodayAttempts = 0 }`。

## Risks / Trade-offs

- [上游风控] 无上限重试最多 24 次/天/账号 → 1h 间隔 + 5~20s 抖动；`relogin_required` 跳过；保留 `TodayAttempts` 以便随时加回上限。
- [通知时间不可预期] 跨午夜的首个 tick 可能在 00:xx 签到并弹「签到成功」→ 保留成功通知（用户可见当天已完成）；若扰民，后续可对夜间成功通知静默，本变更不做。
- [上游日界线] 若本地午夜与上游「自然日」不一致，00:xx 签到可能计入上游前一天 → 目标用户为 UTC+8，本地午夜与上游一致；其他时区为已知边界，暂不处理。
- [余额重复请求] 签到成功 tick 内多刷一次余额 → 每账号每天多一次，可接受。
- [settings.json 旧字段] 旧文件含 `checkinHour/checkinMinute` → Go 反序列化忽略未知字段，向后兼容，无需迁移。

## Migration Plan

无数据迁移。旧 `settings.json` 的多余字段被忽略，下次 `SaveSettings` 时自然清除。回滚：还原代码即可，签到时间字段可重新加入且旧值仍在文件中。

## Open Questions

- 失败通知是否还需按夜间时段（如 23:00–07:00）静默？本变更只做「当日首次失败」节流，暂不按时段静默。
- 是否需要为无上限重试预设一个 `MaxAttemptsPerDay`（如 6）？暂不设，按风险项随时加回。
