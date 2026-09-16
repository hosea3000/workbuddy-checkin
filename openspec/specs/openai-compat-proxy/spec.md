# openai-compat-proxy Specification

## Purpose

提供一个仅监听回环地址的 OpenAI 兼容 HTTP 服务，用设置中唯一的「当前凭证」转发聊天补全请求到上游，并在本地完成流式/非流式、错误与日志脱敏的适配。

## Requirements

### Requirement: 代理服务监听与生命周期

系统 SHALL 提供一个 OpenAI 兼容的本地 HTTP 服务，监听地址 MUST 为 `127.0.0.1:<端口>`（MUST NOT 绑定 `0.0.0.0` 或其他网卡），端口取自设置项并 SHALL 校验为 1024–65535。服务 SHALL 仅在设置为启用时运行，SHALL 在应用退出时停止，且 SHALL NOT 因主窗口关闭（隐藏到后台）而停止。

系统 SHALL 提供绑定方法查询服务的**实际运行状态**（是否运行、实际端口、错误信息），供前端轮询展示；SHALL NOT 仅返回持久化的开关意图。

#### Scenario: 启用后开始监听
- **WHEN** 用户在设置页开启「模型代理」且端口合法
- **THEN** 服务在 `127.0.0.1:<端口>` 开始监听，状态查询返回运行中

#### Scenario: 端口被占用
- **WHEN** 指定端口已被其他程序占用
- **THEN** 服务不启动，状态查询返回可读的错误信息，设置页展示该错误

#### Scenario: 端口非法
- **WHEN** 用户填入小于 1024 或大于 65535 的端口
- **THEN** 系统拒绝保存或拒绝启动并提示端口范围，不产生绑定失败

#### Scenario: 修改端口重启监听
- **WHEN** 服务运行中用户修改端口并保存
- **THEN** 旧监听被关闭、新端口开始监听，状态查询反映新端口

#### Scenario: 关闭开关停止服务
- **WHEN** 用户关闭「模型代理」
- **THEN** 监听关闭，端口不再可连接

#### Scenario: 隐藏窗口不停止服务
- **WHEN** 用户关闭主窗口使应用隐藏到后台
- **THEN** 代理服务继续监听，客户端请求仍可正常处理

#### Scenario: 退出应用停止服务
- **WHEN** 用户从托盘退出应用
- **THEN** 监听关闭，进程不残留

#### Scenario: 仅本机可访问
- **WHEN** 同一局域网内的其他主机尝试连接该端口
- **THEN** 连接失败（服务仅绑定回环地址）

### Requirement: OpenAI 兼容聊天补全端点

系统 SHALL 提供 `POST /v1/chat/completions`，接受 OpenAI Chat Completions 格式请求体（`model`、`messages`、`stream`、`temperature` 等），并返回 OpenAI 格式响应。该端点 SHALL NOT 要求任何鉴权凭据。

#### Scenario: 流式请求成功
- **WHEN** 客户端请求 `stream=true`
- **THEN** 系统以 `text/event-stream` 返回 OpenAI chunk 流（`data: {...}` 行，以 `data: [DONE]` 结束），内容与上游事件实时对应

#### Scenario: 非流式请求成功
- **WHEN** 客户端请求 `stream=false`
- **THEN** 系统消费完上游流后返回单个 OpenAI Chat Completion JSON，`choices` 与 `usage` 字段齐全

#### Scenario: 请求体不合法
- **WHEN** 请求缺少 `messages` 或 `messages` 为空
- **THEN** 系统返回 400 与 OpenAI 错误格式的 `invalid_request_error`

#### Scenario: 未指定模型
- **WHEN** 请求体未提供 `model`
- **THEN** 系统使用默认模型（模型列表首项）；若模型列表为空则返回 400

### Requirement: 非流式请求聚合上游流

由于上游仅提供流式响应，系统 MUST 对 `stream=false` 的请求消费完上游 SSE 流后聚合为完整响应，且 SHALL 在客户端主动断连时取消上游请求。

#### Scenario: 聚合正文与结束原因
- **WHEN** 上游流输出多个内容分片后以完成标记结束
- **THEN** 聚合结果的 `choices[0].message.content` 为全部分片拼接，`finish_reason` 为上游最终值

#### Scenario: 客户端断连取消上游
- **WHEN** 非流式请求进行中客户端断开连接
- **THEN** 系统取消上游 HTTP 请求，不再继续消费 SSE 流

### Requirement: 上游 SSE 事件到 OpenAI chunk 的转换

系统 SHALL 将上游 SSE 事件转换为 OpenAI chunk 格式，语义与参考实现对齐，包括：`reasoning_content` 与正文 `content` 分离、tool_call 增量索引（按 index 聚合 id/name/arguments 分片）、`finish_reason` 透传、usage 事件转 OpenAI `usage` 字段。

#### Scenario: reasoning 内容分离
- **WHEN** 上游事件包含推理内容片段与正文内容片段
- **THEN** 对应 chunk 的 delta 中分别输出 `reasoning_content` 与 `content` 字段

#### Scenario: tool_call 增量聚合
- **WHEN** 上游流式输出多个 tool_call 分片（含 index、id、function.name、arguments 增量）
- **THEN** 输出 chunk 的 `delta.tool_calls` 按 index 正确对应，非流式聚合结果中 `arguments` 拼接为完整字符串

#### Scenario: 流未正常结束
- **WHEN** 上游流在未出现完成标记的情况下中断
- **THEN** 系统向客户端输出 `upstream_incomplete` 类别的 error 事件，而非静默结束

