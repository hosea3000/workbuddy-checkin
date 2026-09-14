//go:build !windows

package main

import "log"

// stubNotifier 在非 Windows 平台上把通知写入日志，保证 Linux 下可编译/可测试。
type stubNotifier struct{}

func (stubNotifier) Notify(title, body string) {
	log.Printf("[notify] %s: %s", title, body)
}

func newNotifier() Notifier { return stubNotifier{} }
