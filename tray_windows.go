//go:build windows

package main

import (
	"log"
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

// Win32 常量。
const (
	wmApp                = 0x8000
	wmDestroy            = 0x0002
	wmLButtonDblClk      = 0x0203
	wmRButtonUp          = 0x0205
	wmPowerBroadcast     = 0x0218
	wmWTSessionChange    = 0x02B1
	pbtApmResumeAuto     = 0x0007
	pbtApmResumeSuspend  = 0x0012
	notifyForThisSession = 0x0000
	wtsSessionLock       = 0x0007
	wtsSessionUnlock     = 0x0008
	nimAdd               = 0x0000
	nimModify            = 0x0001
	nimDelete            = 0x0002
	nifMessage           = 0x0001
	nifIcon              = 0x0002
	nifTip               = 0x0004
	appIconResourceID    = 3 // Wails 打包时以 winres.RT_ICON(=3) 写入图标组
	mfString             = 0x0000
	mfSeparator          = 0x0800
	tpmRightButton       = 0x0002
	tpmRetToCmd          = 0x0100
)

// 托盘菜单命令 ID。
const (
	cmdOpen = 1 + iota
	cmdQuit
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	wtsapi32 = syscall.NewLazyDLL("wtsapi32.dll")

	procRegisterClassEx  = user32.NewProc("RegisterClassExW")
	procCreateWindowEx   = user32.NewProc("CreateWindowExW")
	procDefWindowProc    = user32.NewProc("DefWindowProcW")
	procDestroyWindow    = user32.NewProc("DestroyWindow")
	procGetMessage       = user32.NewProc("GetMessageW")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procDispatchMessage  = user32.NewProc("DispatchMessageW")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")
	procCreatePopupMenu  = user32.NewProc("CreatePopupMenu")
	procAppendMenu       = user32.NewProc("AppendMenuW")
	procTrackPopupMenu   = user32.NewProc("TrackPopupMenu")
	procGetCursorPos     = user32.NewProc("GetCursorPos")
	procSetForegroundWnd = user32.NewProc("SetForegroundWindow")
	procLoadIcon         = user32.NewProc("LoadIconW")
	procGetModuleHandle  = kernel32.NewProc("GetModuleHandleW")

	procRegisterWindowMessage = user32.NewProc("RegisterWindowMessageW")

	procShellNotifyIcon = shell32.NewProc("Shell_NotifyIconW")
	procWTSRegister     = wtsapi32.NewProc("WTSRegisterSessionNotification")
)

type wndClassEx struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     syscall.Handle
	hIcon         syscall.Handle
	hCursor       syscall.Handle
	hbrBackground syscall.Handle
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       syscall.Handle
}

type notifyIconData struct {
	cbSize           uint32
	hWnd             syscall.Handle
	uID              uint32
	uFlags           uint32
	uCallbackMessage uint32
	hIcon            syscall.Handle
	szTip            [128]uint16
	dwState          uint32
	dwStateMask      uint32
	szInfo           [256]uint16
	uTimeoutOrVer    uint32
	szInfoTitle      [64]uint16
	dwInfoFlags      uint32
	guidItem         [16]byte
	hBalloonIcon     syscall.Handle
}

type msg struct {
	hwnd    syscall.Handle
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ x, y int32 }
}

type point struct{ x, y int32 }

// Tray 是 Windows 托盘实现。
type Tray struct {
	hwnd syscall.Handle

	onOpen func()
	onQuit func()
	onWake func()

	menu      syscall.Handle
	wmTaskbar uint32

	mu    sync.Mutex
	tip   string
	added bool
}

var (
	trayInstance *Tray
	trayWndProc  = syscall.NewCallback(trayWndProcFn)
)

// setActivationPolicyAccessory 仅在 macOS 有意义，Windows 为空操作。
func setActivationPolicyAccessory() {}

func newTray(onOpen, onQuit, onWake func()) *Tray {
	return &Tray{onOpen: onOpen, onQuit: onQuit, onWake: onWake}
}

// Start 创建消息窗口与托盘图标（须在独立 OS 线程上运行消息循环）。
func (t *Tray) Start(tip string) {
	t.tip = tip
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		if err := t.create(); err != nil {
			log.Printf("[tray] create failed: %v", err)
			return
		}
		t.loop()
	}()
}

