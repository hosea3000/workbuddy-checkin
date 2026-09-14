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
	a.tray = newTray(a.showWindow, a.quit, a.wake)
	return a
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// 清理上次更新遗留的 .part 残渣与孤儿版本标记（保留待应用的 .new）。
	if exePath, err := os.Executable(); err == nil {
		cleanupUpdateArtifacts(exePath)
	}
	a.scheduler.Start(ctx)
	// 窗口可见性由 main.go 的 StartHidden（--hidden 标记）决定，此处不再补 show。
	a.tray.Start(a.trayTip())
	a.tray.SetTip(a.trayTip())
	// 启动补签（托盘就绪后）
	go func() {
		time.Sleep(2 * time.Second)
		a.scheduler.CatchUp(a.ctx)
		a.refreshTrayTip()
	}()
	log.Printf("workbuddy-checkin started, data dir: %s", a.store.Dir())
}

func (a *App) shutdown(ctx context.Context) {
	if a.scheduler != nil {
		a.scheduler.Stop()
	}
}

// beforeClose 返回 true 阻止窗口关闭（即隐藏到托盘）。
// 主动退出时 quitting 已置位，须放行，否则最小化到托盘会吞掉退出。
func (a *App) beforeClose(ctx context.Context) bool {
	if a.quitting.Load() {
		return false
	}
	if !a.store.GetSettings().MinimizeToTray {
		return false
	}
	a.notifyCloseTipOnce()
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
		a.scheduler.Wake(a.ctx)
		a.refreshTrayTip()
	}()
}

func (a *App) notifyCloseTipOnce() {
	a.mu.Lock()
	first := !a.closeTipShown
	a.closeTipShown = true
	a.mu.Unlock()
	if first && a.notifier != nil {
		a.notifier.Notify("workbuddy-checkin", "已最小化到托盘，应用仍在后台运行")
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
		return "workbuddy-checkin — 有账号需重新登录"
	}
	return fmt.Sprintf("workbuddy-checkin — 今日已签到 %d/%d", checked, len(creds))
}

func (a *App) refreshTrayTip() {
	if a.tray != nil {
		a.tray.SetTip(a.trayTip())
	}
}

func (a *App) userDataDir() string { return a.store.Dir() }
