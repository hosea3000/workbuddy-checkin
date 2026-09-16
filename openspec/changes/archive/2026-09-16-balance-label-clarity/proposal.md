## Why

账号卡片的积分余额区域把标签「积分余额」排在数值下方、并把刷新时间用 `·` 拼接其后（如 `1,234` / `积分余额 · 09:30`），读起来像「积分余额发生在 09:30」，值、单位、时间三者归属不清，存在歧义。

## What Changes

- 数值行改为「值 + 单位」同行：`1,234 积分`；无余额时保持 `—`。
- 时间行改为 `HH:MM 刷新` 形式，明确是余额刷新时刻；从未刷新过时保留空占位，SHALL NOT 造成行高跳动。
- 不再使用「积分余额 ·」这种「标签+时间」拼接写法。

## Capabilities

### New Capabilities
<!-- 无新增能力 -->

### Modified Capabilities
- `account-management`: 账号卡片列表要求中，积分余额的展示方式由「数值 + 带标签与时间的副行」改为「值与单位同行 + `HH:MM 刷新` 时间行（无时间时占位不跳行）」。

## Impact

- 前端：`frontend/src/components/AccountCard.vue`（`balance` / `balanceSub` 计算属性与模板中余额区两行）。
- 不变：`⟳` 刷新图标按钮与其 `刷新余额` hover 文案、`refreshQuota` action 与成功提示、其余图标按钮。
