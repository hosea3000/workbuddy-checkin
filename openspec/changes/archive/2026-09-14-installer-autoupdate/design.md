## Context

当前产品定位为「单文件免安装 exe」：用户下载 `workbuddy-checkin.exe` 双击运行，升级靠手动重新下载。`docs/PRD.md` 明确把「图形化安装器」列为非目标。为推广给普通用户，需要降低「首次安装」与「后续升级」的门槛。

已有的技术基础：

- `main.go` 已有 `version` 变量，CI 通过 `-ldflags "-X main.version=<tag>"` 注入；`wails.json` 已有 `info` 块供 Windows 版本资源使用。
- `.github/workflows/release.yml` 已照抄 health-tool：tag → `wails build` → `gh release create`，当前只发布单个 exe。
- Wails v2.13 内置 NSIS 安装器生成（`wails build -nsis`），支持 `-installscope user|machine` 与 WebView2 引导。
- 参考实现 health-tool（`updater*.go`）已验证「下载新版 exe → `.bat` 自替换 → 重启」的更新器，且带目录可写探测。

约束：仓库不引入新依赖（运行期纯 stdlib + Wails 运行时）；优先复用 health-tool 已验证实现；系统集成需保留 `*_stub.go` 以保证 Linux 下 `go test ./...` 可跑。

## Goals / Non-Goals

**Goals:**

- 普通用户能通过中文安装向导完成首次安装，得到桌面/开始菜单快捷方式与控制面板卸载项。
- 安装后的用户能在应用内一键检查并升级到新版本，无需手动重新下载。
- 同时保留单文件 exe 作为绿色版与自动更新的下载资产。
- 发布流程由 tag 自动触发，产出两个资产。

**Non-Goals:**

- 不制作 MSI（无企业 GPO/Intune/SCCM 批量部署需求）。
- 不做代码签名。
- 不做更新包哈希/签名校验。
- 不做版本回滚。
- 不做定时轮询更新检查。
- 不做增量/差量更新。
- 不做 arm64 产物（沿用 PRD：仅 x64）。

## Decisions

### 决策 1：安装范围用 `user`（非 `machine`）

`-installscope user` 装到 `%LOCALAPPDATA%\Programs\workbuddy-checkin`，`RequestExecutionLevel=user`，不触发 UAC。

**理由**：该目录对当前用户可写，是「下载新版 exe 覆盖自身」这一更新机制成立的前提；同时普通用户不会被 UAC 吓退。

**替代方案**：`machine` 装到 `Program Files`，需管理员且目录不可写，会让自替换更新器失效，只能退化为「跳转 GitHub 手动更新」或改走安装器静默重装。放弃。

### 决策 2：Release 同时发布两个资产

发布 `workbuddy-checkin.exe`（更新器与绿色版使用）与 `workbuddy-checkin-amd64-installer.exe`（首次安装使用）。

**理由**：一次 `wails build -nsis` 同时产出两者；更新器只认 exe 资产，避免下载安装器；绿色版保留给进阶用户。

**替代方案**：只发 exe（回到现状，无安装体验）；只发 installer（更新需下载安装器，体感更重）。均放弃。

### 决策 3：升级走「下载 raw exe + 自替换」，不走安装器静默重装

复用 health-tool 的更新器：下载到 `.part` → 落位 `.new` + `.new.version` → 确认重启 → `.bat` 等待进程退出后 `move /y` 覆盖 → 启动新版。

**理由**：改动最小、逐字可移植、静默无 UAC；安装路径不变，快捷方式与卸载项天然继续有效。

**替代方案**：

- 下载安装器 `installer.exe /S` 静默重装：体积大、用户自定义目录会错位、machine 下弹 UAC、Wails 模板装完不自动启动。放弃。
- 独立 updater 进程（复制自身到 temp 执行 `--apply-update`）：更可控、易做校验/回滚，但引入额外进程模型，偏离「复用 health-tool」约束。暂不采用。

### 决策 4：仅信任 HTTPS，不做哈希/签名校验

下载来源为 GitHub Release 的 `browser_download_url`（HTTPS）。

**理由**：个人工具场景下 HTTPS + `releases/latest` 已足够；引入 digest/`WinVerifyTrust` 会显著增加复杂度，且未签名时签名校验也无法使用。

**替代方案**：比对 GitHub API 的资产 `digest`（sha256）或 `WinVerifyTrust`。留作后续加固，不在本变更。

### 决策 5：不做回滚

不保留旧版 exe 的 `.bak`。

**理由**：与 health-tool 一致；覆盖失败已有重试与回退旧版逻辑，足以避免「程序损坏」。

**替代方案**：保留 `.bak` 并检测启动失败后回滚。复杂度高，收益有限，放弃。

### 决策 6：保留安装向导的目录选择页

**理由**：保留用户对安装位置的自主权，符合 Wails 默认模板；去掉目录页会改变默认向导行为，且用户仍可在别处指定。

