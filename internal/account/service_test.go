package account

import (
	"context"
	"testing"
	"time"

	"github.com/hosea3000/workbuddy-checkin/model"
	"github.com/hosea3000/workbuddy-checkin/store"
	"github.com/hosea3000/workbuddy-checkin/upstream/codebuddy"
)

type fakeClient struct {
	startFn    func(ctx context.Context) (*codebuddy.AuthState, error)
	tokenFn    func(ctx context.Context, state string) (*codebuddy.TokenData, bool, error)
	accountFn  func(ctx context.Context, state string, td *codebuddy.TokenData) (*codebuddy.Account, bool, error)
	accountsFn func(ctx context.Context, td *codebuddy.TokenData) ([]codebuddy.Account, error)
}

func (f *fakeClient) StartAuth(ctx context.Context) (*codebuddy.AuthState, error) {
	return f.startFn(ctx)
}
func (f *fakeClient) PollToken(ctx context.Context, state string) (*codebuddy.TokenData, bool, error) {
	return f.tokenFn(ctx, state)
}
func (f *fakeClient) PollAccount(ctx context.Context, state string, td *codebuddy.TokenData) (*codebuddy.Account, bool, error) {
	return f.accountFn(ctx, state, td)
}
func (f *fakeClient) PollAccounts(ctx context.Context, td *codebuddy.TokenData) ([]codebuddy.Account, error) {
	if f.accountsFn == nil {
		return nil, nil
	}
	return f.accountsFn(ctx, td)
}

func validToken() *codebuddy.TokenData {
	exp := int64(4102444800)
	return &codebuddy.TokenData{AccessToken: "access-token-abcdefgh", RefreshToken: "ref", ExpiresAt: &exp}
}

func newService(t *testing.T, fc *fakeClient) (*Service, *store.Store) {
	t.Helper()
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return NewService(st, fc, nil), st
}

func TestLoginSuccessAndDedup(t *testing.T) {
	fc := &fakeClient{
		startFn: func(context.Context) (*codebuddy.AuthState, error) {
			return &codebuddy.AuthState{State: "st", AuthURL: "https://copilot.tencent.com/authorize?state=st"}, nil
		},
		tokenFn: func(context.Context, string) (*codebuddy.TokenData, bool, error) { return validToken(), false, nil },
		accountFn: func(context.Context, string, *codebuddy.TokenData) (*codebuddy.Account, bool, error) {
			return &codebuddy.Account{UID: "acc1", Type: "personal", Nickname: "Hosea"}, false, nil
		},
	}
	svc, st := newService(t, fc)

	if _, err := svc.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	status := pollUntilDone(t, svc)
	if !status.Done {
		t.Fatalf("expected done, got %+v", status)
	}
	if got := st.ListCredentials(); len(got) != 1 || got[0].UserID != "uid_acc1" || got[0].Status != model.StatusActive {
		t.Fatalf("credential not saved: %+v", got)
	}

	// 再次登录同一账号 → 原位更新，不重复
	svc2 := NewService(st, fc, nil)
	if _, err := svc2.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	pollUntilDone(t, svc2)
	if got := st.ListCredentials(); len(got) != 1 {
		t.Fatalf("duplicate login must not create new card, got %d", len(got))
	}
}

// pollUntilDone 最多推进若干次轮询直到 done/failed。
func pollUntilDone(t *testing.T, svc *Service) model.LoginStatus {
	t.Helper()
	for i := 0; i < 5; i++ {
		s := svc.Stage(context.Background())
		if s.Done || s.Stage == "failed" || s.Stage == "canceled" {
			return s
		}
	}
	return svc.Stage(context.Background())
}

func TestLoginPendingStages(t *testing.T) {
	calls := 0
	fc := &fakeClient{
		startFn: func(context.Context) (*codebuddy.AuthState, error) {
			return &codebuddy.AuthState{State: "st", AuthURL: "https://x.com/a"}, nil
		},
		tokenFn: func(context.Context, string) (*codebuddy.TokenData, bool, error) {
			if calls == 0 {
				calls++
				return nil, true, nil // code=11217 pending
			}
			return validToken(), false, nil
		},
		accountFn: func(context.Context, string, *codebuddy.TokenData) (*codebuddy.Account, bool, error) {
			return nil, true, nil // code=12151 pending
		},
	}
	svc, _ := newService(t, fc)
	_, _ = svc.Start(context.Background())

	if s := svc.Stage(context.Background()); s.Stage != "awaiting_login" {
		t.Errorf("expected awaiting_login, got %q", s.Stage)
	}
	if s := svc.Stage(context.Background()); s.Stage != "awaiting_account" {
		t.Errorf("expected awaiting_account, got %q", s.Stage)
	}
}

