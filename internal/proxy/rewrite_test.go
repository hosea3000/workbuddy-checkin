package proxy

import "testing"

func TestRewriteSystemPromptReplacesClaudeCodeIdentity(t *testing.T) {
	payload := map[string]any{
		"messages": []any{
			map[string]any{"role": "system", "content": claudeCodeIdentity + "\nDo things."},
			map[string]any{"role": "user", "content": "hi"},
		},
	}
	rewriteSystemPromptMessages(payload)
	msgs := payload["messages"].([]any)
	sys := msgs[0].(map[string]any)
	content, _ := sys["content"].(string)
	if content != neutralCLIIdentity+"\nDo things." {
		t.Fatalf("身份声明未替换: %q", content)
	}
}

func TestRewriteSystemPromptStripsAttributionLine(t *testing.T) {
	payload := map[string]any{
		"messages": []any{
			map[string]any{"role": "system", "content": attributionHeaderPrefix + " abc\nrest of prompt"},
		},
	}
	rewriteSystemPromptMessages(payload)
	msgs := payload["messages"].([]any)
	sys := msgs[0].(map[string]any)
	if got := sys["content"].(string); got != "rest of prompt" {
		t.Fatalf("attribution 行未移除: %q", got)
	}
}

func TestRewriteSystemPromptDropsMessageWhenOnlyAttribution(t *testing.T) {
	payload := map[string]any{
		"messages": []any{
			map[string]any{"role": "system", "content": attributionHeaderPrefix + " abc"},
			map[string]any{"role": "user", "content": "hi"},
		},
	}
	rewriteSystemPromptMessages(payload)
	msgs := payload["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("仅含 attribution 的 system 消息应被删除，得到 %d 条", len(msgs))
	}
	if msgs[0].(map[string]any)["role"] != "user" {
		t.Fatalf("剩余消息应为 user，得到 %#v", msgs[0])
	}
}

func TestRewriteSystemPromptLeavesUnrelatedPromptUntouched(t *testing.T) {
	payload := map[string]any{
		"messages": []any{
			map[string]any{"role": "system", "content": "You are Cline, a coding agent."},
		},
	}
	rewriteSystemPromptMessages(payload)
	msgs := payload["messages"].([]any)
	if got := msgs[0].(map[string]any)["content"].(string); got != "You are Cline, a coding agent." {
		t.Fatalf("不匹配的 prompt 不应被改写: %q", got)
	}
}

func TestRewriteSystemPromptHandlesBlockContent(t *testing.T) {
	payload := map[string]any{
		"messages": []any{
			map[string]any{"role": "system", "content": []any{
				map[string]any{"type": "text", "text": claudeCodeIdentity},
				map[string]any{"type": "image", "url": "x"},
			}},
		},
	}
	rewriteSystemPromptMessages(payload)
	msgs := payload["messages"].([]any)
	blocks := msgs[0].(map[string]any)["content"].([]any)
	if len(blocks) != 2 {
		t.Fatalf("应保留两个 block，得到 %d", len(blocks))
	}
	if blocks[0].(map[string]any)["text"] != neutralCLIIdentity {
		t.Fatalf("text block 未改写: %#v", blocks[0])
	}
}
