package telemetry

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hosea3000/workbuddy-checkin/model"
	"github.com/hosea3000/workbuddy-checkin/store"
)

// recorder 捕获上报请求，供断言。
type recorder struct {
	mu       sync.Mutex
	bodies   [][]byte
	status   int
	requests int
}

func (r *recorder) handler(w http.ResponseWriter, req *http.Request) {
	body, _ := io.ReadAll(req.Body)
	r.mu.Lock()
	r.bodies = append(r.bodies, body)
	r.requests++
	r.mu.Unlock()
	status := r.status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
}

func (r *recorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.requests
}

func (r *recorder) last() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.bodies) == 0 {
		return nil
	}
	return r.bodies[len(r.bodies)-1]
}

// newTestService 构造一个指向 httptest 的 Service，并返回 store 与捕获器。
func newTestService(t *testing.T, version string) (*Service, *store.Store, *recorder) {
	t.Helper()
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	rec := &recorder{}
	srv := httptest.NewServer(http.HandlerFunc(rec.handler))
	t.Cleanup(srv.Close)
	oldURL := pingURL
	pingURL = srv.URL
	t.Cleanup(func() { pingURL = oldURL })

	return NewService(st, version), st, rec
}

var uuidV4RE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// 6.6：dev 版本不发起请求
func TestDevVersionDoesNotReport(t *testing.T) {
	svc, _, rec := newTestService(t, "dev")
	svc.checkAndReport(context.Background())
	if rec.count() != 0 {
		t.Fatalf("dev 版本不应上报，实际 %d 次", rec.count())
	}
}

// 6.1：距上次上报 < 24h 时不发请求
func TestSkipsWithin24Hours(t *testing.T) {
	svc, st, rec := newTestService(t, "0.1.4")
	now := time.Now()
	settings := st.GetSettings()
	settings.TelemetryID = "test-id"
	settings.TelemetryLastAt = now.Add(-23 * time.Hour).Unix()
	if err := st.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}
	svc.SetNow(func() time.Time { return now })

	svc.checkAndReport(context.Background())
	if rec.count() != 0 {
		t.Fatalf("24h 内不应上报，实际 %d 次", rec.count())
	}
}

// 6.2：距上次上报 > 24h 且开关开启时发出一次请求，字段与值正确
func TestReportsAfter24Hours(t *testing.T) {
	svc, st, rec := newTestService(t, "0.1.4")
	now := time.Now()
	settings := st.GetSettings()
	settings.TelemetryID = "test-id"
	settings.TelemetryLastAt = now.Add(-25 * time.Hour).Unix()
	if err := st.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveCredential(model.Credential{ID: "c1", UserID: "uid_a"}); err != nil {
		t.Fatal(err)
	}
	if err := st.SaveCredential(model.Credential{ID: "c2", UserID: "uid_b"}); err != nil {
		t.Fatal(err)
	}
	svc.SetNow(func() time.Time { return now })

	svc.checkAndReport(context.Background())
	if rec.count() != 1 {
		t.Fatalf("应上报 1 次，实际 %d 次", rec.count())
	}

	var got payload
	if err := json.Unmarshal(rec.last(), &got); err != nil {
		t.Fatalf("请求体不是合法 JSON: %v", err)
	}
	if got.ID != "test-id" {
		t.Errorf("id = %q, 期望 test-id", got.ID)
	}
	if got.Version != "0.1.4" {
		t.Errorf("v = %q, 期望 0.1.4", got.Version)
	}
	if got.OS == "" || got.Arch == "" {
		t.Errorf("os/arch 不应为空: %q/%q", got.OS, got.Arch)
	}
	if got.Accounts != 2 {
		t.Errorf("accounts = %d, 期望 2", got.Accounts)
	}
	// 成功后记录时间戳
	if st.GetSettings().TelemetryLastAt != now.Unix() {
		t.Errorf("成功后应记录 TelemetryLastAt")
	}
}

// 6.3：请求体不含令牌、邮箱、账号 UID、凭证 ID 等敏感字段
func TestPayloadHasNoSensitiveFields(t *testing.T) {
	svc, st, rec := newTestService(t, "0.1.4")
	settings := st.GetSettings()
	settings.TelemetryID = "test-id"
	settings.TelemetryLastAt = 0
	if err := st.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}
	secret := "super-secret-token"
	if err := st.SaveCredential(model.Credential{
		ID:           "cred-id-should-not-leak",
		UserID:       "uid_account_should_not_leak",
		Email:        "user@example.com",
		Nickname:     "昵称不该泄露",
		AccessToken:  secret,
		RefreshToken: "refresh-should-not-leak",
	}); err != nil {
		t.Fatal(err)
	}

	svc.checkAndReport(context.Background())
	body := string(rec.last())
	for _, forbidden := range []string{
		secret, "refresh-should-not-leak", "user@example.com",
		"uid_account_should_not_leak", "cred-id-should-not-leak", "昵称不该泄露",
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("请求体泄露了敏感值 %q: %s", forbidden, body)
		}
	}
	// 只允许约定字段
	var m map[string]any
	if err := json.Unmarshal(rec.last(), &m); err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{"id": true, "v": true, "os": true, "arch": true, "accounts": true}
	for k := range m {
		if !allowed[k] {
			t.Errorf("出现未约定字段 %q", k)
		}
	}
}

