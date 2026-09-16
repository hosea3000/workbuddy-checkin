## 1. AccountCard.vue 余额区重组

- [x] 1.1 调整 `balance` 计算属性：有余额时返回 `1,234 积分`（数值 + 单位同行），`creditBalance == null` 时保持 `—` 且不附加单位
- [x] 1.2 调整 `balanceSub` 计算属性：有 `creditBalanceAt` 时返回 `HH:MM 刷新`，缺失时返回空字符串（占位不跳行），移除「积分余额 ·」标签拼接
- [x] 1.3 核对模板中余额区两行容器的 class 无需改动（两行仍固定渲染，行高稳定）

## 2. 验证

- [x] 2.1 在 `frontend/` 运行 `npm run build`（`vue-tsc --noEmit && vite build`）通过
- [x] 2.2 代码核对：有余额显示 `1,234 积分` / `09:30 刷新`；从未刷新时数值行正常、时间行为空且不跳行；无余额显示 `—` / 时间行
