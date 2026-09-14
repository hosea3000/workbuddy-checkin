## MODIFIED Requirements

### Requirement: 静默启动
应用 SHALL 仅在由开机自启拉起时静默启动（不弹窗），直接进入托盘；手动启动（双击 exe）SHALL 显示主窗口，不得因已启用自启而静默。启动方式 SHALL 通过命令行标记 `--hidden` 区分。

#### Scenario: 自启静默
- **WHEN** 应用由开机自启启动（启动命令带 `--hidden`）
- **THEN** 不显示主窗口、直接进入托盘

#### Scenario: 手动启动显示窗口
- **WHEN** 用户手动双击 exe（无 `--hidden`），即使已启用开机自启
- **THEN** 主窗口正常显示
