package main

import (
	"runtime"

	"github.com/hosea3000/workbuddy-checkin/model"
)

// downloadAndApplyDarwin 在 macOS 上把更新 dmg 下载到 ~/Downloads，完成后弹确认框，
// 确认则挂载 dmg 引导安装并退出应用。返回空串表示已开始下载，否则为同步失败文案。
func (a *App) downloadAndApplyDarwin(url, exePath string) string {
	if !dirWritable(updateDownloadDir(exePath)) {
		return "下载目录不可写，请通过「前往 GitHub 查看」手动更新"
	}
	partPath, dmgPath := downloadPaths(exePath, updateAssetNameFor(runtime.GOOS, runtime.GOARCH))
	a.setUpdateProgress(model.UpdateDownloadEvent{Phase: model.UpdateDownloadPhaseDownloading})
	go func() {
		if err := downloadUpdate(updateDownloadClient, url, partPath, dmgPath, a.setUpdateProgress); err != nil {
			a.setUpdateProgress(model.UpdateDownloadEvent{
				Phase:   model.UpdateDownloadPhaseError,
				Message: err.Error(),
			})
			return
		}
		a.mu.Lock()
		latest := a.updateLatestVersion
		a.mu.Unlock()
		ok, err := confirmRestartDialog(a.ctx, latest)
		if err != nil {
			a.setUpdateProgress(model.UpdateDownloadEvent{
				Phase:   model.UpdateDownloadPhaseError,
				Message: "确认对话框打开失败，请稍后重试",
			})
			return
		}
		if !ok {
			a.setUpdateProgress(model.UpdateDownloadEvent{
				Phase:   model.UpdateDownloadPhaseCancelled,
				Message: "新版本已下载到「下载」文件夹，可稍后安装",
			})
			return
		}
		if msg := applyUpdateAndRestart(exePath, a.quit); msg != "" {
			a.setUpdateProgress(model.UpdateDownloadEvent{
				Phase:   model.UpdateDownloadPhaseError,
				Message: msg,
			})
		}
	}()
	return ""
}
