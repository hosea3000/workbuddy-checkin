package model

// AccountView 是暴露给前端的账号视图，令牌只以末 8 位形式出现。
type AccountView struct {
	ID                 string    `json:"id"`
	Nickname           string    `json:"nickname"`
	Email              string    `json:"email"`
	Status             string    `json:"status"`
	TokenSuffix        string    `json:"tokenSuffix"`
	ExpiresAt          int64     `json:"expiresAt"`
	Today              TodayView `json:"today"`
	CreditBalance      *float64  `json:"creditBalance"`
	CreditBalanceTotal *float64  `json:"creditBalanceTotal"`
	CreditBalanceAt    int64     `json:"creditBalanceAt"`
}

// TodayView 是账号今日签到状态的视图。
type TodayView struct {
	CheckedIn bool     `json:"checkedIn"`
	Time      string   `json:"time"`
	Credit    *float64 `json:"credit"`
	Message   string   `json:"message"`
}

// CheckinResult 是一次签到的结果（手动 / 自动 / 批量）。
type CheckinResult struct {
	ID          string   `json:"id"`
	Success     bool     `json:"success"`
	Message     string   `json:"message"`
	Credit      *float64 `json:"credit"`
	CheckedInAt int64    `json:"checkedInAt"`
}

// CheckinSummary 是托盘 tooltip 与顶栏计数的聚合。
type CheckinSummary struct {
	CheckedIn       int `json:"checkedIn"`
	Total           int `json:"total"`
	ReloginRequired int `json:"reloginRequired"`
}

// LoginStart 是启动登录的结果。
type LoginStart struct {
	AuthURL   string `json:"authUrl"`
	ExpiresIn int    `json:"expiresIn"`
}

// LoginStatus 是登录轮询状态（前端每 1.5s 调用）。
type LoginStatus struct {
	Stage   string       `json:"stage"` // idle|awaiting_login|awaiting_account|done|failed|canceled
	Done    bool         `json:"done"`
	Error   string       `json:"error"`
	Account *AccountView `json:"account,omitempty"`
}

// QuotaView 是余额查询结果。
type QuotaView struct {
	Balance *float64 `json:"balance"`
	Total   *float64 `json:"total"`
	At      int64    `json:"at"`
}

// TokenSuffix 返回令牌末 8 位（不足 8 位则全遮）；永不返回完整令牌。
func TokenSuffix(token string) string {
	if len(token) <= 8 {
		return "********"[:len(token)]
	}
	return token[len(token)-8:]
}
