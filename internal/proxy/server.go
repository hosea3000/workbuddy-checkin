package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/hosea3000/workbuddy-checkin/upstream/codebuddy"
)

const (
	minPort = 1024
	maxPort = 65535

	// maxRequestBody 限制请求体大小。
	maxRequestBody = 4 << 20
)

// Status 是代理服务的实际运行状态（供设置页展示）。
type Status struct {
	Running bool   `json:"running"`
	Port    int    `json:"port"`
	Error   string `json:"error"`
}

// Service 是本地 OpenAI 兼容代理服务。
type Service struct {
	executor *Executor
	models   *ModelsService

	mu      sync.Mutex
	srv     *http.Server
	ln      net.Listener
	port    int
	lastErr string
}

func NewService(client *codebuddy.Client, creds CredentialProvider, fresh Refresher) *Service {
	models := NewModelsService(client, creds, 30*time.Second)
	return &Service{
		executor: NewExecutor(client, creds, models),
		models:   models,
	}
}

// ValidatePort 校验端口范围（1024–65535，避开特权端口）。
func ValidatePort(port int) error {
	if port < minPort || port > maxPort {
		return fmt.Errorf("端口必须在 %d-%d 之间", minPort, maxPort)
	}
	return nil
}

// listenAddr 显式绑定回环地址：绝不用 ":port"（那会绑定所有网卡）。
func listenAddr(port int) string {
	return fmt.Sprintf("127.0.0.1:%d", port)
}

// Start 在 127.0.0.1:<port> 开始监听；已在运行则先停止。
func (s *Service) Start(port int) error {
	if err := ValidatePort(port); err != nil {
		return err
	}
	s.Stop()
	ln, err := net.Listen("tcp", listenAddr(port))
	if err != nil {
		s.setErr(err)
		return fmt.Errorf("监听 %s 失败：%w", listenAddr(port), err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/chat/completions", s.handleChat)
	mux.HandleFunc("GET /v1/models", s.handleModels)
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}

	s.mu.Lock()
	s.ln, s.srv, s.port, s.lastErr = ln, srv, port, ""
	s.mu.Unlock()

	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.mu.Lock()
			if s.srv == srv {
				s.srv, s.ln, s.port = nil, nil, 0
			}
			s.lastErr = err.Error()
			s.mu.Unlock()
		}
	}()
	return nil
}

// Stop 关闭监听（幂等）。
func (s *Service) Stop() {
	s.mu.Lock()
	srv := s.srv
	s.srv, s.ln, s.port = nil, nil, 0
	s.mu.Unlock()
	if srv == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}

// Status 返回实际运行状态。
func (s *Service) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Status{Running: s.srv != nil, Port: s.port, Error: s.lastErr}
}

func (s *Service) setErr(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastErr = err.Error()
}

func (s *Service) handleChat(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBody)).Decode(&body); err != nil {
		writeError(w, errInvalid("请求体不是合法 JSON"))
		return
	}
	s.executor.Execute(r.Context(), body, w)
}

func (s *Service) handleModels(w http.ResponseWriter, r *http.Request) {
	list := s.models.Available(r.Context())
	created := time.Now().Unix()
	data := make([]any, 0, len(list))
	for _, id := range list {
		data = append(data, map[string]any{
			"id":       id,
			"object":   "model",
			"created":  created,
			"owned_by": "codebuddy",
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"object": "list", "data": data})
}
