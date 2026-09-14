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
	calls     int32
	jitter    time.Duration
	failFirst int32
}

func (f *fakeSvc) PerformCheckin(_ context.Context, id string) (model.CheckinResult, error) {
	n := atomic.AddInt32(&f.calls, 1)
	if f.failFirst > 0 && n <= f.failFirst {
		return model.CheckinResult{ID: id, Success: false, Message: "签到失败"}, nil
	}
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

func TestRunAllSignsUnsignedImmediately(t *testing.T) {
	s, st, svc := newScheduler(t)
	_ = st.SaveCredential(model.Credential{ID: "1", UserID: "uid_1", Status: model.StatusActive})

	// 无签到时间条件：启动/唤醒直接巡检即签。
	s.RunAll(context.Background())
	if svc.calls != 1 {
		t.Errorf("expected immediate sweep call, got %d", svc.calls)
	}
}

func TestRunAllRetriesUnsignedNextSweep(t *testing.T) {
	s, st, svc := newScheduler(t)
	svc.failFirst = 1
	_ = st.SaveCredential(model.Credential{ID: "1", UserID: "uid_1", Status: model.StatusActive})

	first := s.RunAll(context.Background())
	if len(first) != 1 || first[0].Success {
		t.Fatalf("first sweep should fail, got %+v", first)
	}
	// 下一次巡检（每小时）再次尝试，无当日次数上限。
	second := s.RunAll(context.Background())
	if len(second) != 1 || !second[0].Success {
		t.Fatalf("second sweep should retry and succeed, got %+v", second)
	}
	if svc.calls != 2 {
		t.Errorf("expected 2 upstream calls, got %d", svc.calls)
	}
}
