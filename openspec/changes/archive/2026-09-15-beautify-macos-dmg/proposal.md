## Why

macOS 的 dmg 目前用 `hdiutil create -srcfolder` 直接打包，磁盘映像里只有一个孤零零的 `.app`，没有「应用程序」快捷方式、没有固定窗口布局。用户双击打开后只能从只读的临时挂载盘直接运行，重启后盘卸载就找不到 app，体验上也不像常规 macOS 安装包。需要改成标准做法：dmg 打开后呈现「app 图标 + 应用程序文件夹」的拖拽布局。

## What Changes

- 用 `create-dmg` 替代当前的 `hdiutil create -srcfolder`，产出带拖拽布局的 dmg：
  - 窗口 660×400、图标 100px、app 图标位于 `(165,200)`、Applications 拖放链接位于 `(495,200)`、无背景图。
  - 卷名由中文 `WorkBuddy 自动签到` 改为 `workbuddycheckin`（纯小写、无空格，避免与其他软件卷名冲突，且更适配 CI 挂载路径）。
- `create-dmg` 通过下载官方 release tarball 直接调用脚本，不用 `brew install`、不写入系统。
- dmg 文件名（`WorkBuddy-checkin-amd64.dmg` / `WorkBuddy-checkin-arm64.dmg`）保持不变，应用内更新查找逻辑不受影响。

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `ci-autorelease`: 「dmg 打包使用系统工具」这条要求与新引入的 `create-dmg` 打包方式冲突，需改写为「dmg 打包产出标准拖拽布局」，并明确依赖引入边界（CI 脚本层、非 Go 模块）与卷名取值。

## Impact

- 仅影响 `.github/workflows/release.yml` 的 macOS `Package dmg` 步骤；Windows 产物与上传步骤不变。
- 不新增运行时依赖，不改 Go 代码，不签名不公证（与现有硬约束一致）。
- 需要下载 `create-dmg` 脚本（CI 网络依赖），版本需与所用参数对齐。
