package model

// Settings 应用设置。零值不对应默认值，需经 DefaultSettings 填充。
type Settings struct {
	CheckinHour   int  `json:"checkinHour"`
	CheckinMinute int  `json:"checkinMinute"`
	AutoStart     bool `json:"autoStart"`
}

// DefaultSettings 返回设计文档约定的默认值（PRD F7）。
func DefaultSettings() Settings {
	return Settings{
		CheckinHour:   9,
		CheckinMinute: 30,
		AutoStart:     false,
	}
}
