## Why

应用是开源工具但界面上没有任何仓库入口，用户看不到源码、找不到反馈 issue 的地方，也无法确认许可条件。仓库当前也没有 `LICENSE` 文件，开源授权状态不明。设置页的「软件更新」卡片同时承担了版本展示与更新操作，信息归属混乱。

## What Changes

- 新增 `LICENSE` 文件（MIT，`Copyright (c) 2026 hosea3000`），补上仓库缺失的授权声明。
- 设置页新增「关于」卡片，包含：开源地址（可点击，用系统默认浏览器打开 `https://github.com/hosea3000/workbuddy-checkin`）、许可证（MIT）、当前版本号。
- 版本号从「软件更新」卡片移除，该卡片只保留更新状态与操作按钮（检查更新 / 立即更新 / 重启更新）和 GitHub 加速代理设置。

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `settings`: 设置页新增「关于」区块（开源地址跳转、许可证、版本号），并规定版本号的展示位置由更新区块移至关于区块。

## Impact

- `LICENSE`（新增，仓库根）
- `frontend/src/views/SettingsView.vue`（新增关于卡片、移除更新卡片中的版本号行）
- 依赖已有的 Wails runtime `BrowserOpenURL`，无需新增 Go 绑定方法，不涉及 `app*.go`，无需 `wails generate module`
- `openspec/specs/settings/spec.md`