// 6.4：上报失败不写 TelemetryLastAt，下次检查仍会尝试
func TestFailureDoesNotRecordTimestamp(t *testing.T) {
	for _, status := range []int{http.StatusInternalServerError, http.StatusNotFound} {
		svc, st, rec := newTestService(t, "0.1.4")
		rec.status = status
		settings := st.GetSettings()
		settings.TelemetryID = "test-id"
		settings.TelemetryLastAt = 0
		if err := st.SaveSettings(settings); err != nil {
			t.Fatal(err)
		}

		svc.checkAndReport(context.Background())
		if rec.count() != 1 {
			t.Fatalf("status=%d 应尝试 1 次，实际 %d", status, rec.count())
		}
		if st.GetSettings().TelemetryLastAt != 0 {
			t.Errorf("status=%d 失败不应记录时间戳", status)
		}
		// 下次检查仍会尝试
		svc.checkAndReport(context.Background())
		if rec.count() != 2 {
			t.Errorf("失败后下次检查应重试，实际总 %d 次", rec.count())
		}
	}
}

// 6.5：开关关闭时不发请求；关闭再开启后 TelemetryID 不变
func TestDisabledStopsReporting(t *testing.T) {
	svc, st, rec := newTestService(t, "0.1.4")
	settings := st.GetSettings()
	settings.TelemetryID = "stable-id"
	settings.TelemetryEnabled = false
	if err := st.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}

	svc.checkAndReport(context.Background())
	svc.checkAndReport(context.Background())
	if rec.count() != 0 {
		t.Fatalf("关闭后不应上报，实际 %d 次", rec.count())
	}

	// 重新开启沿用同一 ID
	svc.ensureID()
	if got := st.GetSettings().TelemetryID; got != "stable-id" {
		t.Errorf("重新开启后 ID = %q, 期望 stable-id", got)
	}
}

// 6.8：Start 后必存在 TelemetryID（与开关状态无关），且为合法 v4 UUID
func TestEnsureIDGeneratesUUID(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		svc, st, _ := newTestService(t, "0.1.4")
		settings := st.GetSettings()
		settings.TelemetryEnabled = enabled
		if err := st.SaveSettings(settings); err != nil {
			t.Fatal(err)
		}

		svc.ensureID()
		id := st.GetSettings().TelemetryID
		if !uuidV4RE.MatchString(id) {
			t.Errorf("enabled=%v: id = %q, 不是合法 v4 UUID", enabled, id)
		}
	}
}

// 6.9：TelemetryID 落盘不覆盖其他字段
func TestEnsureIDPreservesOtherSettings(t *testing.T) {
	svc, st, _ := newTestService(t, "0.1.4")
	settings := st.GetSettings()
	settings.AutoStart = true
	settings.UpdateProxy = "https://gh-proxy.com"
	settings.ProxyPort = 19090
	settings.ProxyEnabled = true
	settings.ActiveCredentialID = "active-1"
	if err := st.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}

	svc.ensureID()

	got := st.GetSettings()
	if got.TelemetryID == "" {
		t.Fatal("应生成 TelemetryID")
	}
	if !got.AutoStart || got.UpdateProxy != "https://gh-proxy.com" ||
		got.ProxyPort != 19090 || !got.ProxyEnabled || got.ActiveCredentialID != "active-1" {
		t.Errorf("落盘 TelemetryID 时覆盖了其他字段: %+v", got)
	}
}

// 6.10：日志中不出现完整 device_id
func TestFullDeviceIDNotLogged(t *testing.T) {
	svc, st, _ := newTestService(t, "0.1.4")

	var buf strings.Builder
	oldWriter := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(oldWriter) })

	svc.ensureID()
	id := st.GetSettings().TelemetryID
	if id == "" {
		t.Fatal("应生成 TelemetryID")
	}
	if strings.Contains(buf.String(), id) {
		t.Errorf("日志中出现了完整 device_id: %s", buf.String())
	}
	if !strings.Contains(buf.String(), id[:8]) {
		t.Errorf("日志应包含 id 前 8 位用于排查: %s", buf.String())
	}
}

// 6.7：旧 settings.json 缺字段时读出 TelemetryEnabled == true
func TestLegacySettingsDefaultTelemetryEnabled(t *testing.T) {
	dir := t.TempDir()
	legacy := `{"autoStart":false,"updateProxy":"","proxyEnabled":false,"proxyPort":18080,"activeCredentialId":""}`
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := store.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := st.GetSettings()
	if !got.TelemetryEnabled {
		t.Errorf("旧配置应默认开启上报，实际 %v", got.TelemetryEnabled)
	}
	if got.TelemetryID != "" {
		t.Errorf("旧配置不应凭空产生 ID，实际 %q", got.TelemetryID)
	}
}

// Stop 幂等且未 Start 时安全
func TestStopIsIdempotent(t *testing.T) {
	svc, _, _ := newTestService(t, "0.1.4")
	svc.Stop() // 未 Start
	svc.Start(context.Background())
	svc.Stop()
	svc.Stop()
}
