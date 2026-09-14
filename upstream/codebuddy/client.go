package codebuddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client CodeBuddy 上游客户端（聊天/模型/签到）。
type Client struct {
	Endpoint   string // https://copilot.tencent.com
	CLIVersion string
	HTTP       *http.Client

	// Logf 可选：记录请求诊断（不打印令牌）。为 nil 时不记录。
	Logf func(format string, args ...any)
}

// SetLogger 设置诊断日志函数。
func (c *Client) SetLogger(fn func(format string, args ...any)) { c.Logf = fn }

func (c *Client) logf(format string, args ...any) {
	if c.Logf != nil {
		c.Logf(format, args...)
	}
}

func NewClient(endpoint, cliVersion string) *Client {
	if cliVersion == "" {
		cliVersion = "2.107.0"
	}
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	return &Client{
		Endpoint:   strings.TrimRight(endpoint, "/"),
		CLIVersion: cliVersion,
		HTTP: &http.Client{
			Transport: &http.Transport{
				DialContext:         (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
				MaxIdleConns:        100,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
	}
}

// ChatURL 聊天端点。
func (c *Client) ChatURL() string { return c.Endpoint + "/v2/chat/completions" }

// doJSON 执行 JSON 请求。readTimeout 为整体请求超时（<=0 默认 30s）。
func (c *Client) doJSON(ctx context.Context, method, rawURL string, headers map[string]string, payload any, readTimeout time.Duration) (*http.Response, error) {
	if readTimeout <= 0 {
		readTimeout = 30 * time.Second
	}
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal payload: %w", err)
		}
		body = bytes.NewReader(b)
	}
	// 请求绑定调用方 ctx：不能在 doJSON 返回前 cancel，否则调用方读 resp.Body 会被取消
	// （对齐 work2api 原实现：整体超时由调用方 ctx 控制，连接阶段由 Transport.DialContext 负责）。
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	// http.Transport 会覆盖 Host 头取 req.URL.Host；伪装 Host 用 req.Host 字段
	if h := headers["Host"]; h != "" {
		if u, err := url.Parse(rawURL); err == nil && u.Host == h {
			req.Host = h
		}
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// handleNon200 将非 200 响应转为受控错误；401/403 → 凭证失效。
func handleNon200(resp *http.Response) error {
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	message, errType, code := ParseUpstreamErrorBody(string(raw))
	mapped := MapUpstreamStatus(resp.StatusCode, message)
	if errType != "" && mapped.ErrType == "upstream_error" {
		mapped.ErrType = errType
	}
	_ = code
	return mapped
}

// Checkin 执行每日签到，返回 (success, code, message, credit, err)。
// 成功判定与参考实现一致：code==0 或 msg 含"已签到"。
func (c *Client) Checkin(ctx context.Context, cred CredentialSnapshot) (bool, *int, string, *float64, error) {
	headers, err := GenerateHeaders(cred, ConversationIDs{}, c.CLIVersion)
	if err != nil {
		return false, nil, "credential error", nil, err
	}
	resp, err := c.doJSON(ctx, http.MethodPost, c.Endpoint+"/billing/meter/daily-checkin", headers, map[string]any{}, 30*time.Second)
	if err != nil {
		return false, nil, "无法连接签到服务", nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false, nil, fmt.Sprintf("签到服务返回 HTTP %d", resp.StatusCode), nil, handleNon200(resp)
	}
	var body struct {
		Code *int   `json:"code"`
		Msg  string `json:"msg"`
		Data *struct {
			Credit *float64 `json:"credit"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return false, nil, "签到服务响应格式无效", nil, &UpstreamError{StatusCode: 502, ErrType: ErrCategoryInvalidResp, Message: "checkin response invalid"}
	}
	code := body.Code
	msg := body.Msg
	if msg == "" {
		msg = "签到服务响应缺少有效消息"
	}
	success := (code != nil && *code == 0) || strings.Contains(msg, "已签到")
	var credit *float64
	if code != nil && *code == 0 {
		credit = body.Data.Credit
	}
	return success, code, msg, credit, nil
}
