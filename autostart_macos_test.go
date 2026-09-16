package main

import (
	"strings"
	"testing"
)

func TestBuildLaunchAgentPlist(t *testing.T) {
	exe := "/Applications/WorkBuddy 自动签到.app/Contents/MacOS/workbuddy-checkin"
	got := buildLaunchAgentPlist(exe)

	wants := []string{
		"<key>Label</key>",
		"<string>" + launchAgentLabel + "</string>",
		"<key>ProgramArguments</key>",
		"<string>" + exe + "</string>",
		"<string>" + autoStartHiddenFlag + "</string>",
		"<key>RunAtLoad</key>",
		"<true/>",
	}
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Fatalf("plist 缺少片段 %q\n%s", w, got)
		}
	}
}
