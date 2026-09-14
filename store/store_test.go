package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hosea3000/workbuddy-checkin/model"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestSaveAndReloadCredential(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	cred := model.Credential{ID: "id1", UserID: "uid_acc", AccessToken: "secret-token-value"}
	if err := s.SaveCredential(cred); err != nil {
		t.Fatal(err)
	}
	// 文件权限 0600
	fi, err := os.Stat(filepath.Join(dir, credentialsFile))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("expected 0600, got %o", fi.Mode().Perm())
	}
	// 重新加载
	s2, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := s2.GetCredential("id1")
	if !ok || got.AccessToken != "secret-token-value" || got.CreatedAt == 0 {
		t.Errorf("reload mismatch: %+v ok=%v", got, ok)
	}
}

func TestSaveCredentialUpdatesInPlace(t *testing.T) {
	s := newTestStore(t)
	_ = s.SaveCredential(model.Credential{ID: "id1", Nickname: "a"})
	_ = s.SaveCredential(model.Credential{ID: "id1", Nickname: "b"})
	if got := s.ListCredentials(); len(got) != 1 || got[0].Nickname != "b" {
		t.Errorf("in-place update failed: %+v", got)
	}
}

func TestCorruptFileSelfHeals(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, credentialsFile), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := New(dir)
	if err != nil {
		t.Fatalf("corrupt file must not block startup: %v", err)
	}
	if len(s.ListCredentials()) != 0 {
		t.Error("expected empty credentials after corruption")
	}
	matches, _ := filepath.Glob(filepath.Join(dir, credentialsFile+".corrupt-*"))
	if len(matches) != 1 {
		t.Errorf("expected one .corrupt backup, got %v", matches)
	}
}

func TestSettingsDefaultsWhenMissing(t *testing.T) {
	s := newTestStore(t)
	got := s.GetSettings()
	if got != model.DefaultSettings() {
		t.Errorf("expected defaults, got %+v", got)
	}
}

func TestSettingsPartialFileFillsDefaults(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, settingsFile), []byte(`{"checkinHour":7,"checkinMinute":5}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := s.GetSettings()
	if got.CheckinHour != 7 || got.CheckinMinute != 5 {
		t.Errorf("explicit fields lost: %+v", got)
	}
	if !got.CatchUpOnStart || !got.MinimizeToTray {
		t.Errorf("missing fields should keep defaults: %+v", got)
	}
}

func TestSettingsPersist(t *testing.T) {
	dir := t.TempDir()
	s, _ := New(dir)
	custom := model.DefaultSettings()
	custom.CheckinHour = 21
	if err := s.SaveSettings(custom); err != nil {
		t.Fatal(err)
	}
	s2, _ := New(dir)
	if got := s2.GetSettings(); got.CheckinHour != 21 {
		t.Errorf("settings not persisted: %+v", got)
	}
}

func TestCrossDayInvalidation(t *testing.T) {
	now := time.Date(2026, 9, 14, 10, 0, 0, 0, time.Local)
	cred := model.Credential{TodayDate: "2026-09-14", TodaySuccess: true}
	if !cred.HasCheckedInToday(now) {
		t.Error("same-day success should count")
	}
	nextDay := now.Add(24 * time.Hour)
	if cred.HasCheckedInToday(nextDay) {
		t.Error("previous-day success must be invalidated")
	}
}
