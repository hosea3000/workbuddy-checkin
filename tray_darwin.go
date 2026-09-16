//go:build darwin

package main

/*
#cgo darwin CFLAGS: -DDARWIN -x objective-c -fobjc-arc
#cgo darwin LDFLAGS: -framework Cocoa

#include <stdlib.h>
#include "tray_darwin.h"
*/
import "C"

import (
	_ "embed"
	"unsafe"
)

//go:embed build/darwin/trayTemplate@2x.png
var trayIconPNG []byte

// Tray 是 macOS 菜单栏（NSStatusItem）实现，接口与 tray_windows.go 保持一致。
type Tray struct {
	onOpen func()
	onQuit func()
	onWake func()

	started bool
	tip     string
}

var trayInstance *Tray

func newTray(onOpen, onQuit, onWake func()) *Tray {
	return &Tray{onOpen: onOpen, onQuit: onQuit, onWake: onWake}
}

// Start 在主线程创建菜单栏图标。tip 仅作初始 tooltip；systemtray 就绪后由 SetTip 更新。
func (t *Tray) Start(tip string) {
	trayInstance = t
	t.tip = tip
	var icon unsafe.Pointer
	if len(trayIconPNG) > 0 {
		icon = unsafe.Pointer(&trayIconPNG[0])
	}
	C.woTrayStart(icon, C.int(len(trayIconPNG)))
	t.started = true
}

// SetTip 更新菜单栏 tooltip（可空实现：macOS 状态栏项无原生 tooltip，保留接口一致性）。
func (t *Tray) SetTip(tip string) {
	t.tip = tip
	cs := C.CString(tip)
	C.woTraySetTip(cs)
	C.free(unsafe.Pointer(cs))
}

// setActivationPolicyAccessory 把应用切到 Accessory（无 Dock 图标），
// 用于兜底覆盖 Wails 在 applicationWillFinishLaunching 里写死的 Regular。
func setActivationPolicyAccessory() {
	C.woSetAccessoryPolicy()
}

//export woTrayOpen
func woTrayOpen() {
	if trayInstance != nil && trayInstance.onOpen != nil {
		trayInstance.onOpen()
	}
}

//export woTrayQuit
func woTrayQuit() {
	if trayInstance != nil && trayInstance.onQuit != nil {
		trayInstance.onQuit()
	}
}
