package main

import (
	"github.com/hosea3000/workbuddy-checkin/model"
)

// CheckinNow 对指定账号执行一次签到。
func (a *App) CheckinNow(id string) (model.CheckinResult, error) {
	res, err := a.checkin.PerformCheckin(a.ctx, id)
	a.refreshTrayTip()
	return res, err
}

// CheckinAll 对全部账号执行签到（托盘菜单 / 批量）。
func (a *App) CheckinAll() []model.CheckinResult {
	results := a.scheduler.RunAll(a.ctx)
	a.refreshTrayTip()
	return results
}

// RefreshQuota 手动刷新指定账号积分余额。
func (a *App) RefreshQuota(id string) (model.QuotaView, error) {
	view, err := a.checkin.RefreshQuota(a.ctx, id)
	a.refreshTrayTip()
	return view, err
}
