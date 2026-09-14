//go:build !windows

package main

import "log"

// Tray 是托盘在非 Windows 平台的空实现，保证 Linux 下可编译/可测试。
type Tray struct {
	onOpen func()
	onQuit func()
	onWake func()
}

func newTray(onOpen, onQuit, onWake func()) *Tray {
	return &Tray{onOpen: onOpen, onQuit: onQuit, onWake: onWake}
}

// Start 在非 Windows 平台上记录日志并立即返回。
func (t *Tray) Start(tip string) { log.Printf("[tray] stub start: %s", tip) }

// SetTip 在非 Windows 平台上为空操作。
func (t *Tray) SetTip(string) {}
