//go:build !windows

package main

// applyUpdateAndRestart 在非 Windows 平台不可用，返回提示文案。
func applyUpdateAndRestart(exePath string, quit func()) string {
	return "当前平台不支持自动更新，请通过 GitHub 手动更新"
}

// cleanupUpdateArtifacts 在非 Windows 平台为空操作。
func cleanupUpdateArtifacts(exePath string) {}
