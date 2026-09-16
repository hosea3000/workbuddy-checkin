## 1. 模型与设置

- [x] 1.1 `model/settings.go`：删除 `CheckinHour`/`CheckinMinute` 字段及 `DefaultSettings` 中的赋值
- [x] 1.2 `internal/checkin/service.go`：`PerformCheckin` 写入新状态前，若 `c.TodayDate != model.Today(now)` 则归零 `c.TodayAttempts`
- [x] 1.3 `internal/checkin/service.go`：失败通知按「当日首次失败」节流（首次尝试判定放在自增 `TodayAttempts` 之前）

## 2. 调度器

- [x] 2.1 `internal/scheduler/schedule.go`：删除 `NextTrigger`、`ShouldCatchUp`、`ShouldRetry`、`RetryDelay`、`MaxRetriesPerDay`；保留 `Jitter`
- [x] 2.2 `internal/scheduler/scheduler.go`：`loop` 改为单个 `time.NewTicker(time.Hour)`，每 tick 依次执行 `RunAll` + `RefreshAllQuotas`
- [x] 2.3 `internal/scheduler/scheduler.go`：删除 `Wake`、`CatchUp`、`scheduleRetry`；`RunAll` 去掉 `ShouldRetry` 分支
- [x] 2.4 `app.go`：启动补签（`app.go:87`）与托盘 `wake`（`app.go:119`）改调 `scheduler.RunAll`

## 3. 前端

- [x] 3.1 `frontend/src/views/SettingsView.vue`：删除「每日签到时间」输入块与 `form.checkinHour/checkinMinute` 字段
- [x] 3.2 `frontend/src/views/AccountsView.vue`：删除签到时间展示逻辑（约 19–20 行）
- [x] 3.3 重跑 `wails generate module` 更新 `frontend/wailsjs/go/models.ts`

## 4. 测试

- [x] 4.1 `internal/scheduler/schedule_test.go`：删除 `NextTrigger`/`ShouldCatchUp`/`ShouldRetry` 相关用例
- [x] 4.2 `internal/scheduler/scheduler_test.go`：改为验证巡检跳过已签/待重登、失败账号下次巡检重试、启动与唤醒直接 `RunAll`
- [x] 4.3 `internal/checkin/service_test.go`：新增 `TodayAttempts` 跨天归零用例
- [x] 4.4 `internal/checkin/service_test.go`：新增失败通知仅当日首次发送的用例
- [x] 4.5 `store/store_test.go`：更新引用了已删 `CheckinHour/CheckinMinute` 字段的设置用例（实现时发现的连带修改）

## 5. 文档

- [x] 5.1 `docs/PRD.md`：F4 改为每小时巡检、F5 去掉时间条件、F7 删除「签到时间」行、§2 场景 2 改写
- [x] 5.2 `docs/DESIGN.md`：§7.2/§7.3 更新触发源与重试说明

## 6. 验证

- [x] 6.1 `go vet ./... && go test ./...` 通过（Linux stub 分支）
- [x] 6.2 `GOOS=windows GOARCH=amd64 go build ./...` 通过
- [x] 6.3 人工：应用启动即巡检签到；把系统日期调到次日 → 下一次巡检自动补签且通知
