package checkin

import (
	"context"
	"errors"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hosea3000/workbuddy-checkin/model"
	"github.com/hosea3000/workbuddy-checkin/store"
	"github.com/hosea3000/workbuddy-checkin/upstream/codebuddy"
)

// Client 是签到所需的上游能力（便于测试注入假实现）。
type Client interface {
	Checkin(ctx context.Context, cred codebuddy.CredentialSnapshot) (bool, *int, string, *float64, error)
	FetchQuotaPersonal(ctx context.Context, cred codebuddy.CredentialSnapshot) (float64, float64, error)
	RefreshToken(ctx context.Context, accessToken, domain, refreshToken string) (*codebuddy.TokenData, error)
}

// Notifier 是通知能力（Windows Toast / Linux stub）。
type Notifier interface {
	Notify(title, body string)
}

// refreshMarginSeconds 是代理侧续期的触发余量：距过期不足该秒数即视为临期。
const refreshMarginSeconds = 60

// Service 执行单账号签到、续期与余额刷新。
type Service struct {
	store  *store.Store
	client Client
	notify Notifier

	now func() time.Time

	// ponytail: 单凭证场景下一个全局锁足够（应用只有当前凭证会走代理续期）。
	// 若将来代理支持多凭证并发，改为 per-credential 锁。
	refreshMu sync.Mutex
}

// NewService 构造签到服务。now 为 nil 时使用 time.Now。
func NewService(st *store.Store, client Client, notify Notifier) *Service {
	return &Service{store: st, client: client, notify: notify, now: time.Now}
}

// snapshot 由凭证映射为上游头生成器需要的视图。
// 注意：X-User-Id 实际取 AccountUID；UserID（带 uid_ 前缀）仅兜底。
func snapshot(c model.Credential) codebuddy.CredentialSnapshot {
	return codebuddy.CredentialSnapshot{
		BearerToken:  c.AccessToken,
		UserID:       c.UserID,
		AccountUID:   c.AccountUID,
		Domain:       c.Domain,
		EnterpriseID: c.EnterpriseID,
	}
}

// refreshTokens 执行令牌交换并回写存储。无副作用：不置状态、不通知、不判定临期。
func (s *Service) refreshTokens(ctx context.Context, c model.Credential) (model.Credential, error) {
	td, err := s.client.RefreshToken(ctx, c.AccessToken, c.Domain, c.RefreshToken)
	if err != nil {
		return c, err
	}
	c.AccessToken = td.AccessToken
	if td.RefreshToken != "" {
		c.RefreshToken = td.RefreshToken
	}
	if td.ExpiresAt != nil {
		c.ExpiresAt = *td.ExpiresAt
	}
	if td.RefreshExpiresAt != nil {
		c.RefreshExpiresAt = *td.RefreshExpiresAt
	}
	if td.Domain != "" {
		c.Domain = td.Domain
	}
	_ = s.store.SaveCredential(c)
	return c, nil
}

// refreshIfNeeded 在凭证临期（<24h）且存在 refresh_token 时续期。
// 返回更新后的凭证（未续期则原样返回；续期失败被拒时置 relogin_required）。
func (s *Service) refreshIfNeeded(ctx context.Context, c model.Credential) model.Credential {
	if c.RefreshToken == "" {
		return c
	}
	if c.ExpiresAt == 0 || s.now().Unix() < c.ExpiresAt-24*3600 {
		return c
	}
	updated, err := s.refreshTokens(ctx, c)
	if err != nil {
		return s.handleRefreshError(c, err)
	}
	return updated
}

// EnsureFresh 供代理调用的无副作用续期入口：仅在令牌临期（距过期不足 60 秒或已过期）
// 且存在 refresh_token 时续期。失败直接返回错误，不置 relogin_required、不发通知。
// 触发窗口比签到路径（24h）窄：令牌仍可用时不应因续期失败而拒绝请求。
func (s *Service) EnsureFresh(ctx context.Context, id string) (model.Credential, error) {
	c, ok := s.store.GetCredential(id)
	if !ok {
		return model.Credential{}, ErrAccountNotFound
	}
	if !s.stale(c) {
		return c, nil
	}
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()
	// 等锁期间可能已被并发请求刷新，重新读取后再判定
	if cur, ok := s.store.GetCredential(id); ok {
		c = cur
	}
	if !s.stale(c) {
		return c, nil
	}
	return s.refreshTokens(ctx, c)
}

// stale 判断凭证是否需要续期：有 refresh_token 且有效期已知且已临期。
func (s *Service) stale(c model.Credential) bool {
	if c.RefreshToken == "" || c.ExpiresAt == 0 {
		return false
	}
	return s.now().Unix() >= c.ExpiresAt-refreshMarginSeconds
}

// handleRefreshError 只在 unauthorized（凭证被上游拒绝）时置 relogin_required 并通知一次。
// 返回更新后的凭证，避免调用方用旧副本覆盖状态。
func (s *Service) handleRefreshError(c model.Credential, err error) model.Credential {
	var ae *codebuddy.AuthError
	if !errors.As(err, &ae) || ae.Name != "unauthorized" {
		log.Printf("[refresh] account=%s failed: %v", shortID(c), err)
		return c
	}
	wasRelogin := c.IsReloginRequired()
	c.Status = model.StatusReloginRequired
	_ = s.store.SaveCredential(c)
	if !wasRelogin && s.notify != nil {
		s.notify.Notify("WorkBuddy 自动签到", "账号需重新登录："+displayName(c))
	}
	return c
}

