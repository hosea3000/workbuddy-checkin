package checkin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hosea3000/workbuddy-checkin/model"
	"github.com/hosea3000/workbuddy-checkin/store"
	"github.com/hosea3000/workbuddy-checkin/upstream/codebuddy"
)

type fakeClient struct {
	checkinFn func(ctx context.Context, c codebuddy.CredentialSnapshot) (bool, *int, string, *float64, error)
	quotaFn   func(ctx context.Context, c codebuddy.CredentialSnapshot) (float64, float64, error)
	refreshFn func(ctx context.Context, a, d, r string) (*codebuddy.TokenData, error)

	lastSnapshot codebuddy.CredentialSnapshot
}

func (f *fakeClient) Checkin(ctx context.Context, c codebuddy.CredentialSnapshot) (bool, *int, string, *float64, error) {
	f.lastSnapshot = c
	return f.checkinFn(ctx, c)
}
func (f *fakeClient) FetchQuotaPersonal(ctx context.Context, c codebuddy.CredentialSnapshot) (float64, float64, error) {
	if f.quotaFn == nil {
		return 0, 0, errors.New("no quota fn")
	}
	return f.quotaFn(ctx, c)
}
func (f *fakeClient) RefreshToken(ctx context.Context, a, d, r string) (*codebuddy.TokenData, error) {
	return f.refreshFn(ctx, a, d, r)
}

type fakeNotifier struct{ titles []string }

func (n *fakeNotifier) Notify(title, body string) { n.titles = append(n.titles, title) }

func newFixture(t *testing.T, c model.Credential, fc *fakeClient) (*Service, *fakeNotifier, *store.Store) {
	t.Helper()
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SaveCredential(c); err != nil {
		t.Fatal(err)
	}
	n := &fakeNotifier{}
	return NewService(st, fc, n), n, st
}

func TestPerformCheckinSuccess(t *testing.T) {
	credit := 5.0
	fc := &fakeClient{checkinFn: func(_ context.Context, _ codebuddy.CredentialSnapshot) (bool, *int, string, *float64, error) {
		return true, nil, "ok", &credit, nil
	}}
	svc, n, st := newFixture(t, model.Credential{ID: "1", UserID: "uid_a", Status: model.StatusActive, AccessToken: "tok"}, fc)

	res, err := svc.PerformCheckin(context.Background(), "1")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Success || res.Credit == nil || *res.Credit != 5 {
		t.Errorf("unexpected result: %+v", res)
	}
	got, _ := st.GetCredential("1")
	if !got.TodaySuccess || got.TodayAttempts != 1 || got.TodayDate != model.Today(time.Now()) {
		t.Errorf("credential not updated: %+v", got)
	}
	if len(n.titles) != 1 || n.titles[0] != "签到成功" {
		t.Errorf("expected success notification, got %v", n.titles)
	}
}

func TestPerformCheckinAlreadyCheckedIn(t *testing.T) {
	fc := &fakeClient{checkinFn: func(_ context.Context, _ codebuddy.CredentialSnapshot) (bool, *int, string, *float64, error) {
		return false, nil, "今日已签到", nil, nil
	}}
	svc, _, st := newFixture(t, model.Credential{ID: "1", UserID: "uid_a", Status: model.StatusActive}, fc)

	res, _ := svc.PerformCheckin(context.Background(), "1")
	if !res.Success {
		t.Errorf("「已签到」应判为成功: %+v", res)
	}
	got, _ := st.GetCredential("1")
	if !got.TodaySuccess {
		t.Error("today_success should be true")
	}
}

func TestPerformCheckinFailureNotifies(t *testing.T) {
	fc := &fakeClient{checkinFn: func(_ context.Context, _ codebuddy.CredentialSnapshot) (bool, *int, string, *float64, error) {
		return false, nil, "", nil, &codebuddy.UpstreamError{ErrType: codebuddy.ErrCategoryUpstream5xx, Message: "boom"}
	}}
	svc, n, st := newFixture(t, model.Credential{ID: "1", UserID: "uid_a", Status: model.StatusActive}, fc)

	res, _ := svc.PerformCheckin(context.Background(), "1")
	if res.Success {
		t.Error("should fail")
	}
	if len(n.titles) != 1 || n.titles[0] != "签到失败" {
		t.Errorf("expected failure notification, got %v", n.titles)
	}
	got, _ := st.GetCredential("1")
	if got.TodayAttempts != 1 || got.TodaySuccess {
		t.Errorf("attempt recorded wrong: %+v", got)
	}
}

func TestSnapshotMapping(t *testing.T) {
	fc := &fakeClient{checkinFn: func(_ context.Context, _ codebuddy.CredentialSnapshot) (bool, *int, string, *float64, error) {
		return true, nil, "ok", nil, nil
	}}
	cred := model.Credential{ID: "1", UserID: "uid_acc", AccountUID: "acc", Domain: "d.com", EnterpriseID: "e1", AccessToken: "tk"}
	svc, _, _ := newFixture(t, cred, fc)

	_, _ = svc.PerformCheckin(context.Background(), "1")
	s := fc.lastSnapshot
	if s.BearerToken != "tk" || s.AccountUID != "acc" || s.UserID != "uid_acc" || s.Domain != "d.com" || s.EnterpriseID != "e1" {
		t.Errorf("snapshot mapping wrong: %+v", s)
	}
}

