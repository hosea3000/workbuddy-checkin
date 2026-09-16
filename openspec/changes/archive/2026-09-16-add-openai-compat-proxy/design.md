## Context

`workbuddy-checkin` 是单 exe 桌面工具（Wails v2 + Go + Vue3），当前只做账号管理与签到。`upstream/codebuddy/` 已具备聊天所需的大部分能力，但**没有调用者**：

```
upstream/codebuddy/
  client.go     ChatURL() 已存在，无调用者；HTTP *http.Client 已配好连接池
  headers.go    GenerateHeaders(cred, ids, cliVersion) 已导出
  endpoint.go   HostFromEndpoint / stainlessOS / stainlessArch 齐备
  errors.go     UpstreamError{StatusCode, ErrType, CredInvalid} 分类齐备
  ✗ sse.go      缺失（上游 SSE 行解析 / 格式化）
  ✗ events.go   缺失（上游 chunk → 协议中立语义事件）
  ✗ FetchModels 缺失（GET /v3/config）
```

同时 `../work2api`（Go，Gin+GORM）已有一份**验证过**的纯 Go 聊天链路，且与 Web 框架解耦：

| work2api 文件 | 行数 | 外部依赖 | 可迁移性 |
|---|---|---|---|
| `service/chat_executor.go` | 436 | `google/uuid` + 上游包 | 直接可用（uuid 换 `crypto/rand`） |
| `service/request_processor.go` | 206 | 无 | 直接可用（需裁剪策略） |
| `service/system_prompt_rewriter.go` | 112 | 无 | 直接可用 |
| `service/models_service.go` | 91 | 上游包 | 直接可用 |
| `upstream/codebuddy/sse.go` | 128 | 无 | 直接可用 |
| `upstream/codebuddy/events.go` | 282 | 无 | 直接可用 |
| `handler/openai_handler.go` | 101 | Gin | 需改写为 `net/http` |
| `service/credential_pool.go` | 223 | GORM/viper | **不复用**（见决策 3） |

约束（来自 `docs/DESIGN.md`，不可偏离）：不引入 gin/gorm/viper/wire/redis/sqlite 及任何 DI 框架；上游客户端纯 `net/http`；依赖方向 `main → internal/* → store → model`，`internal/* → upstream/codebuddy`；令牌明文 0600 存储但日志/UI/通知永不出现完整令牌；系统集成须同时提供 `*_stub.go` 保证 Linux 下 `go test ./...` 可跑。

## Goals / Non-Goals

**Goals:**

- 在应用内提供 `127.0.0.1:<端口>` 上的 OpenAI 兼容接口，供 Cline / Continue 等客户端使用，从而可以关闭外部的 `codebuddy2api` 进程
- 复用 `workbuddy-checkin` 已有的账号、凭证存储与续期能力，不新建第二套凭证体系
- 用设置页的开关与端口控制服务，状态可观测、失败可解释
- 保持单 exe、零新增第三方依赖

**Non-Goals:**

- Anthropic Messages 协议（`/v1/messages`）、Claude Code 接入——本次不做
- API Key 鉴权、多用户、会话登录——仅本机回环访问，不做
- 凭证池、round-robin 轮换、失败自动降级到其他账号——不做
- 用量统计、请求计数、限流、管理台
- 代理侧的额度查询与签到（仍由既有调度器负责）

## Decisions

### 1. 从 work2api 回迁，不从 codebuddy2api（Python）重写

`work2api` 的聊天链路已经是 Python 参考实现的 Go 翻译（`sse.go` 注释即写明「语义对齐 sse.py」），且 85% 的文件零框架依赖。从 Go 回迁 = 复制 + 替换 `google/uuid`，预计 ~1000 行；从 Python 重写需重新翻译 asyncio 流式管道，预计 2000+ 行且引入新的翻译错误。

**替代方案**：直接调 `codebuddy2api` 作为子进程——被否，目标是单 exe、去掉 Python 运行时。