func (t *Tray) create() error {
	hInst, _, _ := procGetModuleHandle.Call(0)
	className, _ := syscall.UTF16PtrFromString("WorkbuddyCheckinTray")

	wc := wndClassEx{
		cbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		lpfnWndProc:   trayWndProc,
		hInstance:     syscall.Handle(hInst),
		lpszClassName: className,
	}
	if atom, _, err := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc))); atom == 0 {
		return err
	}
	hwnd, _, err := procCreateWindowEx.Call(0, uintptr(unsafe.Pointer(className)), 0, 0, 0, 0, 0, 0, 0, 0, hInst, 0)
	if hwnd == 0 {
		return err
	}
	t.hwnd = syscall.Handle(hwnd)
	trayInstance = t
	// 注册 explorer.exe 重启广播消息（TaskbarCreated），用于重挂图标。
	if name, err := syscall.UTF16PtrFromString("TaskbarCreated"); err == nil {
		if res, _, _ := procRegisterWindowMessage.Call(uintptr(unsafe.Pointer(name))); res != 0 {
			t.wmTaskbar = uint32(res)
		}
	}
	t.buildMenu()
	t.addIcon()
	// 订阅会话解锁/锁定通知（WM_WTSSESSION_CHANGE）。
	procWTSRegister.Call(uintptr(t.hwnd), notifyForThisSession)
	return nil
}

func (t *Tray) buildMenu() {
	hMenu, _, _ := procCreatePopupMenu.Call()
	if hMenu == 0 {
		return
	}
	t.menu = syscall.Handle(hMenu)
	appendMenu(hMenu, mfString, cmdOpen, "打开主界面")
	appendMenu(hMenu, mfSeparator, 0, "")
	appendMenu(hMenu, mfString, cmdQuit, "退出")
}

func (t *Tray) addIcon() {
	hInst, _, _ := procGetModuleHandle.Call(0)
	hIcon, _, _ := procLoadIcon.Call(hInst, appIconResourceID)
	nid := notifyIconData{
		cbSize:           uint32(unsafe.Sizeof(notifyIconData{})),
		hWnd:             t.hwnd,
		uID:              1,
		uFlags:           nifMessage | nifIcon | nifTip,
		uCallbackMessage: wmApp + 1,
		hIcon:            syscall.Handle(hIcon),
	}
	copy(nid.szTip[:], syscall.StringToUTF16(t.tip))
	procShellNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))
	t.added = true
}

// SetTip 更新托盘 tooltip。
func (t *Tray) SetTip(tip string) {
	t.mu.Lock()
	t.tip = tip
	added := t.added
	t.mu.Unlock()
	if !added {
		return
	}
	nid := notifyIconData{
		cbSize: uint32(unsafe.Sizeof(notifyIconData{})),
		hWnd:   t.hwnd,
		uID:    1,
		uFlags: nifTip,
	}
	copy(nid.szTip[:], syscall.StringToUTF16(tip))
	procShellNotifyIcon.Call(nimModify, uintptr(unsafe.Pointer(&nid)))
}

func (t *Tray) remove() {
	nid := notifyIconData{cbSize: uint32(unsafe.Sizeof(notifyIconData{})), hWnd: t.hwnd, uID: 1}
	procShellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
}

func (t *Tray) loop() {
	var m msg
	for {
		ret, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
}

func (t *Tray) showMenu() {
	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procSetForegroundWnd.Call(uintptr(t.hwnd))
	// TPM_RETURNCMD：选中命令通过返回值返回，系统不投递 WM_COMMAND。
	res, _, _ := procTrackPopupMenu.Call(uintptr(t.menu), tpmRightButton|tpmRetToCmd, uintptr(pt.x), uintptr(pt.y), 0, uintptr(t.hwnd), 0)
	switch uint32(res) {
	case cmdOpen:
		if t.onOpen != nil {
			t.onOpen()
		}
	case cmdQuit:
		t.remove()
		if t.onQuit != nil {
			t.onQuit()
		}
	}
}

func appendMenu(hMenu uintptr, flags uintptr, id uintptr, text string) {
	ptr, _ := syscall.UTF16PtrFromString(text)
	procAppendMenu.Call(hMenu, flags, id, uintptr(unsafe.Pointer(ptr)))
}

func trayWndProcFn(hwnd syscall.Handle, message uint32, wParam, lParam uintptr) uintptr {
	t := trayInstance
	if t == nil {
		ret, _, _ := procDefWindowProc.Call(uintptr(hwnd), uintptr(message), wParam, lParam)
		return ret
	}
	switch message {
	case wmApp + 1:
		switch uint32(lParam) {
		case wmLButtonDblClk:
			if t.onOpen != nil {
				t.onOpen()
			}
		case wmRButtonUp:
			t.showMenu()
		}
	case t.wmTaskbar:
		// explorer.exe 重启后重挂图标。
		t.addIcon()
	case wmPowerBroadcast:
		if uint32(wParam) == pbtApmResumeAuto || uint32(wParam) == pbtApmResumeSuspend {
			t.addIcon()
			if t.onWake != nil {
				t.onWake()
			}
		}
	case wmWTSessionChange:
		// 会话解锁（0x8）后重挂图标并按补签逻辑处理。
		if uint32(wParam) == wtsSessionUnlock {
			t.addIcon()
			if t.onWake != nil {
				t.onWake()
			}
		}
	case wmDestroy:
		procPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := procDefWindowProc.Call(uintptr(hwnd), uintptr(message), wParam, lParam)
	return ret
}
