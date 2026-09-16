package proxy

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hosea3000/workbuddy-checkin/model"
	"github.com/hosea3000/workbuddy-checkin/upstream/codebuddy"
)

// CredentialProvider 解析代理要使用的凭证（失败返回 *HTTPError）。
type CredentialProvider interface {
	Active(ctx context.Context) (model.Credential, error)
}

// ModelsProvider 提供模型列表，首项作为默认模型。
type ModelsProvider interface {
	Available(ctx context.Context) []string
}

// Executor 执行聊天请求：选凭证 → 构造头 → 上游流式请求 → 转换 / 聚合。
type Executor struct {
	client *codebuddy.Client
	creds  CredentialProvider
	models ModelsProvider
}

func NewExecutor(client *codebuddy.Client, creds CredentialProvider, models ModelsProvider) *Executor {
	return &Executor{client: client, creds: creds, models: models}
}

// ResponseContext 客户端可见响应信封。
type ResponseContext struct {
	ResponseID string
	Created    int64
	Model      string
}

func newResponseContext(responseModel string) ResponseContext {
	return ResponseContext{
		ResponseID: newResponseID(),
		Created:    time.Now().Unix(),
		Model:      responseModel,
	}
}

// newResponseID 生成 chatcmpl-<32 hex>（不引入 google/uuid）。
func newResponseID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return "chatcmpl-" + hex.EncodeToString(b)
}

// maxUpstreamPayloadBytes 是发给上游的 payload 上限。
//
// 上游对超出模型上下文窗口的请求返回 "Illegal API invocation from an unapproved
// channel"（伪装成渠道校验失败的策略性拒绝），客户端无法识别，会话会持续增长到
// 永久失败。实测 1.16 MB / 520 条消息即触发；官方 CLI 在窗口满前会自行压缩，
// 不会发出这种请求，因此这里按客户端预期主动拦截。
const maxUpstreamPayloadBytes = 1 << 20

