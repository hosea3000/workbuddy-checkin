package model

// Credential 是一个 CodeBuddy 账号的本地凭证记录。
// 令牌明文存储（0600），日志/UI/通知永不出现完整令牌。
type Credential struct {
	ID                string `json:"id"`      // 自产 UUID
	UserID            string `json:"user_id"` // "uid_<account_uid>"，稳定主键
	AccountUID        string `json:"account_uid"`
	Nickname          string `json:"nickname"`
	PreferredUsername string `json:"preferred_username"`
	Email             string `json:"email"`
	EnterpriseID      string `json:"enterprise_id"`
	Domain            string `json:"domain"` // JWT iss / 上游下发
	AccessToken       string `json:"access_token"`
	RefreshToken      string `json:"refresh_token"`
	ExpiresAt         int64  `json:"expires_at"`         // Unix 秒
	RefreshExpiresAt  int64  `json:"refresh_expires_at"` // Unix 秒
	Status            string `json:"status"`             // active | relogin_required

	// 今日签到状态（无历史表，幂等/补签判定读这里；跨天自动失效）
	TodayDate        string   `json:"today_date"` // 本地时区 YYYY-MM-DD
	TodaySuccess     bool     `json:"today_success"`
	TodayMessage     string   `json:"today_message"`
	TodayCredit      *float64 `json:"today_credit"` // 本次获得积分，可能没有
	TodayAttemptedAt int64    `json:"today_attempted_at"`
	TodayAttempts    int      `json:"today_attempts"` // 当日尝试次数（F4 重试上限 3；跨天清零）

	CreditBalance      *float64 `json:"credit_balance"`       // 积分余额 = remaining
	CreditBalanceTotal *float64 `json:"credit_balance_total"` // 周期总额度
	CreditBalanceAt    int64    `json:"credit_balance_at"`    // 上次成功查询余额时间

	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

// 凭证状态。
const (
	StatusActive          = "active"
	StatusReloginRequired = "relogin_required"
)

// IsReloginRequired 判断账号是否需人工重新登录。
func (c Credential) IsReloginRequired() bool { return c.Status == StatusReloginRequired }
