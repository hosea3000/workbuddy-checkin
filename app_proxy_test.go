package main

import (
	"testing"

	"github.com/hosea3000/workbuddy-checkin/model"
	"github.com/hosea3000/workbuddy-checkin/store"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return &App{store: st}
}

func TestSaveSettingsRejectsInvalidProxyPort(t *testing.T) {
	a := newTestApp(t)
	for _, port := range []int{80, 1023, 70000} {
		err := a.SaveSettings(model.Settings{ProxyPort: port})
		if err == nil {
			t.Fatalf("端口 %d 应被拒绝", port)
		}
	}
	if got := a.store.GetSettings().ProxyPort; got != model.DefaultProxyPort {
		t.Fatalf("非法端口不应被持久化，得到 %d", got)
	}
}

func TestSaveSettingsNormalizesZeroPort(t *testing.T) {
	a := newTestApp(t)
	if err := a.SaveSettings(model.Settings{ProxyPort: 0}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	if got := a.store.GetSettings().ProxyPort; got != model.DefaultProxyPort {
		t.Fatalf("零端口应回落到默认值 %d，得到 %d", model.DefaultProxyPort, got)
	}
}

func TestSaveSettingsPersistsProxyFields(t *testing.T) {
	a := newTestApp(t)
	if err := a.SaveSettings(model.Settings{ProxyEnabled: false, ProxyPort: 19000}); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	got := a.store.GetSettings()
	if got.ProxyPort != 19000 || got.ProxyEnabled {
		t.Fatalf("设置未持久化: %#v", got)
	}
}

func TestProxyStatusWithoutService(t *testing.T) {
	a := newTestApp(t)
	st := a.ProxyStatus()
	if st.Running || st.Port != 0 || st.Error != "" {
		t.Fatalf("无服务时应返回零值状态，得到 %#v", st)
	}
}

func TestSetActiveCredential(t *testing.T) {
	a := newTestApp(t)
	if err := a.store.SaveCredential(model.Credential{ID: "a", Status: model.StatusActive}); err != nil {
		t.Fatal(err)
	}
	if err := a.store.SaveCredential(model.Credential{ID: "b", Status: model.StatusActive}); err != nil {
		t.Fatal(err)
	}

	if err := a.SetActiveCredential("b"); err != nil {
		t.Fatalf("SetActiveCredential: %v", err)
	}
	if got := a.store.GetSettings().ActiveCredentialID; got != "b" {
		t.Fatalf("当前凭证应为 b，得到 %q", got)
	}
	// 幂等
	if err := a.SetActiveCredential("b"); err != nil {
		t.Fatalf("重复设置应幂等: %v", err)
	}
	if err := a.SetActiveCredential("missing"); err == nil {
		t.Fatal("不存在的账号应返回错误")
	}
	if got := a.store.GetSettings().ActiveCredentialID; got != "b" {
		t.Fatalf("失败不应改变当前凭证，得到 %q", got)
	}
}

func TestDeleteActiveCredentialReassignsToFirstActive(t *testing.T) {
	a := newTestApp(t)
	_ = a.store.SaveCredential(model.Credential{ID: "a", Status: model.StatusActive})
	_ = a.store.SaveCredential(model.Credential{ID: "b", Status: model.StatusReloginRequired})
	_ = a.store.SaveCredential(model.Credential{ID: "c", Status: model.StatusActive})
	if err := a.SetActiveCredential("a"); err != nil {
		t.Fatal(err)
	}

	if err := a.DeleteAccount("a"); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	// b 是 relogin_required，应跳过；顺位到首个 active（c）
	if got := a.store.GetSettings().ActiveCredentialID; got != "c" {
		t.Fatalf("应顺位到首个 active 账号 c，得到 %q", got)
	}
}

func TestDeleteLastActiveCredentialClearsActive(t *testing.T) {
	a := newTestApp(t)
	_ = a.store.SaveCredential(model.Credential{ID: "a", Status: model.StatusActive})
	_ = a.SetActiveCredential("a")

	if err := a.DeleteAccount("a"); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	if got := a.store.GetSettings().ActiveCredentialID; got != "" {
		t.Fatalf("无剩余账号时当前凭证应置空，得到 %q", got)
	}
}

func TestDeleteOtherCredentialKeepsActive(t *testing.T) {
	a := newTestApp(t)
	_ = a.store.SaveCredential(model.Credential{ID: "a", Status: model.StatusActive})
	_ = a.store.SaveCredential(model.Credential{ID: "b", Status: model.StatusActive})
	_ = a.SetActiveCredential("a")

	if err := a.DeleteAccount("b"); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	if got := a.store.GetSettings().ActiveCredentialID; got != "a" {
		t.Fatalf("删除非当前凭证不应改变当前凭证，得到 %q", got)
	}
}

func TestListAccountsMarksActiveCredential(t *testing.T) {
	a := newTestApp(t)
	_ = a.store.SaveCredential(model.Credential{ID: "a", UserID: "uid_a", Status: model.StatusActive})
	_ = a.store.SaveCredential(model.Credential{ID: "b", UserID: "uid_b", Status: model.StatusActive})
	_ = a.SetActiveCredential("b")

	views := a.ListAccounts()
	if len(views) != 2 {
		t.Fatalf("期望 2 个账号，得到 %d", len(views))
	}
	for _, v := range views {
		want := v.ID == "b"
		if v.IsActive != want {
			t.Fatalf("账号 %s 的 isActive 应为 %v，得到 %v", v.ID, want, v.IsActive)
		}
	}
}
