package codebuddy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestDoJSONBodyReadableAfterReturn 回归登录轮询偶发 "context canceled"：
// doJSON 返回后调用方才能读 resp.Body；若请求 ctx 在 doJSON 返回时被 cancel，
// 慢响应体读取会失败，登录轮询被误判为瞬态错误而永远停在"登录中"。
func TestDoJSONBodyReadableAfterReturn(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":11217`))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte(`,"msg":"pending"}`))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "2.107.0")
	resp, err := c.doJSON(context.Background(), http.MethodGet, srv.URL, nil, nil, 30*time.Second)
	if err != nil {
		t.Fatalf("doJSON: %v", err)
	}
	defer resp.Body.Close()
	body, err := readAllBody(resp)
	if err != nil {
		t.Fatalf("reading body after doJSON returned must not be canceled: %v", err)
	}
	if want := `{"code":11217,"msg":"pending"}`; string(body) != want {
		t.Fatalf("body = %q, want %q", body, want)
	}
}

func readAllBody(resp *http.Response) ([]byte, error) {
	buf := make([]byte, 0, 64)
	tmp := make([]byte, 32)
	for {
		n, err := resp.Body.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			if err.Error() == "EOF" {
				return buf, nil
			}
			return buf, err
		}
	}
}
