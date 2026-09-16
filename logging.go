package main

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

// 日志轮转参数：单文件 5MB 切分，保留 3 个历史文件、最长 30 天，历史文件 gzip 压缩。
const (
	logMaxSizeMB  = 5
	logMaxBackups = 3
	logMaxAgeDays = 30
)

// setupLogging 将日志写入数据目录的 app.log（不含令牌），超过 5MB 自动轮转。
// Wails 构建的是 windowsgui 子系统（无控制台），os.Stderr 是无效句柄；
// io.MultiWriter 遇到第一个错误就停止，会导致文件也写不进去。
// 因此：文件为主输出，stderr 仅在可写时附加（wails dev 下有控制台）。
// lumberjack 惰性建文件，目录不可写时不阻断启动（与原 OpenFile 失败即 return 等价）。
func setupLogging(dir string) {
	log.SetFlags(log.LstdFlags)
	lj := &lumberjack.Logger{
		Filename:   filepath.Join(dir, "app.log"),
		MaxSize:    logMaxSizeMB,
		MaxBackups: logMaxBackups,
		MaxAge:     logMaxAgeDays,
		Compress:   true,
	}
	if stderrWorks() {
		log.SetOutput(io.MultiWriter(lj, os.Stderr))
	} else {
		log.SetOutput(lj)
	}
}

// stderrWorks 探测 stderr 是否可用（windowsgui 子系统下不可用）。
func stderrWorks() bool {
	_, err := os.Stderr.WriteString("")
	return err == nil
}
