package proxy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// stubModels 提供固定的模型列表，避免测试依赖真实上游。
type stubModels struct{ list []string }

func (s stubModels) Available(context.Context) []string { return s.list }

// 超限请求必须返回 context_length_exceeded，客户端据此压缩上下文后重试。
func TestExecuteRejectsOversizedPayload(t *testing.T) {
	big := strings.Repeat("x", maxUpstreamPayloadBytes+1)
	body := map[string]any{
		"model":    "glm-5.2",
		"messages": []any{map[string]any{"role": "user", "content": big}},
	}

	rec := httptest.NewRecorder()
	exec := NewExecutor(nil, nil, stubModels{list: []string{"glm-5.2"}})
	exec.Execute(t.Context(), body, rec)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("状态码应为 400，得到 %d", rec.Code)
	}
	var resp struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("响应不是合法 JSON: %v (%s)", err, rec.Body.String())
	}
	if resp.Error.Code != errCodeContextLengthExceeded {
		t.Fatalf("错误码应为 %q，得到 %q", errCodeContextLengthExceeded, resp.Error.Code)
	}
}

func TestMessageAndToolCount(t *testing.T) {
	payload := map[string]any{
		"messages": []any{map[string]any{"role": "user"}, map[string]any{"role": "assistant"}},
		"tools":    []any{map[string]any{"type": "function"}},
	}
	if got := messageCount(payload); got != 2 {
		t.Fatalf("messageCount 应为 2，得到 %d", got)
	}
	if got := toolCount(payload); got != 1 {
		t.Fatalf("toolCount 应为 1，得到 %d", got)
	}
	if got := messageCount(map[string]any{}); got != 0 {
		t.Fatalf("缺失字段应返回 0，得到 %d", got)
	}
}
