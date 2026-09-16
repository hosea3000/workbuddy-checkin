//go:build darwin

package main

import (
	"os"
	"os/exec"
	"runtime"
)

// applyUpdateAndRestart 在 macOS 上打开已下载的更新 dmg，由系统挂载并引导用户
// 拖入 Applications 完成安装，随后回调 quit 退出应用。不做原地替换。
// 返回空串表示成功，否则返回面向用户的失败文案。
func applyUpdateAndRestart(exePath string, quit func()) string {
	_, dmgPath := downloadPaths(exePath, updateAssetNameFor(runtime.GOOS, runtime.GOARCH))
	if _, err := os.Stat(dmgPath); err != nil {
		return "未找到已下载的更新文件，请先点击「立即更新」"
	}
	if err := exec.Command("open", dmgPath).Start(); err != nil {
		return "打开安装包失败，请手动打开「下载」文件夹中的 dmg"
	}
	quit()
	return ""
}

// cleanupUpdateArtifacts 启动时清理 ~/Downloads 中的下载残渣（.part）。
func cleanupUpdateArtifacts(exePath string) {
	partPath, _ := downloadPaths(exePath, updateAssetNameFor(runtime.GOOS, runtime.GOARCH))
	os.Remove(partPath)
}
