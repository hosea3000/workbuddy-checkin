## 1. AccountCard.vue 图标化

- [x] 1.1 在 `frontend/src/components/AccountCard.vue` 的 `@lucide/vue` import 中加入 `Pin`、`CalendarCheck`、`Loader2`
- [x] 1.2 将「设为当前凭证」文字按钮改为图标按钮（`Pin`），使用删除按钮同款 class 与 `title="设为当前凭证"`，保留 `v-if="!account.isActive"` 与 `@click="emit('setActive', account.id)"`
- [x] 1.3 将「立即签到」文字按钮改为图标按钮（`CalendarCheck`），`title` 为 `立即签到`；`checking` 为真时改为渲染 `Loader2`（`animate-spin`）且 `title="签到中…"`，保留 `:disabled="checking"`
- [x] 1.4 删除 relogin 分支的 `<template v-else>`「重新登录」按钮，并移除签到按钮上按 `account.status` 的 `v-if` 条件，统一为单个图标按钮

## 2. 文档修订

- [x] 2.1 更新 `docs/PRD.md` §53 中「用户点『重新登录』」的描述，改为「用户点『添加账号』重走 OAuth」
- [x] 2.2 更新 `docs/PRD.md` §122 中「卡片红色徽章 + 『重新登录』按钮」的描述，移除按钮表述、保留红色徽章与自动签到跳过

## 3. 验证

- [x] 3.1 在 `frontend/` 运行 `npm run build`（`vue-tsc --noEmit && vite build`）通过
- [x] 3.2 手动核对：普通卡片显示 `Pin`/`CalendarCheck` 两个图标，hover 出对应文案；relogin 卡片只显示红色标识与过期文案、无图标操作按钮（除删除与余额）