**落点**：`upstream/codebuddy/` 补 `sse.go`、`events.go`、`FetchModels`、`GenerateIDEConfigHeaders`；新建 `internal/proxy/` 承载服务与协议适配。依赖方向保持 `internal/proxy → upstream/codebuddy`、`internal/proxy → store → model`。

### 2. 端点路径用 `/v1/chat/completions` 与 `/v1/models`

客户端 base_url 习惯填到 `/v1` 为止，路径越短越通用。`work2api` 用 `/codebuddy/openai/v1/...` 是因为它同时挂载多个 provider；本服务只有一个用途，无需前缀。

### 3. 单一「当前凭证」指针，不做凭证池

代理只用 `Settings.ActiveCredentialID` 指向的那一个凭证。账号卡片提供「设为当前凭证」按钮。

**替代方案**：回迁 `work2api` 的 `CredentialPool`（内存副本 + round-robin + `MarkExpired` 摘除）——被否，理由有三：池子需要 `LoadAll`/`Refresh`/`RefreshWithCurrent` 三个同步入口维护「内存副本 vs 持久层」的一致性，是一整类 bug 的来源；桌面单用户场景下轮换不可预期（用户不知道这次请求走了哪个账号）；且 `store.GetCredential(id)` 已是微秒级的本地 JSON 读，缓存副本没有收益。

**行为**（已在探索中确认）：

| 情况 | 行为 |
|---|---|
| 未设当前凭证 / ID 悬空 | 503，提示设置当前凭证 |
| 状态为 `relogin_required` | 502，提示重新登录；**不自动降级到其他账号** |
| 已过期 | 请求前续期，失败则报错给客户端（见决策 6） |
| 凭证正常 | 直接使用 |

**自动维护**：登录成功且当前为空时自动设该账号；删除的正是当前凭证时顺位到列表首个 `active`，无则置空。

### 4. 无鉴权，显式绑定 `127.0.0.1`

监听地址硬编码 `"127.0.0.1:" + port`，**绝不使用 `":port"`**（那会绑定所有网卡）。

**替代方案**：静态 `sk-` key（约 40 行）——用户明确否决。记录此风险：本机任意程序与浏览器页面均可访问该端口并消耗账号额度；绑定回环地址限制了非本机访问，但不限制本机其他程序。端口 SHALL 校验为 1024–65535，避免用户填入特权端口后启动失败却得到难懂的报错。

### 5. 服务生命周期：设置页启停，状态以实际为准

沿用「开机自启以系统实际状态为准」的既有模式（`GetSettings` 覆盖持久化值）。新增绑定方法返回**实际运行状态**（`running` / `port` / `error`），而非持久化意图，前端轮询显示（本仓库约定：前端不用 Wails 事件传业务状态）。

- 应用启动时若设置为启用则启动服务
- 保存设置时若开关或端口变化则重启监听
- 端口被占用 / 绑定失败 → 服务不启动，错误在设置页可读
- 应用退出时停止服务
- 关闭主窗口（隐藏到托盘）**不停**服务——应用仍在后台运行
- 托盘菜单新增开关项，与设置页开关共享同一状态

### 6. 续期走新增的静默入口，不复用签到路径的副作用

`internal/checkin/service.go` 的 `refreshIfNeeded` 在失败且错误名为 `unauthorized` 时会置 `relogin_required` **并发送系统通知**（`handleRefreshError`）。若代理直接调用它，一次 API 请求就可能弹出桌面通知，且客户端重试会反复触发。

**方案**：抽出纯续期逻辑，两条调用路径各自包装：

```
refreshTokens(ctx, c) (model.Credential, error)   ← 新增：令牌交换 + 回写 store，无副作用
   ├─ refreshIfNeeded    → 失败时 handleRefreshError（置位 + 通知一次）  ← 签到路径，不变
   └─ EnsureFresh        → 失败时直接返回 error                        ← 代理路径，新增
```

