package main

import (
	"flag"
	"io"
)

// autoStartHiddenFlag 是自启命令行中用于静默启动的参数。
const autoStartHiddenFlag = "--hidden"

// buildAutoStartCommand 构造写入 Run key 的启动命令：exe 路径整体加引号（兼容含空格路径），追加 --hidden。
func buildAutoStartCommand(exePath string) string {
	return `"` + exePath + `" ` + autoStartHiddenFlag
}

// hiddenFromArgs 从命令行参数中解析 --hidden 标记；解析失败时视为未指定。
func hiddenFromArgs(args []string) bool {
	fs := flag.NewFlagSet("workbuddy-checkin", flag.ContinueOnError)
	hidden := fs.Bool("hidden", false, "启动时隐藏主窗口，直接进入托盘驻留")
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return false
	}
	return *hidden
}