// PerformCheckin 对单个账号执行签到。cred 为已从 store 取出的副本。
func (s *Service) PerformCheckin(ctx context.Context, id string) (model.CheckinResult, error) {
	c, ok := s.store.GetCredential(id)
	if !ok {
		return model.CheckinResult{}, ErrAccountNotFound
	}
	if c.IsReloginRequired() {
		return model.CheckinResult{ID: id, Success: false, Message: "需重新登录"}, nil
	}
	c = s.refreshIfNeeded(ctx, c)
	if c.IsReloginRequired() {
		return model.CheckinResult{ID: id, Success: false, Message: "需重新登录"}, nil
	}

	now := s.now()
	// 当日首次尝试（用于失败通知节流）：跨天或今日尚未尝试过。
	firstToday := c.TodayDate != model.Today(now) || c.TodayAttempts == 0

	success, _, msg, credit, err := s.client.Checkin(ctx, snapshot(c))
	if err != nil {
		msg = humanizeCheckinError(err)
	}
	// 幂等：上游消息含「已签到」视为成功。
	if !success && strings.Contains(msg, "已签到") {
		success = true
	}

	if c.TodayDate != model.Today(now) {
		c.TodayAttempts = 0
	}
	c.TodayDate = model.Today(now)
	c.TodaySuccess = success
	c.TodayMessage = msg
	c.TodayAttemptedAt = now.Unix()
	c.TodayAttempts++
	if success && credit != nil {
		c.TodayCredit = credit
	}
	if err := s.store.SaveCredential(c); err != nil {
		return model.CheckinResult{}, err
	}

	if success {
		s.refreshQuota(ctx, c)
	}

	res := model.CheckinResult{
		ID:          id,
		Success:     success,
		Message:     buildResultMessage(success, now, c.TodayCredit, msg),
		Credit:      c.TodayCredit,
		CheckedInAt: c.TodayAttemptedAt,
	}
	s.notifyResult(c, res, firstToday)
	return res, nil
}

// refreshQuota 查询并写入积分余额；失败仅记日志，不改 Status、不通知。
func (s *Service) refreshQuota(ctx context.Context, c model.Credential) {
	total, remaining, err := s.client.FetchQuotaPersonal(ctx, snapshot(c))
	if err != nil {
		log.Printf("[quota] account=%s failed: %v", shortID(c), err)
		return
	}
	c.CreditBalance = &remaining
	c.CreditBalanceTotal = &total
	c.CreditBalanceAt = s.now().Unix()
	_ = s.store.SaveCredential(c)
}

// RefreshQuota 供绑定方法手动刷新余额，返回最新视图。
func (s *Service) RefreshQuota(ctx context.Context, id string) (model.QuotaView, error) {
	c, ok := s.store.GetCredential(id)
	if !ok {
		return model.QuotaView{}, ErrAccountNotFound
	}
	if c.IsReloginRequired() {
		return model.QuotaView{Balance: c.CreditBalance, Total: c.CreditBalanceTotal, At: c.CreditBalanceAt}, nil
	}
	c = s.refreshIfNeeded(ctx, c)
	s.refreshQuota(ctx, c)
	updated, _ := s.store.GetCredential(id)
	return model.QuotaView{Balance: updated.CreditBalance, Total: updated.CreditBalanceTotal, At: updated.CreditBalanceAt}, nil
}

// notifyResult 发送签到结果通知。失败仅在当日首次失败时通知，避免每小时巡检重复弹窗。
func (s *Service) notifyResult(c model.Credential, res model.CheckinResult, firstToday bool) {
	if s.notify == nil {
		return
	}
	if res.Success {
		s.notify.Notify("签到成功", displayName(c)+" "+res.Message)
	} else if firstToday {
		s.notify.Notify("签到失败", displayName(c)+"："+res.Message)
	}
}

func buildResultMessage(success bool, now time.Time, credit *float64, msg string) string {
	if !success {
		if msg == "" {
			return "签到失败"
		}
		return msg
	}
	base := "已签到 " + now.Format("15:04")
	if credit != nil {
		base += "，+" + formatCredit(*credit) + " 积分"
	}
	return base
}

func formatCredit(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// humanizeCheckinError 将受控错误转为人话。
func humanizeCheckinError(err error) string {
	var ue *codebuddy.UpstreamError
	if errors.As(err, &ue) {
		switch ue.ErrType {
		case codebuddy.ErrCategoryAuth:
			return "需重新登录"
		case codebuddy.ErrCategoryRateLimit:
			return "请求过于频繁，请稍后再试"
		case codebuddy.ErrCategoryUpstream5xx, codebuddy.ErrCategoryProtocol:
			return "上游服务异常，请稍后重试"
		case codebuddy.ErrCategoryTimeout, codebuddy.ErrCategoryTransport:
			return "网络异常，请检查网络后重试"
		}
	}
	return "签到失败：" + err.Error()
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

func shortID(c model.Credential) string {
	if len(c.UserID) > 8 {
		return c.UserID[:8]
	}
	return c.UserID
}
