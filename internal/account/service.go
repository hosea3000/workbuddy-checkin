package account

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hosea3000/workbuddy-checkin/model"
	"github.com/hosea3000/workbuddy-checkin/store"
	"github.com/hosea3000/workbuddy-checkin/upstream/codebuddy"
)

// Client 是登录所需的上游能力（便于测试注入假实现）。
type Client interface {
	StartAuth(ctx context.Context) (*codebuddy.AuthState, error)
	PollToken(ctx context.Context, state string) (*codebuddy.TokenData, bool, error)
	PollAccount(ctx context.Context, state string, td *codebuddy.TokenData) (*codebuddy.Account, bool, error)
	PollAccounts(ctx context.Context, td *codebuddy.TokenData) ([]codebuddy.Account, error)
}

// PollInterval 是轮询上游的间隔。
const PollInterval = 5 * time.Second

// TTL 是登录会话的最长有效期。
const TTL = 10 * time.Minute

// Service 编排 OAuth 设备授权登录。
type Service struct {
	store  *store.Store
	client Client
	now    func() time.Time

	mu        sync.Mutex
	session   *session
	openURLFn func(string) error
}

type session struct {
	stage     string
	state     string
	authURL   string
	deadline  time.Time
	errorMsg  string
	accountID string
	canceled  bool
	tokenData *codebuddy.TokenData
}

// NewService 构造账号服务。
func NewService(st *store.Store, client Client, openURL func(string) error) *Service {
	return &Service{store: st, client: client, now: time.Now, openURLFn: openURL}
}

// Start 启动一次登录会话，返回授权链接；若已有进行中的会话则复用。
func (s *Service) Start(ctx context.Context) (model.LoginStart, error) {
	s.mu.Lock()
	if s.session != nil && !s.session.canceled && !s.session.expired(s.now()) &&
		(s.session.stage == "awaiting_login" || s.session.stage == "awaiting_account") {
		res := model.LoginStart{AuthURL: s.session.authURL, ExpiresIn: int(TTL.Seconds())}
		s.mu.Unlock()
		return res, nil
	}
	s.mu.Unlock()

	state, err := s.client.StartAuth(ctx)
	if err != nil {
		return model.LoginStart{}, humanizeAuthError(err)
	}
	s.mu.Lock()
	s.session = &session{
		stage:    "awaiting_login",
		state:    state.State,
		authURL:  state.AuthURL,
		deadline: s.now().Add(TTL),
	}
	s.mu.Unlock()
	log.Printf("[login] check state=%s: 授权链接 %s", state.State, state.AuthURL)

	if s.openURLFn != nil {
		_ = s.openURLFn(state.AuthURL)
	}
	return model.LoginStart{AuthURL: state.AuthURL, ExpiresIn: int(TTL.Seconds())}, nil
}

// Stage 推进一次轮询并返回当前状态。前端每 5s 调用，与 PRD F1 的上游轮询间隔一致；
// 每次调用都真实请求上游，无缓存层。
func (s *Service) Stage(ctx context.Context) model.LoginStatus {
	s.mu.Lock()
	sess := s.session
	s.mu.Unlock()
	if sess == nil {
		log.Printf("[login] poll state: no session (idle)")
		return model.LoginStatus{Stage: "idle"}
	}
	if sess.canceled {
		log.Printf("[login] poll state: canceled")
		return model.LoginStatus{Stage: "canceled"}
	}
	if sess.expired(s.now()) {
		log.Printf("[login] poll state: session expired (state=%s)", sess.state)
		s.clear()
		return model.LoginStatus{Stage: "failed", Error: "已过期，请重试"}
	}
	log.Printf("[login] poll state: stage=%s state=%s", sess.stage, sess.state)
	switch sess.stage {
	case "awaiting_login":
		return s.pollToken(ctx, sess)
	case "awaiting_account":
		return s.pollAccount(ctx, sess)
	default:
		return s.statusLocked(sess)
	}
}

func (s *Service) pollToken(ctx context.Context, sess *session) model.LoginStatus {
	log.Printf("[login] check state=%s: GET auth/token (等待用户完成浏览器登录)", sess.state)
	td, pending, err := s.client.PollToken(ctx, sess.state)
	if err != nil {
		// 认证服务暂时不可用属瞬态：保持会话继续轮询，不中断登录。
		if isTransientAuthError(err) {
			log.Printf("[login] check state=%s: 瞬态错误，继续等待: %v", sess.state, err)
			return model.LoginStatus{Stage: "awaiting_login"}
		}
		msg := humanizeAuthError(err).Error()
		log.Printf("[login] check state=%s: 失败: %v", sess.state, err)
		s.fail(sess, msg)
		return model.LoginStatus{Stage: "failed", Error: msg}
	}
	if pending {
		log.Printf("[login] check state=%s: 上游 code=11217 尚未登录，继续等待", sess.state)
		return model.LoginStatus{Stage: "awaiting_login"}
	}
	s.mu.Lock()
	sess.tokenData = td
	sess.stage = "awaiting_account"
	s.mu.Unlock()
	log.Printf("[login] check state=%s: ✅ 已登录！拿到 access_token（尾号 %s），进入账号信息阶段",
		sess.state, tokenTail(td.AccessToken))
	return model.LoginStatus{Stage: "awaiting_account"}
}