// Execute 处理一次聊天请求并把响应写入 w（流式 SSE 或非流式 JSON）。
func (x *Executor) Execute(ctx context.Context, body map[string]any, w http.ResponseWriter) {
	if verr := ValidateChatRequest(body); verr != nil {
		writeError(w, verr)
		return
	}
	defaultModel := ""
	if list := x.models.Available(ctx); len(list) > 0 {
		defaultModel = list[0]
	}
	payload, responseModel := PrepareChatPayload(body, defaultModel)
	if stringOr(payload["model"], "") == "" {
		writeError(w, errInvalid("model is required"))
		return
	}
	payloadJSON := mustJSON(payload)
	if len(payloadJSON) > maxUpstreamPayloadBytes {
		log.Printf("[proxy] payload too large: bytes=%d limit=%d msgs=%d — 已按 context_length_exceeded 拒绝，等待客户端压缩后重试",
			len(payloadJSON), maxUpstreamPayloadBytes, messageCount(payload))
		writeError(w, errContextLengthExceeded(fmt.Sprintf(
			"request body is %d bytes, exceeding the %d byte limit for this model; compact the conversation and retry",
			len(payloadJSON), maxUpstreamPayloadBytes)))
		return
	}

	cred, err := x.creds.Active(ctx)
	if err != nil {
		writeError(w, asHTTPError(err))
		return
	}
	headers, err := codebuddy.GenerateHeaders(snapshot(cred), codebuddy.ConversationIDs{}, x.client.CLIVersion)
	if err != nil {
		writeError(w, errCredential("credential header generation failed"))
		return
	}

	respCtx := newResponseContext(responseModel)
	clientWantsStream := boolOr(body["stream"])

	reqCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	httpReq, err := http.NewRequestWithContext(reqCtx, http.MethodPost, x.client.ChatURL(), strings.NewReader(payloadJSON))
	if err != nil {
		writeError(w, &HTTPError{Status: http.StatusBadGateway, Type: errTypeUpstream, Message: "upstream request build failed"})
		return
	}
	for k, v := range headers {
		httpReq.Header.Set(k, v)
	}
	resp, err := x.client.HTTP.Do(httpReq)
	if err != nil {
		writeError(w, &HTTPError{Status: http.StatusBadGateway, Type: errTypeUpstream, Message: "upstream transport error"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw := readLimited(resp.Body, 64*1024)
		he := upstreamHTTPErrorFromBody(resp.StatusCode, raw)
		log.Printf("[proxy] upstream rejected: status=%d type=%s model=%s msgs=%d bytes=%d tools=%d",
			resp.StatusCode, he.Type, stringOr(payload["model"], "?"), messageCount(payload), len(payloadJSON), toolCount(payload))
		writeError(w, he)
		return
	}

	if clientWantsStream {
		streamToClient(w, resp, respCtx)
		return
	}
	result, aerr := aggregate(resp, respCtx)
	if aerr != nil {
		writeError(w, aerr)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// streamToClient 把上游 SSE 转换为 OpenAI chunk 流并实时写出。
func streamToClient(w http.ResponseWriter, resp *http.Response, respCtx ResponseContext) {
	flusher, _ := w.(http.Flusher)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	normalizer := codebuddy.NewStreamNormalizer()
	indexState := codebuddy.NewToolCallIndexState()
	sawDone := false
	var finishSeen bool

	onEvent := func(ev any) bool {
		switch e := ev.(type) {
		case codebuddy.SSEDone:
			sawDone = true
			return false
		case map[string]any:
			if _, hasErr := e["error"]; hasErr {
				writeSSE(w, flusher, codebuddy.FormatSSEError("CodeBuddy upstream stream error", errTypeUpstream))
				return false
			}
			event := codebuddy.ParseEvent(e)
			if event.FinishReason != nil {
				finishSeen = true
			}
			converted := codebuddy.AddOpenAIToolCallIndexes(event, indexState)
			converted = codebuddy.NormalizeChunkEnvelope(converted, respCtx.ResponseID, respCtx.Created, respCtx.Model)
			for _, chunk := range normalizer.Normalize(converted) {
				writeSSE(w, flusher, codebuddy.FormatSSEEvent(chunk))
			}
		}
		return true
	}

	reader := bufio.NewReaderSize(resp.Body, 64*1024)
	var buffer strings.Builder
	buf := make([]byte, 32*1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			buffer.Write(buf[:n])
			for {
				s := buffer.String()
				idx := strings.Index(s, "\n")
				if idx < 0 {
					break
				}
				line := s[:idx]
				rest := s[idx+1:]
				buffer.Reset()
				buffer.WriteString(rest)
				parsed, perr := codebuddy.ParseSSELine(line)
				if perr != nil {
					writeSSE(w, flusher, codebuddy.FormatSSEError(perr.Error(), errTypeProtocol))
					return
				}
				if parsed == nil {
					continue
				}
				if !onEvent(parsed) {
					goto streamEnd
				}
			}
		}
		if err != nil {
			break
		}
	}
streamEnd:
	if sawDone || finishSeen {
		writeSSE(w, flusher, codebuddy.FormatSSEDone())
	} else {
		writeSSE(w, flusher, codebuddy.FormatSSEError(
			"CodeBuddy upstream stream ended without a completion marker", codebuddy.ErrCategoryIncomplete))
	}
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, s string) {
	if s == "" {
		return
	}
	_, _ = w.Write([]byte(s))
	if flusher != nil {
		flusher.Flush()
	}
}

// aggregate 消费上游流并聚合为非流式 OpenAI 响应（对齐 StreamResponseAggregator）。
func aggregate(resp *http.Response, respCtx ResponseContext) (map[string]any, *HTTPError) {
	agg := newAggregator(respCtx)
	sawDone := false
	var finishSeen bool
	var streamErr string

	onEvent := func(ev any) bool {
		switch e := ev.(type) {
		case codebuddy.SSEDone:
			sawDone = true
			return false
		case map[string]any:
			if raw, hasErr := e["error"]; hasErr {
				streamErr = errorMessage(raw)
				return false
			}
			event := codebuddy.ParseEvent(e)
			if event.FinishReason != nil {
				finishSeen = true
			}
			agg.Process(event)
		}
		return true
	}
	if err := codebuddy.IterSSEEventsScanner(resp.Body, onEvent); err != nil {
		return nil, &HTTPError{Status: http.StatusBadGateway, Type: errTypeProtocol, Message: err.Error()}
	}
	if streamErr != "" {
		return nil, &HTTPError{Status: http.StatusBadGateway, Type: errTypeUpstream, Message: streamErr}
	}
	if !sawDone && !finishSeen {
		return nil, &HTTPError{
			Status:  http.StatusBadGateway,
			Type:    codebuddy.ErrCategoryIncomplete,
			Message: "CodeBuddy upstream stream ended without a completion marker",
		}
	}
	return agg.Finalize(), nil
}

// errorMessage 尽力从上游流内 error 字段提取可读消息。
func errorMessage(raw any) string {
	switch t := raw.(type) {
	case string:
		return t
	case map[string]any:
		if m, ok := t["message"].(string); ok && m != "" {
			return m
		}
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return "CodeBuddy upstream stream error"
	}
	return string(b)
}

func readLimited(r io.Reader, n int64) string {
	b, _ := io.ReadAll(io.LimitReader(r, n))
	return string(b)
}

func messageCount(payload map[string]any) int {
	if list, ok := payload["messages"].([]any); ok {
		return len(list)
	}
	return 0
}

func toolCount(payload map[string]any) int {
	if list, ok := payload["tools"].([]any); ok {
		return len(list)
	}
	return 0
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// snapshot 由凭证映射为上游头生成器需要的视图。
// 注意：X-User-Id 实际取 AccountUID；UserID（带 uid_ 前缀）仅兜底。
func snapshot(c model.Credential) codebuddy.CredentialSnapshot {
	return codebuddy.CredentialSnapshot{
		BearerToken:  c.AccessToken,
		UserID:       c.UserID,
		AccountUID:   c.AccountUID,
		Domain:       c.Domain,
		EnterpriseID: c.EnterpriseID,
	}
}

// aggregator 非流式聚合状态。
type aggregator struct {
	ctx          ResponseContext
	content      strings.Builder
	reasoning    strings.Builder
	finishReason *string
	usage        any
	systemFp     any
	indexState   *codebuddy.ToolCallIndexState
	toolCallMap  map[int]map[string]any
}

func newAggregator(ctx ResponseContext) *aggregator {
	return &aggregator{
		ctx:         ctx,
		indexState:  codebuddy.NewToolCallIndexState(),
		toolCallMap: map[int]map[string]any{},
	}
}

func (a *aggregator) Process(event codebuddy.ResponseEvent) {
	if fp, ok := event.ChunkData["system_fingerprint"]; ok && fp != nil {
		a.systemFp = fp
	}
	if u := event.Usage(); u != nil {
		a.usage = u
	}
	if !event.HasChoice() {
		return
	}
	if rc, ok := event.ReasoningContent().(string); ok && rc != "" {
		a.reasoning.WriteString(rc)
	}
	if c, ok := event.Content().(string); ok && c != "" {
		a.content.WriteString(c)
	}
	for _, tc := range event.ToolCalls() {
		tcm, ok := tc.(map[string]any)
		if !ok {
			continue
		}
		fn, ok := tcm["function"].(map[string]any)
		if !ok {
			continue
		}
		idx := a.indexState.Resolve(tcm)
		if idx == nil {
			continue
		}
		toolID, _ := tcm["id"].(string)
		cur, exists := a.toolCallMap[*idx]
		if !exists {
			typ, _ := tcm["type"].(string)
			if typ == "" {
				typ = "function"
			}
			cur = map[string]any{
				"id":       toolID,
				"type":     typ,
				"function": map[string]any{"name": "", "arguments": ""},
			}
			a.toolCallMap[*idx] = cur
		} else if toolID != "" {
			cur["id"] = toolID
		}
		if t, ok := tcm["type"].(string); ok && t != "" {
			cur["type"] = t
		}
		fnMap := cur["function"].(map[string]any)
		if name, ok := fn["name"].(string); ok && name != "" {
			fnMap["name"] = name
		}
		if args, ok := fn["arguments"].(string); ok && args != "" {
			fnMap["arguments"] = fnMap["arguments"].(string) + args
		}
	}
	if event.FinishReason != nil {
		a.finishReason = event.FinishReason
	}
}

func (a *aggregator) Finalize() map[string]any {
	toolCalls := make([]any, 0)
	for i := 0; i <= maxToolIndex(a.toolCallMap); i++ {
		if tc, ok := a.toolCallMap[i]; ok {
			toolCalls = append(toolCalls, tc)
		}
	}

	finalMessage := map[string]any{"role": "assistant", "content": a.content.String()}
	if a.reasoning.Len() > 0 {
		finalMessage["reasoning_content"] = a.reasoning.String()
	}
	if len(toolCalls) > 0 {
		finalMessage["tool_calls"] = toolCalls
	}
	finishReason := "stop"
	if len(toolCalls) > 0 {
		finishReason = "tool_calls"
	}
	if a.finishReason != nil {
		finishReason = *a.finishReason
	}

	resp := map[string]any{
		"id":      a.ctx.ResponseID,
		"object":  "chat.completion",
		"created": a.ctx.Created,
		"model":   a.ctx.Model,
		"choices": []any{map[string]any{
			"index":         0,
			"message":       finalMessage,
			"finish_reason": finishReason,
			"logprobs":      nil,
		}},
	}
	if a.usage != nil {
		resp["usage"] = a.usage
	}
	if a.systemFp != nil {
		resp["system_fingerprint"] = a.systemFp
	}
	return resp
}

func maxToolIndex(m map[int]map[string]any) int {
	max := -1
	for k := range m {
		if k > max {
			max = k
		}
	}
	return max
}
