package main

import "runtime"

// isWindows 判断当前平台。
func isWindows() bool { return runtime.GOOS == "windows" }
