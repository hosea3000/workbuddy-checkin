//go:build darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const launchAgentName = launchAgentLabel + ".plist"

// launchAgentDir 返回 ~/Library/LaunchAgents 目录。
func launchAgentDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents"), nil
}

// launchAgentPath 返回 plist 完整路径。
func launchAgentPath() (string, error) {
	dir, err := launchAgentDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, launchAgentName), nil
}

// setAutoStart 写入/移除 LaunchAgent，并经 launchctl 注册/注销（幂等）。
func setAutoStart(enable bool) error {
	path, err := launchAgentPath()
	if err != nil {
		return err
	}
	if !enable {
		// 未注册时 bootout 返回非零，忽略即可。
		exec.Command("launchctl", "bootout", "gui/"+uid(), path).Run()
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if dir, err := launchAgentDir(); err != nil {
		return err
	} else if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(buildLaunchAgentPlist(exe)), 0o644); err != nil {
		return err
	}
	// 重复 bootstrap 会报错，先 bootout 清理旧注册再注册。
	exec.Command("launchctl", "bootout", "gui/"+uid(), path).Run()
	if out, err := exec.Command("launchctl", "bootstrap", "gui/"+uid(), path).CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl bootstrap: %w: %s", err, out)
	}
	return nil
}

// autoStartEnabled 以 plist 是否实际存在为准。
func autoStartEnabled() bool {
	path, err := launchAgentPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// uid 返回当前用户 UID 字符串，用于 launchctl 的 gui/<uid> 目标。
func uid() string { return fmt.Sprint(os.Getuid()) }