func TestRefreshIfNeededOnExpiringToken(t *testing.T) {
	fc := &fakeClient{
		checkinFn: func(_ context.Context, _ codebuddy.CredentialSnapshot) (bool, *int, string, *float64, error) {
			return true, nil, "ok", nil, nil
		},
		refreshFn: func(_ context.Context, _, _, _ string) (*codebuddy.TokenData, error) {
			exp := time.Now().Unix() + 9000
			return &codebuddy.TokenData{AccessToken: "newtok", RefreshToken: "newref", ExpiresAt: &exp}, nil
		},
	}
	svc, _, st := newFixture(t, model.Credential{ID: "1", UserID: "uid_a", Status: model.StatusActive, AccessToken: "old", RefreshToken: "ref", ExpiresAt: time.Now().Unix() + 3600}, fc)

	_, _ = svc.PerformCheckin(context.Background(), "1")
	got, _ := st.GetCredential("1")
	if got.AccessToken != "newtok" || got.RefreshToken != "newref" {
		t.Errorf("token not refreshed: %+v", got)
	}
}

func TestRefreshUnauthorizedMarksRelogin(t *testing.T) {
	fc := &fakeClient{
		checkinFn: func(_ context.Context, _ codebuddy.CredentialSnapshot) (bool, *int, string, *float64, error) {
			return false, nil, "需重新登录", nil, nil
		},
		refreshFn: func(_ context.Context, _, _, _ string) (*codebuddy.TokenData, error) {
			return nil, &codebuddy.AuthError{Name: "unauthorized", Description: "rejected", HTTPStatus: 401}
		},
	}
	svc, n, st := newFixture(t, model.Credential{ID: "1", UserID: "uid_a", Status: model.StatusActive, AccessToken: "old", RefreshToken: "ref", ExpiresAt: time.Now().Unix() + 3600}, fc)

	_, _ = svc.PerformCheckin(context.Background(), "1")
	got, _ := st.GetCredential("1")
	if got.Status != model.StatusReloginRequired {
		t.Errorf("expected relogin_required, got %q", got.Status)
	}
	if len(n.titles) == 0 || n.titles[0] != "workbuddy-checkin" {
		t.Errorf("expected relogin notification, got %v", n.titles)
	}
}

func TestRefreshNetworkErrorKeepsActive(t *testing.T) {
	fc := &fakeClient{
		checkinFn: func(_ context.Context, _ codebuddy.CredentialSnapshot) (bool, *int, string, *float64, error) {
			return true, nil, "ok", nil, nil
		},
		refreshFn: func(_ context.Context, _, _, _ string) (*codebuddy.TokenData, error) {
			return nil, &codebuddy.AuthError{Name: "auth_unavailable", Description: "down", HTTPStatus: 503}
		},
	}
	svc, _, st := newFixture(t, model.Credential{ID: "1", UserID: "uid_a", Status: model.StatusActive, AccessToken: "old", RefreshToken: "ref", ExpiresAt: time.Now().Unix() + 3600}, fc)

	_, _ = svc.PerformCheckin(context.Background(), "1")
	got, _ := st.GetCredential("1")
	if got.Status != model.StatusActive {
		t.Errorf("non-unauthorized error must not set relogin_required, got %q", got.Status)
	}
}

func TestReloginRequiredSkipsUpstream(t *testing.T) {
	called := false
	fc := &fakeClient{checkinFn: func(_ context.Context, _ codebuddy.CredentialSnapshot) (bool, *int, string, *float64, error) {
		called = true
		return true, nil, "ok", nil, nil
	}}
	svc, _, _ := newFixture(t, model.Credential{ID: "1", UserID: "uid_a", Status: model.StatusReloginRequired}, fc)

	res, _ := svc.PerformCheckin(context.Background(), "1")
	if called {
		t.Error("must not call upstream for relogin_required account")
	}
	if res.Success {
		t.Error("should report failure")
	}
}

func TestQuotaFailureKeepsBalanceAndStatus(t *testing.T) {
	old := 100.0
	fc := &fakeClient{
		checkinFn: func(_ context.Context, _ codebuddy.CredentialSnapshot) (bool, *int, string, *float64, error) {
			return true, nil, "ok", nil, nil
		},
		quotaFn: func(_ context.Context, _ codebuddy.CredentialSnapshot) (float64, float64, error) {
			return 0, 0, errors.New("network down")
		},
	}
	svc, _, st := newFixture(t, model.Credential{ID: "1", UserID: "uid_a", Status: model.StatusActive, CreditBalance: &old}, fc)

	_, _ = svc.PerformCheckin(context.Background(), "1")
	got, _ := st.GetCredential("1")
	if got.CreditBalance == nil || *got.CreditBalance != 100 {
		t.Errorf("balance should be preserved, got %+v", got.CreditBalance)
	}
	if got.Status != model.StatusActive {
		t.Errorf("quota failure must not change status, got %q", got.Status)
	}
}
