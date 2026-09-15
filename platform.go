package main

import "runtime"

// isWindows 判断当前平台。
func isWindows() bool { return runtime.GOOS == "windows" }

// isMac 判断当前平台是否为 macOS。
func isMac() bool { return runtime.GOOS == "darwin" }
