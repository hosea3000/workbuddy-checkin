package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/hosea3000/workbuddy-checkin/model"
)

// 一键更新相关的文件名后缀与事件名常量。
const (
	updatePartSuffix    = ".part"        // 下载中的临时文件后缀
	updateNewSuffix     = ".new"         // 下载完成、待确认替换的文件后缀
	updateVersionSuffix = ".new.version" // 与 .new 配套的版本元数据文件
	updateBatName       = "workbuddy-checkin-update.bat"
	updateMoveRetries   = 3 // 覆盖 exe 的最大重试次数
)

// exeUpdatePaths 计算 exe 同目录下的 .part / .new / 版本元数据 / bat 脚本路径。
func exeUpdatePaths(exePath string) (partPath, newPath, versionPath, batPath string) {
	dir := filepath.Dir(exePath)
	base := filepath.Base(exePath)
	return filepath.Join(dir, base+updatePartSuffix),
		filepath.Join(dir, base+updateNewSuffix),
		filepath.Join(dir, base+updateVersionSuffix),
		filepath.Join(dir, updateBatName)
}

// updateDownloadDir 返回下载落位目录：Windows 为 exe 同目录（便于自替换），
// macOS 为 ~/Downloads（.app 内不可写，且 dmg 由用户拖入安装）。
func updateDownloadDir(exePath string) string {
	if isMac() {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, "Downloads")
		}
	}
	return filepath.Dir(exePath)
}

// downloadPaths 计算下载中/落位文件的路径（不含 Windows 专属的版本标记与 bat）。
func downloadPaths(exePath, assetName string) (partPath, newPath string) {
	dir := updateDownloadDir(exePath)
	return filepath.Join(dir, assetName+updatePartSuffix),
		filepath.Join(dir, assetName)
}

// dirWritable 探测目录可写性：尝试创建并删除临时文件，失败返回 false。
func dirWritable(dir string) bool {
	f, err := os.CreateTemp(dir, ".workbuddy-checkin-write-probe-*")
	if err != nil {
		return false
	}
	name := f.Name()
	f.Close()
	os.Remove(name)
	return true
}

// removePartArtifact 删除 exe 同目录的 .part 下载残渣（不存在时忽略）。
func removePartArtifact(exePath string) {
	partPath, _, _, _ := exeUpdatePaths(exePath)
	os.Remove(partPath)
}

// cleanupPendingUpdateFiles 删除 .new 与配套的 .new.version（已是最新时的清理）。
func cleanupPendingUpdateFiles(exePath string) {
	_, newPath, versionPath, _ := exeUpdatePaths(exePath)
	os.Remove(newPath)
	os.Remove(versionPath)
}

// cleanupStaleVersionArtifact 清理孤儿 .new.version：.new 不存在时其版本标记无意义。
func cleanupStaleVersionArtifact(exePath string) {
	_, newPath, versionPath, _ := exeUpdatePaths(exePath)
	if _, err := os.Stat(newPath); err != nil {
		os.Remove(versionPath)
	}
}

// pendingUpdateVersion 读取 .new 配套的版本号；.new 不存在时返回空串。
func pendingUpdateVersion(exePath string) string {
	_, newPath, versionPath, _ := exeUpdatePaths(exePath)
	if _, err := os.Stat(newPath); err != nil {
		return ""
	}
	v, err := os.ReadFile(versionPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(v))
}

