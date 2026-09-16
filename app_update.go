package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/hosea3000/workbuddy-checkin/model"
	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// updateClient 是检查更新使用的 HTTP 客户端，包级变量便于测试替换。
var updateClient = &http.Client{Timeout: updateCheckTimeout}

// updateDownloadClient 是一键更新下载使用的 HTTP 客户端，超时远宽于检查请求。
var updateDownloadClient = &http.Client{Timeout: updateDownloadTimeout}

// GetVersion 返回当前应用版本号（发布构建由 CI 注入，本地开发为 dev）。
func (a *App) GetVersion() string { return version }

// CheckUpdate 检查 GitHub Release 最新版本；dev 版本短路，不发起网络请求。
// 发现新版本时缓存资产下载地址，供 DownloadAndApplyUpdate 直接使用。
func (a *App) CheckUpdate() model.UpdateCheckResult {
	if version == "" || version == "dev" {
		return model.UpdateCheckResult{
			Status:         model.UpdateStatusUpToDate,
			CurrentVersion: version,
			Message:        "当前为开发版本，跳过更新检查",
		}
	}
	proxy := ""
	if a.store != nil {
		proxy = a.store.GetSettings().UpdateProxy
	}
	result := checkForUpdates(updateClient, version, updateAPIBaseURL, proxy, runtime.GOOS, runtime.GOARCH)
	a.mu.Lock()
	if result.Status == model.UpdateStatusAvailable {
		a.updateDownloadURL = result.DownloadURL
		a.updateLatestVersion = result.LatestVersion
	} else {
		a.updateDownloadURL = ""
		a.updateLatestVersion = ""
	}
	a.mu.Unlock()
	if result.Status == model.UpdateStatusUpToDate {
		if exePath, err := os.Executable(); err == nil {
			cleanupPendingUpdateFiles(exePath)
		}
	}
	return result
}

// UpdateProgress 返回最近一次更新下载的进度（前端轮询）。
func (a *App) UpdateProgress() model.UpdateDownloadEvent {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.updateProgress
}

// PendingUpdateInfo 返回 exe 同目录是否存在已下载待应用（.new）的更新及其版本号。
func (a *App) PendingUpdateInfo() model.PendingUpdateInfo {
	if version == "" || version == "dev" || !isWindows() {
		return model.PendingUpdateInfo{}
	}
	exePath, err := os.Executable()
	if err != nil {
		return model.PendingUpdateInfo{}
	}
	_, newPath, _, _ := exeUpdatePaths(exePath)
	if _, err := os.Stat(newPath); err != nil {
		return model.PendingUpdateInfo{}
	}
	return model.PendingUpdateInfo{
		Exists:  true,
		Version: pendingUpdateVersion(exePath),
	}
}

