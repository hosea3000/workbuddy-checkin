## Why

账号卡片行尾的「设为当前凭证」「立即签到」两个文字按钮占据较多横向空间，账号列表在窄窗口下显得拥挤；而同样行尾的删除按钮已经是图标 + `title` 的形态，风格不统一。

## What Changes

- 「设为当前凭证」文字按钮改为图标按钮（`Pin`），hover 通过原生 `title` 显示文案 `设为当前凭证`。
- 「立即签到」文字按钮改为图标按钮（`CalendarCheck`），hover 通过原生 `title` 显示文案 `立即签到`；签到中状态改为展示 spinner（`Loader2`）并禁用，`title` 显示 `签到中…`。
- **BREAKING（UI）**：删除 `relogin_required` 账号上的「重新登录」按钮。凭证过期账号的红色 ring、红色头像、`需重新登录` 徽章、「凭证已过期，自动签到已跳过该账号」提示文字全部保留；重新登录改由页面顶部「添加账号」按钮重走 OAuth。相应更新 `docs/PRD.md` 中描述该按钮的段落。

## Capabilities

### New Capabilities
<!-- 无新增能力，本次仅调整既有卡片的行为呈现方式 -->

### Modified Capabilities
- `account-management`: 账号卡片行尾操作入口的呈现要求——「设为当前凭证」「立即签到」以图标 + hover 文案呈现，并移除 relogin 卡片的「重新登录」按钮。

## Impact

- 前端：`frontend/src/components/AccountCard.vue`（按钮模板与图标 import；`setActive`/`checkin` 事件契约不变）。
- 文档：`docs/PRD.md`（§53、§122 中「重新登录」按钮的描述）。
- 不变：`checkinNow` / `setActiveCredential` 绑定方法、`accounts` store、`relogin_required` 状态机、`StatusBadge`、托盘 tooltip、`CheckinSummary()` 均不受影响。
