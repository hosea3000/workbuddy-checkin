package codebuddy

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
)

// SSEDone 表示上游流结束标记（data: [DONE]）。
type SSEDone struct{}

// SSEDataError 上游 SSE data 字段不是可解析 JSON。
type SSEDataError struct{ Message string }

func (e *SSEDataError) Error() string { return e.Message }

// ParseSSELine 解析单行 SSE，返回 JSON 对象(map)、SSEDone{} 或 nil。
// 语义对齐参考实现的 parse_sse_event。
func ParseSSELine(line string) (any, error) {
	stripped := strings.TrimSpace(line)
	if stripped == "" || strings.HasPrefix(stripped, ":") {
		return nil, nil
	}
	if stripped != "data:" && !strings.HasPrefix(stripped, "data: ") {
		return nil, nil
	}
	data := strings.TrimSpace(stripped[5:])
	if data == "" {
		return nil, nil
	}
	if data == "[DONE]" {
		return SSEDone{}, nil
	}
	var v any
	if err := json.Unmarshal([]byte(data), &v); err != nil {
		return nil, &SSEDataError{Message: "upstream SSE data contains invalid JSON"}
	}
	return v, nil
}

// IterSSEEventsScanner 逐行流式解析 SSE，onEvent 返回 false 时提前终止。
// 空行与注释行被跳过；最后一行无需换行结尾（bufio.Scanner 会产出未终止的末行）。
func IterSSEEventsScanner(r io.Reader, onEvent func(ev any) bool) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		ev, err := ParseSSELine(line)
		if err != nil {
			return err
		}
		if ev == nil {
			continue
		}
		if !onEvent(ev) {
			return nil
		}
	}
	return scanner.Err()
}

// FormatSSEEvent 格式化 data-only SSE 事件（OpenAI 客户端依赖空行边界）。
func FormatSSEEvent(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return "data: " + string(b) + "\n\n"
}

// FormatSSEDone 结束标记。
func FormatSSEDone() string { return "data: [DONE]\n\n" }

// FormatSSEError 流中错误事件格式。
func FormatSSEError(message, errorType string) string {
	return FormatSSEEvent(map[string]any{
		"error": map[string]any{"message": message, "type": errorType},
	})
}