### 决策 7：安装器界面改为简体中文

Wails 首次 `-nsis` 构建会生成 `build/windows/installer/project.nsi`。将其中的 `!insertmacro MUI_LANGUAGE "English"` 改为 `"SimpChinese"` 后**提交进仓库**；Wails 之后会优先使用仓库中的 `project.nsi`。`wails_tools.nsh` 每次构建由 Wails 覆盖，属派生文件；WebView2 bootstrapper 为每次构建下载物，二者不提交。

**理由**：面向国内普通用户，中文向导降低理解成本。CI 的 `nsis` apt 包自带 `SimpChinese` 语言文件。

### 决策 8：更新触发为「启动检查（可关）+ 手动检查」，无定时器

沿用已有的设置项「启动时检查更新」（默认开），并在设置页新增手动检查与当前版本展示。

**理由**：签到工具常驻托盘、启动频率足够，定时轮询收益低且增加网络请求。

### 决策 9：更新器逐字移植 health-tool

移植 `updater.go`（检查/版本比较）、`updater_apply.go`（下载/脚本）、`updater_apply_windows.go`、`updater_apply_stub.go`，仅改常量：`updateRepoOwner` / `updateRepoName` / `updateAssetName = "workbuddy-checkin.exe"` / `updateBatName`。

**理由**：该实现已在 health-tool 验证；DESIGN 已把更新器列为 P2（约半天移植）。

### 决策 10：发布前构建门禁

发布工作流在构建前执行 `go vet ./...` 与 `go test ./...`。

**理由**：决策 5 放弃回滚后，坏版本一旦发布只能手动重装，发布前测试的边际价值更高。

### 决策 11：卸载时拒绝运行中的应用

卸载脚本在开始时用 NSIS 内置 `FindWindow` 按托盘窗口类名 `WorkbuddyCheckinTray` 检测应用是否在运行；命中则弹提示并用 `Quit` 立即结束卸载器（而非 `Abort`——`Abort` 只中止当前 section，不会退出卸载器进程）。

**理由**：应用运行中会锁住 exe，继续卸载会残留文件或失败；托盘消息窗口在应用运行时始终存在（即使主窗口已隐藏），类名稳定可靠。

**替代方案**：按进程名检测需引入 `nsProcess` 插件；按 exe 文件占用检测不稳定。均放弃。

## Risks / Trade-offs

- [未签名导致 SmartScreen/杀软拦截] → 开源可查、README 说明、仅写 HKCU、不触碰敏感注册表位置；签名留待后续评估。
- [安装范围与更新机制强耦合] → 锁定 `-installscope user`，在文档与工作流中注明：改 machine 会破坏自替换更新。
- [`.bat` 自替换可能触发 AV 启发式] → 无窗口启动（`CREATE_NO_WINDOW`）、覆盖失败重试、失败兜底为「前往 GitHub 手动更新」。
- [无校验 + 无回滚，坏版本影响面大] → 发布前 `go vet && go test` 门禁；必要时手动重装。
- [国内下载 GitHub 资产可能缓慢/失败] → 下载失败给出中文提示并提供 Release 页面入口；镜像留作后续。
- [绿色版与安装版共用 `%APPDATA%` 与单实例锁] → 二者不会同时运行；要求数据 schema 前后兼容。
- [`wails_tools.nsh` 被构建覆盖、bootstrapper 被重复下载] → 加入 `.gitignore`，只提交定制后的 `project.nsi`。
- [CI 缺少 `makensis` 会导致静默跳过安装器] → 工作流显式安装 `nsis`；若产物缺失应视为失败。

## Migration Plan

1. 更新 `docs/PRD.md`（形态改为「安装器 + 绿色版双产物」、移除非目标中的「图形化安装器」、更新风险与验收）与 `docs/DESIGN.md`（能力清单加入 `ci-autorelease`、`app-update`）。
2. 实现更新器移植与绑定方法，设置页增加版本展示与手动检查；Linux 下 `go vet ./... && go test ./...` 通过。
3. 首次本地 `wails build -nsis -installscope user` 生成 `build/windows/installer/`，改 `project.nsi` 为中文并提交，补充 `.gitignore`。
4. 更新 `release.yml`：安装 `nsis`、加测试门禁、构建加 `-nsis -installscope user`、上传双资产。
5. 打 `v0.2.0` tag 验证端到端发布与安装/升级流程。

回滚策略：若发布异常，删除该 Release 与 tag，重新打 tag 发布；应用侧无回滚，用户按提示手动重装上一版本。

## Open Questions

- 是否在后续加入代码签名（OV/EV），以缓解 SmartScreen 与杀软误报？
- 是否需要为国内用户提供 GitHub 资产镜像或加速下载？
- 若未来确需企业批量部署，是否再评估 MSI/MSIX（与本变更的 user-scope 安装器并存）？
