package proxy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hosea3000/workbuddy-checkin/model"
	"github.com/hosea3000/workbuddy-checkin/upstream/codebuddy"
)

type fakeCreds struct {
	cred model.Credential
	err  error
}

func (f *fakeCreds) Active(context.Context) (model.Credential, error) { return f.cred, f.err }

type fakeModels struct{ list []string }

func (f *fakeModels) Available(context.Context) []string { return f.list }

func testCred() model.Credential {
	return model.Credential{
		ID: "c1", UserID: "uid_a", AccountUID: "a",
		AccessToken: "tok", Status: model.StatusActive,
	}
}

// upstream 是可捕获请求体的假上游。
type upstream struct {
	*httptest.Server
	lastBody map[string]any
	calls    int
}

func newUpstream(t *testing.T, status int, body string) *upstream {
	t.Helper()
	u := &upstream{}
	u.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u.calls++
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &u.lastBody)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(u.Close)
	return u
}

func newTestExecutor(t *testing.T, up *upstream, creds CredentialProvider) *Executor {
	t.Helper()
	return NewExecutor(codebuddy.NewClient(up.URL, ""), creds, &fakeModels{list: []string{"glm-5.2"}})
}

const okStream = "data: {\"choices\":[{\"delta\":{\"role\":\"assistant\",\"content\":\"Hel\"}}]}\n\n" +
	"data: {\"choices\":[{\"delta\":{\"content\":\"lo\"}}]}\n\n" +
	"data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
	"data: [DONE]\n\n"

func chatBody(extra map[string]any) map[string]any {
	body := map[string]any{
		"messages": []any{map[string]any{"role": "user", "content": "hi"}},
	}
	for k, v := range extra {
		body[k] = v
	}
	return body
}

func TestExecuteNonStreamAggregates(t *testing.T) {
	up := newUpstream(t, http.StatusOK, okStream)
	x := newTestExecutor(t, up, &fakeCreds{cred: testCred()})

	rec := httptest.NewRecorder()
	x.Execute(context.Background(), chatBody(nil), rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("状态码 %d，body=%s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("响应非 JSON: %v", err)
	}
	if got["object"] != "chat.completion" {
		t.Fatalf("object 不符: %#v", got["object"])
	}
	choices := got["choices"].([]any)
	msg := choices[0].(map[string]any)["message"].(map[string]any)
	if msg["content"] != "Hello" {
		t.Fatalf("内容未聚合: %#v", msg["content"])
	}
	if choices[0].(map[string]any)["finish_reason"] != "stop" {
		t.Fatalf("finish_reason 不符: %#v", choices[0].(map[string]any)["finish_reason"])
	}
}

func TestExecuteStreamWritesSSE(t *testing.T) {
	up := newUpstream(t, http.StatusOK, okStream)
	x := newTestExecutor(t, up, &fakeCreds{cred: testCred()})

	rec := httptest.NewRecorder()
	x.Execute(context.Background(), chatBody(map[string]any{"stream": true}), rec)

	if rec.Code != http.StatusOK {
		t.Fatalf("状态码 %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type 不符: %q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "chat.completion.chunk") {
		t.Fatalf("缺少 chunk 信封: %s", body)
	}
	if !strings.HasSuffix(body, "data: [DONE]\n\n") {
		t.Fatalf("未以 [DONE] 结束: %s", body)
	}
}

func TestExecuteForwardsTemperatureAndForcesStream(t *testing.T) {
	up := newUpstream(t, http.StatusOK, okStream)
	x := newTestExecutor(t, up, &fakeCreds{cred: testCred()})

	x.Execute(context.Background(), chatBody(map[string]any{"temperature": 0.2}), httptest.NewRecorder())

	if up.lastBody["temperature"] != 0.2 {
		t.Fatalf("temperature 应透传，上游收到 %#v", up.lastBody["temperature"])
	}
	if up.lastBody["stream"] != true {
		t.Fatalf("上游应收到 stream=true，得到 %#v", up.lastBody["stream"])
	}
}

func TestExecuteRejectsInvalidRequestWithoutUpstreamCall(t *testing.T) {
	up := newUpstream(t, http.StatusOK, okStream)
	x := newTestExecutor(t, up, &fakeCreds{cred: testCred()})

	rec := httptest.NewRecorder()
	x.Execute(context.Background(), map[string]any{}, rec)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，得到 %d", rec.Code)
	}
	if up.calls != 0 {
		t.Fatalf("校验失败不应调用上游，实际调用 %d 次", up.calls)
	}
}

func TestExecuteRejectsStop(t *testing.T) {
	up := newUpstream(t, http.StatusOK, okStream)
	x := newTestExecutor(t, up, &fakeCreds{cred: testCred()})

	rec := httptest.NewRecorder()
	x.Execute(context.Background(), chatBody(map[string]any{"stop": []any{"x"}}), rec)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("期望 400，得到 %d", rec.Code)
	}
}

