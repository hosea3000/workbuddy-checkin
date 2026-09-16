## RENAMED Requirements

- FROM: `### Requirement: 首次提示最小化到托盘`
- TO: `### Requirement: 首次提示隐藏到后台`

## MODIFIED Requirements

### Requirement: 首次提示隐藏到后台

系统 SHALL 在用户首次关闭主窗口、窗口被隐藏到后台（Windows 系统托盘 / macOS 菜单栏）时提示一次该行为，此后不再提示；提示文案 SHALL 平台无关地表达「应用仍在后台运行」。

#### Scenario: Windows 首次关闭窗口
- **WHEN** Windows 用户首次关闭主窗口
- **THEN** 系统提示一次「应用仍在后台运行」，此后不再提示

#### Scenario: macOS 首次关闭窗口
- **WHEN** macOS 用户首次关闭主窗口
- **THEN** 系统提示一次「应用仍在后台运行」，此后不再提示，且应用不退出

#### Scenario: 仅提示一次
- **WHEN** 用户第二次及以后关闭主窗口
- **THEN** 不再出现该提示