// downloadUpdate 将 url 流式下载到 partPath，完成后重命名为 newPath，并沿
// progress 回调推送进度。任何失败都会清理 .part 残渣并返回错误。
func downloadUpdate(client *http.Client, url, partPath, newPath string, progress func(model.UpdateDownloadEvent)) error {
	os.Remove(partPath)
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("下载请求失败：%w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载响应异常（HTTP %d）", resp.StatusCode)
	}
	total := resp.ContentLength
	if total < 0 {
		total = 0
	}
	out, err := os.Create(partPath)
	if err != nil {
		return fmt.Errorf("创建下载文件失败：%w", err)
	}
	// 节流推送进度：每累计 256 KiB 推送一次，避免高频回调开销。
	// 总量未知（无 Content-Length 或 gzip 自动解压）时 percent 恒为 0，
	// 前端按已下载字节显示进度。
	const emitEvery = 256 * 1024
	buf := make([]byte, 64*1024)
	var downloaded, lastEmitted int64
	emit := func() {
		if downloaded-lastEmitted < emitEvery && downloaded < total {
			return
		}
		lastEmitted = downloaded
		percent := 0
		if total > 0 {
			percent = int(downloaded * 100 / total)
		}
		if progress != nil {
			progress(model.UpdateDownloadEvent{
				Phase:      model.UpdateDownloadPhaseDownloading,
				Downloaded: downloaded,
				Total:      total,
				Percent:    percent,
			})
		}
	}
	// 开始即推送一次，让 UI 立即进入下载反馈（连接阶段不再零反馈）
	emit()
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				out.Close()
				os.Remove(partPath)
				return fmt.Errorf("写入下载文件失败：%w", writeErr)
			}
			downloaded += int64(n)
			emit()
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			out.Close()
			os.Remove(partPath)
			return fmt.Errorf("读取下载内容失败：%w", readErr)
		}
	}
	if err := out.Close(); err != nil {
		os.Remove(partPath)
		return fmt.Errorf("关闭下载文件失败：%w", err)
	}
	if err := os.Rename(partPath, newPath); err != nil {
		os.Remove(partPath)
		return fmt.Errorf("下载文件落位失败：%w", err)
	}
	if progress != nil {
		progress(model.UpdateDownloadEvent{
			Phase:      model.UpdateDownloadPhaseCompleted,
			Downloaded: downloaded,
			Total:      total,
			Percent:    100,
		})
	}
	return nil
}

// escapeBatName 转义 bat 内嵌的文件名：% 需写作 %%（引号内的其他特殊字符安全）。
// 路径一律通过 %~dp0 引用，脚本内容不拼接绝对路径，减少注入面。
func escapeBatName(name string) string {
	return strings.ReplaceAll(name, "%", "%%")
}

// buildUpdateBat 生成一次性更新脚本：等待旧进程退出 → move /y 覆盖 exe
// （失败重试 updateMoveRetries 次，仍失败则启动旧版并保留 .new）→ 启动新版本
// → 删除脚本自身。脚本与 exe 同目录，使用 %~dp0 引用自身路径。
func buildUpdateBat(exeName string) string {
	name := escapeBatName(exeName)
	var b strings.Builder
	b.WriteString("@echo off\r\n")
	b.WriteString("setlocal EnableExtensions\r\n")
	b.WriteString("rem 等待应用进程完全退出（单实例锁随之释放）\r\n")
	b.WriteString(":wait_loop\r\n")
	b.WriteString("tasklist /fi \"imagename eq " + name + "\" 2>nul | findstr /i \"" + name + "\" >nul\r\n")
	b.WriteString("if not errorlevel 1 (\r\n")
	b.WriteString("  ping -n 2 127.0.0.1 >nul\r\n")
	b.WriteString("  goto wait_loop\r\n")
	b.WriteString(")\r\n")
	b.WriteString("rem 覆盖 exe，失败重试；重试耗尽则启动旧版并保留 .new\r\n")
	b.WriteString("set tries=0\r\n")
	b.WriteString(":move_loop\r\n")
	b.WriteString("move /y \"%~dp0" + name + updateNewSuffix + "\" \"%~dp0" + name + "\" >nul 2>&1\r\n")
	b.WriteString("if not errorlevel 1 goto move_done\r\n")
	b.WriteString("set /a tries+=1\r\n")
	b.WriteString("if %tries% lss " + fmt.Sprint(updateMoveRetries) + " (\r\n")
	b.WriteString("  ping -n 2 127.0.0.1 >nul\r\n")
	b.WriteString("  goto move_loop\r\n")
	b.WriteString(")\r\n")
	b.WriteString("start \"\" \"%~dp0" + name + "\"\r\n")
	b.WriteString("goto cleanup\r\n")
	b.WriteString(":move_done\r\n")
	b.WriteString("del \"%~dp0" + name + updateVersionSuffix + "\" >nul 2>&1\r\n")
	b.WriteString("start \"\" \"%~dp0" + name + "\"\r\n")
	b.WriteString(":cleanup\r\n")
	b.WriteString("del \"%~f0\" >nul 2>&1\r\n")
	b.WriteString("exit /b 0\r\n")
	return b.String()
}
