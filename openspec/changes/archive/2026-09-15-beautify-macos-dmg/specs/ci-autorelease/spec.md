## MODIFIED Requirements

### Requirement: dmg 打包使用标准拖拽布局

CI SHALL 为 macOS 产出带标准拖拽安装布局的 dmg：磁盘映像内含 `.app` 与指向 `/Applications` 的拖放链接，窗口尺寸 660×400、图标尺寸 100px、app 图标位于 `(165,200)`、应用程序链接位于 `(495,200)`、无背景图，卷名为 `workbuddycheckin`。打包 SHALL 使用 `create-dmg` 脚本（脚本层 CI 依赖，通过下载官方 release tarball 调用，SHALL NOT 用 `brew install` 或写入系统），SHALL NOT 引入新的 Go 模块依赖。

#### Scenario: 产出拖拽布局 dmg
- **WHEN** macOS 构建 job 完成 dmg 打包
- **THEN** 挂载该 dmg 可见 `.app` 图标与「应用程序」拖放链接，图标位置与窗口尺寸符合上述取值

#### Scenario: 卷名不与常见软件冲突
- **WHEN** 用户挂载 dmg
- **THEN** 卷名为 `workbuddycheckin`

#### Scenario: 不以 brew 引入依赖
- **WHEN** macOS 构建 job 执行 dmg 打包
- **THEN** 仅通过下载的 `create-dmg` 脚本与 runner 预装系统工具完成打包，未执行 `brew install`，且未新增 Go 模块依赖
