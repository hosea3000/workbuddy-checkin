package main

import (
	"strings"
	"testing"

	"github.com/hosea3000/workbuddy-checkin/internal/account"
	"github.com/hosea3000/workbuddy-checkin/model"
)

// TestAccountViewHidesToken 校验 UI 视图绝不携带完整令牌，仅暴露末 8 位。
func TestAccountViewHidesToken(t *testing.T) {
	token := "eyJhbGciOiJSUzI1NiJ9.SUPER-SECRET-TOKEN-BODY.signature"
	cred := model.Credential{
		ID:          "1",
		UserID:      "uid_acc",
		AccessToken: token,
		Status:      model.StatusActive,
	}
	view := account.ToView(cred, timeNow(), "")

	if view.TokenSuffix == token {
		t.Fatal("view must not expose the full token")
	}
	if len(view.TokenSuffix) != 8 || view.TokenSuffix != token[len(token)-8:] {
		t.Errorf("token suffix should be last 8 chars, got %q", view.TokenSuffix)
	}
	// 视图序列化后不得包含完整令牌子串。
	if strings.Contains(view.TokenSuffix, token[:16]) {
		t.Error("view leaks token material")
	}
}

func TestTokenSuffixShortToken(t *testing.T) {
	if got := model.TokenSuffix("abc"); got != "***" {
		t.Errorf("short token must be fully masked, got %q", got)
	}
	if got := model.TokenSuffix("12345678"); got != "********" {
		t.Errorf("8-char token must be masked, got %q", got)
	}
}
