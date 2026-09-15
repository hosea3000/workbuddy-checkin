package main

// launchAgentLabel 是 macOS LaunchAgent 的唯一标识。
const launchAgentLabel = "com.hosea3000.workbuddy-checkin"

// buildLaunchAgentPlist 生成 LaunchAgent plist 内容：登录时以 --hidden 静默启动。
// 放在平台无关文件中，便于在任意平台做单元测试。
func buildLaunchAgentPlist(exePath string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>` + launchAgentLabel + `</string>
	<key>ProgramArguments</key>
	<array>
		<string>` + exePath + `</string>
		<string>` + autoStartHiddenFlag + `</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
</dict>
</plist>
`
}
