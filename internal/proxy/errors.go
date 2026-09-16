package proxy

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/hosea3000/workbuddy-checkin/upstream/codebuddy"
)

// 对客户端可见的 OpenAI 错误类别。
const (
	errTypeInvalidRequest = "invalid_request_error"
	errTypeAuth           = "authentication_error"
	errTypeRateLimit      = "rate_limit_error"
	errTypeUpstream       = "upstream_server_error"
	errTypeUnavailable    = "service_unavailable_error"
	errTypeProtocol       = "upstream_protocol_error"
)

// 对客户端可见的 OpenAI 错误码。客户端据此识别可自愈的错误并采取动作
// （例如 context_length_exceeded 触发上下文压缩）。
const errCodeContextLengthExceeded = "context_length_exceeded"

// HTTPError 是可直接序列化为 OpenAI 错误体的错误。
type HTTPError struct {
	Status  int
	Type    string
	Message string
	Code    string
}

func (e *HTTPError) Error() string { return e.Message }

func errInvalid(msg string) *HTTPError {
	return &HTTPError{Status: http.StatusBadRequest, Type: errTypeInvalidRequest, Message: msg}
}

func errUnavailable(msg string) *HTTPError {
	return &HTTPError{Status: http.StatusServiceUnavailable, Type: errTypeUnavailable, Message: msg}
}

func errCredential(msg string) *HTTPError {
	return &HTTPError{Status: http.StatusBadGateway, Type: errTypeAuth, Message: msg}
}

// errContextLengthExceeded 表示请求体超出上游可接受的规模。
// 带 context_length_exceeded 错误码，客户端（如 opencode）会据此触发上下文压缩并重试；
// 若只透传上游的拒绝信息，客户端无法识别，会话会持续膨胀到永久失败。
func errContextLengthExceeded(msg string) *HTTPError {
	return &HTTPError{
		Status:  http.StatusBadRequest,
		Type:    errTypeInvalidRequest,
		Message: msg,
		Code:    errCodeContextLengthExceeded,
	}
}

// asHTTPError 把任意错误归一为 HTTPError（未知错误按上游故障处理）。
func asHTTPError(err error) *HTTPError {
	var he *HTTPError
	if errors.As(err, &he) {
		return he
	}
	return &HTTPError{Status: http.StatusBadGateway, Type: errTypeUpstream, Message: "upstream request failed"}
}

// writeError 写出 OpenAI 错误体。
func writeError(w http.ResponseWriter, e *HTTPError) {
	if e == nil {
		e = &HTTPError{Status: http.StatusInternalServerError, Type: errTypeUpstream, Message: "internal error"}
	}
	var code any
	if e.Code != "" {
		code = e.Code
	}
	writeJSON(w, e.Status, map[string]any{
		"error": map[string]any{"message": e.Message, "type": e.Type, "code": code},
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// mapUpstreamHTTP 把上游受控错误映射为客户端可见错误（设计决策 9）。
func mapUpstreamHTTP(ue *codebuddy.UpstreamError) *HTTPError {
	if ue == nil {
		return &HTTPError{Status: http.StatusBadGateway, Type: errTypeUpstream, Message: "upstream error"}
	}
	if ue.CredInvalid {
		return errCredential(ue.Message)
	}
	status := ue.StatusCode
	if status < 400 || status > 599 {
		status = http.StatusBadGateway
	}
	typ := ue.ErrType
	if typ == "" {
		typ = errTypeUpstream
	}
	return &HTTPError{Status: status, Type: typ, Message: ue.Message}
}

// upstreamHTTPErrorFromBody 解析上游错误体后映射。
func upstreamHTTPErrorFromBody(statusCode int, raw string) *HTTPError {
	message, errType, _ := codebuddy.ParseUpstreamErrorBody(raw)
	ue := codebuddy.MapUpstreamStatus(statusCode, message)
	if errType != "" && ue.ErrType == "upstream_error" {
		ue.ErrType = errType
	}
	return mapUpstreamHTTP(ue)
}
