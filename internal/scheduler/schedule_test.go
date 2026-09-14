package scheduler

import (
	"math/rand"
	"testing"
	"time"

	"github.com/hosea3000/workbuddy-checkin/model"
)

func TestNextTriggerToday(t *testing.T) {
	now := time.Date(2026, 9, 14, 8, 0, 0, 0, time.Local)
	got := NextTrigger(now, 9, 30)
	want := time.Date(2026, 9, 14, 9, 30, 0, 0, time.Local)
	if !got.Equal(want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestNextTriggerTomorrowWhenPast(t *testing.T) {
	now := time.Date(2026, 9, 14, 10, 0, 0, 0, time.Local)
	got := NextTrigger(now, 9, 30)
	want := time.Date(2026, 9, 15, 9, 30, 0, 0, time.Local)
	if !got.Equal(want) {
		t.Errorf("got %v want %v", got, want)
	}
}

func TestNextTriggerExactlyNow(t *testing.T) {
	now := time.Date(2026, 9, 14, 9, 30, 0, 0, time.Local)
	got := NextTrigger(now, 9, 30)
	if !got.Equal(time.Date(2026, 9, 15, 9, 30, 0, 0, time.Local)) {
		t.Errorf("exact trigger should roll to tomorrow, got %v", got)
	}
}

func TestShouldCatchUp(t *testing.T) {
	now := time.Date(2026, 9, 14, 11, 0, 0, 0, time.Local)
	settings := model.DefaultSettings()

	unsign := model.Credential{Status: model.StatusActive}
	if !ShouldCatchUp(unsign, now, settings) {
		t.Error("should catch up when past trigger and unsigned")
	}

	signed := model.Credential{Status: model.StatusActive, TodayDate: "2026-09-14", TodaySuccess: true}
	if ShouldCatchUp(signed, now, settings) {
		t.Error("should skip already checked-in account")
	}
}

func TestShouldCatchUpBeforeTrigger(t *testing.T) {
	now := time.Date(2026, 9, 14, 8, 0, 0, 0, time.Local)
	unsign := model.Credential{Status: model.StatusActive}
	if ShouldCatchUp(unsign, now, model.DefaultSettings()) {
		t.Error("before trigger must not catch up")
	}
}

func TestShouldRetryLimit(t *testing.T) {
	now := time.Date(2026, 9, 14, 11, 0, 0, 0, time.Local)

	c := model.Credential{Status: model.StatusActive, TodayDate: "2026-09-14", TodayAttempts: 1}
	if !ShouldRetry(c, now) {
		t.Error("attempts=1 should retry")
	}
	c.TodayAttempts = MaxRetriesPerDay
	if ShouldRetry(c, now) {
		t.Error("at limit must not retry")
	}

	c.TodayAttempts = 1
	c.Status = model.StatusReloginRequired
	if ShouldRetry(c, now) {
		t.Error("relogin_required must not retry")
	}

	c.Status = model.StatusActive
	c.TodaySuccess = true
	if ShouldRetry(c, now) {
		t.Error("already successful today must not retry")
	}
}

func TestJitterRange(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 200; i++ {
		d := Jitter(rng)
		if d < 5*time.Second || d > 20*time.Second {
			t.Fatalf("jitter out of range: %v", d)
		}
	}
}
