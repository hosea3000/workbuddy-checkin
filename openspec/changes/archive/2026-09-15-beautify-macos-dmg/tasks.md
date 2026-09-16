## 1. 确认 create-dmg 版本与参数

- [x] 1.1 选定并记录要锁定的 `create-dmg` release tag（`v1.2.1`）
- [x] 1.2 本地/临时环境解出该版本脚本，跑 `--help` 确认 `--volname`、`--window-size`、`--icon-size`、`--icon`、`--app-drop-link` 参数名一致（已对照 v1.2.1 源码确认；脚本需同目录 `support/`，故下载整包 tarball）

## 2. 改写 macOS dmg 打包步骤

- [x] 2.1 在 `release.yml` 的 macOS job 增加下载并解出 `create-dmg` 脚本的步骤（固定 tarball URL，不用 brew）
- [x] 2.2 将 `Package dmg` 的 `hdiutil create -srcfolder` 替换为 `create-dmg` 调用，参数：卷名 `workbuddycheckin`、窗口 660×400、图标 100px、app `(165,200)`、Applications `(495,200)`、无背景图
- [x] 2.3 确认 dmg 输出文件名仍为 `WorkBuddy-checkin-<arch>.dmg`，上传步骤不变

## 3. 验证

- [x] 3.1 本地（macOS）或一次测试 tag 上跑通 macOS job，确认 dmg 挂载后含 app + 应用程序链接且位置正确、卷名为 `workbuddycheckin`
- [x] 3.2 确认 Release 资产名与应用内更新查找的 `WorkBuddy-checkin-<arch>.dmg` 一致
- [x] 3.3 `openspec validate --changes "beautify-macos-dmg"` 通过后归档 change