#### Scenario: 上游返回流内错误
- **WHEN** 上游事件体中包含 `error` 字段
- **THEN** 系统向客户端输出 `upstream_error` 类别的 error 事件并结束该流

### Requirement: 请求预处理策略

系统 SHALL 对 `temperature` 原样透传，MUST NOT 改写客户端提供的值。系统 SHALL 对非空 `stop` 返回 400（上游无法正确报告停止序列命中）。系统 SHALL 在客户端未表态时默认开启 `enable_thinking`，并在客户端显式禁用时尊重其选择。系统 MUST 强制 `stream=true`（上游仅支持流式）并附加 `stream_options.include_usage=true`。系统 SHALL 对 system 消息执行指纹改写（移除 Claude Code 身份声明与 attribution 行），不匹配时原样保留。系统 SHALL 在转发前拦截超出上游可接受规模的请求体，返回 `400` 且错误码为 `context_length_exceeded`，MUST NOT 把上游的策略性拒绝原样透传给客户端。

#### Scenario: temperature 透传
- **WHEN** 客户端请求携带 `temperature=0.2`
- **THEN** 转发给上游的请求体中 `temperature` 仍为 `0.2`，未被改写

#### Scenario: 超长请求触发上下文压缩
- **WHEN** 序列化后的上游请求体超过代理允许的上限
- **THEN** 系统返回 `400`，错误体 `code` 为 `context_length_exceeded`，且 SHALL NOT 向上游发出该请求
- **AND** 客户端据此压缩上下文后重试可正常完成

#### Scenario: 非空停止序列被拒绝
- **WHEN** 请求体携带非空 `stop` 数组
- **THEN** 系统返回 400，错误说明不支持停止序列

#### Scenario: 思考开关尊重显式禁用
- **WHEN** 客户端请求携带 `enable_thinking=false`
- **THEN** 转发给上游的请求体中不含 `enable_thinking` 或为禁用状态

#### Scenario: 思考开关默认开启
- **WHEN** 客户端请求未提及 `enable_thinking`
- **THEN** 转发给上游的请求体中 `enable_thinking` 为 `true`

#### Scenario: system 消息指纹改写
- **WHEN** 请求的 system 消息包含 Claude Code 身份声明或 attribution 头行
- **THEN** 转发给上游的文本中该指纹被替换或移除

### Requirement: 模型列表端点

系统 SHALL 提供 `GET /v1/models`，返回本地配置模型与上游 `GET /v3/config` 实际模型的有序去重并集，结果 SHALL 缓存（TTL 30 秒）。上游查询失败时 SHALL 回退到缓存，无缓存时回退到配置模型列表。

#### Scenario: 返回合并模型列表
- **WHEN** 客户端请求模型列表且上游可用
- **THEN** 响应包含配置模型与上游实际模型的并集，每项含 `id` 与 `object: "model"`

#### Scenario: 上游不可用时回退缓存
- **WHEN** 上游模型查询失败但缓存仍在
- **THEN** 返回缓存中的模型列表

#### Scenario: 无缓存且上游失败
- **WHEN** 上游模型查询失败且无缓存
- **THEN** 返回配置模型列表而非错误

#### Scenario: 命中缓存不重复请求上游
- **WHEN** 距上次成功查询未超过 TTL 时再次请求模型列表
- **THEN** 系统直接返回缓存，不发起上游请求

### Requirement: 代理侧凭证选取

系统 SHALL 使用设置中的「当前凭证」作为唯一上游凭证，MUST NOT 实现凭证池、轮换或自动降级到其他账号。当当前凭证缺失、已标记为需重新登录、或令牌无法续期时，系统 SHALL 返回明确错误而非改选其他账号。

#### Scenario: 使用当前凭证
- **WHEN** 已设置当前凭证且其状态为 `active`
- **THEN** 系统使用该凭证的令牌构造上游请求

#### Scenario: 未设置当前凭证
- **WHEN** 设置中不存在当前凭证或其指向的凭证已被删除
- **THEN** 系统返回 503，提示需要设置当前凭证

#### Scenario: 当前凭证需重新登录
- **WHEN** 当前凭证状态为 `relogin_required`
- **THEN** 系统返回 502，提示该凭证已被上游拒绝需重新登录，且不改选其他账号

#### Scenario: 令牌临期自动续期
- **WHEN** 当前凭证的 `ExpiresAt` 已到（或距过期不足 60 秒）
- **THEN** 系统在转发前续期；成功则使用新令牌继续处理请求

#### Scenario: 续期失败返回错误
- **WHEN** 令牌续期失败
- **THEN** 系统向客户端返回错误，且 SHALL NOT 将账号状态置为 `relogin_required`、SHALL NOT 发送系统通知

### Requirement: 代理错误映射与日志脱敏

系统 SHALL 将上游与本地错误映射为 OpenAI 错误体 `{"error": {"message", "type", "code"}}`。系统 SHALL 在日志中记录错误类别与账号短码，MUST NOT 记录完整 `access_token` / `refresh_token`。

#### Scenario: 上游限流
- **WHEN** 上游返回 429
- **THEN** 系统返回 429 且错误体 `type` 为 `rate_limit_error`

#### Scenario: 上游服务异常
- **WHEN** 上游返回 5xx
- **THEN** 系统返回 502 且错误体 `type` 为 `upstream_server_error`

#### Scenario: 日志不含令牌
- **WHEN** 代理处理任意请求（含失败）
- **THEN** 应用日志中不出现完整令牌值