`EnsureFresh` 的触发条件比签到路径窄：**仅在 `now >= ExpiresAt - 60s` 时续期**，而非签到路径的 24 小时窗口。理由：24 小时窗口下，一个仍可正常使用 20 小时的令牌若因网络抖动续期失败，代理会白白拒绝请求；窄窗口下失败只发生在令牌确实不可用时，报错才是正确的。

并发：单一凭证只需一个 `sync.Mutex` 串行化续期，避免并发请求重复刷新同一账号。持锁跨网络调用会短暂串行化请求——可接受（续期罕见且快）。`// ponytail:` 标注此天花板：若将来多凭证或高并发，改为 per-credential 锁。

### 7. 请求策略裁剪

`work2api` 的 `PrepareChatPayload` 含若干产品策略，逐条裁决：

| 策略 | 处置 | 理由 |
|---|---|---|
| `ForcedTemperature` 强制改写 | **删除** | 用户明确要求透传客户端值；静默改写会让客户端配置「撒谎」 |
| `ForcedReasoningModels` 强制 `reasoning_effort=max` | **删除** | 该分支在 `work2api` 中从未配置，是死代码 |
| `enable_thinking` 默认 true | **保留** | 仅当客户端未表态时给默认值；客户端显式 `false` 时尊重（`is_thinking_explicitly_disabled` 判定），不构成静默覆盖 |
| 非空 `stop` 返回 400 | **保留** | 上游不报告停止序列命中；接受但静默失效比拒绝更糟 |
| `stream=true` 强制 + `stream_options.include_usage=true` | **保留** | 上游仅支持流式；usage 需显式索取 |
| 单条 user 消息补 `system` | **保留** | 防上游对无 system 请求报错 |
| `StripModelNamespace` | **保留** | 无害 |
| system prompt 反指纹改写 | **保留** | 零风险（不匹配即原样返回），且是将来接 Claude Code 的现成资产 |

### 8. 模型列表来自上游 `/v3/config`

`GET /v1/models` 返回「配置模型 ∪ 上游实际模型」的有序去重并集，TTL 30 秒缓存，上游失败时回退缓存、再无缓存则回退配置模型。

需要新增 `GenerateIDEConfigHeaders`（`/v3/config` 用 `CodeBuddyIDE` 变体头集，与聊天头集不同）。默认模型：取并集首项；若为空则要求客户端显式指定 `model`。

**替代方案**：只返回配置模型（写死列表）——被否，上游模型会变，写死会很快过期。

### 9. 错误映射

复用 `upstream/codebuddy/errors.go` 的 `ErrCategory*` 与 `CredInvalid`，映射为 OpenAI 错误体 `{"error": {"message", "type", "code"}}`：

| 上游 / 本地情况 | HTTP | type |
|---|---|---|
| 无可用凭证 / 未设当前凭证 | 503 | `service_unavailable_error` |
| 凭证被拒绝（`CredInvalid`） | 502 | `authentication_error` |
| 限流（429） | 429 | `rate_limit_error` |
| 上游 5xx | 502 | `upstream_server_error` |
| SSE 协议错误 / 流未正常结束 | 流内 error 事件 | `upstream_protocol_error` / `upstream_incomplete` |
| 请求体不合法 | 400 | `invalid_request_error` |

日志中记录错误类别与账号短码，**不得记录令牌**。

## Risks / Trade-offs

