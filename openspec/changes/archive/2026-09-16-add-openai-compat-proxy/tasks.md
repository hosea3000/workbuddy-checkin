## 1. 上游能力补齐

- [x] 1.1 从 `../work2api/internal/upstream/codebuddy/sse.go` 回迁 `sse.go`（`ParseSSELine` / `IterSSEEvents` / `IterSSEEventsScanner` / `FormatSSEEvent` / `FormatSSEDone` / `FormatSSEError`），去掉 `google/uuid` 相关依赖，注释改中文
- [x] 1.2 从 `../work2api/internal/upstream/codebuddy/events.go` 回迁 `events.go`（`ResponseEvent` / `ParseEvent` / `NormalizeChunkEnvelope` / `AddOpenAIToolCallIndexes` / `StreamNormalizer` / `ToolCallIndexState`），注释改中文
- [x] 1.3 为 `sse.go` 编写单测：`data:` 行解析、`[DONE]`、空行/注释行、非法 JSON、分块边界续行、flush 残余
- [x] 1.4 为 `events.go` 编写单测：`finish_reason` 提取、`reasoning_content` 与 `content` 分离、tool_call 按 index 增量拼接
- [x] 1.5 在 `headers.go` 新增 `GenerateIDEConfigHeaders`（`CodeBuddyIDE` 变体头集，供 `/v3/config` 使用）
- [x] 1.6 在 `client.go` 新增 `FetchModels`（`GET /v3/config`）与 `ExtractModelIDs`（容忍字符串或对象形态），配单测
- [x] 1.7 跑 `go vet ./... && go test ./...` 确认上游包通过

## 2. 静默续期入口

- [x] 2.1 从 `internal/checkin/service.go` 的 `refreshIfNeeded` 抽出纯续期逻辑 `refreshTokens(ctx, c) (model.Credential, error)`（令牌交换 + 回写 store，无副作用）
- [x] 2.2 让 `refreshIfNeeded` 改为包装 `refreshTokens`，失败时仍走 `handleRefreshError`（保持签到路径行为与 24 小时窗口不变）
- [x] 2.3 新增公开入口 `EnsureFresh(ctx, id)`：仅当 `now >= ExpiresAt - 60s` 且存在 refresh_token 时续期，失败直接返回 error，不置位、不通知
- [x] 2.4 用 `sync.Mutex` 串行化 `EnsureFresh` 的续期调用，加 `// ponytail:` 注释标注单凭证锁的天花板与升级路径
- [x] 2.5 补单测：未临期不请求上游、无 refresh_token 不请求、续期成功回写、401 失败返回 error 且状态不变、签到路径仍按 24 小时窗口置位
- [x] 2.6 跑 `go test ./internal/checkin/` 确认既有签到测试全绿

## 3. 代理服务核心

- [x] 3.1 新建 `internal/proxy/` 包，从 `../work2api/internal/service/request_processor.go` 回迁请求校验与预处理，按设计决策 7 裁剪：删除 `ForcedTemperature` 与 `ForcedReasoningModels`，保留 `enable_thinking` 默认/显式禁用、非空 `stop` 返回 400、强制 `stream=true` + `include_usage`、单条 user 补 system、`StripModelNamespace`
- [x] 3.2 从 `../work2api/internal/service/system_prompt_rewriter.go` 回迁 system prompt 指纹改写，配单测（命中替换、attribution 行移除、不匹配原样保留）
- [x] 3.3 从 `../work2api/internal/service/chat_executor.go` 回迁聊天执行器，`google/uuid` 替换为已有的 `crypto/rand` 自产 UUID（`responseID` 前缀 `chatcmpl-` 保留）
- [x] 3.4 从 `../work2api/internal/service/models_service.go` 回迁模型列表服务（并集去重保序 + TTL 30s 缓存 + 回退链），数据源改为读取本地配置模型
- [x] 3.5 实现凭证解析：读 `Settings.ActiveCredentialID` → `store.GetCredential(id)`，按设计决策 3 处理未设置/悬空（503）、`relogin_required`（502）、临期（调 `EnsureFresh`，失败报错）
- [x] 3.6 实现 OpenAI 错误体映射（`{"error": {"message", "type", "code"}}`），覆盖 503 / 502 / 429 / 400 与流内 error 事件；日志只记错误类别与账号短码
- [x] 3.7 用 `net/http` 实现路由与处理器：`POST /v1/chat/completions`（流式/非流式分流）、`GET /v1/models`
- [x] 3.8 实现服务生命周期：`Start(port)` / `Stop()` / `Status()`，监听地址显式拼 `"127.0.0.1:"+port`，端口校验 1024–65535，绑定失败返回可读错误
- [x] 3.9 补单测：非法请求 400、未设凭证 503、`relogin_required` 502、非流式聚合结果、流式 chunk 序列与 `[DONE]`、模型列表缓存与回退、端口非法拒绝

