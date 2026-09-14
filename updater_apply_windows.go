//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows"
)

// applyUpdateAndRestart 生成一次性更新脚本并脱离启动，随后回调 quit 触发完整退出流程。
// 脚本等待进程退出后覆盖 exe 并启动新版本，因此 quit 在脚本启动后立即调用即可。
// 返回空串表示成功，否则返回面向用户的失败文案。
func applyUpdateAndRestart(exePath string, quit func()) string {
	_, newPath, _, batPath := exeUpdatePaths(exePath)
	if _, err := os.Stat(newPath); err != nil {
		return "未找到已下载的更新文件，请先点击「立即更新」"
	}
	if err := os.WriteFile(batPath, []byte(buildUpdateBat(filepath.Base(exePath))), 0o644); err != nil {
		return "生成更新脚本失败，请通过 GitHub 手动更新"
	}
	// CREATE_NO_WINDOW：脚本以无窗口方式运行，避免控制台窗口闪现。
	cmd := exec.Command("cmd", "/c", batPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: windows.CREATE_NO_WINDOW}
	if err := cmd.Start(); err != nil {
		os.Remove(batPath)
		return "启动更新脚本失败，请通过 GitHub 手动更新"
	}
	quit()
	return ""
}

// cleanupUpdateArtifacts 启动时清理 exe 同目录的 .part 下载残渣与孤儿 .new.version，
// 保留 .new 待确认缓存及其配套版本标记。
func cleanupUpdateArtifacts(exePath string) {
	removePartArtifact(exePath)
	cleanupStaleVersionArtifact(exePath)
}
