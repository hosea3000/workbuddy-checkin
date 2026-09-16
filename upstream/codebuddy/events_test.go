package codebuddy

import (
	"encoding/json"
	"testing"
)

func chunk(t *testing.T, raw string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("测试数据非法: %v", err)
	}
	return m
}

func TestParseEventFinishReasonAndDelta(t *testing.T) {
	ev := ParseEvent(chunk(t, `{"choices":[{"delta":{"content":"hi"},"finish_reason":"stop"}]}`))
	if !ev.HasChoice() {
		t.Fatal("应识别 choice")
	}
	if ev.Content() != "hi" {
		t.Fatalf("content 不符: %#v", ev.Content())
	}
	if ev.FinishReason == nil || *ev.FinishReason != "stop" {
		t.Fatalf("finish_reason 不符: %#v", ev.FinishReason)
	}
}

func TestParseEventEmptyFinishReasonIsNil(t *testing.T) {
	ev := ParseEvent(chunk(t, `{"choices":[{"delta":{},"finish_reason":""}]}`))
	if ev.FinishReason != nil {
		t.Fatalf("空 finish_reason 应为 nil，得到 %q", *ev.FinishReason)
	}
}

func TestParseEventNoChoices(t *testing.T) {
	ev := ParseEvent(chunk(t, `{"usage":{"total_tokens":5}}`))
	if ev.HasChoice() {
		t.Fatal("无 choices 时不应有 choice")
	}
	if ev.Usage() == nil {
		t.Fatal("usage 应可读取")
	}
}

func TestParseEventSeparatesReasoningAndContent(t *testing.T) {
	ev := ParseEvent(chunk(t, `{"choices":[{"delta":{"reasoning_content":"think","content":"say"}}]}`))
	if ev.ReasoningContent() != "think" {
		t.Fatalf("reasoning 不符: %#v", ev.ReasoningContent())
	}
	if ev.Content() != "say" {
		t.Fatalf("content 不符: %#v", ev.Content())
	}
}

func TestToolCallIndexResolvePrefersUpstreamIndex(t *testing.T) {
	st := NewToolCallIndexState()
	idx := st.Resolve(map[string]any{"id": "a", "index": float64(3)})
	if idx == nil || *idx != 3 {
		t.Fatalf("应沿用上游 index=3，得到 %#v", idx)
	}
	// 同 id 再次出现且无 index 时应回到 3
	again := st.Resolve(map[string]any{"id": "a"})
	if again == nil || *again != 3 {
		t.Fatalf("应按 id 映射回 3，得到 %#v", again)
	}
}

func TestToolCallIndexResolveGeneratesStableIndex(t *testing.T) {
	st := NewToolCallIndexState()
	first := st.Resolve(map[string]any{"id": "a"})
	if first == nil || *first != 0 {
		t.Fatalf("首个无 index 的 tool_call 应得 0，得到 %#v", first)
	}
	second := st.Resolve(map[string]any{"id": "b"})
	if second == nil || *second != 1 {
		t.Fatalf("第二个应得 1，得到 %#v", second)
	}
	// 无 id 无 index 时沿用最近一次
	third := st.Resolve(map[string]any{})
	if third == nil || *third != 1 {
		t.Fatalf("应沿用最近 index，得到 %#v", third)
	}
}

func TestAddOpenAIToolCallIndexesFillsIndex(t *testing.T) {
	st := NewToolCallIndexState()
	ev := ParseEvent(chunk(t, `{"choices":[{"delta":{"tool_calls":[{"id":"a","function":{"name":"f","arguments":"{"}}]}}]}`))
	out := AddOpenAIToolCallIndexes(ev, st)
	choices, _ := out["choices"].([]any)
	delta, _ := choices[0].(map[string]any)["delta"].(map[string]any)
	tcs, _ := delta["tool_calls"].([]any)
	if len(tcs) != 1 {
		t.Fatalf("tool_calls 数量不符: %#v", tcs)
	}
	tc, _ := tcs[0].(map[string]any)
	if tc["index"] != 0 {
		t.Fatalf("index 应被补齐为 0，得到 %#v", tc["index"])
	}
	// 原事件不被修改
	if _, has := ev.Delta["tool_calls"].([]any)[0].(map[string]any)["index"]; has {
		t.Fatal("原事件不应被就地修改")
	}
}

func TestStreamNormalizerEmitsRoleOnce(t *testing.T) {
	n := NewStreamNormalizer()
	first := n.Normalize(chunk(t, `{"choices":[{"delta":{"content":"a"}}]}`))
	if len(first) != 2 {
		t.Fatalf("首个内容 chunk 应产出 role + content 两个 chunk，得到 %d", len(first))
	}
	roleDelta := deltaOf(t, first[0])
	if roleDelta["role"] != "assistant" {
		t.Fatalf("第 1 个 chunk 应补 assistant 角色，得到 %#v", roleDelta)
	}
	second := n.Normalize(chunk(t, `{"choices":[{"delta":{"content":"b"}}]}`))
	if len(second) != 1 {
		t.Fatalf("后续 chunk 不应重复补角色，得到 %d", len(second))
	}
}

func TestStreamNormalizerDropsEmptyDelta(t *testing.T) {
	n := NewStreamNormalizer()
	n.Normalize(chunk(t, `{"choices":[{"delta":{"content":"a"}}]}`))
	out := n.Normalize(chunk(t, `{"choices":[{"delta":{"content":""}}]}`))
	if len(out) != 0 {
		t.Fatalf("空 delta 且无 finish/usage 时不应产出 chunk，得到 %#v", out)
	}
}

func TestStreamNormalizerKeepsFinishReason(t *testing.T) {
	n := NewStreamNormalizer()
	out := n.Normalize(chunk(t, `{"choices":[{"delta":{},"finish_reason":"stop"}]}`))
	if len(out) != 1 {
		t.Fatalf("应产出 1 个 finish chunk，得到 %d", len(out))
	}
	choice := out[0]["choices"].([]any)[0].(map[string]any)
	if choice["finish_reason"] != "stop" {
		t.Fatalf("finish_reason 应保留，得到 %#v", choice["finish_reason"])
	}
}

func TestNormalizeChunkEnvelope(t *testing.T) {
	out := NormalizeChunkEnvelope(chunk(t, `{"choices":[]}`), "chatcmpl-x", 123, "glm-5.2")
	if out["id"] != "chatcmpl-x" || out["object"] != "chat.completion.chunk" ||
		out["created"] != int64(123) || out["model"] != "glm-5.2" {
		t.Fatalf("信封不符: %#v", out)
	}
}

func deltaOf(t *testing.T, c map[string]any) map[string]any {
	t.Helper()
	choices, _ := c["choices"].([]any)
	if len(choices) == 0 {
		t.Fatalf("无 choices: %#v", c)
	}
	choice, _ := choices[0].(map[string]any)
	d, _ := choice["delta"].(map[string]any)
	return d
}