func (s *Service) pollAccount(ctx context.Context, sess *session) model.LoginStatus {
	s.mu.Lock()
	td := sess.tokenData
	s.mu.Unlock()

	log.Printf("[login] check state=%s: GET login/account (获取账号信息)", sess.state)
	acc, pending, err := s.client.PollAccount(ctx, sess.state, td)
	if err != nil {
		if isTransientAuthError(err) {
			log.Printf("[login] check state=%s: 账号服务瞬态错误，继续等待: %v", sess.state, err)
			return model.LoginStatus{Stage: "awaiting_account"}
		}
		msg := humanizeAuthError(err).Error()
		log.Printf("[login] check state=%s: 账号请求失败: %v", sess.state, err)
		s.fail(sess, msg)
		return model.LoginStatus{Stage: "failed", Error: msg}
	}
	if pending {
		log.Printf("[login] check state=%s: 上游 code=12151 账号信息准备中，继续等待", sess.state)
		return model.LoginStatus{Stage: "awaiting_account"}
	}

	// 第三段：账号列表仅用于确认账号已就绪，其结果一期不用；
	// 失败不阻断（PollAccount 已返回有效账号）。
	if _, err := s.client.PollAccounts(ctx, td); err != nil {
		log.Printf("[login] check state=%s: PollAccounts 失败（忽略，不阻断）: %v", sess.state, err)
	}

	log.Printf("[login] check state=%s: ✅ 账号信息就绪 uid=%s nickname=%q，组装凭证", sess.state, acc.UID, acc.Nickname)
	cred, err := s.assemble(acc, td)
	if err != nil {
		log.Printf("[login] check state=%s: 凭证保存失败: %v", sess.state, err)
		s.fail(sess, "凭证保存失败，请重新认证")
		return model.LoginStatus{Stage: "failed", Error: "凭证保存失败，请重新认证"}
	}

	s.mu.Lock()
	sess.stage = "done"
	sess.accountID = cred.ID
	s.mu.Unlock()
	view := ToView(cred, s.now())
	log.Printf("[login] check state=%s: ✅ 登录完成，凭证已入库 id=%s user=%s", sess.state, cred.ID, cred.UserID)
	return model.LoginStatus{Stage: "done", Done: true, Account: &view}
}

// tokenTail 返回令牌末 4 位用于日志（绝不打印完整令牌）。
func tokenTail(token string) string {
	if len(token) < 4 {
		return "****"
	}
	return "..." + token[len(token)-4:]
}

// assemble 组装凭证并做去重双查原位更新。
func (s *Service) assemble(acc *codebuddy.Account, td *codebuddy.TokenData) (model.Credential, error) {
	accountUID := acc.UID
	userID := oauthUserID(td.AccessToken, accountUID)

	// 去重：先按 uid_<account_uid>，再按 account_uid 兜底。
	existing, found := s.findExisting(userID, accountUID)

	id := newUUID()
	if found {
		id = existing.ID
	}

	cred := model.Credential{
		ID:               id,
		UserID:           userID,
		AccountUID:       accountUID,
		Status:           model.StatusActive,
		AccessToken:      td.AccessToken,
		RefreshToken:     td.RefreshToken,
		ExpiresAt:        expiresAt(td),
		RefreshExpiresAt: deref(td.RefreshExpiresAt),
		Domain:           td.Domain,
		EnterpriseID:     td.EnterpriseID,
		Nickname:         acc.Nickname,
	}
	// personal 账号不带企业信息。
	if acc.Type == "personal" {
		cred.EnterpriseID = ""
	} else if acc.EnterpriseID != "" {
		cred.EnterpriseID = acc.EnterpriseID
	}

	// JWT 身份补充昵称/邮箱。
	ident := applyIdentity(td.AccessToken)
	if ident.Nickname != "" {
		cred.Nickname = ident.Nickname
	}
	cred.PreferredUsername = ident.PreferredUsername
	cred.Email = ident.Email
	if cred.Nickname == "" && ident.PreferredUsername != "" {
		cred.Nickname = ident.PreferredUsername
	}
	if cred.Domain == "" {
		if domain, ent, ok := extractIssuerInfo(td.AccessToken); ok {
			cred.Domain = domain
			if cred.EnterpriseID == "" {
				cred.EnterpriseID = ent
			}
		}
	}
	if found {
		cred.CreatedAt = existing.CreatedAt
	}
	if err := s.store.SaveCredential(cred); err != nil {
		return model.Credential{}, err
	}
	return cred, nil
}

func (s *Service) findExisting(userID, accountUID string) (model.Credential, bool) {
	if c, ok := s.store.FindCredential(func(c model.Credential) bool { return c.UserID == userID }); ok {
		return c, true
	}
	if accountUID != "" {
		if c, ok := s.store.FindCredential(func(c model.Credential) bool { return c.AccountUID == accountUID }); ok {
			return c, true
		}
	}
	return model.Credential{}, false
}

// Cancel 取消当前登录会话。
func (s *Service) Cancel() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.session != nil {
		s.session.canceled = true
		s.session.stage = "canceled"
	}
}

func (s *Service) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.session = nil
}

func (s *Service) fail(sess *session, msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess.stage = "failed"
	sess.errorMsg = msg
}

func (s *Service) statusLocked(sess *session) model.LoginStatus {
	if sess.stage == "done" {
		if c, ok := s.store.GetCredential(sess.accountID); ok {
			v := ToView(c, s.now())
			return model.LoginStatus{Stage: "done", Done: true, Account: &v}
		}
	}
	if sess.stage == "failed" {
		return model.LoginStatus{Stage: "failed", Error: sess.errorMsg}
	}
	return model.LoginStatus{Stage: sess.stage}
}

func (sess *session) expired(now time.Time) bool { return now.After(sess.deadline) }

func expiresAt(td *codebuddy.TokenData) int64 {
	if td.ExpiresAt != nil {
		return *td.ExpiresAt
	}
	if td.ExpiresIn != nil {
		return time.Now().Unix() + *td.ExpiresIn
	}
	return 0
}

func deref(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

// newUUID 生成 RFC 4122 v4 UUID（crypto/rand，不引入 google/uuid）。
func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
