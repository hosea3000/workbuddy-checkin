package main

import (
	"os/exec"

	"github.com/hosea3000/workbuddy-checkin/model"
)

// GetSettings 返回当前设置；AutoStart 以 HKCU Run 的真实状态为准（不信任持久化值）。
func (a *App) GetSettings() model.Settings {
	s := a.store.GetSettings()
	s.AutoStart = autoStartEnabled()
	return s
}

// SaveSettings 持久化设置，并每次同步 HKCU Run（幂等，可修复外部改动导致的不一致）。
func (a *App) SaveSettings(s model.Settings) error {
	if err := setAutoStart(s.AutoStart); err != nil {
		return err
	}
	return a.store.SaveSettings(s)
}

// OpenDataDir 用系统资源管理器打开数据目录。
func (a *App) OpenDataDir() error {
	return openDir(a.store.Dir())
}

// openDir 打开目录（Windows 用 explorer，其他平台用 xdg-open；stub 平台仅尝试）。
func openDir(dir string) error {
	return exec.Command(dirOpenCmd(), dir).Start()
}

func dirOpenCmd() string {
	if isWindows() {
		return "explorer"
	}
	return "xdg-open"
}
