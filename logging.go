package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
)

// setupLogging 将日志写入数据目录的 app.log（不含令牌）。
// Wails 构建的是 windowsgui 子系统（无控制台），os.Stderr 是无效句柄；
// io.MultiWriter 遇到第一个错误就停止，会导致文件也写不进去。
// 因此：文件为主输出，stderr 仅在可写时附加（wails dev 下有控制台）。
func setupLogging(dir string) {
	log.SetFlags(log.LstdFlags)
	f, err := os.OpenFile(filepath.Join(dir, "app.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return
	}
	if stderrWorks() {
		log.SetOutput(io.MultiWriter(f, os.Stderr))
	} else {
		log.SetOutput(f)
	}
}

// stderrWorks 探测 stderr 是否可用（windowsgui 子系统下不可用）。
func stderrWorks() bool {
	_, err := os.Stderr.WriteString("")
	return err == nil
}
