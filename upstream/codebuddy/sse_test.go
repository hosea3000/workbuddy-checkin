package codebuddy

import (
	"errors"
	"strings"
	"testing"
)

func TestParseSSELine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantNil  bool
		wantDone bool
		wantErr  bool
	}{
		{name: "空行", line: "", wantNil: true},
		{name: "仅空白", line: "   ", wantNil: true},
		{name: "注释行", line: ": keep-alive", wantNil: true},
		{name: "非 data 字段", line: "event: message", wantNil: true},
		{name: "data 无内容", line: "data:", wantNil: true},
		{name: "data 空格后无内容", line: "data:   ", wantNil: true},
		{name: "data 无空格分隔", line: "data:{\"a\":1}", wantNil: true},
		{name: "done 标记", line: "data: [DONE]", wantDone: true},
		{name: "合法 JSON", line: `data: {"choices":[{"delta":{"content":"hi"}}]}`},
		{name: "非法 JSON", line: "data: {not json", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ev, err := ParseSSELine(tc.line)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("期望错误，得到 %#v", ev)
				}
				var de *SSEDataError
				if !errors.As(err, &de) {
					t.Fatalf("期望 SSEDataError，得到 %T", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("意外错误: %v", err)
			}
			if tc.wantNil {
				if ev != nil {
					t.Fatalf("期望 nil，得到 %#v", ev)
				}
				return
			}
			if tc.wantDone {
				if _, ok := ev.(SSEDone); !ok {
					t.Fatalf("期望 SSEDone，得到 %T", ev)
				}
				return
			}
			if _, ok := ev.(map[string]any); !ok {
				t.Fatalf("期望 map，得到 %T", ev)
			}
		})
	}
}

func TestParseSSELineTrimBehaviour(t *testing.T) {
	ev, err := ParseSSELine("  data:   {\"a\":1}  ")
	if err != nil {
		t.Fatal(err)
	}
	m, ok := ev.(map[string]any)
	if !ok {
		t.Fatalf("期望 map，得到 %T", ev)
	}
	if _, has := m["a"]; !has {
		t.Fatalf("未解析出字段: %#v", m)
	}
}

func collect(t *testing.T, input string) []any {
	t.Helper()
	var got []any
	err := IterSSEEventsScanner(strings.NewReader(input), func(ev any) bool {
		got = append(got, ev)
		return true
	})
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	return got
}

func TestIterSSEEventsScanner(t *testing.T) {
	input := ": comment\n\n" +
		`data: {"n":1}` + "\n\n" +
		"event: ignored\n" +
		`data: {"n":2}` + "\n\n" +
		"data: [DONE]\n"
	got := collect(t, input)
	if len(got) != 3 {
		t.Fatalf("期望 3 个事件，得到 %d: %#v", len(got), got)
	}
	first, ok := got[0].(map[string]any)
	if !ok || first["n"].(float64) != 1 {
		t.Fatalf("第 1 个事件不符: %#v", got[0])
	}
	if _, ok := got[2].(SSEDone); !ok {
		t.Fatalf("第 3 个事件应为 SSEDone，得到 %T", got[2])
	}
}

func TestIterSSEEventsScannerLastLineWithoutNewline(t *testing.T) {
	got := collect(t, `data: {"n":1}`+"\n"+`data: {"n":2}`)
	if len(got) != 2 {
		t.Fatalf("末行（无换行结尾）应被处理一次，得到 %d 个事件: %#v", len(got), got)
	}
}

func TestIterSSEEventsScannerEarlyStop(t *testing.T) {
	var got []any
	err := IterSSEEventsScanner(strings.NewReader("data: {\"n\":1}\ndata: {\"n\":2}\ndata: {\"n\":3}\n"),
		func(ev any) bool {
			got = append(got, ev)
			return false
		})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("应提前终止于第 1 个事件，得到 %d", len(got))
	}
}

func TestIterSSEEventsScannerPropagatesError(t *testing.T) {
	err := IterSSEEventsScanner(strings.NewReader("data: {bad\n"), func(any) bool { return true })
	var de *SSEDataError
	if !errors.As(err, &de) {
		t.Fatalf("期望 SSEDataError，得到 %v", err)
	}
}

func TestFormatSSE(t *testing.T) {
	got := FormatSSEEvent(map[string]any{"a": 1})
	if !strings.HasPrefix(got, "data: ") || !strings.HasSuffix(got, "\n\n") {
		t.Fatalf("格式不符: %q", got)
	}
	if FormatSSEDone() != "data: [DONE]\n\n" {
		t.Fatalf("done 格式不符: %q", FormatSSEDone())
	}
	errEvent := FormatSSEError("boom", "upstream_error")
	if !strings.Contains(errEvent, "boom") || !strings.Contains(errEvent, "upstream_error") {
		t.Fatalf("错误事件内容不符: %q", errEvent)
	}
}
