## Context

主窗口 820×640，最小 720×520（`main.go`）。当前 App.vue 是「顶栏 + 单个 router-view」，设置页把系统集成与模型代理混在一起。模型代理已是一等能力（`openspec/specs/openai-compat-proxy/`），却藏在设置页里。

拆分是**纯前端**改动：`GetSettings` / `SaveSettings` / `ProxyStatus` / `SetActiveCredential` 四个绑定方法已完全覆盖拆分后两个页面的需求，无后端改动。

## Goals / Non-Goals

**Goals:**
- 左侧紧凑导航栏 + 右侧内容区，三个顶层入口各归其位。
- 模型代理配置从设置页迁出到独立页，设置页只留系统集成。
- 模型代理页只读展示当前凭证，不新增写入入口。

**Non-Goals:**
- 不改任何 Go 绑定方法签名或行为。
- 不引入新依赖（继续用已有 `@lucide/vue` 图标）。
- 不做窄窗口响应式/折叠导航（`MinWidth: 720` 下 160px 侧栏足够，YAGNI）。
- 不改托盘菜单及其代理开关。

## Decisions

### D1 布局放在 App.vue，welcome 走例外分支

用 `route.name === 'welcome'` 条件渲染整个外壳，而不是把导航栏做成嵌套路由 layout。

考虑过的替代：给非 welcome 路由加嵌套 layout 组件。收益是 welcome 自动隔离；成本是 router 结构与 store/header 逻辑都要重组。**选条件渲染**——改动集中在 App.vue 一处，welcome 是唯一例外，一个 `v-if` 就够。

```
App.vue
├─ v-if route.name === 'welcome'  → <router-view/> 整页
└─ v-else                        → header 通栏 + aside 导航 + <router-view/>
```

### D2 导航栏约 160px，图标 + 文字

`MinWidth: 720` 下占 ~22%，内容区还有 560px，够用。纯图标栏（56px）省空间但可发现性差，而这是给普通用户的小工具。不做折叠态。

### D3 当前凭证：只读展示 + 跳转链接

`activeCredentialId` 的写入仍独占在账号卡片的大头针（`AccountCard.vue` → `accounts.setActive` → `api.setActiveCredential`）。代理页从 `ListAccounts` 里找出 `isActive` 的那个账号展示，另附「去账号页设置」链接。

理由：两个写入入口需要同步两份状态，而用户心智里「签到哪个账号」和「代理用哪个账号」是两个动作。方案 A 零同步成本。

### D4 保存必须读全量再改自己那部分（关键陷阱）

`Settings` 是整份覆盖写，`store.save(form)` 直接提交整个对象。拆页后若某个页面只构造自己管的字段就提交，会清掉另一页的设置。

规定两个页面的写入都遵循：`await store.load()` → `Object.assign(全量, 本页字段)` → `store.save()`。

设置页保存的是 `autoStart` / `updateProxy`。模型代理页**没有保存按钮**——启停按钮本身就是写入动作，同样先 `store.load()` 取全量、只覆盖 `proxyPort` 与 `proxyEnabled` 后提交（见 D6）。同时，`SettingsView.vue:62` 现有的 `activeCredentialId: store.settings?.activeCredentialId ?? ''` 原样回传逻辑在 D4 模式下**可以删除**——因为 `store.load()` 已经把真实值取回来了。

### D6 代理启停用单一按钮，点击即读最新配置

代理页不做「启用开关 + 保存」两步式，改为运行中显示「停止」、否则显示「启动」的单一按钮。

原因：开关是持久化意图，状态却是实际运行结果，两者分离会让用户看到「开关已开但服务未运行」的困惑态（端口被占用时尤其明显）。直接用实际运行状态驱动按钮语义，消除这一中间态。

按钮逻辑：`proxyEnabled = !proxyStatus.running`，并把页面上的 `proxyPort` 一并提交。写入前仍先 `store.load()`，因此**启动时以最新持久化配置为准**——即便用户停在代理页期间托盘菜单改了开关，点击按钮也会基于最新值重新计算，不会拿旧快照覆盖。

端口改动需先停止再启动才生效（运行中不热切换），页面在该情形给出一句提示。

### D5 路由 `/proxy`，名 `proxy`

与既有 `/accounts`、`/settings` 命名一致。

## Risks / Trade-offs

- **[覆盖写 bug]** 两个页面的保存都共享同一份 `Settings`，任何一处忘记先 load 就会互相覆盖 → 按 D4 统一模式，并在两个保存函数各写一句注释点明。验收时手动交叉验证：改代理设置 → 保存 → 去设置页看 autoStart/updateProxy 是否还在，反之亦然。
- **[轮询泄漏]** ProxyView 的 2s 轮询若不在 `onUnmounted` 清理，离开页面后仍在跑 → 沿用 `SettingsView.vue:47-49` 已有的 `clearInterval` 模式。
- **[顶栏签到汇总错位]** 汇总额只在账号页显示，原判据 `route.name === 'accounts'` 依旧成立，无需改动；但要确认拆分后 `route.name` 仍是 `accounts`。
- **[文档漂移]** `docs/PRD.md` F7 表格把模型代理两行列为设置项，拆页后不再准确 → 归档前同步 PRD（F7 去掉模型代理行，导航结构补一句）。

## Migration Plan

纯前端、无数据迁移。旧设置文件字段不变，默认值逻辑不变。回滚 = 还原前端提交。

## Open Questions

无（1–5 已与用户逐条确认）。
