package account

import (
	"errors"
	"fmt"
	"time"

	"github.com/hosea3000/workbuddy-checkin/model"
	"github.com/hosea3000/workbuddy-checkin/upstream/codebuddy"
)

// ToView 将凭证映射为前端视图（令牌只暴露末 8 位）。
// activeID 为设置中的当前凭证 ID，用于标记模型代理正在使用的账号。
func ToView(c model.Credential, now time.Time, activeID string) model.AccountView {
	today := model.TodayView{
		CheckedIn: c.HasCheckedInToday(now),
		Message:   c.TodayMessage,
	}
	if today.CheckedIn {
		today.Time = time.Unix(c.TodayAttemptedAt, 0).Format("15:04")
	}
	return model.AccountView{
		ID:                 c.ID,
		Nickname:           displayName(c),
		Email:              c.Email,
		Status:             normalizeStatus(c.Status),
		TokenSuffix:        model.TokenSuffix(c.AccessToken),
		ExpiresAt:          c.ExpiresAt,
		IsActive:           activeID != "" && c.ID == activeID,
		Today:              today,
		CreditBalance:      c.CreditBalance,
		CreditBalanceTotal: c.CreditBalanceTotal,
		CreditBalanceAt:    c.CreditBalanceAt,
	}
}

func displayName(c model.Credential) string {
	if c.Nickname != "" {
		return c.Nickname
	}
	if len(c.UserID) > 12 {
		return c.UserID[:12]
	}
	return c.UserID
}

func normalizeStatus(s string) string {
	if s == model.StatusReloginRequired {
		return model.StatusReloginRequired
	}
	return model.StatusActive
}

// isTransientAuthError 判断错误是否属可重试的瞬态故障（服务不可用 / 网络）。
// 登录轮询遇到该类错误应保持会话继续轮询，而非中断。
func isTransientAuthError(err error) bool {
	var ae *codebuddy.AuthError
	if !errors.As(err, &ae) {
		return false
	}
	switch ae.Name {
	case "auth_unavailable":
		return true
	}
	return false
}

// humanizeAuthError 将登录受控错误转为人话（PRD F1 业务码映射）。
func humanizeAuthError(err error) error {
	var ae *codebuddy.AuthError
	if errors.As(err, &ae) {
		switch ae.Name {
		case "license_seat_limit", "license_expired", "trial_expired", "ip_restricted":
			return errors.New(ae.Description)
		case "unauthorized":
			return errors.New("凭证已被上游拒绝，请重试")
		case "auth_unavailable":
			if ae.Cause != nil {
				return fmt.Errorf("认证服务暂时不可用（%v）", ae.Cause)
			}
			return errors.New("认证服务暂时不可用，请稍后重试")
		}
		if ae.Description != "" {
			return errors.New(ae.Description)
		}
	}
	return fmt.Errorf("登录失败：%v", err)
}
