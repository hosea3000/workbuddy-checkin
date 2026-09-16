package checkin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hosea3000/workbuddy-checkin/model"
	"github.com/hosea3000/workbuddy-checkin/upstream/codebuddy"
)

// countingRefresh 返回一个记录调用次数、并转发到 inner 的 refreshFn。
func countingRefresh(calls *int, inner func(context.Context, string, string, string) (*codebuddy.TokenData, error)) func(context.Context, string, string, string) (*codebuddy.TokenData, error) {
	return func(ctx context.Context, a, d, r string) (*codebuddy.TokenData, error) {
		*calls++
		if inner == nil {
			return nil, errors.New("refresh should not be called")
		}
		return inner(ctx, a, d, r)
	}
}

func TestEnsureFreshNotStaleSkipsUpstream(t *testing.T) {
	now := time.Now().Unix()
	calls := 0
	fc := &fakeClient{refreshFn: countingRefresh(&calls, nil)}
	svc, _, _ := newFixture(t, model.Credential{
		ID: "1", UserID: "uid_a", Status: model.StatusActive,
		AccessToken: "old", RefreshToken: "ref", ExpiresAt: now + 3600,
	}, fc)
	svc.now = func() time.Time { return time.Unix(now, 0) }

	got, err := svc.EnsureFresh(context.Background(), "1")
	if err != nil {
		t.Fatalf("EnsureFresh: %v", err)
	}
	if calls != 0 {
		t.Fatalf("未临期不应请求上游，调用 %d 次", calls)
	}
	if got.AccessToken != "old" {
		t.Fatalf("应返回原凭证，得到 %+v", got)
	}
}

func TestEnsureFreshWithinMarginRefreshes(t *testing.T) {
	now := time.Now().Unix()
	calls := 0
	fc := &fakeClient{refreshFn: countingRefresh(&calls, func(_ context.Context, _, _, _ string) (*codebuddy.TokenData, error) {
		exp := now + 7200
		return &codebuddy.TokenData{AccessToken: "newtok", ExpiresAt: &exp}, nil
	})}
	svc, _, _ := newFixture(t, model.Credential{
		ID: "1", UserID: "uid_a", Status: model.StatusActive,
		AccessToken: "old", RefreshToken: "ref", ExpiresAt: now + 30,
	}, fc)
	svc.now = func() time.Time { return time.Unix(now, 0) }

	got, err := svc.EnsureFresh(context.Background(), "1")
	if err != nil {
		t.Fatalf("EnsureFresh: %v", err)
	}
	if calls != 1 || got.AccessToken != "newtok" {
		t.Fatalf("余量内应续期一次，calls=%d got=%+v", calls, got)
	}
}

func TestEnsureFreshWithoutRefreshTokenSkipsUpstream(t *testing.T) {
	now := time.Now().Unix()
	calls := 0
	fc := &fakeClient{refreshFn: countingRefresh(&calls, nil)}
	svc, _, _ := newFixture(t, model.Credential{
		ID: "1", UserID: "uid_a", Status: model.StatusActive,
		AccessToken: "old", ExpiresAt: now - 10,
	}, fc)
	svc.now = func() time.Time { return time.Unix(now, 0) }

	if _, err := svc.EnsureFresh(context.Background(), "1"); err != nil {
		t.Fatalf("EnsureFresh: %v", err)
	}
	if calls != 0 {
		t.Fatalf("无 refresh_token 不应请求上游，调用 %d 次", calls)
	}
}

func TestEnsureFreshZeroExpirySkipsUpstream(t *testing.T) {
	calls := 0
	fc := &fakeClient{refreshFn: countingRefresh(&calls, nil)}
	svc, _, _ := newFixture(t, model.Credential{
		ID: "1", UserID: "uid_a", Status: model.StatusActive,
		AccessToken: "old", RefreshToken: "ref",
	}, fc)

	if _, err := svc.EnsureFresh(context.Background(), "1"); err != nil {
		t.Fatalf("EnsureFresh: %v", err)
	}
	if calls != 0 {
		t.Fatalf("有效期未知时不应请求上游，调用 %d 次", calls)
	}
}

func TestEnsureFreshRefreshesExpiredToken(t *testing.T) {
	now := time.Now().Unix()
	calls := 0
	fc := &fakeClient{refreshFn: countingRefresh(&calls, func(_ context.Context, a, _, r string) (*codebuddy.TokenData, error) {
		if a != "old" || r != "ref" {
			t.Fatalf("传入的令牌不符: access=%q refresh=%q", a, r)
		}
		exp := now + 7200
		return &codebuddy.TokenData{AccessToken: "newtok", RefreshToken: "newref", ExpiresAt: &exp}, nil
	})}
	svc, n, st := newFixture(t, model.Credential{
		ID: "1", UserID: "uid_a", Status: model.StatusActive,
		AccessToken: "old", RefreshToken: "ref", ExpiresAt: now - 5,
	}, fc)
	svc.now = func() time.Time { return time.Unix(now, 0) }

	got, err := svc.EnsureFresh(context.Background(), "1")
	if err != nil {
		t.Fatalf("EnsureFresh: %v", err)
	}
	if got.AccessToken != "newtok" || got.RefreshToken != "newref" || got.ExpiresAt != now+7200 {
		t.Fatalf("返回的凭证未更新: %+v", got)
	}
	saved, _ := st.GetCredential("1")
	if saved.AccessToken != "newtok" || saved.RefreshToken != "newref" {
		t.Fatalf("未回写存储: %+v", saved)
	}
	if len(n.titles) != 0 {
		t.Fatalf("静默续期不应发通知，得到 %v", n.titles)
	}
}

func TestEnsureFreshUnauthorizedReturnsErrorWithoutSideEffects(t *testing.T) {
	now := time.Now().Unix()
	fc := &fakeClient{refreshFn: func(_ context.Context, _, _, _ string) (*codebuddy.TokenData, error) {
		return nil, &codebuddy.AuthError{Name: "unauthorized", Description: "rejected", HTTPStatus: 401}
	}}
	svc, n, st := newFixture(t, model.Credential{
		ID: "1", UserID: "uid_a", Status: model.StatusActive,
		AccessToken: "old", RefreshToken: "ref", ExpiresAt: now - 5,
	}, fc)
	svc.now = func() time.Time { return time.Unix(now, 0) }

	if _, err := svc.EnsureFresh(context.Background(), "1"); err == nil {
		t.Fatal("续期失败应返回错误")
	}
	got, _ := st.GetCredential("1")
	if got.Status != model.StatusActive {
		t.Fatalf("静默续期失败不得置 relogin_required，得到 %q", got.Status)
	}
	if len(n.titles) != 0 {
		t.Fatalf("静默续期失败不应发通知，得到 %v", n.titles)
	}
}

func TestEnsureFreshUnknownAccount(t *testing.T) {
	fc := &fakeClient{refreshFn: countingRefresh(new(int), nil)}
	svc, _, _ := newFixture(t, model.Credential{ID: "1", Status: model.StatusActive}, fc)
	if _, err := svc.EnsureFresh(context.Background(), "missing"); !errors.Is(err, ErrAccountNotFound) {
		t.Fatalf("期望 ErrAccountNotFound，得到 %v", err)
	}
}
