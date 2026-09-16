//go:build !windows && !darwin

package main

import "log"

// Tray 是托盘在既非 Windows 也非 macOS 平台的空实现，保证 Linux 下可编译/可测试。
type Tray struct {
	onOpen func()
	onQuit func()
	onWake func()

	onToggleProxy func()
}

func newTray(onOpen, onQuit, onWake, onToggleProxy func()) *Tray {
	return &Tray{onOpen: onOpen, onQuit: onQuit, onWake: onWake, onToggleProxy: onToggleProxy}
}

// SetProxyState 在无托盘实现的平台上为空操作。
func (t *Tray) SetProxyState(bool) {}

// Start 在无托盘实现的平台上记录日志并立即返回。
func (t *Tray) Start(tip string) { log.Printf("[tray] stub start: %s", tip) }

// SetTip 在无托盘实现的平台上为空操作。
func (t *Tray) SetTip(string) {}

// setActivationPolicyAccessory 仅在 macOS 有意义，其余平台为空操作。
func setActivationPolicyAccessory() {}
