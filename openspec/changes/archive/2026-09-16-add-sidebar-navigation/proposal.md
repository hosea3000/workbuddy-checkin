## Why

当前设置页把系统集成（开机自启、更新、GitHub 加速代理）与模型代理（开关、端口、运行状态）混在一页，而模型代理已经是一个独立的产品能力（对外提供 OpenAI 兼容服务），继续挂在「设置」下让用户难以发现，也让设置在导航上失了重心。改为左侧导航栏后，「账号签到 / 模型代理 / 设置」三个顶层能力各归其位。

## What Changes

- App 布局改为左侧紧凑导航栏（约 160px）+ 右侧内容区；顶栏跨通栏保留。
- 导航项：**账号签到**（`/accounts`）、**模型代理**（`/proxy`，新增）、**设置**（`/settings`）。
- 新增 `ProxyView.vue`，从 `SettingsView.vue` 迁出：监听端口、运行状态（2s 轮询）、客户端 Base URL 提示；启停由**单一按钮**直接驱动（不做「开关 + 保存」两步式），点击时读取最新配置。
- 新增只读展示「当前凭证」：显示当前用于代理转发的账号，附「去账号页设置」链接。**不提供切换入口**，写入仍只在账号卡片的大头针按钮。
- `SettingsView.vue` 瘦身：仅保留开机自启、软件更新、GitHub 加速代理、打开数据目录、保存按钮；移除返回箭头（导航栏取代）。
- 顶栏调整：移除右侧「设置」⚙ 按钮；logo、标题、版本更新提示、签到汇总（仅账号页显示）保持原位。
- 欢迎页 `WelcomeView`（首启引导）不套导航栏，维持整页居中。
- 前端为纯布局/路由重构，**不新增或修改任何 Go 绑定方法**。

## Capabilities

### New Capabilities
- `app-navigation`: 应用外壳的导航结构——左侧导航栏的布局、三个顶层入口、欢迎页不套导航栏的例外。

### Modified Capabilities
- `settings`: 设置页不再包含模型代理相关配置项（开关 / 端口 / 运行状态），这些迁往模型代理页；设置页仅保留开机自启、更新与数据目录。
- `openai-compat-proxy`: 新增「当前凭证」在模型代理页的只读展示要求；明确凭证选取的**唯一写入入口**仍在账号卡片。代理服务本身的行为不变。

## Impact

- 前端：`App.vue`（布局）、`router.ts`（新增 `/proxy` 路由）、新增 `views/ProxyView.vue`、`views/SettingsView.vue`（瘦身）、`stores/settings.ts`（复用，无需改动）。
- 陷阱：`Settings` 是整份覆盖写。拆页后每个页面保存前必须 `store.load()` 取全量、只改自己负责的字段，否则会清掉另一页的设置。`SettingsView.vue` 现有的 `activeCredentialId` 原样回传逻辑需重新审视。
- 后端：无改动。`GetSettings` / `SaveSettings` / `ProxyStatus` 已满足拆分后两个页面的全部需求。
- 文档：`docs/PRD.md` F7 设置的表需要挪走模型代理两行；导航结构可能需在 PRD 补一句。
