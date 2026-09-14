package model

import "time"

// Today 返回本地时区的 YYYY-MM-DD。
func Today(now time.Time) string { return now.Format("2006-01-02") }

// HasCheckedInToday 判断凭证今日是否已成功签到；TodayDate 非今天时视为失效。
func (c Credential) HasCheckedInToday(now time.Time) bool {
	return c.TodaySuccess && c.TodayDate == Today(now)
}
