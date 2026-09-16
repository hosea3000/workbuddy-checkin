## 1. 许可证

- [x] 1.1 在仓库根新建 `LICENSE`：MIT 全文，`Copyright (c) 2026 hosea3000`

## 2. 设置页关于区块

- [x] 2.1 `SettingsView.vue` 新增「关于」卡片（`开机自启` → `软件更新` → `关于` 顺序），展示开源地址、许可证 MIT、当前版本号
- [x] 2.2 开源地址做成可点击元素，点击调用 `BrowserOpenURL('https://github.com/hosea3000/workbuddy-checkin')`，窗口内不导航
- [x] 2.3 从「软件更新」卡片移除「当前版本 vX」文案，保留更新状态提示与操作按钮

## 3. 验证

- [x] 3.1 `cd frontend && npm run build`（含 `vue-tsc --noEmit`）通过
- [x] 3.2 `go vet ./... && go test ./...` 通过（本次未改 Go，作回归）
- [x] 3.3 手动核对：设置页版本号只在关于区块出现；点击开源地址在系统浏览器打开仓库