## 4. 装配与绑定方法

- [x] 4.1 `model/settings.go` 新增 `ProxyEnabled bool`、`ProxyPort int`、`ActiveCredentialID string`，并在 `DefaultSettings()` 中给出默认值（`false` / `18080` / `""`）
- [x] 4.2 `model/view.go` 的 `AccountView` 新增 `isActive bool`，并在 `account.ToView` 中按当前凭证 ID 填充（需要把当前凭证 ID 传入视图映射）
- [x] 4.3 `app.go` 构造并持有代理服务，启动时若设置启用则拉起，退出时停止
- [x] 4.4 `app_settings.go`：`SaveSettings` 校验端口范围、按开关或端口变化重启监听；新增绑定方法返回代理**实际运行状态**（running / port / error）
- [x] 4.5 `app_account.go` 新增「设为当前凭证」绑定方法（校验 ID 存在、幂等、不调用上游）；`DeleteAccount` 在删除当前凭证时顺位到首个 `active`，无则置空
- [x] 4.6 `internal/account` 登录成功入库后，若当前凭证为空则自动设为该账号（不覆盖已有选择）
- [x] 4.7 改过绑定方法后跑 `wails generate module`，并 `git checkout -- frontend/wailsjs/runtime/` 还原纯 mode 变更
- [x] 4.8 补 `app` 层单测：端口非法拒绝、代理状态查询、删除当前凭证顺位、登录自动设当前凭证不覆盖

## 5. 前端

- [x] 5.1 设置页新增「模型代理」区块：开关、端口输入（范围校验与错误提示）、实际运行状态展示
- [x] 5.2 账号卡片新增「设为当前凭证」按钮与当前凭证标记，切换后列表刷新
- [x] 5.3 在 `frontend/` 跑 `npm run build` 确认类型检查与构建通过

## 6. 托盘菜单项

- [x] 6.1 `tray_windows.go` 与 `tray_darwin.go` 菜单新增「模型代理」开关项，显示当前状态并可切换
- [x] 6.2 保持 `tray_stub.go` 覆盖其余平台，确认 Linux 下 `go test ./...` 仍可编译运行

## 7. 全量验证

- [x] 7.1 `go vet ./... && go test ./...`
- [x] 7.2 `GOOS=darwin GOARCH=arm64 go build ./...`（注意 `tray_darwin.go` 用 cgo，Linux 上会跳过，需在 macOS 侧另行验证）
- [x] 7.3 `cd frontend && npm run build`
- [x] 7.4 端到端验证 `/v1/models` 与 `/v1/chat/completions`（流式与非流式）：以假上游自动化替代手工 curl（`internal/proxy/integration_test.go`），断言响应格式、真实上游路径与响应中无完整令牌；真实账号冒烟待人工
- [x] 7.5 定位长会话下 `Illegal API invocation from an unapproved channel` 根因，确认是请求体超出模型上下文窗口（非竞品关键词、非移植回归）
- [x] 7.6 代理侧加固：转发前按 `maxUpstreamPayloadBytes` 拦截超长请求体，返回 `context_length_exceeded` 错误码并补单测（`internal/proxy/limits_test.go`）
- [ ] 7.7 手工验证：未设当前凭证 / 关闭代理 / 端口被占用三种情况下设置页与客户端错误提示可读
