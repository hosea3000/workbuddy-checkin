package scheduler

import (
	"math/rand"
	"time"

	"github.com/hosea3000/workbuddy-checkin/model"
)

// NextTrigger 计算下次签到触发点：今天 HH:MM，已过则明天 HH:MM（本地时区）。
func NextTrigger(now time.Time, hour, minute int) time.Time {
	today := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !today.After(now) {
		today = today.AddDate(0, 0, 1)
	}
	return today
}

// ShouldCatchUp 判定给定凭证此刻是否需要补签（启动/唤醒时补签为默认行为）。
func ShouldCatchUp(cred model.Credential, now time.Time, settings model.Settings) bool {
	if cred.IsReloginRequired() {
		return false
	}
	trigger := time.Date(now.Year(), now.Month(), now.Day(), settings.CheckinHour, settings.CheckinMinute, 0, 0, now.Location())
	if now.Before(trigger) {
		return false
	}
	return !cred.HasCheckedInToday(now)
}

// ShouldRetry 判定失败后是否应重试：当日尝试未达上限、非待重登、今日未成功（失败重试为默认行为）。
func ShouldRetry(cred model.Credential, now time.Time) bool {
	if cred.IsReloginRequired() {
		return false
	}
	if cred.HasCheckedInToday(now) {
		return false
	}
	return cred.TodayAttempts < MaxRetriesPerDay
}

// MaxRetriesPerDay 是每账号每日自动重试上限。
const MaxRetriesPerDay = 3

// RetryDelay 是失败后的重试间隔。
const RetryDelay = 30 * time.Minute

// Jitter 返回 5~20 秒随机间隔（防风控）。
func Jitter(rng *rand.Rand) time.Duration {
	if rng == nil {
		return 5 * time.Second
	}
	return time.Duration(5+rng.Intn(16)) * time.Second
}