// DownloadAndApplyUpdate 一键更新：异步下载新版本 exe 到 exe 同目录的 .part，
// 完成后落位为 .new 并通过 UpdateProgress 暴露进度。dev 版本与非 Windows 平台
// 短路不发起网络请求。返回空串表示已开始下载，否则返回同步失败文案。
func (a *App) DownloadAndApplyUpdate() string {
	if version == "" || version == "dev" {
		return "当前为开发版本，不支持自动更新"
	}
	if !isWindows() && !isMac() {
		return "当前平台不支持自动更新，请通过 GitHub 手动更新"
	}
	if a.ctx == nil {
		return "应用尚未就绪，请稍后重试"
	}
	a.mu.Lock()
	url := a.updateDownloadURL
	a.mu.Unlock()
	if url == "" {
		return "暂无可用更新，请先检查更新"
	}
	if a.store != nil {
		url = proxiedURL(a.store.GetSettings().UpdateProxy, url)
	}
	exePath, err := os.Executable()
	if err != nil {
		return "无法定位程序路径，无法自动更新"
	}
	if isMac() {
		return a.downloadAndApplyDarwin(url, exePath)
	}
	if !dirWritable(filepath.Dir(exePath)) {
		return "程序目录不可写，请通过「前往 GitHub 查看」手动更新"
	}
	partPath, newPath, versionPath, _ := exeUpdatePaths(exePath)
	a.setUpdateProgress(model.UpdateDownloadEvent{Phase: model.UpdateDownloadPhaseDownloading})
	go func() {
		if err := downloadUpdate(updateDownloadClient, url, partPath, newPath, a.setUpdateProgress); err != nil {
			a.setUpdateProgress(model.UpdateDownloadEvent{
				Phase:   model.UpdateDownloadPhaseError,
				Message: err.Error(),
			})
			return
		}
		// 记录 .new 对应的版本号，供确认弹窗与后续「重启更新」读取。
		a.mu.Lock()
		latest := a.updateLatestVersion
		a.mu.Unlock()
		if latest != "" {
			if err := os.WriteFile(versionPath, []byte(latest), 0o644); err != nil {
				a.setUpdateProgress(model.UpdateDownloadEvent{
					Phase:   model.UpdateDownloadPhaseError,
					Message: "写入版本标记失败，请通过 GitHub 手动更新",
				})
				cleanupPendingUpdateFiles(exePath)
				return
			}
		}
		// 下载完成：弹原生确认框；确认则退出替换重启，取消则等待用户稍后重启。
		ok, err := confirmRestartDialog(a.ctx, pendingUpdateVersion(exePath))
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
				Message: "新版本已下载，可稍后重启",
			})
			return
		}
		exePath, err := os.Executable()
		if err != nil {
			a.setUpdateProgress(model.UpdateDownloadEvent{
				Phase:   model.UpdateDownloadPhaseError,
				Message: "无法定位程序路径，请通过 GitHub 手动更新",
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

// ApplyUpdateAndRestart 确认重启：校验 .new 就绪后弹确认框，确认则生成一次性更新脚本
// 并走完整退出流程。返回空串表示已进入重启流程；「已取消」表示用户选择稍后；其余为失败文案。
func (a *App) ApplyUpdateAndRestart() string {
	if version == "" || version == "dev" {
		return "当前为开发版本，不支持自动更新"
	}
	if a.ctx == nil {
		return "应用尚未就绪，请稍后重试"
	}
	exePath, err := os.Executable()
	if err != nil {
		return "无法定位程序路径，无法自动更新"
	}
	if isMac() {
		return applyUpdateAndRestart(exePath, a.quit)
	}
	_, newPath, _, _ := exeUpdatePaths(exePath)
	if _, err := os.Stat(newPath); err != nil {
		return "未找到已下载的更新文件，请先点击「立即更新」"
	}
	ok, err := confirmRestartDialog(a.ctx, pendingUpdateVersion(exePath))
	if err != nil {
		return "确认对话框打开失败，请稍后重试"
	}
	if !ok {
		return "已取消"
	}
	return applyUpdateAndRestart(exePath, a.quit)
}

func (a *App) setUpdateProgress(ev model.UpdateDownloadEvent) {
	a.mu.Lock()
	a.updateProgress = ev
	a.mu.Unlock()
}

// confirmRestartDialog 弹出「新版本已就绪」原生确认框，返回用户是否选择立即重启。
// 注意：Windows 端 MessageDialog 使用原生 MessageBox，固定 Yes/No 按钮并返回
// "Yes"/"No"（Buttons 自定义文本被忽略）；其他平台返回自定义按钮文本。
// 因此除明确的取消值外一律视为确认。
func confirmRestartDialog(ctx context.Context, latestVersion string) (bool, error) {
	message := "新版本已下载完成，是否立即重启生效？\n（Yes 立即重启，No 稍后处理）"
	if latestVersion != "" {
		message = "新版本 v" + latestVersion + " 已下载完成，是否立即重启生效？\n（Yes 立即重启，No 稍后处理）"
	}
	choice, err := wruntime.MessageDialog(ctx, wruntime.MessageDialogOptions{
		Type:          wruntime.QuestionDialog,
		Title:         "更新就绪",
		Message:       message,
		Buttons:       []string{"立即重启", "稍后"},
		DefaultButton: "立即重启",
		CancelButton:  "稍后",
	})
	if err != nil {
		return false, err
	}
	switch choice {
	case "", "No", "Cancel", "稍后":
		return false, nil
	default:
		return true, nil
	}
}
