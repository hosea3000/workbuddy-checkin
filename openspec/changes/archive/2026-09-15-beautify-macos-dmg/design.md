## Context

`release.yml` 的 macOS job 目前用一行 `hdiutil create -srcfolder ... .app` 打包 dmg：映像里只有一个孤立的 `.app`，既没有「应用程序」链接，也没有固定窗口布局。用户打开后只能从只读挂载盘直接运行，观感与常规 macOS 安装包不符。

约束来自 `AGENTS.md`/`DESIGN.md`：

- macOS 系统集成只用系统命令，不签名不公证，不引入新的第三方 Go 依赖。
- 产物必须仍能被应用内更新按 `WorkBuddy-checkin-<arch>.dmg` 文件名找到。

## Goals / Non-Goals

**Goals**

- dmg 打开后呈现「app 图标 + 应用程序文件夹」拖拽布局，窗口与图标位置固定。
- 不改变 dmg 文件名、不改变上传步骤、不改变 Windows 产物。

**Non-Goals**

- 不做代码签名 / 公证（Gatekeeper 弹窗不在本变更范围）。
- 不加背景图、不加品牌化窗口样式。
- 不改动应用内更新逻辑。

## Decisions

**决策 1：用 `create-dmg` 替代裸 `hdiutil`**

`create-dmg` 内部就是「建可写 dmg → 挂载 → AppleScript 摆图标（含 Applications 软链）→ 转 UDZO」的封装，能把这套易错流程收敛成一条命令。

- 备选 A：纯 `hdiutil` + `osascript` 手写 —— 参数多、易错、无重试，维护成本高。
- 备选 B：预置 `.DS_Store` 模板随仓库提交，CI 只做 `hdiutil create` —— CI 最稳，但布局每次调整都要本地重做模板，迭代笨重。
- 选 `create-dmg` 换取一次性的简单与可读性，接受其对 GUI session 的依赖（见风险）。

**决策 2：下载 tarball 调用脚本，不用 `brew install`**

`create-dmg` 本体是单文件 bash 脚本。下载官方 release tarball 解出后直接调用，避免 brew 在 CI 上的网络/更新不确定性，也不往系统装东西，边界仍属「CI 脚本层依赖」而非 Go 依赖。

**决策 3：卷名用 `workbuddycheckin`**

原名 `WorkBuddy 自动签到` 含空格与中文，挂载路径 `/Volumes/WorkBuddy 自动签到` 在 AppleScript 定位窗口时更脆；`WorkBuddy` 又易与其他软件撞名。取 `workbuddycheckin`：纯小写、无空格、与可执行文件名一致、不冲突。

**决策 4：视觉参数取常见默认**

`--window-size 660 400`、`--icon-size 100`、app `(165,200)`、Applications `(495,200)`、无背景图。

## Risks / Trade-offs

- [无头 runner 上 Finder AppleScript 偶发超时/找不到窗口] → `create-dmg` 内建重试；关注 `--no-return-code` 等开关，若 CI 不稳定再评估方案 B（预置 `.DS_Store`）。
- [`create-dmg` 版本与参数名不一致（如 `--app-drop-link` 在旧 tag 名称不同）] → 落地前先解出目标版本跑 `--help` 对齐参数。
- [下载 create-dmg 需要 CI 出网] → 固定一个 release tag 的 tarball URL，避免 master 漂移。
- [卷名残留在 runner 上造成下次挂成 `workbuddycheckin 1`] → matrix 两个架构在不同 runner 且卷名已无空格，风险低；本地反复测试时手动 `hdiutil detach`。

## Migration Plan

无数据迁移。单次修改 `release.yml` 的 `Package dmg` 步骤；回滚即改回 `hdiutil create -srcfolder` 一行。新逻辑仅在下次推 `v*` tag 时生效。

## Open Questions

- 具体锁定的 `create-dmg` tag 版本号（落地时确认，并据其 `--help` 对齐参数名）。
