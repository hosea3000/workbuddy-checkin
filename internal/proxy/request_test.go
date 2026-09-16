package proxy

import "testing"

func TestValidateChatRequest(t *testing.T) {
	tests := []struct {
		name string
		body map[string]any
		want int
	}{
		{name: "nil body", body: nil, want: 400},
		{name: "缺 messages", body: map[string]any{}, want: 400},
		{name: "messages 非数组", body: map[string]any{"messages": "x"}, want: 400},
		{name: "messages 为空", body: map[string]any{"messages": []any{}}, want: 400},
		{name: "message 非对象", body: map[string]any{"messages": []any{"x"}}, want: 400},
		{name: "缺 role", body: map[string]any{"messages": []any{map[string]any{"content": "hi"}}}, want: 400},
		{name: "缺 content", body: map[string]any{"messages": []any{map[string]any{"role": "user"}}}, want: 400},
		{name: "assistant 仅带 tool_calls", body: map[string]any{"messages": []any{
			map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"id": "a"}}},
		}}},
		{name: "非空 stop 数组", body: map[string]any{
			"messages": []any{map[string]any{"role": "user", "content": "hi"}},
			"stop":     []any{"x"},
		}, want: 400},
		{name: "非空 stop 字符串", body: map[string]any{
			"messages": []any{map[string]any{"role": "user", "content": "hi"}},
			"stop":     "x",
		}, want: 400},
		{name: "空 stop 数组放行", body: map[string]any{
			"messages": []any{map[string]any{"role": "user", "content": "hi"}},
			"stop":     []any{},
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateChatRequest(tc.body)
			if tc.want == 0 {
				if err != nil {
					t.Fatalf("期望通过，得到 %v", err)
				}
				return
			}
			if err == nil || err.Status != tc.want {
				t.Fatalf("期望 %d，得到 %#v", tc.want, err)
			}
		})
	}
}

func TestPrepareChatPayloadPassesTemperatureThrough(t *testing.T) {
	payload, _ := PrepareChatPayload(map[string]any{
		"messages":    []any{map[string]any{"role": "user", "content": "hi"}},
		"temperature": 0.2,
	}, "")
	if payload["temperature"] != 0.2 {
		t.Fatalf("temperature 应原样透传，得到 %#v", payload["temperature"])
	}
}

func TestPrepareChatPayloadForcesStream(t *testing.T) {
	payload, _ := PrepareChatPayload(map[string]any{
		"messages": []any{map[string]any{"role": "user", "content": "hi"}},
	}, "")
	if payload["stream"] != true {
		t.Fatalf("stream 应被强制为 true，得到 %#v", payload["stream"])
	}
	so, _ := payload["stream_options"].(map[string]any)
	if so == nil || so["include_usage"] != true {
		t.Fatalf("应附加 include_usage，得到 %#v", payload["stream_options"])
	}
}

func TestPrepareChatPayloadThinkingDefaultsOn(t *testing.T) {
	payload, _ := PrepareChatPayload(map[string]any{
		"messages": []any{map[string]any{"role": "user", "content": "hi"}},
	}, "")
	if payload["enable_thinking"] != true {
		t.Fatalf("未表态时应默认开启思考，得到 %#v", payload["enable_thinking"])
	}
}

func TestPrepareChatPayloadRespectsExplicitThinkingDisable(t *testing.T) {
	for _, v := range []any{false, "false", 0.0, "disabled"} {
		payload, _ := PrepareChatPayload(map[string]any{
			"messages":        []any{map[string]any{"role": "user", "content": "hi"}},
			"enable_thinking": v,
		}, "")
		if _, has := payload["enable_thinking"]; has {
			t.Fatalf("显式禁用时不应保留 enable_thinking（输入 %#v），得到 %#v", v, payload["enable_thinking"])
		}
	}
}

func TestPrepareChatPayloadThinkingDisabledByThinkingObject(t *testing.T) {
	payload, _ := PrepareChatPayload(map[string]any{
		"messages": []any{map[string]any{"role": "user", "content": "hi"}},
		"thinking": map[string]any{"type": "disabled"},
	}, "")
	if _, has := payload["enable_thinking"]; has {
		t.Fatalf("thinking.type=disabled 时不应保留 enable_thinking")
	}
}

func TestPrepareChatPayloadDefaultModelAndNamespaceStrip(t *testing.T) {
	payload, responseModel := PrepareChatPayload(map[string]any{
		"messages": []any{map[string]any{"role": "user", "content": "hi"}},
	}, "glm-5.2")
	if payload["model"] != "glm-5.2" {
		t.Fatalf("应填入默认模型，得到 %#v", payload["model"])
	}
	if responseModel != "unknown" {
		t.Fatalf("未指定模型时 responseModel 应为 unknown，得到 %q", responseModel)
	}

	payload, responseModel = PrepareChatPayload(map[string]any{
		"messages": []any{map[string]any{"role": "user", "content": "hi"}},
		"model":    "codebuddy/glm-5.2",
	}, "")
	if payload["model"] != "glm-5.2" {
		t.Fatalf("应剥离命名空间，得到 %#v", payload["model"])
	}
	if responseModel != "codebuddy/glm-5.2" {
		t.Fatalf("responseModel 应保留客户端原值，得到 %q", responseModel)
	}
}

func TestPrepareChatPayloadAddsSystemForSingleUserMessage(t *testing.T) {
	payload, _ := PrepareChatPayload(map[string]any{
		"messages": []any{map[string]any{"role": "user", "content": "hi"}},
	}, "")
	msgs, _ := payload["messages"].([]any)
	if len(msgs) != 2 {
		t.Fatalf("单条 user 消息应补 system，得到 %d 条", len(msgs))
	}
	first, _ := msgs[0].(map[string]any)
	if first["role"] != "system" {
		t.Fatalf("首条应为 system，得到 %#v", first)
	}
}

func TestPrepareChatPayloadDoesNotMutateInput(t *testing.T) {
	body := map[string]any{
		"messages": []any{map[string]any{"role": "user", "content": "hi"}},
	}
	PrepareChatPayload(body, "m")
	if _, has := body["stream"]; has {
		t.Fatal("不应修改调用方的请求体")
	}
}
