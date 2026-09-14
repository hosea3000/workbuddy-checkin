package model

// Settings 应用设置。零值不对应默认值，需经 DefaultSettings 填充。
type Settings struct {
	CheckinHour        int  `json:"checkinHour"`
	CheckinMinute      int  `json:"checkinMinute"`
	CatchUpOnStart     bool `json:"catchUpOnStart"`
	RetryOnFailure     bool `json:"retryOnFailure"`
	NotifySuccess      bool `json:"notifySuccess"`
	NotifyFailure      bool `json:"notifyFailure"`
	AutoStart          bool `json:"autoStart"`
	MinimizeToTray     bool `json:"minimizeToTrayOnClose"`
	CheckUpdateOnStart bool `json:"checkUpdateOnStart"`
}

// DefaultSettings 返回设计文档约定的默认值（PRD F7）。
func DefaultSettings() Settings {
	return Settings{
		CheckinHour:        9,
		CheckinMinute:      30,
		CatchUpOnStart:     true,
		RetryOnFailure:     true,
		NotifySuccess:      true,
		NotifyFailure:      true,
		AutoStart:          false,
		MinimizeToTray:     true,
		CheckUpdateOnStart: true,
	}
}
