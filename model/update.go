package model

// 更新检查状态常量。
const (
	UpdateStatusUpToDate  = "up-to-date"
	UpdateStatusAvailable = "update-available"
	UpdateStatusError     = "error"
)

// UpdateCheckResult 是一次更新检查的结果。
type UpdateCheckResult struct {
	// Status 取值：up-to-date / update-available / error。
	Status         string `json:"status"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	ReleaseURL     string `json:"releaseUrl"`
	// DownloadURL 是 workbuddy-checkin.exe 资产的下载地址；资产缺失时为空。
	DownloadURL string `json:"downloadUrl"`
	Message     string `json:"message"`
}

// 更新下载事件阶段常量。
const (
	UpdateDownloadPhaseDownloading = "downloading"
	UpdateDownloadPhaseCompleted   = "completed"
	UpdateDownloadPhaseCancelled   = "cancelled"
	UpdateDownloadPhaseError       = "error"
)

// UpdateDownloadEvent 是一次更新下载的进度事件，通过 Wails events（update:progress）推送给前端。
type UpdateDownloadEvent struct {
	// Phase 取值：downloading / completed / cancelled / error。
	Phase      string `json:"phase"`
	Downloaded int64  `json:"downloaded"`
	Total      int64  `json:"total"`
	Percent    int    `json:"percent"`
	// Message 为错误阶段的提示文案。
	Message string `json:"message"`
}

// PendingUpdateInfo 描述已下载待应用（.new 就绪）的更新状态。
type PendingUpdateInfo struct {
	// Exists 表示 exe 同目录是否存在 .new 待应用文件。
	Exists bool `json:"exists"`
	// Version 是 .new 对应的版本号（来自 .new.version 元数据），读取失败时为空。
	Version string `json:"version"`
}
