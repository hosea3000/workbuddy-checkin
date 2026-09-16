package proxy

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hosea3000/workbuddy-checkin/model"
	"github.com/hosea3000/workbuddy-checkin/store"
	"github.com/hosea3000/workbuddy-checkin/upstream/codebuddy"
)

const secretToken = "SECRET-ACCESS-TOKEN-DO-NOT-LEAK"

// storeRefresher 模拟真实 EnsureFresh：从 store 读回凭证（而非返回零值）。
type storeRefresher struct{ st *store.Store }

func (r *storeRefresher) EnsureFresh(_ context.Context, id string) (model.Credential, error) {
	c, ok := r.st.GetCredential(id)
	if !ok {
		return model.Credential{}, errors.New("credential not found")
	}
	return c, nil
}

// newRealResolver 构造真实 store + CredentialResolver，用于端到端验证。
func newRealResolver(t *testing.T, activeID string) CredentialProvider {
	t.Helper()
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SaveCredential(model.Credential{
		ID: "c1", UserID: "uid_a", AccountUID: "a",
		AccessToken: secretToken, Status: model.StatusActive,
		ExpiresAt: time.Now().Unix() + 3600,
	}); err != nil {
		t.Fatal(err)
	}
	s := st.GetSettings()
	s.ActiveCredentialID = activeID
	if err := st.SaveSettings(s); err != nil {
		t.Fatal(err)
	}
	return NewCredentialResolver(st, &storeRefresher{st: st})
}

// TestProxyEndToEnd 启动真实监听并用真实 HTTP 客户端验证端到端行为
// （等价于任务 7.4 的 curl 验证，但可自动化、无需真实账号）。
func TestProxyEndToEnd(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v3/config":
			// 用一个不在兜底列表里的模型，确保断言的是真实上游结果
			_, _ = w.Write([]byte(`{"code":0,"data":{"models":["upstream-only-model","glm-5.2"]}}`))
		case "/v2/chat/completions":
			if got := r.Header.Get("Authorization"); got != "Bearer "+secretToken {
				t.Errorf("上游未收到正确的 Authorization 头")
			}
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, okStream)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer up.Close()

	svc := NewService(codebuddy.NewClient(up.URL, ""), newRealResolver(t, "c1"), &fakeRefresher{})
	port := freePort(t)
	if err := svc.Start(port); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer svc.Stop()
	base := fmt.Sprintf("http://127.0.0.1:%d", port)

	t.Run("models", func(t *testing.T) {
		body := getBody(t, base+"/v1/models")
		if !strings.Contains(body, "upstream-only-model") {
			t.Fatalf("模型列表应来自上游而非兜底: %s", body)
		}
		assertNoToken(t, body)
	})

	t.Run("chat 非流式", func(t *testing.T) {
		body := postBody(t, base+"/v1/chat/completions",
			`{"model":"glm-5.2","stream":false,"messages":[{"role":"user","content":"hi"}]}`)
		var got map[string]any
		if err := json.Unmarshal([]byte(body), &got); err != nil {
			t.Fatalf("响应非 JSON: %v（body=%s）", err, body)
		}
		choices := got["choices"].([]any)
		msg := choices[0].(map[string]any)["message"].(map[string]any)
		if msg["content"] != "Hello" {
			t.Fatalf("聚合内容不符: %#v", msg["content"])
		}
		assertNoToken(t, body)
	})

	t.Run("chat 流式", func(t *testing.T) {
		body := postBody(t, base+"/v1/chat/completions",
			`{"model":"glm-5.2","stream":true,"messages":[{"role":"user","content":"hi"}]}`)
		if !strings.HasSuffix(body, "data: [DONE]\n\n") {
			t.Fatalf("流未以 [DONE] 结束: %s", body)
		}
		if !strings.Contains(body, "chat.completion.chunk") {
			t.Fatalf("缺少 chunk 信封: %s", body)
		}
		assertNoToken(t, body)
	})

	t.Run("未设当前凭证返回 503", func(t *testing.T) {
		svc2 := NewService(codebuddy.NewClient(up.URL, ""), newRealResolver(t, ""), &fakeRefresher{})
		port2 := freePort(t)
		if err := svc2.Start(port2); err != nil {
			t.Fatalf("Start: %v", err)
		}
		defer svc2.Stop()

		resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/v1/chat/completions", port2),
			"application/json", strings.NewReader(`{"messages":[{"role":"user","content":"hi"}]}`))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("期望 503，得到 %d", resp.StatusCode)
		}
	})
}

// TestProxyOnlyLoopback 验证服务只接受本机回环连接。
func TestProxyOnlyLoopback(t *testing.T) {
	// 用本地假上游，避免测试触达真实网络
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"code":0,"data":{"models":["m"]}}`))
	}))
	defer up.Close()

	svc := NewService(codebuddy.NewClient(up.URL, ""), newRealResolver(t, "c1"), &fakeRefresher{})
	port := freePort(t)
	if err := svc.Start(port); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer svc.Stop()

	// 绑定地址必须精确为回环
	if got := listenAddr(port); got != fmt.Sprintf("127.0.0.1:%d", port) {
		t.Fatalf("监听地址应为回环，得到 %q", got)
	}
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/v1/models", port))
	if err != nil {
		t.Fatalf("回环访问应成功: %v", err)
	}
	_ = resp.Body.Close()
}

func getBody(t *testing.T, url string) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s 状态码 %d", url, resp.StatusCode)
	}
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}

func postBody(t *testing.T, url, payload string) string {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST %s 状态码 %d（body=%s）", url, resp.StatusCode, b)
	}
	reader := bufio.NewReader(resp.Body)
	b, _ := io.ReadAll(reader)
	return string(b)
}

// assertNoToken 断言响应中不出现完整令牌（令牌脱敏约束）。
func assertNoToken(t *testing.T, body string) {
	t.Helper()
	if strings.Contains(body, secretToken) {
		t.Fatal("响应泄漏了完整令牌")
	}
}
