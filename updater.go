package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/hosea3000/workbuddy-checkin/model"
)

// GitHub 仓库信息与检查超时。
const (
	updateRepoOwner       = "hosea3000"
	updateRepoName        = "workbuddy-checkin"
	updateCheckTimeout    = 10 * time.Second
	updateDownloadTimeout = 10 * time.Minute // 下载 exe 的总超时，远宽于检查请求
	updateResponseLimit   = 1 << 20          // 1 MiB，防御异常响应体
)

// updateAPIBaseURL 是 GitHub API 基址；变量便于测试注入 httptest 地址。
var updateAPIBaseURL = "https://api.github.com"

// 待更新的可执行文件资产名（与 CI 发布的产物名一致）。
const updateAssetName = "workbuddy-checkin.exe"

// updateAssetNameFor 按平台选择更新资产名：Windows 为 exe，macOS 按架构选择 dmg。
// 命名须与 CI 发布的产物名严格一致，否则更新会静默失效（由单元测试锁死）。
func updateAssetNameFor(goos, goarch string) string {
	if goos == "darwin" {
		return "WorkBuddy-checkin-" + goarch + ".dmg"
	}
	return updateAssetName
}

// githubAsset 是 GitHub release 资产条目的最小子集。
type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// githubRelease 是 GitHub releases/latest API 响应的最小子集。
type githubRelease struct {
	TagName string        `json:"tag_name"`
	HTMLURL string        `json:"html_url"`
	Assets  []githubAsset `json:"assets"`
}

// findAssetDownloadURL 按文件名精确匹配资产并返回其下载地址；未匹配时返回空串。
func findAssetDownloadURL(assets []githubAsset, name string) string {
	for _, asset := range assets {
		if asset.Name == name {
			return asset.BrowserDownloadURL
		}
	}
	return ""
}

// proxiedURL 用加速代理前缀包装 GitHub 地址（如 gh-proxy.com 系列），检查与下载共用。
// proxy 为空则直连返回原地址；缺少 scheme 时补 https://；忽略末尾斜杠。
func proxiedURL(proxy, rawURL string) string {
	proxy = strings.TrimSpace(proxy)
	if proxy == "" {
		return rawURL
	}
	if !strings.Contains(proxy, "://") {
		proxy = "https://" + proxy
	}
	return strings.TrimRight(proxy, "/") + "/" + rawURL
}

// compareVersions 语义化版本比较：a<b 返回 -1，a==b 返回 0，a>b 返回 1。
// 比较前剥离前导 v；pre-release 后缀视为低于对应的正式版本；任一侧无法解析时返回 0（容错为无更新）。
func compareVersions(a, b string) int {
	va := parseVersionNumbers(a)
	vb := parseVersionNumbers(b)
	if va == nil || vb == nil {
		return 0
	}
	for i := 0; i < 3; i++ {
		if va[i] != vb[i] {
			if va[i] < vb[i] {
				return -1
			}
			return 1
		}
	}
	pa, pb := hasPreRelease(a), hasPreRelease(b)
	switch {
	case pa && !pb:
		return -1
	case !pa && pb:
		return 1
	default:
		return 0
	}
}

// parseVersionNumbers 解析主/次/补丁三段数字；带前导 v 则剥离；- 之后的 pre-release 段忽略。
// 非法版本号（空段、非数字）返回 nil。
func parseVersionNumbers(v string) []int {
	core := strings.TrimPrefix(v, "v")
	if i := strings.IndexByte(core, '-'); i >= 0 {
		core = core[:i]
	}
	parts := strings.Split(core, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return nil
	}
	nums := make([]int, 3)
	for i, part := range parts {
		if part == "" {
			return nil
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return nil
		}
		nums[i] = n
	}
	return nums
}

// hasPreRelease 判断版本号是否带 pre-release 后缀（如 0.2.0-beta）。
func hasPreRelease(v string) bool {
	return strings.Contains(strings.TrimPrefix(v, "v"), "-")
}

// isUpToDate 当前版本不低于最新版本时视为已是最新。
func isUpToDate(current, latest string) bool {
	return compareVersions(current, latest) >= 0
}

// checkForUpdates 请求 GitHub releases/latest 并映射为三态结果。client 与 baseURL 由调用方注入，便于测试。
// 配置了代理时优先经代理请求；代理不可用（部分代理不支持 api.github.com）时回退直连。
func checkForUpdates(client *http.Client, currentVersion, baseURL, proxy string) model.UpdateCheckResult {
	if proxy != "" {
		if result := fetchLatestRelease(client, currentVersion, proxiedURL(proxy, baseURL)); result.Status != model.UpdateStatusError {
			return result
		}
	}
	return fetchLatestRelease(client, currentVersion, baseURL)
}

// fetchLatestRelease 发起一次 releases/latest 请求并解析结果。
func fetchLatestRelease(client *http.Client, currentVersion, baseURL string) model.UpdateCheckResult {
	result := model.UpdateCheckResult{
		Status:         model.UpdateStatusError,
		CurrentVersion: currentVersion,
	}
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", baseURL, updateRepoOwner, updateRepoName)
	resp, err := client.Get(url)
	if err != nil {
		result.Message = "检查更新失败：网络异常，请稍后重试"
		return result
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusNotFound:
		result.Message = "检查更新失败：仓库暂无发布版本"
		return result
	case http.StatusOK:
		// 继续解析
	case http.StatusForbidden, http.StatusTooManyRequests:
		result.Message = "检查更新失败：请求过于频繁，请稍后再试"
		return result
	default:
		result.Message = fmt.Sprintf("检查更新失败：服务暂时不可用（HTTP %d）", resp.StatusCode)
		return result
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, updateResponseLimit))
	if err != nil {
		result.Message = "检查更新失败：响应读取异常"
		return result
	}
	var release githubRelease
	if err := json.Unmarshal(body, &release); err != nil || strings.TrimSpace(release.TagName) == "" {
		result.Message = "检查更新失败：响应解析异常"
		return result
	}
	result.DownloadURL = findAssetDownloadURL(release.Assets, updateAssetNameFor(runtime.GOOS, runtime.GOARCH))
	latest := strings.TrimPrefix(strings.TrimSpace(release.TagName), "v")
	if isUpToDate(currentVersion, latest) {
		result.Status = model.UpdateStatusUpToDate
		result.Message = "已是最新版本"
		return result
	}
	result.Status = model.UpdateStatusAvailable
	result.LatestVersion = latest
	result.ReleaseURL = release.HTMLURL
	result.Message = "发现新版本"
	return result
}
