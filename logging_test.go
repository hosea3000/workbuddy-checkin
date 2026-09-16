package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// newTestLogger 构造一个极小上限的轮转 logger，便于在测试中快速触发轮转。
// 不用 t.TempDir：lumberjack 的 mill 压缩 goroutine 在 Close 之后仍可能继续
// 创建/删除文件（它只在 millCh 关闭时退出，而 Close 不关闭它），会与
// t.TempDir 的 RemoveAll 清理竞态。这里自建目录并容忍清理错误。
func newTestLogger(t *testing.T) (*lumberjack.Logger, string) {
	t.Helper()
	dir, err := os.MkdirTemp("", "wb-log-rotate-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	lj := &lumberjack.Logger{
		Filename:   filepath.Join(dir, "app.log"),
		MaxSize:    1, // 1 MB
		MaxBackups: 3,
		MaxAge:     30,
		Compress:   true,
	}
	t.Cleanup(func() { _ = lj.Close() })
	return lj, dir
}

// listLogs 返回当前文件名集合。
func listLogs(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func count(names []string, pred func(string) bool) int {
	n := 0
	for _, name := range names {
		if pred(name) {
			n++
		}
	}
	return n
}

// isBackup 判断是否为历史文件：lumberjack 把时间戳插在扩展名之前，
// 即 `app-<时间戳>.log`（压缩后 `app-<时间戳>.log.gz`）。
func isBackup(name string) bool {
	return name != "app.log" && strings.HasPrefix(name, "app-") && strings.Contains(name, ".log")
}

// 写入超过单文件上限的内容后，应产生历史文件且当前文件被重置，日志不丢。
func TestLogRotationSplits(t *testing.T) {
	lj, dir := newTestLogger(t)

	chunk := strings.Repeat("x", 512*1024) + "\n" // 0.5 MB
	for i := 0; i < 4; i++ {                      // ~2MB，超过 1MB 上限
		if _, err := lj.Write([]byte(chunk)); err != nil {
			t.Fatal(err)
		}
	}
	if err := lj.Close(); err != nil {
		t.Fatal(err)
	}

	names := listLogs(t, dir)
	if count(names, func(n string) bool { return n == "app.log" }) != 1 {
		t.Fatalf("期望恰好 1 个当前文件，实际文件：%v", names)
	}
	if count(names, isBackup) == 0 {
		t.Fatalf("写入 2MB 超过 1MB 上限，期望产生历史文件，实际文件：%v", names)
	}
}

// Compress=true 时历史文件最终应被 gzip（压缩由 mill goroutine 异步完成）。
func TestLogRotationCompresses(t *testing.T) {
	lj, dir := newTestLogger(t)

	chunk := strings.Repeat("x", 512*1024) + "\n"
	for i := 0; i < 4; i++ {
		if _, err := lj.Write([]byte(chunk)); err != nil {
			t.Fatal(err)
		}
	}
	if err := lj.Close(); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if count(listLogs(t, dir), func(n string) bool { return strings.HasSuffix(n, ".gz") }) > 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("等待 3s 仍未出现 .gz 历史文件，实际文件：%v", listLogs(t, dir))
}

// 历史文件数量不应超过 MaxBackups。
func TestLogRotationCapsBackups(t *testing.T) {
	lj, dir := newTestLogger(t)

	chunk := strings.Repeat("y", 512*1024) + "\n" // 0.5 MB
	for i := 0; i < 20; i++ {                     // ~10MB，远超 1MB×3 的历史容量
		if _, err := lj.Write([]byte(chunk)); err != nil {
			t.Fatal(err)
		}
	}
	if err := lj.Close(); err != nil {
		t.Fatal(err)
	}

	// 压缩/清理异步，给 mill 一点时间收敛后再断言上界。
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if count(listLogs(t, dir), isBackup) <= 3 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("历史文件数超过 MaxBackups=3，实际文件：%v", listLogs(t, dir))
}
