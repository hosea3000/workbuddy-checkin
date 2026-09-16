## 1. 路由与外壳

- [x] 1.1 `router.ts` 新增 `{ path: '/proxy', name: 'proxy', component: ProxyView }`
- [x] 1.2 `App.vue` 改为条件外壳：`route.name === 'welcome'` 走整页 `<router-view/>`，否则渲染 header + 左侧导航栏 + `<router-view/>`
- [x] 1.3 导航栏三项（账号签到 / 模型代理 / 设置），约 160px，图标 + 文字，选中态跟随 `route.name`
- [x] 1.4 顶栏移除设置 ⚙ 按钮，保留 logo / 标题 / 更新提示 / 签到汇总（汇总仍仅账号页显示）

## 2. 模型代理页

- [x] 2.1 新建 `views/ProxyView.vue`，从 `SettingsView.vue` 迁入：端口、状态轮询（`onUnmounted` 清理）、Base URL 提示；启停改为单一按钮（运行中「停止」/否则「启动」）
- [x] 2.2 启停遵循 design D4/D6：`store.load()` → 只覆盖 `proxyPort`/`proxyEnabled` → `store.save()`，启动读取最新持久化配置
- [x] 2.3 只读展示当前凭证（从 `ListAccounts` 取 `isActive` 账号），附「去账号页设置」链接，不提供切换控件

## 3. 设置页瘦身

- [x] 3.1 `SettingsView.vue` 移除模型代理卡片及 `proxyStatus` 轮询
- [x] 3.2 移除返回箭头（导航栏取代）
- [x] 3.3 保存改为 D4 模式（`load()` 后改 `autoStart` / `updateProxy`），删除 `activeCredentialId` 手动回传的旧逻辑

## 4. 验证

- [x] 4.1 `cd frontend && npm run build` 通过（`vue-tsc --noEmit` 无错）
- [x] 4.2 交叉验证覆盖写：启停代理后 → 设置页确认 `autoStart` / `updateProxy` 未丢；改设置页保存 → 代理页确认端口未丢、运行状态不受影响
- [x] 4.3 欢迎页无导航栏；添加账号后导航栏出现
- [x] 4.4 离开模型代理页后状态轮询停止
- [x] 4.5 同步 `docs/PRD.md`：F7 设置表移除模型代理行，补一句导航结构
- [x] 4.6 `go vet ./... && go test ./...` 通过（确认无后端回归）
