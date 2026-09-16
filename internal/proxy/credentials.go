package proxy

import (
	"context"
	"log"

	"github.com/hosea3000/workbuddy-checkin/model"
	"github.com/hosea3000/workbuddy-checkin/store"
)

// Refresher 按需续期（由 internal/checkin.Service.EnsureFresh 提供）。
type Refresher interface {
	EnsureFresh(ctx context.Context, id string) (model.Credential, error)
}

// CredentialResolver 从设置读取「当前凭证」并按需续期（设计决策 3）。
// 无凭证池、无轮换、不自动降级到其他账号。
type CredentialResolver struct {
	store *store.Store
	fresh Refresher
}

func NewCredentialResolver(st *store.Store, fresh Refresher) *CredentialResolver {
	return &CredentialResolver{store: st, fresh: fresh}
}

// Active 返回当前凭证；不可用时返回 *HTTPError。
func (r *CredentialResolver) Active(ctx context.Context) (model.Credential, error) {
	id := r.store.GetSettings().ActiveCredentialID
	if id == "" {
		return model.Credential{}, errUnavailable("未设置当前凭证：请在账号列表中选择一个账号设为当前凭证")
	}
	c, ok := r.store.GetCredential(id)
	if !ok {
		return model.Credential{}, errUnavailable("当前凭证不存在：请重新设置当前凭证")
	}
	if c.IsReloginRequired() {
		return model.Credential{}, errCredential("当前凭证已被上游拒绝，请重新登录该账号")
	}
	fresh, err := r.fresh.EnsureFresh(ctx, id)
	if err != nil {
		log.Printf("[proxy] credential refresh failed: account=%s err=%v", shortID(c), err)
		return model.Credential{}, errCredential("当前凭证续期失败，请重新登录该账号")
	}
	return fresh, nil
}

// shortID 返回账号短码（日志用，不含令牌）。
func shortID(c model.Credential) string {
	if c.UserID != "" {
		if len(c.UserID) > 12 {
			return c.UserID[:12]
		}
		return c.UserID
	}
	return c.ID
}
