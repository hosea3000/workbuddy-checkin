//go:build darwin

package main

import (
	"log"
	"os/exec"
)

// osascriptNotifier 用 macOS 自带的 osascript 发送通知，无需第三方依赖或授权弹窗。
type osascriptNotifier struct{}

func (osascriptNotifier) Notify(title, body string) {
	script := `display notification "` + escapeAppleScript(body) + `" with title "` + escapeAppleScript(title) + `"`
	if err := exec.Command("osascript", "-e", script).Run(); err != nil {
		log.Printf("[notify] osascript failed: %v", err)
	}
}

// escapeAppleScript 转义字符串字面量中的反斜杠与双引号，避免注入与语法错误。
func escapeAppleScript(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r == '\\' || r == '"' {
			out = append(out, '\\')
		}
		out = append(out, r)
	}
	return string(out)
}

func newNotifier() Notifier { return osascriptNotifier{} }
