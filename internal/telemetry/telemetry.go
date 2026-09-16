// Package telemetry 提供匿名使用数据上报：每小时检查一次，距上次成功上报超过
// 24 小时则发起一次 POST。上报内容仅含版本号、系统、架构与账号数量，不含任何
// 账号或个人信息。失败静默且不记录时间，下一小时自然重试。
package telemetry

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/hosea3000/workbuddy-checkin/store"
)

// pingURL 是上报地址。变量便于测试注入 httptest 地址。
// 服务端契约见 docs/DESIGN.md：POST /ping，body 见 payload，收到即返回 200 且无响应体。
var pingURL = "https://workbuddy.frp.hosea123.com/ping"

// 上报间隔：距上次成功上报超过该时长才再次上报。
const reportInterval = 24 * time.Hour

// checkInterval 是检查循环的触发周期。
const checkInterval = time.Hour

// requestTimeout 是单次上报的超时，避免请求悬挂。
const requestTimeout = 10 * time.Second

// payload 是上报请求体。字段顺序固定，便于服务端与测试断言。
type payload struct {
	ID       string `json:"id"`
	Version  string `json:"v"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Accounts int    `json:"accounts"`
}

// Service 按小时检查并上报匿名使用数据。
type Service struct {
	store   *store.Store
	version string

	client *http.Client
	now    func() time.Time

	stop chan struct{}
	done chan struct{}
}

// NewService 构造上报服务。appVersion 为应用运行时版本（本地开发为 dev）。
func NewService(st *store.Store, appVersion string) *Service {
	return &Service{
		store:   st,
		version: appVersion,
		client:  &http.Client{Timeout: requestTimeout},
		now:     time.Now,
		done:    make(chan struct{}),
	}
}

// SetNow 注入时钟（测试用）。
func (s *Service) SetNow(fn func() time.Time) { s.now = fn }

// Start 确保设备标识存在（无条件，与开关无关），启动时先检查一次（仍受 24h
// 间隔与开关约束），随后启动小时级检查循环。
func (s *Service) Start(ctx context.Context) {
	s.ensureID()
	s.stop = make(chan struct{})
	go func() {
		s.checkAndReport(ctx)
		s.loop(ctx)
	}()
}

// Stop 停止检查循环；幂等，未 Start 时调用也安全。
func (s *Service) Stop() {
	if s.stop == nil {
		return
	}
	select {
	case <-s.stop:
		return
	default:
	}
	close(s.stop)
	<-s.done
}

// ensureID 在设备标识缺失时生成 UUID v4 并落盘。始终经 store 的「读→改→整体写回」，
// 不直接写 settings.json，以免覆盖用户同期的其他设置修改。
func (s *Service) ensureID() {
	settings := s.store.GetSettings()
	if settings.TelemetryID != "" {
		return
	}
	id, err := newUUID()
	if err != nil {
		log.Printf("[telemetry] 生成设备标识失败: %v", err)
		return
	}
	settings.TelemetryID = id
	if err := s.store.SaveSettings(settings); err != nil {
		log.Printf("[telemetry] 保存设备标识失败: %v", err)
		return
	}
	log.Printf("[telemetry] 已生成设备标识 %s…", id[:8])
}

// loop 每小时检查一次是否需要上报。
func (s *Service) loop(ctx context.Context) {
	defer close(s.done)
	tick := time.NewTicker(checkInterval)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stop:
			return
		case <-tick.C:
			s.checkAndReport(ctx)
		}
	}
}

// checkAndReport 按开关、版本与间隔门槛决定是否上报。
func (s *Service) checkAndReport(ctx context.Context) {
	settings := s.store.GetSettings()
	if !settings.TelemetryEnabled {
		return
	}
	if s.version == "" || s.version == "dev" {
		return
	}
	now := s.now()
	if settings.TelemetryLastAt != 0 && now.Sub(time.Unix(settings.TelemetryLastAt, 0)) <= reportInterval {
		return
	}
	s.report(ctx, settings.TelemetryID, now)
}

// report 发起一次上报。仅在收到 HTTP 200 时记录时间戳；失败静默、不记录，
// 使下一个小时级检查可再次尝试。
func (s *Service) report(ctx context.Context, id string, now time.Time) {
	body, err := json.Marshal(payload{
		ID:       id,
		Version:  s.version,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		Accounts: len(s.store.ListCredentials()),
	})
	if err != nil {
		log.Printf("[telemetry] 序列化失败: %v", err)
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, pingURL, bytes.NewReader(body))
	if err != nil {
		log.Printf("[telemetry] 构造请求失败: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("[telemetry] 上报失败: %v", err)
		return
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("[telemetry] 上报失败: HTTP %d", resp.StatusCode)
		return
	}
	settings := s.store.GetSettings()
	settings.TelemetryLastAt = now.Unix()
	if err := s.store.SaveSettings(settings); err != nil {
		log.Printf("[telemetry] 记录上报时间失败: %v", err)
		return
	}
	log.Printf("[telemetry] 上报成功 v=%s os=%s arch=%s accounts=%d",
		s.version, runtime.GOOS, runtime.GOARCH, len(s.store.ListCredentials()))
}

// newUUID 生成 RFC 4122 v4 UUID：122 位随机量，不含时间、机器码或主机名。
func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40 // 版本 4
	b[8] = (b[8] & 0x3f) | 0x80 // 变体位
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
