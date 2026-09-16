package model

// DefaultProxyPort 是模型代理的默认监听端口。
const DefaultProxyPort = 18080

// Settings 应用设置。零值不对应默认值，需经 DefaultSettings 填充。
type Settings struct {
	AutoStart   bool   `json:"autoStart"`
	UpdateProxy string `json:"updateProxy"` // GitHub 下载加速代理前缀，空为直连

	ProxyEnabled       bool   `json:"proxyEnabled"`       // 模型代理开关
	ProxyPort          int    `json:"proxyPort"`          // 模型代理监听端口
	ActiveCredentialID string `json:"activeCredentialId"` // 当前凭证 ID，空表示未设置
}

// DefaultSettings 返回设计文档约定的默认值（PRD F7）。
func DefaultSettings() Settings {
	return Settings{
		AutoStart:    false,
		ProxyPort:    DefaultProxyPort,
		ProxyEnabled: false,
	}
}
