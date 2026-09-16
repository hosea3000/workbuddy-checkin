package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hosea3000/workbuddy-checkin/internal/account"
	"github.com/hosea3000/workbuddy-checkin/internal/checkin"
	"github.com/hosea3000/workbuddy-checkin/internal/proxy"
	"github.com/hosea3000/workbuddy-checkin/internal/scheduler"
	"github.com/hosea3000/workbuddy-checkin/model"
	"github.com/hosea3000/workbuddy-checkin/store"
	"github.com/hosea3000/workbuddy-checkin/upstream/codebuddy"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Notifier 是通知能力（Windows Toast / Linux stub）。
type Notifier interface {
	Notify(title, body string)
}

// App 是 Wails 绑定方法的唯一宿主，也是各子服务的装配点。
type App struct {
	ctx context.Context

	store     *store.Store
	client    *codebuddy.Client
	accounts  *account.Service
	checkin   *checkin.Service
	scheduler *scheduler.Scheduler
	proxy     *proxy.Service
	tray      *Tray
	notifier  Notifier

	mu            sync.Mutex
	closeTipShown bool
	quitting      atomic.Bool

	updateDownloadURL   string
	updateLatestVersion string
	updateProgress      model.UpdateDownloadEvent
}

// NewApp 构造应用并初始化存储与各服务。
func NewApp() *App {
	st, err := store.New("")
	if err != nil {
		log.Fatalf("init store: %v", err)
	}
	setupLogging(st.Dir())
	client := codebuddy.NewClient("", "")
	client.SetLogger(func(format string, args ...any) { log.Printf(format, args...) })
	a := &App{
		store:    st,
		client:   client,
		notifier: newNotifier(),
	}
	a.accounts = account.NewService(st, client, func(url string) error {
		if a.ctx == nil {
			return fmt.Errorf("app not started")
		}
		wruntime.BrowserOpenURL(a.ctx, url)
		return nil
	})
	a.checkin = checkin.NewService(st, client, a.notifier)
	a.scheduler = scheduler.New(st, a.checkin)
	a.proxy = proxy.NewService(client, proxy.NewCredentialResolver(st, a.checkin), a.checkin)
	a.tray = newTray(a.showWindow, a.quit, a.wake, a.toggleProxyFromTray)
	return a
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// 清理上次更新遗留的 .part 残渣与孤儿版本标记（保留待应用的 .new）。
	if exePath, err := os.Executable(); err == nil {
		cleanupUpdateArtifacts(exePath)
	}
	a.scheduler.Start(ctx)
	a.startProxyIfEnabled()
	// 窗口可见性由 main.go 的 StartHidden（--hidden 标记）决定，此处不再补 show。
	// macOS 兜底切到 Accessory：Wails 在 applicationWillFinishLaunching 写死 Regular，
	// 会覆盖 Info.plist 的 LSUIElement，故此处再设一次以隐藏 Dock 图标。
	setActivationPolicyAccessory()
	a.tray.Start(a.trayTip())
	a.tray.SetTip(a.trayTip())
	a.tray.SetProxyState(a.store.GetSettings().ProxyEnabled)
	// 启动即巡检（托盘就绪后）：当天未签到的账号立即补签
	go func() {
		time.Sleep(2 * time.Second)
		a.scheduler.RunAll(a.ctx)
		a.refreshTrayTip()
	}()
	log.Printf("workbuddy-checkin started, data dir: %s", a.store.Dir())
}

func (a *App) shutdown(ctx context.Context) {
	if a.scheduler != nil {
		a.scheduler.Stop()
	}
	if a.proxy != nil {
		a.proxy.Stop()
	}
}

// toggleProxyFromTray 由托盘菜单项调用：翻转模型代理开关并同步菜单勾选状态。
func (a *App) toggleProxyFromTray() {
	settings := a.store.GetSettings()
	if err := a.SetProxyEnabled(!settings.ProxyEnabled); err != nil {
		log.Printf("[proxy] 托盘切换失败: %v", err)
	}
}

// startProxyIfEnabled 按设置拉起模型代理服务；端口非法或绑定失败时仅记日志，
// 实际状态由 ProxyStatus() 暴露给设置页。
func (a *App) startProxyIfEnabled() {
	settings := a.store.GetSettings()
	if !settings.ProxyEnabled {
		return
	}
	if err := a.proxy.Start(settings.ProxyPort); err != nil {
		log.Printf("[proxy] 启动失败: %v", err)
		return
	}
	log.Printf("[proxy] 已在 127.0.0.1:%d 监听", settings.ProxyPort)
}

// beforeClose 决定窗口关闭行为。两平台均为「隐藏窗口、应用继续后台运行」：
// Windows 由 HideWindowOnClose 隐藏到托盘；macOS 由这里显式 WindowHide（orderOut）。
// 主动退出时 quitting 已置位，须放行。
func (a *App) beforeClose(ctx context.Context) bool {
	if a.quitting.Load() {
		return false
	}
	a.notifyCloseTipOnce()
	if isMac() {
		wruntime.WindowHide(ctx)
	}
	return true
}

func (a *App) quit() {
	if a.ctx == nil {
		return
	}
	a.quitting.Store(true)
	wruntime.Quit(a.ctx)
}

func (a *App) wake() {
	go func() {
		a.scheduler.RunAll(a.ctx)
		a.refreshTrayTip()
	}()
}

func (a *App) notifyCloseTipOnce() {
	a.mu.Lock()
	first := !a.closeTipShown
	a.closeTipShown = true
	a.mu.Unlock()
	if first && a.notifier != nil {
		a.notifier.Notify("WorkBuddy 自动签到", "已隐藏到后台，应用仍在运行")
	}
}

// showWindow 显示并聚焦主窗口（单实例唤起 / 托盘双击 / 托盘打开）。
func (a *App) showWindow() {
	if a.ctx == nil {
		return
	}
	wruntime.WindowShow(a.ctx)
	wruntime.WindowUnminimise(a.ctx)
}

// trayTip 生成托盘 tooltip。
func (a *App) trayTip() string {
	creds := a.store.ListCredentials()
	checked := 0
	relogin := 0
	now := time.Now()
	for _, c := range creds {
		if c.IsReloginRequired() {
			relogin++
			continue
		}
		if c.HasCheckedInToday(now) {
			checked++
		}
	}
	if relogin > 0 {
		return "WorkBuddy 自动签到 — 有账号需重新登录"
	}
	return fmt.Sprintf("WorkBuddy 自动签到 — 今日已签到 %d/%d", checked, len(creds))
}

func (a *App) refreshTrayTip() {
	if a.tray != nil {
		a.tray.SetTip(a.trayTip())
	}
}

func (a *App) userDataDir() string { return a.store.Dir() }
