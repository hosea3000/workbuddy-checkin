## 1. AccountCard.vue 常驻与激活态

- [x] 1.1 移除「设为当前凭证」按钮上的 `v-if="!account.isActive"`
- [x] 1.2 通过 `:class` 按 `account.isActive` 拼接激活态样式：`text-indigo-600 dark:text-indigo-400 bg-indigo-50 dark:bg-indigo-500/20`，非激活保持现有 `text-slate-400 hover:...` 样式
- [x] 1.3 保持 `title="设为当前凭证"` 与 `@click="emit('setActive', account.id)"` 不变

## 2. 验证

- [x] 2.1 在 `frontend/` 运行 `npm run build`（`vue-tsc --noEmit && vite build`）通过
- [x] 2.2 代码核对：所有卡片均渲染该图标按钮，仅当前凭证卡片带亮色 + 底色；行尾四个图标列对齐
