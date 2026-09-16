package main

import (
	"log"
	"os/exec"

	"github.com/hosea3000/workbuddy-checkin/internal/proxy"
	"github.com/hosea3000/workbuddy-checkin/model"
)

// GetSettings 返回当前设置；AutoStart 以 HKCU Run 的真实状态为准（不信任持久化值）。
func (a *App) GetSettings() model.Settings {
	s := a.store.GetSettings()
	s.AutoStart = autoStartEnabled()
	return s
}

// SaveSettings 持久化设置，并同步开机自启与模型代理监听（幂等，可修复外部改动导致的不一致）。
func (a *App) SaveSettings(s model.Settings) error {
	if s.ProxyPort == 0 {
		s.ProxyPort = model.DefaultProxyPort
	}
	if err := proxy.ValidatePort(s.ProxyPort); err != nil {
		return err
	}
	if err := setAutoStart(s.AutoStart); err != nil {
		return err
	}
	if err := a.store.SaveSettings(s); err != nil {
		return err
	}
	a.applyProxySettings(s)
	if a.tray != nil {
		a.tray.SetProxyState(s.ProxyEnabled)
	}
	return nil
}

// applyProxySettings 按最新设置启停模型代理服务（开关或端口变化时重启监听）。
func (a *App) applyProxySettings(s model.Settings) {
	if a.proxy == nil {
		return
	}
	if !s.ProxyEnabled {
		a.proxy.Stop()
		return
	}
	if err := a.proxy.Start(s.ProxyPort); err != nil {
		log.Printf("[proxy] 启动失败: %v", err)
		return
	}
	log.Printf("[proxy] 已在 127.0.0.1:%d 监听", s.ProxyPort)
}

// ProxyStatus 返回模型代理的实际运行状态（供设置页轮询展示，不返回持久化意图）。
func (a *App) ProxyStatus() proxy.Status {
	if a.proxy == nil {
		return proxy.Status{}
	}
	return a.proxy.Status()
}

// SetProxyEnabled 切换模型代理开关（供托盘菜单调用），复用 SaveSettings 的校验与重启逻辑。
func (a *App) SetProxyEnabled(enabled bool) error {
	s := a.store.GetSettings()
	s.ProxyEnabled = enabled
	return a.SaveSettings(s)
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
	if isMac() {
		return "open"
	}
	return "xdg-open"
}
