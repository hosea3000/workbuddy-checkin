package scheduler

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hosea3000/workbuddy-checkin/model"
	"github.com/hosea3000/workbuddy-checkin/store"
)

type fakeSvc struct {
	calls  int32
	jitter time.Duration
}

func (f *fakeSvc) PerformCheckin(_ context.Context, id string) (model.CheckinResult, error) {
	atomic.AddInt32(&f.calls, 1)
	return model.CheckinResult{ID: id, Success: true, Message: "已签到 09:30"}, nil
}
func (f *fakeSvc) RefreshQuota(_ context.Context, _ string) (model.QuotaView, error) {
	return model.QuotaView{}, nil
}

func newScheduler(t *testing.T) (*Scheduler, *store.Store, *fakeSvc) {
	t.Helper()
	st, err := store.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc := &fakeSvc{}
	s := New(st, svc)
	s.SetJitter(func() time.Duration { return time.Millisecond })
	return s, st, svc
}

func TestRunAllSkipsCheckedInAndRelogin(t *testing.T) {
	s, st, svc := newScheduler(t)
	_ = st.SaveCredential(model.Credential{ID: "1", UserID: "uid_1", Status: model.StatusActive})
	_ = st.SaveCredential(model.Credential{ID: "2", UserID: "uid_2", Status: model.StatusActive, TodayDate: model.Today(time.Now()), TodaySuccess: true})
	_ = st.SaveCredential(model.Credential{ID: "3", UserID: "uid_3", Status: model.StatusReloginRequired})

	results := s.RunAll(context.Background())
	if len(results) != 1 {
		t.Fatalf("expected 1 checkin, got %d", len(results))
	}
	if svc.calls != 1 {
		t.Errorf("expected 1 upstream call, got %d", svc.calls)
	}
}

func TestRunAllSerialOrderJitter(t *testing.T) {
	s, st, _ := newScheduler(t)
	_ = st.SaveCredential(model.Credential{ID: "1", UserID: "uid_1", Status: model.StatusActive})
	_ = st.SaveCredential(model.Credential{ID: "2", UserID: "uid_2", Status: model.StatusActive})

	start := time.Now()
	results := s.RunAll(context.Background())
	elapsed := time.Since(start)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// 账号间至少一次抖动（测试注入 1ms）
	if elapsed < time.Millisecond {
		t.Errorf("expected jitter wait between accounts, elapsed=%v", elapsed)
	}
}

func TestWakeCatchUp(t *testing.T) {
	s, st, svc := newScheduler(t)
	// 已过签到时间（默认 09:30），未签到 → 应补签
	s.SetNow(func() time.Time { return time.Date(2026, 9, 14, 12, 0, 0, 0, time.Local) })
	_ = st.SaveCredential(model.Credential{ID: "1", UserID: "uid_1", Status: model.StatusActive})

	s.Wake(context.Background())
	if svc.calls != 1 {
		t.Errorf("expected catch-up call, got %d", svc.calls)
	}
}

func TestWakeNoCatchUpBeforeTrigger(t *testing.T) {
	s, st, svc := newScheduler(t)
	s.SetNow(func() time.Time { return time.Date(2026, 9, 14, 8, 0, 0, 0, time.Local) })
	_ = st.SaveCredential(model.Credential{ID: "1", UserID: "uid_1", Status: model.StatusActive})

	s.Wake(context.Background())
	if svc.calls != 0 {
		t.Errorf("should not catch up before trigger, got %d calls", svc.calls)
	}
}
