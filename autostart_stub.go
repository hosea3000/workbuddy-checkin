//go:build !windows

package main

// setAutoStart 在非 Windows 平台上为空操作（仅保证可编译）。
func setAutoStart(enable bool) error { return nil }

// autoStartEnabled 在非 Windows 平台上恒为 false。
func autoStartEnabled() bool { return false }
