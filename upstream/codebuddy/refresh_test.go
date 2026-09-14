package codebuddy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientDefaultEndpoint(t *testing.T) {
	if got := NewClient("", "").Endpoint; got != defaultEndpoint {
		t.Errorf("empty endpoint must fall back to default, got %q", got)
	}
	if got := NewClient("https://example.com/", "").Endpoint; got != "https://example.com" {
		t.Errorf("trailing slash should be trimmed, got %q", got)
	}
}

// TestRefreshTokenUnauthorizedPropagates 回归 work2api 的错误分类 bug：
// 上游 401/403 必须归一到 unauthorized（置 relogin_required 依赖此错误名），
// 而非被 RefreshToken 包成 refresh_failed。
func TestRefreshTokenUnauthorizedPropagates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":401,"msg":"invalid token"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "2.107.0")
	_, err := c.RefreshToken(context.Background(), "access", "copilot.tencent.com", "refresh")
	if err == nil {
		t.Fatal("expected error")
	}
	var ae *AuthError
	if !errors.As(err, &ae) {
		t.Fatalf("expected *AuthError, got %T", err)
	}
	if ae.Name != "unauthorized" {
		t.Errorf("expected unauthorized, got %q (desc=%q)", ae.Name, ae.Description)
	}
}
