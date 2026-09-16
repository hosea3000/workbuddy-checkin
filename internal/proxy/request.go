package proxy

import (
	"encoding/json"
	"strings"
)

// ValidateChatRequest 校验 OpenAI 聊天请求（对齐参考实现的 validate_request）。
// 上游无法报告停止序列命中，因此非空 stop 直接拒绝（设计决策 7）。
func ValidateChatRequest(body map[string]any) *HTTPError {
	if err := validateMessages(body); err != nil {
		return err
	}
	if hasNonEmptyStop(body["stop"]) {
		return errInvalid("stop sequences are not supported by CodeBuddy upstream")
	}
	return nil
}

func validateMessages(body map[string]any) *HTTPError {
	if body == nil {
		return errInvalid("Request body must be a JSON object")
	}
	messages, ok := body["messages"].([]any)
	if !ok || len(messages) == 0 {
		return errInvalid("Messages field is required and must be an array")
	}
	for _, m := range messages {
		msg, ok := m.(map[string]any)
		if !ok {
			return errInvalid("Message must be an object")
		}
		if _, hasRole := msg["role"]; !hasRole {
			return errInvalid("Message must have 'role' field")
		}
		if _, hasContent := msg["content"]; hasContent {
			continue
		}
		// 允许 assistant 消息仅带 tool_calls
		tcs, _ := msg["tool_calls"].([]any)
		valid := false
		if role, _ := msg["role"].(string); role == "assistant" && len(tcs) > 0 {
			valid = true
			for _, tc := range tcs {
				if _, ok := tc.(map[string]any); !ok {
					valid = false
					break
				}
			}
		}
		if !valid {
			return errInvalid("Message must have 'content' field")
		}
	}
	return nil
}

func hasNonEmptyStop(v any) bool {
	switch t := v.(type) {
	case string:
		return t != ""
	case []any:
		return len(t) > 0
	default:
		return false
	}
}

// PrepareChatPayload 深拷贝并应用上游协议适配，返回 (上游 payload, 客户端期望的响应 model)。
//
// 相对参考实现裁剪（设计决策 7）：不做 temperature 强制改写、不做推理模型强制 reasoning_effort。
func PrepareChatPayload(requestBody map[string]any, defaultModel string) (map[string]any, string) {
	payload := deepCloneMap(requestBody)
	responseModel := stringOr(payload["model"], "")
	if responseModel == "" {
		responseModel = "unknown"
	}
	if stringOr(payload["model"], "") == "" && defaultModel != "" {
		payload["model"] = defaultModel
	}
	// 命名空间剥离 provider/model → model（客户端常带前缀，上游只认裸模型 ID）
	if m, ok := payload["model"].(string); ok {
		payload["model"] = stripNamespace(m)
	}
	// 思考开关：客户端显式禁用则尊重，未表态则默认开启
	if thinkingExplicitlyDisabled(payload) {
		delete(payload, "enable_thinking")
	} else if _, has := payload["enable_thinking"]; !has {
		payload["enable_thinking"] = true
	}
	// 单条 user 消息补 system，避免上游对无 system 请求报错
	if messages, ok := payload["messages"].([]any); ok && len(messages) == 1 {
		if m, ok := messages[0].(map[string]any); ok && m["role"] == "user" {
			system := map[string]any{"role": "system", "content": "You are a helpful assistant."}
			payload["messages"] = []any{system, messages[0]}
		}
	}
	rewriteSystemPromptMessages(payload)

	// 上游只支持流式
	streamOptions, _ := payload["stream_options"].(map[string]any)
	merged := map[string]any{"include_usage": true}
	for k, v := range streamOptions {
		merged[k] = v
	}
	payload["stream_options"] = merged
	payload["stream"] = true

	return payload, responseModel
}

func stripNamespace(model string) string {
	model = strings.TrimSpace(model)
	if idx := strings.LastIndex(model, "/"); idx >= 0 {
		return model[idx+1:]
	}
	return model
}

func thinkingExplicitlyDisabled(payload map[string]any) bool {
	if v, ok := payload["enable_thinking"]; ok && isFalseLike(v) {
		return true
	}
	if thinking, ok := payload["thinking"].(map[string]any); ok {
		if t, _ := thinking["type"].(string); strings.ToLower(strings.TrimSpace(t)) == "disabled" {
			return true
		}
	}
	return false
}

func isFalseLike(v any) bool {
	switch t := v.(type) {
	case bool:
		return !t
	case float64:
		return t == 0
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		return s == "false" || s == "0" || s == "no" || s == "off" || s == "disabled"
	default:
		return false
	}
}

func stringOr(v any, def string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return def
}

func boolOr(v any) bool {
	b, _ := v.(bool)
	return b
}

func deepCloneMap(m map[string]any) map[string]any {
	b, _ := json.Marshal(m)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	if out == nil {
		out = map[string]any{}
	}
	return out
}