func TestLoginBusinessErrorHumanized(t *testing.T) {
	fc := &fakeClient{
		startFn: func(context.Context) (*codebuddy.AuthState, error) {
			return &codebuddy.AuthState{State: "st", AuthURL: "https://x.com/a"}, nil
		},
		tokenFn: func(context.Context, string) (*codebuddy.TokenData, bool, error) {
			return nil, false, codebuddy.BusinessAuthError(12005)
		},
		accountFn: func(context.Context, string, *codebuddy.TokenData) (*codebuddy.Account, bool, error) {
			return nil, false, nil
		},
	}
	svc, _ := newService(t, fc)
	_, _ = svc.Start(context.Background())

	s := svc.Stage(context.Background())
	if s.Stage != "failed" || s.Error != "企业许可证没有可用席位" {
		t.Errorf("expected humanized business error, got %+v", s)
	}
}

func TestCancelLogin(t *testing.T) {
	fc := &fakeClient{
		startFn: func(context.Context) (*codebuddy.AuthState, error) {
			return &codebuddy.AuthState{State: "st", AuthURL: "https://x.com/a"}, nil
		},
		tokenFn: func(context.Context, string) (*codebuddy.TokenData, bool, error) {
			return nil, true, nil
		},
		accountFn: func(context.Context, string, *codebuddy.TokenData) (*codebuddy.Account, bool, error) {
			return nil, true, nil
		},
	}
	svc, _ := newService(t, fc)
	_, _ = svc.Start(context.Background())
	svc.Cancel()
	if s := svc.Stage(context.Background()); s.Stage != "canceled" {
		t.Errorf("expected canceled, got %q", s.Stage)
	}
}

func TestLoginTTLExpiry(t *testing.T) {
	fc := &fakeClient{
		startFn: func(context.Context) (*codebuddy.AuthState, error) {
			return &codebuddy.AuthState{State: "st", AuthURL: "https://x.com/a"}, nil
		},
		tokenFn: func(context.Context, string) (*codebuddy.TokenData, bool, error) {
			return nil, true, nil
		},
		accountFn: func(context.Context, string, *codebuddy.TokenData) (*codebuddy.Account, bool, error) {
			return nil, true, nil
		},
	}
	svc, _ := newService(t, fc)
	base := time.Now()
	svc.now = func() time.Time { return base }
	_, _ = svc.Start(context.Background())

	svc.now = func() time.Time { return base.Add(TTL + time.Second) }
	s := svc.Stage(context.Background())
	if s.Stage != "failed" || s.Error != "已过期，请重试" {
		t.Errorf("expected TTL expiry failure, got %+v", s)
	}
}

func TestLoginStatusIdleWhenNoSession(t *testing.T) {
	svc, _ := newService(t, &fakeClient{})
	if s := svc.Stage(context.Background()); s.Stage != "idle" {
		t.Errorf("expected idle, got %q", s.Stage)
	}
}

func TestTransientAuthErrorKeepsPolling(t *testing.T) {
	calls := 0
	fc := &fakeClient{
		startFn: func(context.Context) (*codebuddy.AuthState, error) {
			return &codebuddy.AuthState{State: "st", AuthURL: "https://x.com/a"}, nil
		},
		tokenFn: func(context.Context, string) (*codebuddy.TokenData, bool, error) {
			calls++
			if calls == 1 {
				return nil, false, &codebuddy.AuthError{Name: "auth_unavailable", Description: "认证服务暂时不可用"}
			}
			return validToken(), false, nil
		},
		accountFn: func(context.Context, string, *codebuddy.TokenData) (*codebuddy.Account, bool, error) {
			return &codebuddy.Account{UID: "acc1", Type: "personal"}, false, nil
		},
	}
	svc, _ := newService(t, fc)
	_, _ = svc.Start(context.Background())

	// 第一次轮询遇瞬态错误 → 保持 awaiting_login（不失败）
	s := svc.Stage(context.Background())
	if s.Stage != "awaiting_login" || s.Error != "" {
		t.Fatalf("transient error must not fail login, got %+v", s)
	}
	// 续轮询成功 → 进入 awaiting_account
	if s := svc.Stage(context.Background()); s.Stage != "awaiting_account" {
		t.Fatalf("expected recovery to awaiting_account, got %+v", s)
	}
}
