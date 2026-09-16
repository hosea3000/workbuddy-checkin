package proxy

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hosea3000/workbuddy-checkin/model"
	"github.com/hosea3000/workbuddy-checkin/upstream/codebuddy"
)

type fakeRefresher struct {
	cred model.Credential
	err  error
}

func (f *fakeRefresher) EnsureFresh(context.Context, string) (model.Credential, error) {
	return f.cred, f.err
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func TestValidatePort(t *testing.T) {
	for _, port := range []int{0, 80, 1023, 65536, -1} {
		if err := ValidatePort(port); err == nil {
			t.Fatalf("端口 %d 应被拒绝", port)
		}
	}
	for _, port := range []int{1024, 18080, 65535} {
		if err := ValidatePort(port); err != nil {
			t.Fatalf("端口 %d 应被接受，得到 %v", port, err)
		}
	}
}

func TestListenAddrBindsLoopbackOnly(t *testing.T) {
	if got := listenAddr(18080); got != "127.0.0.1:18080" {
		t.Fatalf("必须显式绑定回环地址，得到 %q", got)
	}
	if strings.HasPrefix(listenAddr(18080), ":") {
		t.Fatal("不得使用 \":port\"（会绑定所有网卡）")
	}
}

func TestServiceLifecycle(t *testing.T) {
	port := freePort(t)
	svc := NewService(codebuddy.NewClient("http://127.0.0.1:1", ""), &fakeCreds{cred: testCred()}, &fakeRefresher{})

	if st := svc.Status(); st.Running {
		t.Fatalf("初始状态不应运行: %#v", st)
	}
	if err := svc.Start(port); err != nil {
		t.Fatalf("Start: %v", err)
	}
	st := svc.Status()
	if !st.Running || st.Port != port || st.Error != "" {
		t.Fatalf("启动后状态不符: %#v", st)
	}
	svc.Stop()
	if st := svc.Status(); st.Running {
		t.Fatalf("停止后仍报告运行: %#v", st)
	}
	svc.Stop() // 幂等
}

func TestServiceStartRejectsInvalidPort(t *testing.T) {
	svc := NewService(codebuddy.NewClient("", ""), &fakeCreds{cred: testCred()}, &fakeRefresher{})
	if err := svc.Start(80); err == nil {
		t.Fatal("特权端口应被拒绝")
	}
	if st := svc.Status(); st.Running {
		t.Fatal("拒绝后不应处于运行状态")
	}
}

func TestServiceStartPortInUse(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	svc := NewService(codebuddy.NewClient("", ""), &fakeCreds{cred: testCred()}, &fakeRefresher{})
	if err := svc.Start(port); err == nil {
		t.Fatal("端口被占用时应返回错误")
	}
	st := svc.Status()
	if st.Running {
		t.Fatalf("绑定失败不应报告运行: %#v", st)
	}
	if st.Error == "" {
		t.Fatal("绑定失败应记录可读错误")
	}
}

func TestHandleModels(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/config" {
			t.Errorf("path=%s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"code":0,"data":{"models":["upstream-a","glm-5.2"]}}`))
	}))
	defer up.Close()

	svc := NewService(codebuddy.NewClient(up.URL, ""), &fakeCreds{cred: testCred()}, &fakeRefresher{})
	rec := httptest.NewRecorder()
	svc.handleModels(rec, httptest.NewRequest(http.MethodGet, "/v1/models", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("状态码 %d", rec.Code)
	}
	var got struct {
		Object string           `json:"object"`
		Data   []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("响应非 JSON: %v", err)
	}
	if got.Object != "list" || len(got.Data) == 0 {
		t.Fatalf("列表结构不符: %#v", got)
	}
	ids := map[string]bool{}
	for _, m := range got.Data {
		if m["object"] != "model" {
			t.Fatalf("模型项缺少 object=model: %#v", m)
		}
		ids[m["id"].(string)] = true
	}
	if !ids["upstream-a"] || !ids["glm-5.2"] {
		t.Fatalf("模型列表不符: %#v", ids)
	}
}

func TestHandleModelsFallsBackWhenUpstreamFails(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer up.Close()

	svc := NewService(codebuddy.NewClient(up.URL, ""), &fakeCreds{cred: testCred()}, &fakeRefresher{})
	rec := httptest.NewRecorder()
	svc.handleModels(rec, httptest.NewRequest(http.MethodGet, "/v1/models", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("上游失败时应回退而非报错，状态码 %d", rec.Code)
	}
	var got struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if len(got.Data) != len(fallbackModels) {
		t.Fatalf("应回退到兜底模型列表，得到 %#v", got.Data)
	}
}

func TestHandleChatRejectsBadJSON(t *testing.T) {
	svc := NewService(codebuddy.NewClient("", ""), &fakeCreds{cred: testCred()}, &fakeRefresher{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader("{not json"))
	svc.handleChat(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，得到 %d", rec.Code)
	}
}