- **本机任意程序可白嫖账号额度**（决策 4 的已知代价）→ 绑定 `127.0.0.1` 限制非本机访问；用户已明确接受。若将来需收紧，加静态 `sk-` key 即可（约 40 行，无需改架构）
- **上游协议为逆向所得，可能随时变更** → 伪装头版本号沿用既有可配置的 `cliVersion`；错误分类已覆盖协议异常，失败时表现为客户端可见的错误而非静默错误结果
- **单凭证无冗余**：当前凭证失效则代理整体不可用 → 这是用户明确选择的「点哪个就是哪个」语义；错误信息指向「重新登录」或「换一个账号」，用户手动切换
- **静默续期持锁跨网络调用**会短暂串行化请求 → 仅在令牌临期 60 秒内发生；已标注升级路径
- **`temperature` 透传后若上游对极端值报错**，将直接暴露为客户端错误 → 届时再评估是否加回可配置覆盖；当前不预先假设
- **`system_prompt_rewriter` 对 Cline 等客户端基本不触发**（它针对 Claude Code 的身份声明）→ 保留成本仅为 112 行且零副作用，作为将来接入 Anthropic 协议的资产
- **回迁代码与本仓库风格差异**：`work2api` 用 `google/uuid`、部分英文注释 → 替换为既有 `newUUID()`（`crypto/rand`），注释统一为中文

## Migration Plan

1. 补 `upstream/codebuddy/{sse.go,events.go}` 与 `FetchModels`、`GenerateIDEConfigHeaders`，配套单测（SSE 解析、tool_call 增量索引、模型提取）
2. 抽出 `refreshTokens` 并加 `EnsureFresh`，跑 `internal/checkin` 既有测试确保签到路径行为不变
3. 实现 `internal/proxy/`（服务、适配、执行器、聚合），配 `httptest` 级单测
4. 接入装配：`app.go` 构造代理服务，`model/settings.go` 加字段，`app_settings.go` 加读写与状态查询
5. 前端：设置页（开关 / 端口 / 状态）、账号卡片（设为当前凭证 / 当前标记）
6. 托盘菜单项
7. 全量验证：`go vet ./... && go test ./...`、`GOOS=darwin GOARCH=arm64 go build ./...`、`cd frontend && npm run build`

**回滚**：本功能默认关闭且不修改既有数据语义（`settings.json` 仅新增可选字段，缺失即默认值）。回滚 = 删除 `internal/proxy/` 与前端入口；已写入的 `ActiveCredentialID` 字段被旧版本忽略。

## Open Questions

- 托盘菜单项的文案与位置（「模型代理」子菜单 vs 平铺开关）——实现时定
- 默认模型的最终取值：并集首项 vs 要求客户端显式指定——先按并集首项，若客户端体验不佳再调整
- ~~`/v1/models` 是否需要支持 `owned_by` 等 OpenAI 字段的完整形态——先返回最小合法结构（`id` / `object` / `created`）~~

## 已定结论：超长请求被上游策略性拒绝

**现象**：长会话下上游返回 `Illegal API invocation from an unapproved channel`（HTTP 400）。实测请求形状为 `msgs=520 bytes=1164359 (1.16 MB)`。

**根因**：请求体超出模型上下文窗口。上游把这种「官方 CLI 不可能发出」的请求伪装成渠道校验失败拒绝。

两个曾怀疑但已排除的方向：

1. **竞品关键词检测**——错。`claude`/`anthropic` 在该请求里命中 503 次，但短会话（同样含这些词，如 `msg[0]` 的 system 提示）一直正常；且同一长会话在 `codebuddy2api` 上同样报错，说明与本次移植无关。**结论：关键词不是触发条件。**
2. **`/v1/models` 字段不全**——错，但相关。opencode 判断「context is full」需要知道模型窗口，而 OpenAI 的 `/v1/models` schema 本身不携带窗口信息，代理无法在此声明。**结论：这是客户端侧配置问题，不是代理缺陷。**

**已验证的修复**：在 opencode 配置里给模型声明窗口，客户端随即正常触发压缩并可继续调用：

```jsonc
"provider": { "<id>": { "models": { "<model>": { "limit": { "context": 128000, "output": 32000 } } } } }
```

**代理侧加固**：`Execute` 在发上游前按 `maxUpstreamPayloadBytes`（1 MB）拦截超限请求，返回标准 OpenAI `context_length_exceeded` 错误码，使客户端无需额外配置也能识别并自行压缩，而不是拿到一句无法解读的上游拒绝。
