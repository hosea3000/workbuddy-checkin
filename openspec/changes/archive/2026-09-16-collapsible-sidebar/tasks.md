## 1. 折叠状态

- [x] 1.1 `App.vue` 增加 `collapsed` ref，初值读取 `localStorage['sidebar-collapsed'] === '1'`（`try/catch` 兜底为展开）
- [x] 1.2 切换函数：翻转 `collapsed`，并写回 `localStorage`（写入失败不抛错，仅不记住）

## 2. 导航栏折叠

- [x] 2.1 侧栏宽度 `w-40 ↔ w-14`，加宽度过渡
- [x] 2.2 折叠时隐藏文字节点、图标居中；`title` 始终挂菜单名
- [x] 2.3 保留选中态与 `router.push` 逻辑不变（两种状态一致）
- [x] 2.4 侧栏底部加折叠/展开开关，`w-full` 命中整条，图标与 `title` 随状态切换

## 3. 验证

- [x] 3.1 `cd frontend && npm run build` 通过（`vue-tsc --noEmit` 无错）
- [x] 3.2 首次打开（清空 localStorage）为展开态
- [x] 3.3 折叠后仅见图标，点击图标仍能切页且选中态正确
- [x] 3.4 折叠后重启应用仍为折叠态
- [x] 3.5 欢迎页不出现侧栏与折叠开关
- [x] 3.6 `go vet ./... && go test ./...` 通过（确认无后端回归）
