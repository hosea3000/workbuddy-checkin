package scheduler

import (
	"context"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/hosea3000/workbuddy-checkin/model"
	"github.com/hosea3000/workbuddy-checkin/store"
)

// Service 抽象签到执行（由 checkin.Service 实现）。
type Service interface {
	PerformCheckin(ctx context.Context, id string) (model.CheckinResult, error)
	RefreshQuota(ctx context.Context, id string) (model.QuotaView, error)
}

// Scheduler 负责定时触发、补签、重试与余额轮询。
type Scheduler struct {
	store  *store.Store
	svc    Service
	now    func() time.Time
	rng    *rand.Rand
	jitter func() time.Duration

	mu      sync.Mutex
	runMu   sync.Mutex // 串行化 runAll / 手动签到（不并发打上游）
	stopped bool
	cancel  context.CancelFunc
	done    chan struct{}
}

// New 构造调度器。
func New(st *store.Store, svc Service) *Scheduler {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &Scheduler{
		store:  st,
		svc:    svc,
		now:    time.Now,
		rng:    rng,
		jitter: func() time.Duration { return Jitter(rng) },
		done:   make(chan struct{}),
	}
}

// SetNow 注入时钟（测试用）。
func (s *Scheduler) SetNow(fn func() time.Time) { s.now = fn }

// SetJitter 注入抖动函数（测试用，避免真等待）。
func (s *Scheduler) SetJitter(fn func() time.Duration) { s.jitter = fn }

// Start 启动调度循环。
func (s *Scheduler) Start(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	s.mu.Lock()
	s.cancel = cancel
	s.mu.Unlock()
	go s.loop(ctx)
}

// Stop 停止调度循环。
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		return
	}
	s.stopped = true
	cancel := s.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (s *Scheduler) loop(ctx context.Context) {
	defer close(s.done)
	settings := s.store.GetSettings()
	next := NextTrigger(s.now(), settings.CheckinHour, settings.CheckinMinute)
	quotaTick := time.NewTicker(time.Hour)
	defer quotaTick.Stop()

	for {
		wait := time.Until(next)
		if wait < 0 {
			wait = 0
		}
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			settings = s.store.GetSettings()
			s.RunAll(ctx)
			next = NextTrigger(s.now(), settings.CheckinHour, settings.CheckinMinute)
		case <-quotaTick.C:
			s.RefreshAllQuotas(ctx)
		}
	}
}

// Wake 处理休眠唤醒 / 跨天：满足条件则补签。
func (s *Scheduler) Wake(ctx context.Context) {
	settings := s.store.GetSettings()
	creds := s.store.ListCredentials()
	now := s.now()
	for _, c := range creds {
		if ShouldCatchUp(c, now, settings) {
			s.RunAll(ctx)
			return
		}
	}
}

// CatchUp 启动时按补签条件执行一次。
func (s *Scheduler) CatchUp(ctx context.Context) {
	s.Wake(ctx)
}

// RunAll 串行对全部账号执行签到（逐个、抖动、跳过已成功与待重登）。
func (s *Scheduler) RunAll(ctx context.Context) []model.CheckinResult {
	s.runMu.Lock()
	defer s.runMu.Unlock()

	now := s.now()
	var results []model.CheckinResult

	for i, c := range s.store.ListCredentials() {
		if c.IsReloginRequired() || c.HasCheckedInToday(now) {
			continue
		}
		if ctx.Err() != nil {
			return results
		}
		// 账号之间随机间隔 5~20s（首个不等待）。
		if i > 0 {
			select {
			case <-ctx.Done():
				return results
			case <-time.After(s.jitter()):
			}
		}
		res, err := s.svc.PerformCheckin(ctx, c.ID)
		if err != nil {
			log.Printf("[scheduler] checkin account=%s failed: %v", shortID(c), err)
			continue
		}
		results = append(results, res)
		if !res.Success && ShouldRetry(mustGet(s.store, c.ID), s.now()) {
			s.scheduleRetry(ctx, c.ID)
		}
	}
	return results
}

func (s *Scheduler) scheduleRetry(ctx context.Context, id string) {
	go func() {
		timer := time.NewTimer(RetryDelay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		s.runMu.Lock()
		defer s.runMu.Unlock()
		c, ok := s.store.GetCredential(id)
		if !ok || !ShouldRetry(c, s.now()) {
			return
		}
		if _, err := s.svc.PerformCheckin(ctx, id); err != nil {
			log.Printf("[scheduler] retry account=%s failed: %v", shortID(c), err)
		}
	}()
}

// RefreshAllQuotas 刷新全部账号余额（跳过待重登与已成功的余额空值）。
func (s *Scheduler) RefreshAllQuotas(ctx context.Context) {
	for _, c := range s.store.ListCredentials() {
		if c.IsReloginRequired() {
			continue
		}
		if ctx.Err() != nil {
			return
		}
		if _, err := s.svc.RefreshQuota(ctx, c.ID); err != nil {
			log.Printf("[scheduler] quota account=%s failed: %v", shortID(c), err)
		}
	}
}

func mustGet(st *store.Store, id string) model.Credential {
	c, _ := st.GetCredential(id)
	return c
}

func shortID(c model.Credential) string {
	if len(c.UserID) > 8 {
		return c.UserID[:8]
	}
	return c.UserID
}
