package account

import (
	"context"
	"testing"

	"github.com/hosea3000/workbuddy-checkin/model"
	"github.com/hosea3000/workbuddy-checkin/upstream/codebuddy"
)

func loginClient() *fakeClient {
	return &fakeClient{
		startFn: func(context.Context) (*codebuddy.AuthState, error) {
			return &codebuddy.AuthState{State: "st", AuthURL: "https://copilot.tencent.com/authorize?state=st"}, nil
		},
		tokenFn: func(context.Context, string) (*codebuddy.TokenData, bool, error) { return validToken(), false, nil },
		accountFn: func(context.Context, string, *codebuddy.TokenData) (*codebuddy.Account, bool, error) {
			return &codebuddy.Account{UID: "acc1", Type: "personal", Nickname: "Hosea"}, false, nil
		},
	}
}

func TestFirstLoginBecomesActiveCredential(t *testing.T) {
	svc, st := newService(t, loginClient())
	if _, err := svc.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if status := pollUntilDone(t, svc); !status.Done {
		t.Fatalf("登录未完成: %+v", status)
	}
	creds := st.ListCredentials()
	if len(creds) != 1 {
		t.Fatalf("期望 1 个账号，得到 %d", len(creds))
	}
	if got := st.GetSettings().ActiveCredentialID; got != creds[0].ID {
		t.Fatalf("首个账号应自动成为当前凭证（%s），得到 %q", creds[0].ID, got)
	}
}

func TestLoginDoesNotOverrideExistingActiveCredential(t *testing.T) {
	svc, st := newService(t, loginClient())
	// 预置一个已存在的当前凭证
	_ = st.SaveCredential(model.Credential{ID: "existing", UserID: "uid_other", Status: model.StatusActive})
	settings := st.GetSettings()
	settings.ActiveCredentialID = "existing"
	if err := st.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	pollUntilDone(t, svc)

	if got := st.GetSettings().ActiveCredentialID; got != "existing" {
		t.Fatalf("已有当前凭证不应被新登录覆盖，得到 %q", got)
	}
}
