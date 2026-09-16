package proxy

import (
	"context"
	"sync"
	"time"

	"github.com/hosea3000/workbuddy-checkin/upstream/codebuddy"
)

// fallbackModels 上游不可用且无缓存时的兜底模型列表（与参考实现默认值对齐）。
var fallbackModels = []string{"glm-5.2", "deepseek-v4-pro"}

// ModelsService 模型列表：上游实际模型 ∪ 兜底模型（有序去重），带 TTL 缓存。
type ModelsService struct {
	client *codebuddy.Client
	creds  CredentialProvider
	ttl    time.Duration

	// ponytail: 持锁跨网络调用会串行化 /v1/models；该端点有 TTL 缓存且调用稀疏，够用。
	// 若客户端高频轮询模型列表，改为先查缓存、锁外拉取。
	mu       sync.Mutex
	cache    []string
	cacheAt  time.Time
	hasCache bool
}

func NewModelsService(client *codebuddy.Client, creds CredentialProvider, ttl time.Duration) *ModelsService {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &ModelsService{client: client, creds: creds, ttl: ttl}
}

// Available 返回有序去重的模型列表：命中缓存直接返回，上游失败回退上次成功结果，
// 再无缓存则回退兜底列表。
func (m *ModelsService) Available(ctx context.Context) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.hasCache && time.Since(m.cacheAt) < m.ttl {
		return append([]string{}, m.cache...)
	}
	if actual := m.fetchActual(ctx); len(actual) > 0 {
		m.cache = orderedUnion(actual, fallbackModels)
		m.cacheAt = time.Now()
		m.hasCache = true
		return append([]string{}, m.cache...)
	}
	if m.hasCache {
		return append([]string{}, m.cache...)
	}
	return append([]string{}, fallbackModels...)
}

func (m *ModelsService) fetchActual(ctx context.Context) []string {
	cred, err := m.creds.Active(ctx)
	if err != nil {
		return nil
	}
	models, err := m.client.FetchModels(ctx, snapshot(cred))
	if err != nil {
		return nil
	}
	return models
}

// orderedUnion 按传入顺序去重合并（先出现的优先）。
func orderedUnion(lists ...[]string) []string {
	seen := map[string]bool{}
	var out []string
	for _, list := range lists {
		for _, s := range list {
			if s == "" || seen[s] {
				continue
			}
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