func TestExecuteNoCredential(t *testing.T) {
	up := newUpstream(t, http.StatusOK, okStream)
	x := newTestExecutor(t, up, &fakeCreds{err: errUnavailable("未设置当前凭证")})

	rec := httptest.NewRecorder()
	x.Execute(context.Background(), chatBody(nil), rec)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("期望 503，得到 %d", rec.Code)
	}
	assertErrorType(t, rec, errTypeUnavailable)
	if up.calls != 0 {
		t.Fatal("无凭证时不应调用上游")
	}
}

func TestExecuteReloginRequired(t *testing.T) {
	up := newUpstream(t, http.StatusOK, okStream)
	x := newTestExecutor(t, up, &fakeCreds{err: errCredential("需重新登录")})

	rec := httptest.NewRecorder()
	x.Execute(context.Background(), chatBody(nil), rec)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("期望 502，得到 %d", rec.Code)
	}
	assertErrorType(t, rec, errTypeAuth)
}

func TestExecuteUpstreamErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		upstream   int
		wantStatus int
		wantType   string
	}{
		{name: "限流", upstream: http.StatusTooManyRequests, wantStatus: http.StatusTooManyRequests, wantType: errTypeRateLimit},
		{name: "上游 5xx", upstream: http.StatusInternalServerError, wantStatus: http.StatusBadGateway, wantType: errTypeUpstream},
		{name: "凭证被拒", upstream: http.StatusUnauthorized, wantStatus: http.StatusBadGateway, wantType: errTypeAuth},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			up := newUpstream(t, tc.upstream, `{"message":"boom"}`)
			x := newTestExecutor(t, up, &fakeCreds{cred: testCred()})

			rec := httptest.NewRecorder()
			x.Execute(context.Background(), chatBody(nil), rec)
			if rec.Code != tc.wantStatus {
				t.Fatalf("期望 %d，得到 %d（body=%s）", tc.wantStatus, rec.Code, rec.Body.String())
			}
			assertErrorType(t, rec, tc.wantType)
		})
	}
}

func TestExecuteIncompleteStreamIsError(t *testing.T) {
	up := newUpstream(t, http.StatusOK, "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n")
	x := newTestExecutor(t, up, &fakeCreds{cred: testCred()})

	rec := httptest.NewRecorder()
	x.Execute(context.Background(), chatBody(nil), rec)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("流未正常结束应报错，得到 %d（body=%s）", rec.Code, rec.Body.String())
	}
}

func TestExecuteStreamErrorEventReported(t *testing.T) {
	up := newUpstream(t, http.StatusOK, "data: {\"error\":{\"message\":\"upstream exploded\"}}\n\n")
	x := newTestExecutor(t, up, &fakeCreds{cred: testCred()})

	rec := httptest.NewRecorder()
	x.Execute(context.Background(), chatBody(nil), rec)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("期望 502，得到 %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "upstream exploded") {
		t.Fatalf("应透出上游错误消息: %s", rec.Body.String())
	}
}

func assertErrorType(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("响应非 JSON: %v", err)
	}
	e, ok := got["error"].(map[string]any)
	if !ok {
		t.Fatalf("缺少 error 字段: %s", rec.Body.String())
	}
	if e["type"] != want {
		t.Fatalf("error.type 期望 %q，得到 %#v（body=%s）", want, e["type"], rec.Body.String())
	}
}
