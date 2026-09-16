## ADDED Requirements

### Requirement: 设置页关于区块

设置页 SHALL 提供「关于」区块，展示以下信息：

| 项 | 内容 | 交互 |
|---|---|---|
| 开源地址 | `https://github.com/hosea3000/workbuddy-checkin` | 点击用系统默认浏览器打开 |
| 许可证 | MIT | 仅展示 |
| 版本 | 应用运行时版本（本地开发为 `dev`） | 仅展示 |

点击开源地址 SHALL 通过系统默认浏览器打开该仓库，SHALL NOT 在应用窗口内导航。

版本号 SHALL 只在「关于」区块展示；「软件更新」区块 SHALL NOT 再展示当前版本号，只保留更新状态与操作按钮。

#### Scenario: 关于区块展示三项信息
- **WHEN** 用户打开设置页
- **THEN** 页面显示「关于」区块，依次展示开源地址、许可证（MIT）与当前版本号

#### Scenario: 点击开源地址跳转浏览器
- **WHEN** 用户点击关于区块中的开源地址
- **THEN** 系统默认浏览器打开 `https://github.com/hosea3000/workbuddy-checkin`，应用窗口内容不变

#### Scenario: 版本号只在关于区块展示
- **WHEN** 用户打开设置页
- **THEN** 只有「关于」区块出现「当前版本」字样，「软件更新」区块不出现版本号

#### Scenario: 开发版本显示 dev
- **WHEN** 应用以本地开发方式运行（版本号为 `dev`）
- **THEN** 关于区块的版本显示为 `dev`
