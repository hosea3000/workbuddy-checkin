package main

import (
	"errors"
	"log"

	"github.com/hosea3000/workbuddy-checkin/internal/account"
	"github.com/hosea3000/workbuddy-checkin/model"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// StartLogin 启动 OAuth 设备授权，返回授权链接（自动打开浏览器）。
func (a *App) StartLogin() (model.LoginStart, error) {
	start, err := a.accounts.Start(a.ctx)
	if err != nil {
		log.Printf("[login] StartLogin failed: %v", err)
		return model.LoginStart{}, err
	}
	log.Printf("[login] StartLogin ok, browser should open")
	return start, nil
}

// LoginStatus 推进一次登录轮询并返回当前状态（前端每 1.5s 调用）。
func (a *App) LoginStatus() model.LoginStatus {
	status := a.accounts.Stage(a.ctx)
	if status.Done {
		a.refreshTrayTip()
	}
	return status
}

// CancelLogin 取消当前登录会话。
func (a *App) CancelLogin() {
	a.accounts.Cancel()
}

// OpenAuthURL 用系统浏览器重新打开授权链接。
func (a *App) OpenAuthURL(url string) error {
	if !isSafeAuthURL(url) {
		return errors.New("invalid auth url")
	}
	wruntime.BrowserOpenURL(a.ctx, url)
	return nil
}

// ListAccounts 返回账号视图列表。
func (a *App) ListAccounts() []model.AccountView {
	creds := a.store.ListCredentials()
	now := timeNow()
	out := make([]model.AccountView, 0, len(creds))
	for _, c := range creds {
		out = append(out, account.ToView(c, now))
	}
	return out
}

// CheckinSummary 返回今日签到聚合（托盘 tooltip / 顶栏计数）。
func (a *App) CheckinSummary() model.CheckinSummary {
	creds := a.store.ListCredentials()
	now := timeNow()
	summary := model.CheckinSummary{Total: len(creds)}
	for _, c := range creds {
		if c.IsReloginRequired() {
			summary.ReloginRequired++
			continue
		}
		if c.HasCheckedInToday(now) {
			summary.CheckedIn++
		}
	}
	return summary
}

// DeleteAccount 删除本地凭证（不调用上游）。
func (a *App) DeleteAccount(id string) error {
	if err := a.store.DeleteCredential(id); err != nil {
		return err
	}
	a.refreshTrayTip()
	return nil
}

// HasAccounts 返回是否已有账号（首启引导路由判定）。
func (a *App) HasAccounts() bool {
	return len(a.store.ListCredentials()) > 0
}

// isSafeAuthURL 校验外部授权链接：绝对 http(s)、无 userinfo、无控制字符。
func isSafeAuthURL(raw string) bool { return safeExternalURL(raw) }
