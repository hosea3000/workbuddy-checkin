## Why

当前凭证卡片会隐藏「设为当前凭证」图标按钮（`v-if="!account.isActive"`），导致该行比其余行少一个图标位、行尾图标列左右错位；同时「已激活」这一状态在图标上没有任何呈现。

## What Changes

- 「设为当前凭证」图标按钮改为**常驻显示**，不再随 `isActive` 隐藏。
- 当前凭证卡片的该按钮呈现高亮激活态：`text-indigo-600 dark:text-indigo-400` + 胶囊底色 `bg-indigo-50 dark:bg-indigo-500/20`。
- 激活态仍可点击（幂等，无副作用），hover 文案始终为「设为当前凭证」。

## Capabilities

### New Capabilities
<!-- 无新增能力 -->

### Modified Capabilities
- `account-management`: 账号卡片列表要求中，「设为当前凭证」图标入口由「非当前凭证卡片才提供」改为「所有卡片常驻提供，当前凭证以高亮激活态呈现」。

## Impact

- 前端：`frontend/src/components/AccountCard.vue`（移除按钮上的 `v-if`，按 `account.isActive` 条件拼接 class）。事件契约、`setActive` action 均不变。
- 不变：`checkinNow`/`setActiveCredential` 绑定方法、`accounts` store、`StatusBadge`、「✓ 当前凭证」徽章、签到与余额刷新图标。
